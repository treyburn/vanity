package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/alecthomas/kong"

	"go.treyburn.dev/vanity/internal/cmd"
	"go.treyburn.dev/vanity/internal/config"
)

func main() {
	// Create a context that cancels on SIGINT (ctrl+c) or SIGTERM (CI/container stop).
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var c cmd.CLI
	kongCtx := kong.Parse(&c,
		kong.Name("vanity"),
		kong.Description("A static site generator for Go vanity URLs."),
		kong.UsageOnError(),
		kong.BindTo(ctx, (*context.Context)(nil)),
	)

	// Load config (commands that don't need it, like init, skip this)
	cfg, err := config.Load(".vanity.yaml")
	if err != nil && kongCtx.Command() != "init" {
		kongCtx.FatalIfErrorf(err)
	}

	// Apply CLI/env overrides and set up logging
	var logger = slog.Default()
	if cfg != nil {
		if c.LogLevel != "" {
			cfg.Log.Level = config.LogLevel(c.LogLevel)
		}
		if c.LogFormat != "" {
			cfg.Log.Format = config.LogFormat(c.LogFormat)
		}

		logger, err = cfg.Log.NewLogger()
		if err != nil {
			kongCtx.FatalIfErrorf(err)
		}
		slog.SetDefault(logger)
	}

	err = kongCtx.Run(cfg)
	if err != nil {
		logger.With(slog.Any("error", err)).Error("Failed to execute command")
	}
	kongCtx.FatalIfErrorf(err)
}
