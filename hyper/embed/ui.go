// +build !dev

package embed

import (
	"embed"
	"io/fs"
	"net/http"
)

// Embed the UI static files at build time
// This allows us to distribute a single binary with the UI built-in
// Path is relative to this file (hyper/embed/)
// Points to: ui2/dist (via symlink)
//
//go:embed all:ui2/dist
var UI embed.FS

// GetUIFileSystem returns an http.FileSystem for the embedded UI
// If embedded files are available, they are served directly from the binary
// This enables single-binary deployment without external file dependencies
func GetUIFileSystem() (http.FileSystem, error) {
	// The embedded FS has structure: ui2/dist/index.html, ui2/dist/assets/...
	// Strip the "ui2/dist" prefix to serve files from root
	stripped, err := fs.Sub(UI, "ui2/dist")
	if err != nil {
		return nil, err
	}
	return http.FS(stripped), nil
}

// HasUI checks if UI files were embedded at build time
func HasUI() bool {
	// Try to read index.html from embedded files
	_, err := UI.ReadFile("ui2/dist/index.html")
	return err == nil
}
