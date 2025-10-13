// +build dev

package embed

import (
	"embed"
	"io/fs"
	"net/http"
)

// In dev mode, UI is NOT embedded
// This file is compiled when -tags dev is used
// The UI will be served from Vite dev server instead
var UI embed.FS

// GetUIFileSystem returns an error in dev mode
// UI should be served from Vite dev server at localhost:5173
func GetUIFileSystem() (http.FileSystem, error) {
	// Return nil, will trigger hasEmbedded=false in main.go
	return nil, fs.ErrNotExist
}

// HasUI returns false in dev mode
// This triggers the Vite proxy in http_server.go
func HasUI() bool {
	return false
}
