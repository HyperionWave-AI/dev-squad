use std::env;
use std::fs;
use std::io;
use std::net::{SocketAddr, TcpStream};
use std::path::{Path, PathBuf};
use std::sync::Mutex;
use std::thread;
use std::time::{Duration, Instant};

use tauri::{AppHandle, Manager, RunEvent};
use tauri_plugin_shell::process::{CommandChild, CommandEvent};
use tauri_plugin_shell::ShellExt;

const SIDECAR_NAME: &str = "hyper-sidecar";
const DEFAULT_HTTP_PORT: u16 = 8080;
const BACKEND_READY_TIMEOUT: Duration = Duration::from_secs(45);
const BACKEND_POLL_INTERVAL: Duration = Duration::from_millis(500);

struct BackendProcessState {
    child: Mutex<Option<CommandChild>>,
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    let app = tauri::Builder::default()
        .plugin(
            tauri_plugin_log::Builder::default()
                .level(log::LevelFilter::Info)
                .build(),
        )
        .plugin(tauri_plugin_shell::init())
        .setup(|app| {
            let config_path = resolve_config_path();
            let http_port = resolve_http_port(config_path.as_deref());

            if let Some(path) = &config_path {
                log::info!("Using Hyper config file: {}", path.display());
            } else {
                log::warn!("No .env.hyper found via HYPER_CONFIG, cwd, or executable directory");
            }
            log::info!("Waiting for Hyper backend on 127.0.0.1:{http_port}");

            let child = spawn_backend(app.handle(), config_path.as_deref())?;
            app.manage(BackendProcessState {
                child: Mutex::new(Some(child)),
            });

            wait_for_backend_and_redirect(app.handle().clone(), http_port);
            Ok(())
        })
        .build(tauri::generate_context!())
        .expect("failed to build tauri app");

    app.run(|app_handle, event| {
        if matches!(event, RunEvent::Exit | RunEvent::ExitRequested { .. }) {
            stop_backend_process(app_handle);
        }
    });
}

fn resolve_config_path() -> Option<PathBuf> {
    if let Ok(explicit_path) = env::var("HYPER_CONFIG") {
        let candidate = PathBuf::from(explicit_path);
        if candidate.is_file() {
            return Some(candidate);
        }
    }

    let mut search_dirs = Vec::new();

    if let Ok(cwd) = env::current_dir() {
        search_dirs.push(cwd);
    }

    if let Ok(current_exe) = env::current_exe() {
        if let Some(exe_dir) = current_exe.parent() {
            search_dirs.push(exe_dir.to_path_buf());
        }
    }

    for dir in search_dirs {
        for file_name in [".env.hyper.hot", ".env.hyper"] {
            let candidate = dir.join(file_name);
            if candidate.is_file() {
                return Some(candidate);
            }
        }
    }

    None
}

fn resolve_http_port(config_path: Option<&Path>) -> u16 {
    if let Ok(value) = env::var("HTTP_PORT") {
        if let Ok(port) = parse_port(&value) {
            return port;
        }
    }

    if let Some(path) = config_path {
        if let Ok(contents) = fs::read_to_string(path) {
            for raw_line in contents.lines() {
                let line = raw_line.trim();
                if line.is_empty() || line.starts_with('#') {
                    continue;
                }

                if let Some((key, value)) = line.split_once('=') {
                    if key.trim() == "HTTP_PORT" {
                        if let Ok(port) = parse_port(value) {
                            return port;
                        }
                    }
                }
            }
        }
    }

    DEFAULT_HTTP_PORT
}

fn parse_port(value: &str) -> Result<u16, std::num::ParseIntError> {
    let trimmed = value.trim();
    let unquoted = if trimmed.len() >= 2
        && ((trimmed.starts_with('"') && trimmed.ends_with('"'))
            || (trimmed.starts_with('\'') && trimmed.ends_with('\'')))
    {
        &trimmed[1..trimmed.len() - 1]
    } else {
        trimmed
    };

    unquoted.parse::<u16>()
}

fn spawn_backend(app: &AppHandle, config_path: Option<&Path>) -> io::Result<CommandChild> {
    let mut command = app
        .shell()
        .sidecar(SIDECAR_NAME)
        .map_err(|err| io::Error::other(format!("failed to prepare sidecar: {err}")))?;

    command = command.args(["--mode", "http"]);

    if let Some(path) = config_path {
        let config_arg = path.to_string_lossy().into_owned();
        command = command.args(["--config", config_arg.as_str()]);
    }

    let (mut receiver, child) = command
        .spawn()
        .map_err(|err| io::Error::other(format!("failed to spawn sidecar: {err}")))?;

    tauri::async_runtime::spawn(async move {
        while let Some(event) = receiver.recv().await {
            match event {
                CommandEvent::Stdout(data) => {
                    let line = String::from_utf8_lossy(&data).trim().to_string();
                    if !line.is_empty() {
                        log::info!("[hyper] {line}");
                    }
                }
                CommandEvent::Stderr(data) => {
                    let line = String::from_utf8_lossy(&data).trim().to_string();
                    if !line.is_empty() {
                        log::warn!("[hyper] {line}");
                    }
                }
                CommandEvent::Error(err) => {
                    log::error!("[hyper] sidecar error: {err}");
                }
                CommandEvent::Terminated(payload) => {
                    log::warn!("[hyper] sidecar terminated: {payload:?}");
                }
                _ => {}
            }
        }
    });

    Ok(child)
}

fn wait_for_backend_and_redirect(app: AppHandle, port: u16) {
    thread::spawn(move || {
        let ui_url = format!("http://127.0.0.1:{port}/ui");
        let start = Instant::now();

        while start.elapsed() < BACKEND_READY_TIMEOUT {
            if is_backend_listening(port) {
                log::info!("Hyper backend is reachable at {ui_url}");
                redirect_main_window(&app, &ui_url);
                return;
            }

            thread::sleep(BACKEND_POLL_INTERVAL);
        }

        let message = format!(
      "Hyper backend did not become ready within {} seconds on port {}.\nCheck sidecar logs for details.",
      BACKEND_READY_TIMEOUT.as_secs(),
      port
    );
        show_startup_error(&app, &message);
    });
}

fn is_backend_listening(port: u16) -> bool {
    let address = SocketAddr::from(([127, 0, 0, 1], port));
    TcpStream::connect_timeout(&address, Duration::from_millis(250)).is_ok()
}

fn redirect_main_window(app: &AppHandle, ui_url: &str) {
    let Some(window) = app.get_webview_window("main") else {
        log::error!("main window not found");
        return;
    };

    let encoded_url = match serde_json::to_string(ui_url) {
        Ok(value) => value,
        Err(err) => {
            log::error!("failed to encode ui URL: {err}");
            return;
        }
    };

    let script = format!("window.location.replace({encoded_url});");
    if let Err(err) = window.eval(&script) {
        log::error!("failed to redirect main window: {err}");
    }
}

fn show_startup_error(app: &AppHandle, message: &str) {
    let Some(window) = app.get_webview_window("main") else {
        log::error!("main window not found while showing startup error");
        return;
    };

    let encoded_message = match serde_json::to_string(message) {
        Ok(value) => value,
        Err(err) => {
            log::error!("failed to encode startup error message: {err}");
            return;
        }
    };

    let script = format!(
        r##"(function() {{
      const msg = {encoded_message};
      document.body.innerHTML = "";
      document.body.style.margin = "0";
      document.body.style.background = "#020617";
      document.body.style.color = "#e2e8f0";
      document.body.style.fontFamily = "monospace";
      const wrap = document.createElement("main");
      wrap.style.padding = "24px";
      wrap.style.maxWidth = "900px";
      const title = document.createElement("h1");
      title.textContent = "Hyper backend failed to start";
      title.style.fontSize = "18px";
      const detail = document.createElement("pre");
      detail.textContent = msg;
      detail.style.whiteSpace = "pre-wrap";
      detail.style.lineHeight = "1.5";
      detail.style.background = "#0f172a";
      detail.style.padding = "12px";
      detail.style.border = "1px solid #334155";
      detail.style.borderRadius = "8px";
      wrap.appendChild(title);
      wrap.appendChild(detail);
      document.body.appendChild(wrap);
    }})();"##
    );

    if let Err(err) = window.eval(&script) {
        log::error!("failed to show startup error in window: {err}");
    }
}

fn stop_backend_process(app_handle: &AppHandle) {
    let Some(state) = app_handle.try_state::<BackendProcessState>() else {
        return;
    };

    let Ok(mut guard) = state.child.lock() else {
        return;
    };

    let Some(child) = guard.take() else {
        return;
    };

    if let Err(err) = child.kill() {
        log::warn!("failed to stop hyper sidecar: {err}");
    } else {
        log::info!("stopped hyper sidecar");
    }
}
