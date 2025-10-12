# Icons

Generate icons using Tauri's icon generator:

```bash
npm install -g @tauri-apps/cli
tauri icon path/to/your-icon.png
```

This will generate all required icon sizes:
- 32x32.png
- 128x128.png
- 128x128@2x.png
- icon.icns (macOS)
- icon.ico (Windows)
- icon.png (Linux)

For now, placeholder icons will be created during the build process.

You can also generate icons manually at: https://tauri.app/v1/guides/features/icons/
