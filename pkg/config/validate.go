package config

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	html "html/template"

	"go.treyburn.dev/vanity/pkg/vcs"
)

// ValidationError wraps a validation error and provides an exit code of 2
// via Kong's ExitCoder interface.
type ValidationError struct {
	error
}

// ExitCode returns 2 to signal a validation failure to Kong.
func (v ValidationError) ExitCode() int {
	return 2
}

// NewValidationError creates a ValidationError with the given message.
func NewValidationError(msg string) ValidationError {
	return ValidationError{errors.New(msg)}
}

// ValidateBasic performs fast, local validation. Called by check, generate,
// and preview before doing work.
func ValidateBasic(cfg *Config) error {
	var errs error

	// Log config enum checks
	switch cfg.Log.Level {
	case LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError:
		// valid
	default:
		errs = errors.Join(errs, ValidationError{fmt.Errorf("log.level: unknown value %q (must be debug, info, warn, or error)", cfg.Log.Level)})
	}

	switch cfg.Log.Format {
	case LogFormatText, LogFormatJSON:
		// valid
	default:
		errs = errors.Join(errs, ValidationError{fmt.Errorf("log.format: unknown value %q (must be text or json)", cfg.Log.Format)})
	}

	// Required fields
	if cfg.Domain == "" {
		errs = errors.Join(errs, ValidationError{fmt.Errorf("domain is required")})
	}
	if len(cfg.Modules) == 0 {
		errs = errors.Join(errs, ValidationError{fmt.Errorf("at least one module is required")})
	}

	// Per-module checks
	seen := make(map[string]bool)
	seenRepo := make(map[string]bool)
	for i, m := range cfg.Modules {
		if m.Name == "" {
			errs = errors.Join(errs, ValidationError{fmt.Errorf("modules[%d].name is required", i)})
		}
		if m.Repo == "" {
			errs = errors.Join(errs, ValidationError{fmt.Errorf("modules[%d].repo is required", i)})
		} else if err := validateRepoURL(m.Repo); err != nil {
			errs = errors.Join(errs, ValidationError{fmt.Errorf("modules[%d].repo: %w", i, err)})
		}

		if m.Name != "" && seen[m.Name] {
			errs = errors.Join(errs, ValidationError{fmt.Errorf("duplicate module name: %q", m.Name)})
		}
		seen[m.Name] = true

		if m.Repo != "" && seenRepo[m.Repo] {
			errs = errors.Join(errs, ValidationError{fmt.Errorf("duplicate repo URL: %q", m.Repo)})
		}
		seenRepo[m.Repo] = true

		if m.Subpackages != nil && m.Subpackages.Mode != "" {
			switch m.Subpackages.Mode {
			case SubpackageModeOff, SubpackageModeAuto, SubpackageModeExplicit:
				// valid
			default:
				errs = errors.Join(errs, ValidationError{fmt.Errorf("modules[%d].subpackages.mode: unknown value %q (must be off, auto, or explicit)", i, m.Subpackages.Mode)})
			}

			if m.Subpackages.Mode == SubpackageModeExplicit && len(m.Subpackages.Paths) == 0 {
				errs = errors.Join(errs, ValidationError{fmt.Errorf("modules[%d].subpackages.paths is required when mode is explicit", i)})
			}
		}
	}

	// Template validation
	if cfg.Templates.HasCustomTemplates() || len(cfg.Templates.Assets) > 0 {
		errs = errors.Join(errs, validateTemplates(&cfg.Templates))
	}

	// Asset collision detection (purely local, no network needed)
	if len(cfg.Templates.Assets) > 0 {
		errs = errors.Join(errs, validateAssetCollisions(cfg))
	}

	return errs
}

// validationFuncMap is a stub FuncMap that registers the same function names
// as the generator's FuncMap. This allows template parse-checking to succeed
// without the config package depending on the generator package.
// The implementations are trivial — actual behavior comes from the generator.
var validationFuncMap = html.FuncMap{
	"upper":     func(s string) string { return s },
	"lower":     func(s string) string { return s },
	"title":     func(s string) string { return s },
	"join":      func(elems []string, sep string) string { return "" },
	"sprintf":   fmt.Sprintf,
	"now":       func() any { return nil },
	"year":      func() int { return 0 },
	"contains":  func(s, substr string) bool { return false },
	"hasPrefix": func(s, prefix string) bool { return false },
	"hasSuffix": func(s, suffix string) bool { return false },
	"replace":   func(s, old, new string) string { return s },
	"trimSpace": func(s string) string { return s },
}

// validateTemplates checks that template files exist, parse correctly, and
// that asset paths exist. It also checks for collisions between assets and
// generated output files.
func validateTemplates(t *TemplatesConfig) error {
	var errs error

	// Check that all referenced template files exist and parse
	for _, path := range t.AllTemplatePaths() {
		info, err := os.Stat(path)
		if err != nil {
			errs = errors.Join(errs, ValidationError{fmt.Errorf("templates: %q does not exist", path)})
			continue
		}
		if info.IsDir() {
			errs = errors.Join(errs, ValidationError{fmt.Errorf("templates: %q is a directory, expected a file", path)})
			continue
		}
		// Parse with the validation FuncMap so custom function calls don't cause false errors
		if _, err := html.New(filepath.Base(path)).Funcs(validationFuncMap).Parse(string(mustReadFile(path))); err != nil {
			errs = errors.Join(errs, ValidationError{fmt.Errorf("templates: %q has invalid syntax: %w", path, err)})
		}
	}

	// Check that asset paths exist
	for _, path := range t.Assets {
		if _, err := os.Stat(path); err != nil {
			errs = errors.Join(errs, ValidationError{fmt.Errorf("templates.assets: %q does not exist", path)})
		}
	}

	return errs
}

// mustReadFile reads a file and returns its contents. It panics on error,
// but callers should have already verified the file exists via os.Stat.
func mustReadFile(path string) []byte {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		panic(fmt.Sprintf("reading %q after stat succeeded: %v", path, err))
	}
	return data
}

// ValidateFull runs ValidateBasic plus expensive remote checks.
// Default for `vanity check`. Skipped with --skip-repo-validation.
func ValidateFull(ctx context.Context, cfg *Config) error {
	if err := ValidateBasic(cfg); err != nil {
		return err
	}

	var errs error

	for i, m := range cfg.Modules {
		if err := vcs.ValidateRemote(ctx, m.Repo); err != nil {
			errs = errors.Join(errs, ValidationError{fmt.Errorf("modules[%d].repo: %w", i, err)})
		}

		if m.LocalPath != "" {
			info, err := os.Stat(m.LocalPath)
			if err != nil {
				errs = errors.Join(errs, ValidationError{fmt.Errorf("modules[%d].local_path: %q does not exist", i, m.LocalPath)})
			} else if !info.IsDir() {
				errs = errors.Join(errs, ValidationError{fmt.Errorf("modules[%d].local_path: %q is not a directory", i, m.LocalPath)})
			} else if err := vcs.ValidateLocalRepo(m.LocalPath); err != nil {
				errs = errors.Join(errs, ValidationError{fmt.Errorf("modules[%d].local_path: %w", i, err)})
			}
		}
	}

	return errs
}

// validateAssetCollisions checks that asset paths don't collide with
// generated output files (index.html, 404.html, robots.txt, sitemap.xml,
// and module directories). Since assets are copied preserving their relative
// path, we check that no asset path matches a generated output path.
func validateAssetCollisions(cfg *Config) error {
	// Build a set of top-level generated output names
	generated := make(map[string]bool)
	if cfg.Output.Index {
		generated["index.html"] = true
	}
	if cfg.Output.NotFound {
		generated["404.html"] = true
	}
	if cfg.Output.Robots {
		generated["robots.txt"] = true
	}
	if cfg.Output.Sitemap {
		generated["sitemap.xml"] = true
	}
	for _, m := range cfg.Modules {
		generated[m.Name] = true
	}

	var errs error
	for _, assetPath := range cfg.Templates.Assets {
		info, err := os.Stat(assetPath)
		if err != nil {
			continue // already caught by validateTemplates
		}

		// For collision purposes, check the base name of the asset path
		// against generated top-level names.
		baseName := filepath.Base(assetPath)
		if info.IsDir() {
			// A directory asset's base name could collide with a module directory
			if generated[baseName] {
				errs = errors.Join(errs, ValidationError{
					fmt.Errorf("templates.assets: directory %q would collide with generated path %q", assetPath, baseName),
				})
			}
		} else {
			// A file asset's base name could collide with a generated file
			if generated[baseName] {
				errs = errors.Join(errs, ValidationError{
					fmt.Errorf("templates.assets: file %q would collide with generated file %q", assetPath, baseName),
				})
			}
		}
	}

	return errs
}

// validateRepoURL checks that repo is a valid URL with a scheme.
func validateRepoURL(repo string) error {
	u, err := url.Parse(repo)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme == "" {
		return fmt.Errorf("missing scheme (e.g., https://)")
	}
	if u.Host == "" {
		return fmt.Errorf("missing host")
	}
	return nil
}
