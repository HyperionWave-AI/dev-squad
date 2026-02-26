use std::env;
use std::fs;
use std::path::PathBuf;

fn ensure_sidecar_placeholder() {
    let manifest_dir =
        PathBuf::from(env::var("CARGO_MANIFEST_DIR").expect("missing CARGO_MANIFEST_DIR"));
    let target = env::var("TARGET").expect("missing TARGET");

    let mut sidecar_path = manifest_dir
        .join("binaries")
        .join(format!("hyper-sidecar-{target}"));

    if target.contains("windows") {
        sidecar_path.set_extension("exe");
    }

    if sidecar_path.exists() {
        return;
    }

    if let Some(parent) = sidecar_path.parent() {
        let _ = fs::create_dir_all(parent);
    }

    let placeholder = if target.contains("windows") {
        b"".as_slice()
    } else {
        b"#!/bin/sh\nexit 0\n".as_slice()
    };

    if fs::write(&sidecar_path, placeholder).is_ok() {
        #[cfg(unix)]
        {
            use std::os::unix::fs::PermissionsExt;
            let _ = fs::set_permissions(&sidecar_path, fs::Permissions::from_mode(0o755));
        }
        println!(
            "cargo:warning=Created placeholder sidecar at {}",
            sidecar_path.display()
        );
    }
}

fn main() {
    ensure_sidecar_placeholder();
    tauri_build::build()
}
