//go:build windows

package main

import (
	"context"
	"os"
	"os/signal"
)

// setupSignalHandler creates a context that cancels on interrupt signals
func setupSignalHandler() (context.Context, func()) {
	// Windows: only handle os.Interrupt (Ctrl+C)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	return ctx, stop
}
