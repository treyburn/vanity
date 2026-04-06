package config

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"

	"go.treyburn.dev/vanity/internal/vcs"
)

// ValidationError wraps a validation error and provides an exit code of 2
// via Kong's ExitCoder interface.
type ValidationError struct {
	error
}

func (v ValidationError) ExitCode() int {
	return 2
}

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

	return errs
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

		if m.Subpackages != nil && m.Subpackages.LocalPath != "" {
			info, err := os.Stat(m.Subpackages.LocalPath)
			if err != nil {
				errs = errors.Join(errs, ValidationError{fmt.Errorf("modules[%d].subpackages.local_path: %q does not exist", i, m.Subpackages.LocalPath)})
			} else if !info.IsDir() {
				errs = errors.Join(errs, ValidationError{fmt.Errorf("modules[%d].subpackages.local_path: %q is not a directory", i, m.Subpackages.LocalPath)})
			} else if err := vcs.ValidateLocalRepo(m.Subpackages.LocalPath); err != nil {
				errs = errors.Join(errs, ValidationError{fmt.Errorf("modules[%d].subpackages.local_path: %w", i, err)})
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
