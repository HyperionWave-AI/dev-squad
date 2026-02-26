#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DESKTOP_APP_DIR="$ROOT_DIR/desktop-app"
TAURI_MANIFEST="$DESKTOP_APP_DIR/src-tauri/Cargo.toml"
TAURI_BINARIES_DIR="$ROOT_DIR/desktop-app/src-tauri/binaries"
BUILD_NATIVE_SCRIPT="$ROOT_DIR/build-native.sh"

MODE="dev"
PLATFORMS_RAW=""

usage() {
  cat <<'EOF'
Usage:
  scripts/desktop.sh [--mode dev|build|install] [--platform <platform>] [--platforms "<p1 p2 ...>"]

Platform aliases:
  macos-arm64     darwin-arm64     aarch64-apple-darwin
  macos-amd64     darwin-amd64     x86_64-apple-darwin
  linux-amd64     x86_64-unknown-linux-gnu
  linux-arm64     aarch64-unknown-linux-gnu
  windows-amd64   x86_64-pc-windows-msvc

Examples:
  scripts/desktop.sh --mode dev
  scripts/desktop.sh --mode build --platform macos-arm64
  scripts/desktop.sh --mode build --platforms "macos-arm64 windows-amd64 linux-amd64"
  scripts/desktop.sh --mode install

Install notes:
  - Set INSTALL_DEST to override install destination.
  - install mode accepts a single platform only (defaults to host).
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --mode)
      MODE="$2"
      shift 2
      ;;
    --platform)
      PLATFORMS_RAW="${PLATFORMS_RAW} $2"
      shift 2
      ;;
    --platforms)
      PLATFORMS_RAW="${PLATFORMS_RAW} $2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown option: $1" >&2
      usage
      exit 1
      ;;
  esac
done

if [[ "$MODE" != "dev" && "$MODE" != "build" && "$MODE" != "install" ]]; then
  echo "Invalid mode: $MODE. Use dev, build, or install." >&2
  exit 1
fi

if [[ ! -f "$TAURI_MANIFEST" ]]; then
  echo "Tauri manifest not found at: $TAURI_MANIFEST" >&2
  exit 1
fi

if [[ ! -x "$BUILD_NATIVE_SCRIPT" ]]; then
  echo "Build script not executable: $BUILD_NATIVE_SCRIPT" >&2
  exit 1
fi

mkdir -p "$TAURI_BINARIES_DIR"

map_platform() {
  local platform="$1"
  case "$platform" in
    macos-arm64|darwin-arm64|aarch64-apple-darwin)
      echo "aarch64-apple-darwin|darwin-arm64"
      ;;
    macos-amd64|darwin-amd64|x86_64-apple-darwin)
      echo "x86_64-apple-darwin|darwin-amd64"
      ;;
    linux-amd64|x86_64-unknown-linux-gnu)
      echo "x86_64-unknown-linux-gnu|linux-amd64"
      ;;
    linux-arm64|aarch64-unknown-linux-gnu)
      echo "aarch64-unknown-linux-gnu|linux-arm64"
      ;;
    windows-amd64|x86_64-pc-windows-msvc)
      echo "x86_64-pc-windows-msvc|windows-amd64"
      ;;
    *)
      return 1
      ;;
  esac
}

detect_host_target() {
  rustc -vV | awk '/^host:/ {print $2}'
}

build_sidecar_for_target() {
  local input_platform="$1"
  local mapping
  local rust_target
  local go_platform
  local built_binary
  local sidecar_dest

  if ! mapping="$(map_platform "$input_platform")"; then
    echo "Unsupported platform: $input_platform" >&2
    exit 1
  fi

  IFS='|' read -r rust_target go_platform <<<"$mapping"

  echo ""
  echo "=== Building Hyper sidecar for $rust_target ==="
  "$BUILD_NATIVE_SCRIPT" --platform "$go_platform"

  built_binary="$ROOT_DIR/bin/hyper"
  if [[ "$go_platform" == windows-* ]]; then
    built_binary="$ROOT_DIR/bin/hyper.exe"
  fi

  if [[ ! -f "$built_binary" ]]; then
    echo "Expected build output not found: $built_binary" >&2
    exit 1
  fi

  sidecar_dest="$TAURI_BINARIES_DIR/hyper-sidecar-$rust_target"
  if [[ "$rust_target" == *windows* ]]; then
    sidecar_dest="${sidecar_dest}.exe"
  fi

  cp "$built_binary" "$sidecar_dest"
  if [[ "$rust_target" != *windows* ]]; then
    chmod +x "$sidecar_dest"
  fi

  echo "Sidecar ready: $sidecar_dest"
}

run_dev() {
  local host_target
  host_target="$(detect_host_target)"

  if [[ -z "$host_target" ]]; then
    echo "Failed to detect host Rust target." >&2
    exit 1
  fi

  build_sidecar_for_target "$host_target"

  echo ""
  echo "=== Launching Tauri desktop app (dev) ==="
  (
    cd "$DESKTOP_APP_DIR"
    cargo tauri dev
  )
}

run_build() {
  local -a raw_items
  local -a targets
  local host_target
  local item
  local mapping
  local rust_target

  read -r -a raw_items <<<"${PLATFORMS_RAW//,/ }"

  if [[ ${#raw_items[@]} -eq 0 ]]; then
    host_target="$(detect_host_target)"
    raw_items=("$host_target")
  fi

  for item in "${raw_items[@]}"; do
    if [[ -z "$item" ]]; then
      continue
    fi
    if ! mapping="$(map_platform "$item")"; then
      echo "Unsupported platform: $item" >&2
      exit 1
    fi
    IFS='|' read -r rust_target _ <<<"$mapping"
    targets+=("$rust_target")
  done

  if [[ ${#targets[@]} -eq 0 ]]; then
    echo "No valid platforms provided." >&2
    exit 1
  fi

  for rust_target in "${targets[@]}"; do
    ensure_bundle_for_target "$rust_target"
  done
}

ensure_bundle_for_target() {
  local rust_target="$1"
  build_sidecar_for_target "$rust_target"

  echo ""
  echo "=== Building desktop bundle for $rust_target ==="
  rustup target add "$rust_target"
  (
    cd "$DESKTOP_APP_DIR"
    cargo tauri build --target "$rust_target"
  )
}

install_bundle_for_target() {
  local rust_target="$1"

  case "$rust_target" in
    *apple-darwin)
      local app_path="$DESKTOP_APP_DIR/src-tauri/target/$rust_target/release/bundle/macos/Hyper Desktop.app"
      local dest_dir="${INSTALL_DEST:-}"

      if [[ ! -d "$app_path" ]]; then
        ensure_bundle_for_target "$rust_target"
      fi

      if [[ -z "$dest_dir" ]]; then
        if [[ -w "/Applications" ]]; then
          dest_dir="/Applications"
        else
          dest_dir="$HOME/Applications"
        fi
      fi

      mkdir -p "$dest_dir"
      rm -rf "$dest_dir/Hyper Desktop.app"
      cp -R "$app_path" "$dest_dir/"
      echo ""
      echo "Installed: $dest_dir/Hyper Desktop.app"
      ;;

    *unknown-linux-gnu)
      local appimage_dir="$DESKTOP_APP_DIR/src-tauri/target/$rust_target/release/bundle/appimage"
      local appimage_file=""
      local install_path="${INSTALL_DEST:-$HOME/.local/bin/hyper-desktop.AppImage}"

      if [[ -d "$appimage_dir" ]]; then
        appimage_file="$(find "$appimage_dir" -maxdepth 1 -type f -name '*.AppImage' | head -n 1)"
      fi

      if [[ -z "$appimage_file" ]]; then
        ensure_bundle_for_target "$rust_target"
        appimage_file="$(find "$appimage_dir" -maxdepth 1 -type f -name '*.AppImage' | head -n 1)"
      fi

      if [[ -z "$appimage_file" ]]; then
        echo "Failed to locate AppImage bundle for $rust_target" >&2
        exit 1
      fi

      mkdir -p "$(dirname "$install_path")"
      cp "$appimage_file" "$install_path"
      chmod +x "$install_path"
      echo ""
      echo "Installed AppImage: $install_path"
      ;;

    *windows*)
      local msi_dir="$DESKTOP_APP_DIR/src-tauri/target/$rust_target/release/bundle/msi"
      local msi_file=""

      if [[ -d "$msi_dir" ]]; then
        msi_file="$(find "$msi_dir" -maxdepth 1 -type f -name '*.msi' | head -n 1)"
      fi

      if [[ -z "$msi_file" ]]; then
        ensure_bundle_for_target "$rust_target"
        msi_file="$(find "$msi_dir" -maxdepth 1 -type f -name '*.msi' | head -n 1)"
      fi

      if [[ -z "$msi_file" ]]; then
        echo "Failed to locate MSI bundle for $rust_target" >&2
        exit 1
      fi

      if command -v msiexec >/dev/null 2>&1; then
        echo ""
        echo "Installing MSI with msiexec..."
        msiexec /i "$msi_file"
      else
        echo ""
        echo "MSI ready (manual install): $msi_file"
      fi
      ;;

    *)
      echo "Unsupported install target: $rust_target" >&2
      exit 1
      ;;
  esac
}

run_install() {
  local -a raw_items
  local -a targets
  local host_target
  local item
  local mapping
  local rust_target

  read -r -a raw_items <<<"${PLATFORMS_RAW//,/ }"

  if [[ ${#raw_items[@]} -eq 0 ]]; then
    host_target="$(detect_host_target)"
    raw_items=("$host_target")
  fi

  for item in "${raw_items[@]}"; do
    if [[ -z "$item" ]]; then
      continue
    fi
    if ! mapping="$(map_platform "$item")"; then
      echo "Unsupported platform: $item" >&2
      exit 1
    fi
    IFS='|' read -r rust_target _ <<<"$mapping"
    targets+=("$rust_target")
  done

  if [[ ${#targets[@]} -ne 1 ]]; then
    echo "install mode accepts exactly one platform. Use --platform or leave empty for host." >&2
    exit 1
  fi

  install_bundle_for_target "${targets[0]}"
}

if [[ "$MODE" == "dev" ]]; then
  run_dev
elif [[ "$MODE" == "build" ]]; then
  run_build
else
  run_install
fi
