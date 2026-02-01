package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/sglre6355/sgrbot/internal/bot"
	_ "github.com/sglre6355/sgrbot/internal/modules/test"
)

// version is set at build time via ldflags:
// go build -ldflags "-X main.version=1.0.0" ./cmd/sgrbot
var version = "dev"

func main() {
	// Configure JSON logging
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	slog.Info("starting sgrbot", "version", version)

	// Load configuration
	cfg, err := bot.LoadConfig()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Create and configure bot
	b := bot.NewBot(cfg)
	b.LoadModules()

	// Start bot
	if err := b.Start(); err != nil {
		slog.Error("failed to start bot", "error", err)
		os.Exit(1)
	}

	// Wait for shutdown signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	slog.Info("received termination signal, shutting down")
	if err := b.Stop(); err != nil {
		slog.Error("failed to shutdown", "error", err)
	}

	slog.Info("completed bot shutdown")
	os.Exit(0)
}
