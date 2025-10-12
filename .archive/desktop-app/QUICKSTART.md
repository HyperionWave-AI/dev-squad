# Desktop App Quick Start

Get the Hyperion Coordinator desktop app running in 5 minutes!

## Prerequisites

Install these one-time:

```bash
# 1. Install Rust
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh

# 2. Verify installations
rustc --version
node --version  # Should be 18+
```

## Run in Development

From the project root:

```bash
# Build native binary + run desktop app
make desktop
```

That's it! The app will:
1. Build the `hyper` binary (if needed)
2. Install npm dependencies
3. Launch the desktop app
4. Start the Hyperion server automatically
5. Load the UI in a native window

## Build for Distribution

```bash
# From project root
make desktop-build
```

**Find your app:**
- **macOS**: `desktop-app/src-tauri/target/release/bundle/dmg/`
- **Windows**: `desktop-app/src-tauri/target/release/bundle/msi/`
- **Linux**: `desktop-app/src-tauri/target/release/bundle/appimage/`

## Configuration

Before first run, configure MongoDB:

```bash
# 1. Copy template
cp .env.native .env.native

# 2. Edit with your settings
vim .env.native

# 3. Add your MongoDB URI
MONGODB_URI=mongodb+srv://user:pass@cluster.mongodb.net/coordinator_db
```

## Troubleshooting

### "Binary not found"

```bash
make native  # Build the binary first
```

### "Failed to start server"

Check if port 7095 is already in use:
```bash
lsof -i :7095
```

### Build errors on macOS

```bash
xcode-select --install
```

### Build errors on Linux

```bash
# Ubuntu/Debian
sudo apt install libwebkit2gtk-4.0-dev build-essential curl wget libssl-dev

# Fedora
sudo dnf install webkit2gtk3-devel openssl-devel

# Arch
sudo pacman -S webkit2gtk base-devel curl wget openssl
```

## Next Steps

- See [README.md](./README.md) for full documentation
- Configure system tray and auto-update
- Build for multiple platforms
- Publish to app stores

## Manual Build (Alternative)

If `make` commands don't work:

```bash
# 1. Build native binary
cd ..
./build-native.sh

# 2. Install desktop app dependencies
cd desktop-app
npm install

# 3. Run in development
npm run dev

# 4. Or build for production
npm run build
```

## Features

✅ Native desktop app (macOS, Windows, Linux)
✅ System tray icon
✅ Auto-starts Hyperion server
✅ Graceful shutdown
✅ Small bundle size (~15 MB)
✅ Native performance
✅ Uses system webview

Enjoy! 🚀
