package cmd

import (
	"context"
	"log/slog"

	"go.treyburn.dev/vanity/internal/config"
	"go.treyburn.dev/vanity/internal/generator"
	"go.treyburn.dev/vanity/internal/vcs"
)

type GenerateCmd struct{}

func (g *GenerateCmd) Run(ctx context.Context, cfg *config.Config) error {
	gen, err := generator.New()
	if err != nil {
		return err
	}

	subpackages, err := discoverSubpackages(ctx, cfg)
	if err != nil {
		return err
	}

	slog.Info("generating static site", "output", cfg.Output.Dir, "modules", len(cfg.Modules))
	if err = gen.Generate(cfg, subpackages); err != nil {
		return err
	}

	slog.Info("generation complete", "output", cfg.Output.Dir)
	return nil
}

// discoverSubpackages resolves subpackage paths for all modules that have
// subpackage discovery enabled (auto or explicit mode).
func discoverSubpackages(ctx context.Context, cfg *config.Config) (map[string][]string, error) {
	result := make(map[string][]string)

	for _, m := range cfg.Modules {
		if m.Subpackages == nil {
			continue
		}

		switch m.Subpackages.Mode {
		case config.SubpackageModeAuto:
			slog.Info("discovering subpackages", "module", m.Name)
			var opts []vcs.Option
			if m.Subpackages.LocalPath != "" {
				opts = append(opts, vcs.WithLocalPath(m.Subpackages.LocalPath))
			}
			pkgs, err := vcs.DiscoverSubpackages(ctx, m.Repo, m.Subpackages.Exclude, opts...)
			if err != nil {
				return nil, err
			}
			slog.Info("discovered subpackages", "module", m.Name, "count", len(pkgs))
			result[m.Name] = pkgs

		case config.SubpackageModeExplicit:
			result[m.Name] = m.Subpackages.Paths
		default:
			// nothing to do is not enabled
			continue
		}
	}

	return result, nil
}
