package cmd

import (
	"context"
	"log/slog"

	"go.treyburn.dev/vanity/internal/config"
)

type CheckCmd struct {
	SkipRepoValidation bool `name:"skip-repo-validation" default:"false" help:"Skip repository validation (remote reachability/local path existence)."`
}

func (c *CheckCmd) Run(ctx context.Context, cfg *config.Config) error {
	if c.SkipRepoValidation {
		if err := config.ValidateBasic(cfg); err != nil {
			return err
		}
		slog.Info("basic validation passed")
		return nil
	}

	if err := config.ValidateFull(ctx, cfg); err != nil {
		return err
	}
	slog.Info("full validation passed")
	return nil
}
