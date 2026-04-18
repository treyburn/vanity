package cmd

import (
	"context"
	"fmt"
	"log/slog"

	"go.treyburn.dev/vanity/pkg/config"
	"go.treyburn.dev/vanity/pkg/generator"
	"go.treyburn.dev/vanity/pkg/vcs"
)

// GenerateCmd produces static HTML files from configuration.
type GenerateCmd struct{}

// Run generates the static site.
func (g *GenerateCmd) Run(ctx context.Context, cfg *config.Config) error {
	var opts []generator.NewOption
	if cfg.Templates.HasCustomTemplates() {
		opts = append(opts, generator.WithTemplates(cfg.Templates))
	}

	gen, err := generator.New(opts...)
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
			if m.LocalPath != "" {
				opts = append(opts, vcs.WithLocalPath(m.LocalPath))
			}
			pkgs, err := vcs.DiscoverSubpackages(ctx, m.Repo, m.Subpackages.Exclude, opts...)
			if err != nil {
				return nil, err
			}
			slog.Info("discovered subpackages", "module", m.Name, "count", len(pkgs))
			result[m.Name] = pkgs

		case config.SubpackageModeExplicit:
			result[m.Name] = m.Subpackages.Paths
		case config.SubpackageModeOff:
			// not enabled - so nothing to do
			continue
		default:
			return nil, config.NewValidationError(fmt.Sprintf("unknown subpackage mode: %s", m.Subpackages.Mode))
		}
	}

	return result, nil
}
