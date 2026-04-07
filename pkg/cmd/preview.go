package cmd

import (
	"context"
	"log/slog"
	"testing/fstest"

	"go.treyburn.dev/vanity/pkg/config"
	"go.treyburn.dev/vanity/pkg/generator"
	"go.treyburn.dev/vanity/pkg/server"
)

type PreviewCmd struct {
	Port  int  `default:"8080"  help:"Port for the local HTTP server."                name:"port"`
	Quiet bool `default:"false" help:"Suppress the startup banner and curl examples." name:"quiet"`
}

func (p *PreviewCmd) Run(ctx context.Context, cfg *config.Config) error {
	if err := config.ValidateBasic(cfg); err != nil {
		return err
	}

	gen, err := generator.New()
	if err != nil {
		return err
	}

	subpackages, err := discoverSubpackages(ctx, cfg)
	if err != nil {
		return err
	}

	memFS := make(fstest.MapFS)
	slog.Info("generating in-memory site", "modules", len(cfg.Modules))
	if err = gen.Generate(cfg, subpackages, generator.WithInMemory(memFS)); err != nil {
		return err
	}

	srv := server.New(p.Port, p.Quiet)
	return srv.Serve(ctx, memFS)
}
