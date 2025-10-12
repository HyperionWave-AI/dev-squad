// Prevents additional console window on Windows in release, DO NOT REMOVE!!
#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

use std::process::{Child, Command};
use std::sync::Mutex;
use std::env;
use std::path::PathBuf;
use tauri::{Manager, State};

struct HyperionProcess(Mutex<Option<Child>>);

fn get_hyper_binary_path(app_handle: &tauri::AppHandle) -> Result<PathBuf, String> {
    // In development, use the binary from bin/
    if cfg!(debug_assertions) {
        let mut path = env::current_dir().map_err(|e| e.to_string())?;
        path.pop(); // Go up from desktop-app/
        path.push("bin");
        path.push("hyper");
        return Ok(path);
    }

    // In production, the binary is bundled with the app
    let resource_path = app_handle
        .path()
        .resource_dir()
        .map_err(|e| e.to_string())?;

    let binary_name = if cfg!(target_os = "windows") {
        "hyper.exe"
    } else {
        "hyper"
    };

    Ok(resource_path.join(binary_name))
}

fn get_env_path(app_handle: &tauri::AppHandle) -> Result<PathBuf, String> {
    // In development, look for .env.hyper in bin/ directory
    if cfg!(debug_assertions) {
        let mut path = env::current_dir().map_err(|e| e.to_string())?;
        path.pop(); // Go up from desktop-app/
        path.push("bin");
        path.push(".env.hyper");
        return Ok(path);
    }

    // In production, look for .env.hyper in the resource directory
    let resource_path = app_handle
        .path()
        .resource_dir()
        .map_err(|e| e.to_string())?;

    Ok(resource_path.join(".env.hyper"))
}

fn start_hyperion_server(app_handle: &tauri::AppHandle) -> Result<Child, String> {
    let binary_path = get_hyper_binary_path(app_handle)?;

    if !binary_path.exists() {
        return Err(format!("Hyperion binary not found at {:?}", binary_path));
    }

    println!("Starting Hyperion server from: {:?}", binary_path);

    // Check if .env.hyper exists
    let env_path = get_env_path(app_handle).ok();

    let mut cmd = Command::new(&binary_path);
    cmd.arg("--mode=http");

    if let Some(env_file) = env_path {
        if env_file.exists() {
            println!("Using config file: {:?}", env_file);
            // The binary will find .env.hyper automatically in its directory
        } else {
            println!("Warning: .env.hyper not found at {:?}", env_file);
        }
    }

    let child = cmd
        .stdout(std::process::Stdio::inherit())
        .stderr(std::process::Stdio::inherit())
        .spawn()
        .map_err(|e| format!("Failed to start Hyperion server: {}", e))?;

    println!("Hyperion server started with PID: {}", child.id());

    Ok(child)
}

fn stop_hyperion_server(process: &Mutex<Option<Child>>) {
    let mut child_opt = process.lock().unwrap();
    if let Some(mut child) = child_opt.take() {
        println!("Stopping Hyperion server (PID: {})...", child.id());
        let _ = child.kill();
        let _ = child.wait();
        println!("Hyperion server stopped");
    }
}

#[tauri::command]
fn get_server_url() -> String {
    "http://localhost:7095/ui".to_string()
}

#[tauri::command]
async fn check_server_health() -> Result<String, String> {
    let client = reqwest::Client::new();
    match client.get("http://localhost:7095/health").send().await {
        Ok(response) => {
            if response.status().is_success() {
                Ok("healthy".to_string())
            } else {
                Err(format!("Server returned status: {}", response.status()))
            }
        }
        Err(e) => Err(format!("Health check failed: {}", e)),
    }
}

// MCP Tool Commands - Direct access to MCP tools from desktop app

#[tauri::command]
async fn call_mcp_tool(name: String, arguments: serde_json::Value) -> Result<serde_json::Value, String> {
    let client = reqwest::Client::new();

    let payload = serde_json::json!({
        "name": name,
        "arguments": arguments
    });

    match client
        .post("http://localhost:7095/api/mcp/tools/call")
        .json(&payload)
        .send()
        .await
    {
        Ok(response) => {
            if response.status().is_success() {
                response.json().await.map_err(|e| e.to_string())
            } else {
                Err(format!("MCP tool call failed: {}", response.status()))
            }
        }
        Err(e) => Err(format!("Failed to call MCP tool: {}", e)),
    }
}

#[tauri::command]
async fn create_human_task(prompt: String) -> Result<serde_json::Value, String> {
    call_mcp_tool("coordinator_create_human_task".to_string(), serde_json::json!({ "prompt": prompt })).await
}

#[tauri::command]
async fn create_agent_task(
    human_task_id: String,
    agent_name: String,
    role: String,
    context_summary: Option<String>,
    files_modified: Option<Vec<String>>,
    todos: Option<Vec<serde_json::Value>>,
) -> Result<serde_json::Value, String> {
    let mut args = serde_json::json!({
        "humanTaskId": human_task_id,
        "agentName": agent_name,
        "role": role
    });

    if let Some(summary) = context_summary {
        args["contextSummary"] = serde_json::Value::String(summary);
    }
    if let Some(files) = files_modified {
        args["filesModified"] = serde_json::json!(files);
    }
    if let Some(todos_list) = todos {
        args["todos"] = serde_json::json!(todos_list);
    }

    call_mcp_tool("coordinator_create_agent_task".to_string(), args).await
}

#[tauri::command]
async fn list_human_tasks() -> Result<serde_json::Value, String> {
    call_mcp_tool("coordinator_list_human_tasks".to_string(), serde_json::json!({})).await
}

#[tauri::command]
async fn list_agent_tasks(agent_name: Option<String>, human_task_id: Option<String>) -> Result<serde_json::Value, String> {
    let mut args = serde_json::json!({});

    if let Some(name) = agent_name {
        args["agentName"] = serde_json::Value::String(name);
    }
    if let Some(id) = human_task_id {
        args["humanTaskId"] = serde_json::Value::String(id);
    }

    call_mcp_tool("coordinator_list_agent_tasks".to_string(), args).await
}

#[tauri::command]
async fn update_task_status(task_id: String, status: String, notes: Option<String>) -> Result<serde_json::Value, String> {
    let mut args = serde_json::json!({
        "taskId": task_id,
        "status": status
    });

    if let Some(notes_text) = notes {
        args["notes"] = serde_json::Value::String(notes_text);
    }

    call_mcp_tool("coordinator_update_task_status".to_string(), args).await
}

#[tauri::command]
async fn upsert_knowledge(
    collection: String,
    text: String,
    metadata: Option<serde_json::Value>,
) -> Result<serde_json::Value, String> {
    let mut args = serde_json::json!({
        "collection": collection,
        "text": text
    });

    if let Some(meta) = metadata {
        args["metadata"] = meta;
    }

    call_mcp_tool("coordinator_upsert_knowledge".to_string(), args).await
}

#[tauri::command]
async fn query_knowledge(
    collection: String,
    query: String,
    limit: Option<i32>,
) -> Result<serde_json::Value, String> {
    let mut args = serde_json::json!({
        "collection": collection,
        "query": query
    });

    if let Some(lim) = limit {
        args["limit"] = serde_json::json!(lim);
    }

    call_mcp_tool("coordinator_query_knowledge".to_string(), args).await
}

fn main() {
    tauri::Builder::default()
        .manage(HyperionProcess(Mutex::new(None)))
        .setup(|app| {
            let app_handle = app.handle().clone();

            // In development mode, assume server is already running via make desktop
            // In production mode (release build), start the server ourselves
            if !cfg!(debug_assertions) {
                // Production mode - start Hyperion server
                match start_hyperion_server(&app_handle) {
                    Ok(child) => {
                        let process_state: State<HyperionProcess> = app.state();
                        *process_state.0.lock().unwrap() = Some(child);
                        println!("✓ Hyperion Coordinator started successfully");

                        // Give server a moment to start (5 seconds to allow for collection recreation if needed)
                        std::thread::sleep(std::time::Duration::from_secs(5));
                    }
                    Err(e) => {
                        eprintln!("Failed to start Hyperion server: {}", e);
                        eprintln!("Please ensure the binary is built: make native");
                        return Err(e.into());
                    }
                }
            } else {
                // Development mode - server should already be running
                println!("Development mode: expecting server to be running on http://localhost:7095");
            }

            Ok(())
        })
        .on_window_event(|_window, event| {
            // Allow normal window close behavior
            match event {
                tauri::WindowEvent::CloseRequested { .. } => {
                    // Window will close normally
                }
                _ => {}
            }
        })
        .invoke_handler(tauri::generate_handler![
            get_server_url,
            check_server_health,
            call_mcp_tool,
            create_human_task,
            create_agent_task,
            list_human_tasks,
            list_agent_tasks,
            update_task_status,
            upsert_knowledge,
            query_knowledge
        ])
        .build(tauri::generate_context!())
        .expect("error while running tauri application")
        .run(|app_handle, event| match event {
            tauri::RunEvent::Exit => {
                // Stop Hyperion server on app exit
                let process_state: State<HyperionProcess> = app_handle.state();
                stop_hyperion_server(&process_state.0);
            }
            _ => {}
        });
}
