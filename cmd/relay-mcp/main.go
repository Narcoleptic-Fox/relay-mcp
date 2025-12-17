package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

    "github.com/joho/godotenv"
    "github.com/Narcoleptic-Fox/relay-mcp/internal/config"
    "github.com/Narcoleptic-Fox/relay-mcp/internal/providers"
    "github.com/Narcoleptic-Fox/relay-mcp/internal/server"
)

func main() {
    // Load .env file if present
    _ = godotenv.Load()

    // Initialize structured logging
    logLevel := slog.LevelInfo
    if os.Getenv("LOG_LEVEL") == "debug" {
        logLevel = slog.LevelDebug
    }

    logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
        Level: logLevel,
    }))
    slog.SetDefault(logger)

    // Load configuration
    cfg, err := config.Load()
    if err != nil {
        slog.Error("failed to load configuration", "error", err)
        os.Exit(1)
    }

    // Initialize provider registry
    registry := providers.NewRegistry(cfg)
    if err := registry.Initialize(); err != nil {
        slog.Error("failed to initialize providers", "error", err)
        os.Exit(1)
    }

    // Create MCP server
    srv := server.New(cfg, registry)

    // Setup graceful shutdown
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

    go func() {
        <-sigCh
        slog.Info("shutting down...")
        cancel()
    }()

    // Run server (blocks on stdio)
    slog.Info("starting RELAY MCP server", "version", cfg.Version)
    if err := srv.Run(ctx); err != nil {
        slog.Error("server error", "error", err)
        os.Exit(1)
    }
}
