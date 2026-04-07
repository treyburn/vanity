package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"
	"github.com/lmittmann/tint"
)

// ConfigFileName is the default configuration file name.
const ConfigFileName = ".vanity.yml"

// LogLevel is an enum for log verbosity.
type LogLevel string

// Log level constants.
const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

// LogFormat is an enum for log output format.
type LogFormat string

// Log format constants.
const (
	LogFormatText LogFormat = "text"
	LogFormatJSON LogFormat = "json"
)

// SubpackageMode is an enum for subpackage discovery behavior.
type SubpackageMode string

// Subpackage mode constants.
const (
	SubpackageModeOff      SubpackageMode = "off"
	SubpackageModeAuto     SubpackageMode = "auto"
	SubpackageModeExplicit SubpackageMode = "explicit"
)

// Config is the top-level vanity configuration.
type Config struct {
	Log      LogConfig      `yaml:"log,omitempty"`
	Output   OutputConfig   `yaml:"output,omitempty"`
	Domain   string         `yaml:"domain"`
	Defaults DefaultsConfig `yaml:"defaults,omitempty"`
	Modules  []Module       `yaml:"modules"`
}

// LogConfig controls logging behavior.
type LogConfig struct {
	Level  LogLevel  `yaml:"level"`
	Format LogFormat `yaml:"format"`
	Color  bool      `yaml:"color"`
}

// NewLogger creates a configured *slog.Logger from the log config.
func (l LogConfig) NewLogger() (*slog.Logger, error) {
	var lvl slog.Level
	switch l.Level {
	case LogLevelDebug:
		lvl = slog.LevelDebug
	case LogLevelInfo:
		lvl = slog.LevelInfo
	case LogLevelWarn:
		lvl = slog.LevelWarn
	case LogLevelError:
		lvl = slog.LevelError
	default:
		return nil, fmt.Errorf("unknown log level: %q", l.Level)
	}

	var handler slog.Handler
	switch l.Format {
	case LogFormatText:
		handler = tint.NewHandler(os.Stderr, &tint.Options{
			Level:   lvl,
			NoColor: !l.Color,
		})
	case LogFormatJSON:
		handler = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: lvl})
	default:
		return nil, fmt.Errorf("unknown log format: %q", l.Format)
	}

	return slog.New(handler), nil
}

// OutputConfig controls generated output settings.
type OutputConfig struct {
	Dir      string `yaml:"dir"`
	Clean    bool   `yaml:"clean"`
	Index    bool   `yaml:"index"`
	NotFound bool   `yaml:"not_found"`
	Robots   bool   `yaml:"robots"`
	Sitemap  bool   `yaml:"sitemap"`
}

// DefaultsConfig holds default values inherited by modules.
type DefaultsConfig struct {
	Branch       string `yaml:"branch"`
	GoSource     bool   `yaml:"go_source"`
	RedirectRoot string `yaml:"redirect_root"`
}

// Module represents a single Go module to host.
type Module struct {
	Name        string            `yaml:"name"`
	Repo        string            `yaml:"repo"`
	Branch      string            `yaml:"branch,omitempty"`
	GoSource    *bool             `yaml:"go_source,omitempty"`
	Redirect    string            `yaml:"redirect,omitempty"`
	LocalPath   string            `yaml:"local_path,omitempty"`
	Subpackages *SubpackageConfig `yaml:"subpackages,omitempty"`
}

// ImportPath returns the full import path for the module.
func (m Module) ImportPath(domain string) string {
	return domain + "/" + m.Name
}

// SubpackageConfig controls subpackage discovery for a module.
type SubpackageConfig struct {
	Mode    SubpackageMode `yaml:"mode"`
	Exclude []string       `yaml:"exclude,omitempty"`
	Paths   []string       `yaml:"paths,omitempty"`
}

// DefaultConfig returns a Config with all default values populated.
// Domain and Modules are left empty — they must come from the YAML file.
// Used by: config loading (unmarshal into defaults).
func DefaultConfig() *Config {
	return &Config{
		Log: LogConfig{
			Level:  LogLevelInfo,
			Format: LogFormatText,
			Color:  true,
		},
		Output: OutputConfig{
			Dir:      "dist",
			Clean:    true,
			Index:    true,
			NotFound: true,
			Robots:   true,
			Sitemap:  true,
		},
		Defaults: DefaultsConfig{
			Branch:       "main",
			GoSource:     true,
			RedirectRoot: "https://pkg.go.dev",
		},
	}
}

// MinimalConfig returns a Config with only required fields for `vanity init`.
func MinimalConfig() *Config {
	return &Config{
		Domain: "example.com",
		Modules: []Module{
			{
				Name: "my-module",
				Repo: "https://github.com/example/my-module",
			},
		},
	}
}

// ExampleConfig returns a Config with placeholder values for `vanity init --verbose`.
// This is the starting point users edit — includes a sample domain and module.
func ExampleConfig() *Config {
	cfg := DefaultConfig()
	cfg.Domain = "example.com"
	cfg.Modules = []Module{
		{
			Name:      "my-module",
			Repo:      "https://github.com/example/my-module",
			Branch:    "main",
			GoSource:  new(true),
			Redirect:  "https://pkg.go.dev/example.com/my-module",
			LocalPath: "./my-module",
			Subpackages: &SubpackageConfig{
				Mode:    SubpackageModeAuto,
				Exclude: []string{"pkg", "testdata"},
				Paths:   []string{"sub/pkg"},
			},
		},
	}
	return cfg
}

// Load reads .vanity.yml from the given path, applying defaults for
// any unspecified fields, then resolves per-module inheritance.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	// Start with defaults — YAML values overwrite only what's specified
	cfg := DefaultConfig()
	if err = yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	if err = ValidateBasic(cfg); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	cfg.Resolve()
	return cfg, nil
}

// Resolve fills in per-module fields from defaults. After calling Resolve,
// each Module's Branch, GoSource, and Redirect are fully populated —
// no need for Effective* methods elsewhere.
func (c *Config) Resolve() {
	for i := range c.Modules {
		m := &c.Modules[i]
		if m.Branch == "" {
			m.Branch = c.Defaults.Branch
		}
		if m.GoSource == nil {
			v := c.Defaults.GoSource
			m.GoSource = &v
		}
		if m.Redirect == "" {
			m.Redirect = c.Defaults.RedirectRoot + "/" + c.Domain + "/" + m.Name
		}
		if m.Subpackages == nil {
			m.Subpackages = &SubpackageConfig{}
		}
		if m.Subpackages.Mode == "" {
			m.Subpackages.Mode = SubpackageModeAuto
		}
		if m.Subpackages.Mode == SubpackageModeAuto && len(m.Subpackages.Exclude) == 0 {
			m.Subpackages.Exclude = []string{"pkg", "testdata"}
		}
	}
}
