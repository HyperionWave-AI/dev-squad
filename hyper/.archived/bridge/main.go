package main

import (
	"flag"
	"fmt"

	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

func main() {
	// Parse command-line flags
	configPath := flag.String("config", "", "Path to config file (default: .env.hyper)")
	flag.Parse()

	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Load environment variables
	if *configPath != "" {
		// Load from custom config path
		if err := godotenv.Load(*configPath); err != nil {
			logger.Fatal("Failed to load config from custom path",
				zap.String("path", *configPath),
				zap.Error(err))
		}
		logger.Info("Loaded configuration from custom path", zap.String("path", *configPath))
	} else {
		// Load from default location
		if err := godotenv.Load(".env.hyper"); err != nil {
			logger.Warn("Could not load .env.hyper", zap.Error(err))
		}
	}

	logger.Info("Starting Hyper HTTP Bridge")
	// TODO: Implement HTTP bridge logic from hyper/internal/bridge/
	fmt.Println("Bridge server placeholder - use MCP server with TRANSPORT_MODE=http for now")
}
