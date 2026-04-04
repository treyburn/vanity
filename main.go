package main

import (
	"errors"
	"log/slog"
	"os"

	"github.com/alecthomas/kong"

	"go.treyburn.dev/vanity/internal/cmd"
	"go.treyburn.dev/vanity/internal/config"
)

func main() {
	var c cmd.CLI
	kongCtx := kong.Parse(&c,
		kong.Name("vanity"),
		kong.Description("A static site generator for Go vanity URLs."),
		kong.UsageOnError(),
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

	err = kongCtx.Run()
	if err != nil {
		logger.With(slog.Any("error", err)).Error("Failed to execute command")
		if errors.Is(err, cmd.ErrValidationFailed) {
			os.Exit(2)
		}
		os.Exit(1)
	}

	os.Exit(0)

}
