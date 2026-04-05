package cmd

import (
	"context"
	"log/slog"

	"go.treyburn.dev/vanity/internal/config"
)

type CheckCmd struct {
	BasicOnly bool `name:"basic-only" default:"false" help:"Skip repository validation (remote reachability/local path existence)."`
}

func (c *CheckCmd) Run(ctx context.Context, cfg *config.Config) error {
	if c.BasicOnly {
		slog.Info("Running basic validation")
		if err := config.ValidateBasic(cfg); err != nil {
			return err
		}
		slog.Info("Basic validation passed")
		return nil
	}

	slog.Info("Running full validation")
	if err := config.ValidateFull(ctx, cfg); err != nil {
		return err
	}
	slog.Info("Full validation passed")
	return nil
}
