# Hyper Desktop

Tauri desktop shell for Hyper.

The desktop app launches the `hyper` backend as a sidecar process, waits for it to be reachable on `http://127.0.0.1:<port>`, then loads the existing web UI from `/ui`.

## Local usage

```bash
make desktop
```

Build installers:

```bash
make desktop-build
make desktop-build PLATFORMS="macos-arm64 windows-amd64 linux-amd64"
```

## Config

- Set `HYPER_CONFIG` to force a config file path.
- If `HYPER_CONFIG` is not set, the app looks for `.env.hyper.hot` then `.env.hyper` in:
1. current working directory
2. executable directory

The backend reads the same configuration as `bin/hyper`.
