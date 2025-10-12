# Hyperion Coordinator Desktop App

A native desktop application built with Tauri that wraps the Hyperion Coordinator binary.

## Features

- **Native Desktop App** - macOS, Windows, and Linux support
- **Auto-Start Server** - Automatically launches the Hyperion binary
- **Embedded UI** - Uses the embedded React UI from the binary
- **Small Bundle** - ~15 MB (vs 120 MB with Electron)
- **Native Performance** - Uses system webview, no Chromium overhead
- **Direct MCP Access** - Call MCP tools directly from React via Tauri commands

## Quick Start

### Prerequisites

1. **Rust** - Install from https://rustup.rs/
2. **Node.js** - Version 18+ (for npm)
3. **Hyperion Binary** - Built via `make native`

### Development

```bash
# From project root
make desktop

# Or manually
cd desktop-app
npm install
npm run dev
```

This will:
1. Build the native binary if not already built
2. Start the desktop app in development mode
3. Launch the Hyperion server automatically
4. Open the UI in a native window

### Build for Distribution

```bash
# From project root
make desktop-build

# Or manually
cd desktop-app
npm install
npm run build
```

**Output locations:**
- **macOS**: `src-tauri/target/release/bundle/dmg/Hyperion Coordinator.dmg`
- **macOS**: `src-tauri/target/release/bundle/macos/Hyperion Coordinator.app`
- **Windows**: `src-tauri/target/release/bundle/msi/Hyperion Coordinator.msi`
- **Linux**: `src-tauri/target/release/bundle/appimage/hyperion-coordinator.AppImage`

## Configuration

The desktop app uses the same `.env.native` configuration as the binary:

1. **Bundled config** - Place `.env.native` in the app's resource directory
2. **User config** - Or use system environment variables

When building for distribution, you can include `.env.native` by copying it to:
- **macOS**: `Hyperion Coordinator.app/Contents/Resources/.env.native`
- **Windows**: Same directory as the `.exe`
- **Linux**: Same directory as the `.AppImage`

## Architecture

```
┌─────────────────────────────────────────┐
│        Tauri Desktop Window             │
│  ┌───────────────────────────────────┐  │
│  │   Webview (System Browser Engine)│  │
│  │   Loads: http://localhost:7095/ui│  │
│  └───────────────────────────────────┘  │
│         ▲                                │
│         │ HTTP                           │
│         │                                │
│  ┌──────▼──────────────────────────┐    │
│  │  Hyperion Binary (subprocess)   │    │
│  │  • HTTP Server (port 7095)      │    │
│  │  • Embedded UI                  │    │
│  │  • MCP Server                   │    │
│  └─────────────────────────────────┘    │
└─────────────────────────────────────────┘
```

**How it works:**

1. Tauri app starts and launches `hyper` binary as subprocess
2. Binary starts HTTP server on port 7095 with embedded UI
3. Tauri webview loads `http://localhost:7095/ui`
4. UI communicates with backend via HTTP API
5. On app exit, Tauri gracefully shuts down the binary

## Window Behavior

When you close the window, the app exits completely and automatically stops the Hyperion server subprocess.

## Platform-Specific Builds

### macOS

```bash
# Universal binary (Intel + Apple Silicon)
npm run build:mac

# Intel only
npm run build:mac-intel

# Apple Silicon only
npm run build:mac-arm
```

**Output:**
- `.dmg` - Disk image for distribution
- `.app` - Application bundle

**To sign and notarize:**
1. Get Apple Developer account
2. Set `APPLE_CERTIFICATE` and `APPLE_ID` in environment
3. Tauri will automatically sign and notarize

### Windows

```bash
npm run build:windows
```

**Output:**
- `.msi` - Windows installer
- `.exe` - Portable executable

**To sign:**
1. Get code signing certificate
2. Set `WINDOWS_CERTIFICATE` in environment
3. Tauri will automatically sign

### Linux

```bash
npm run build:linux
```

**Output:**
- `.AppImage` - Portable app bundle
- `.deb` - Debian/Ubuntu package
- `.rpm` - Red Hat/Fedora package (if rpmbuild installed)

## Development Tips

### Hot Reload

In development mode, changes to:
- **Rust code** - Requires restart (`npm run dev`)
- **HTML/JS** - Auto-reloads when you save
- **Tauri config** - Requires restart

### Debugging

**Enable DevTools:**
```rust
// In src-tauri/src/main.rs
.invoke_handler(tauri::generate_handler![])
.setup(|app| {
    #[cfg(debug_assertions)]
    {
        let window = app.get_window("main").unwrap();
        window.open_devtools();
    }
    Ok(())
})
```

**View Rust logs:**
```bash
# Development mode shows all logs in terminal
npm run dev
```

**View binary logs:**
- Check console output from the Hyperion binary
- Binary logs appear in the Tauri dev console

### Testing

```bash
# Run Rust tests
cd src-tauri
cargo test

# Build and test the app
npm run build
# Then manually test the built app
```

## Troubleshooting

### "Binary not found" error

**Solution:** Build the native binary first:
```bash
cd ..
make native
```

### "Failed to start Hyperion server"

**Causes:**
1. Port 7095 is already in use
2. `.env.native` is not configured
3. MongoDB URI is invalid

**Solution:**
1. Check if port 7095 is free: `lsof -i :7095`
2. Configure `.env.native` with valid MongoDB URI
3. Test binary directly: `./bin/hyper --mode=http`

### Window doesn't load

**Solution:**
1. Check if binary is running: `ps aux | grep hyper`
2. Check binary logs in console
3. Verify `http://localhost:7095/ui` works in browser
4. Check firewall isn't blocking port 7095

### Build fails on macOS

**Error:** "xcrun: error: invalid active developer path"

**Solution:**
```bash
xcode-select --install
```

### Build fails on Linux

**Error:** Missing dependencies

**Solution:**
```bash
# Ubuntu/Debian
sudo apt install libwebkit2gtk-4.0-dev \
    build-essential \
    curl \
    wget \
    libssl-dev \
    libgtk-3-dev \
    libayatana-appindicator3-dev \
    librsvg2-dev

# Fedora
sudo dnf install webkit2gtk3-devel \
    openssl-devel \
    curl \
    wget \
    libappindicator-gtk3 \
    librsvg2-devel

# Arch
sudo pacman -S webkit2gtk \
    base-devel \
    curl \
    wget \
    openssl \
    appmenu-gtk-module \
    gtk3 \
    libappindicator-gtk3 \
    librsvg
```

## File Structure

```
desktop-app/
├── index.html              # Loading screen + iframe loader
├── package.json            # NPM scripts and dependencies
├── src-tauri/              # Tauri Rust backend
│   ├── Cargo.toml          # Rust dependencies
│   ├── tauri.conf.json     # Tauri configuration
│   ├── build.rs            # Build script
│   ├── icons/              # App icons (all platforms)
│   └── src/
│       └── main.rs         # Main Rust code (launches binary)
└── README.md               # This file
```

## Publishing

### macOS App Store

1. Update `tauri.conf.json` with App Store identifiers
2. Enable sandbox and capabilities
3. Build with `npm run build:mac`
4. Submit via App Store Connect

### Windows Store

1. Generate MSIX package: `npm run build:windows`
2. Submit via Microsoft Partner Center

### Linux Repositories

1. **Flatpak**: Create Flatpak manifest
2. **Snap**: Create snapcraft.yaml
3. **AUR**: Create PKGBUILD for Arch User Repository

## Advanced Configuration

### Custom Window Size

Edit `tauri.conf.json`:
```json
{
  "app": {
    "windows": [{
      "width": 1600,
      "height": 1000,
      "minWidth": 1000,
      "minHeight": 700
    }]
  }
}
```

### Custom System Tray Icon

1. Replace icons in `src-tauri/icons/`
2. Use Tauri icon generator:
   ```bash
   tauri icon path/to/icon.png
   ```

### Bundle .env.native with App

Edit `tauri.conf.json`:
```json
{
  "bundle": {
    "resources": [
      "../bin/hyper",
      "../.env.native"
    ]
  }
}
```

### Auto-Update

Add `tauri-plugin-updater`:
```bash
cd src-tauri
cargo add tauri-plugin-updater
```

Configure in `tauri.conf.json`:
```json
{
  "plugins": {
    "updater": {
      "active": true,
      "endpoints": ["https://your-server.com/updates"],
      "dialog": true,
      "pubkey": "YOUR_PUBLIC_KEY"
    }
  }
}
```

## License

Part of the Hyperion AI Platform. See LICENSE file for details.

## Support

- **Issues**: https://github.com/your-org/hyperion/issues
- **Docs**: See main README.md and README-NATIVE.md
- **Tauri Docs**: https://tauri.app/v1/guides/
