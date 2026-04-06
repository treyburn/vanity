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

	// Load config
	cfg, err := config.Load(".vanity.yaml")
	// init and version cmd's don't require the config to exist - so it's ok to continue
	if err != nil && kongCtx.Command() != "init" && kongCtx.Command() != "version" {
		kongCtx.FatalIfErrorf(err)
	}

	// Apply CLI/env overrides and set up logging
	if cfg != nil {
		if c.LogLevel != "" {
			cfg.Log.Level = config.LogLevel(c.LogLevel)
		}
		if c.LogFormat != "" {
			cfg.Log.Format = config.LogFormat(c.LogFormat)
		}

		logger, logErr := cfg.Log.NewLogger()
		if logErr != nil {
			kongCtx.FatalIfErrorf(err)
		}
		slog.SetDefault(logger)
		slog.Debug("config loaded", "domain", cfg.Domain, "modules", len(cfg.Modules))
	}

	if err = kongCtx.Run(cfg); err != nil {
		slog.Error("failed to execute command", "error", err)
		kongCtx.FatalIfErrorf(err)
	}
	os.Exit(0)
}
