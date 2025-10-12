//go:build unix || darwin

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

// setupSignalHandler creates a context that cancels on interrupt signals
func setupSignalHandler() (context.Context, func()) {
	// Unix/macOS: handle both SIGINT and SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	return ctx, stop
}
