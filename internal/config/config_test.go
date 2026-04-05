package config

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_FullConfig(t *testing.T) {
	yaml := `
domain: go.treyburn.dev
log:
  level: debug
  format: json
  color: false
output:
  dir: public
  clean: false
  index: false
  not_found: false
  robots: false
  sitemap: false
defaults:
  branch: develop
  go_source: false
  redirect_root: https://godoc.org
modules:
  - name: vanity
    repo: https://github.com/treyburn/vanity
    branch: main
    go_source: true
    redirect: https://vanity.treyburn.dev/docs
    subpackages:
      mode: auto
      exclude:
        - "internal/*"
        - "testdata/*"
  - name: other-pkg
    repo: https://github.com/treyburn/other-pkg
`
	cfg := loadFromString(t, yaml)

	assert.Equal(t, "go.treyburn.dev", cfg.Domain)
	assert.Equal(t, LogLevelDebug, cfg.Log.Level)
	assert.Equal(t, LogFormatJSON, cfg.Log.Format)
	assert.False(t, cfg.Log.Color)
	assert.Equal(t, "public", cfg.Output.Dir)
	assert.False(t, cfg.Output.Clean)
	assert.False(t, cfg.Output.Index)
	assert.False(t, cfg.Output.NotFound)
	assert.False(t, cfg.Output.Robots)
	assert.False(t, cfg.Output.Sitemap)
	assert.Equal(t, "develop", cfg.Defaults.Branch)
	assert.False(t, cfg.Defaults.GoSource)
	assert.Equal(t, "https://godoc.org", cfg.Defaults.RedirectRoot)

	require.Len(t, cfg.Modules, 2)

	// First module: all fields explicitly set
	m0 := cfg.Modules[0]
	assert.Equal(t, "vanity", m0.Name)
	assert.Equal(t, "https://github.com/treyburn/vanity", m0.Repo)
	assert.Equal(t, "main", m0.Branch)
	require.NotNil(t, m0.GoSource)
	assert.True(t, *m0.GoSource)
	assert.Equal(t, "https://vanity.treyburn.dev/docs", m0.Redirect)
	require.NotNil(t, m0.Subpackages)
	assert.Equal(t, SubpackageModeAuto, m0.Subpackages.Mode)
	assert.Equal(t, []string{"internal/*", "testdata/*"}, m0.Subpackages.Exclude)

	// Second module: inherits defaults via Resolve()
	m1 := cfg.Modules[1]
	assert.Equal(t, "other-pkg", m1.Name)
	assert.Equal(t, "develop", m1.Branch)
	require.NotNil(t, m1.GoSource)
	assert.False(t, *m1.GoSource)
	assert.Equal(t, "https://godoc.org/go.treyburn.dev/other-pkg", m1.Redirect)
}

func TestLoad_MinimalConfig(t *testing.T) {
	yaml := `
domain: go.example.com
modules:
  - name: foo
    repo: https://github.com/example/foo
`
	cfg := loadFromString(t, yaml)

	// Defaults should be applied
	assert.Equal(t, LogLevelInfo, cfg.Log.Level)
	assert.Equal(t, LogFormatText, cfg.Log.Format)
	assert.True(t, cfg.Log.Color)
	assert.Equal(t, "dist", cfg.Output.Dir)
	assert.True(t, cfg.Output.Clean)
	assert.True(t, cfg.Output.Index)
	assert.True(t, cfg.Output.NotFound)
	assert.True(t, cfg.Output.Robots)
	assert.True(t, cfg.Output.Sitemap)
	assert.Equal(t, "main", cfg.Defaults.Branch)
	assert.True(t, cfg.Defaults.GoSource)
	assert.Equal(t, "https://pkg.go.dev", cfg.Defaults.RedirectRoot)

	require.Len(t, cfg.Modules, 1)
	m := cfg.Modules[0]
	assert.Equal(t, "foo", m.Name)
	assert.Equal(t, "https://github.com/example/foo", m.Repo)
	assert.Equal(t, "main", m.Branch)
	require.NotNil(t, m.GoSource)
	assert.True(t, *m.GoSource)
	assert.Equal(t, "https://pkg.go.dev/go.example.com/foo", m.Redirect)
}

func TestLoad_MissingDomain(t *testing.T) {
	yaml := `
modules:
  - name: foo
    repo: https://github.com/example/foo
`
	path := writeTempConfig(t, yaml)
	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "domain")
}

func TestLoad_MissingModules(t *testing.T) {
	yaml := `
domain: go.example.com
`
	path := writeTempConfig(t, yaml)
	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one module is required")
}

func TestLoad_MissingModuleName(t *testing.T) {
	yaml := `
domain: go.example.com
modules:
  - repo: https://github.com/example/foo
`
	path := writeTempConfig(t, yaml)
	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name")
}

func TestLoad_MissingModuleRepo(t *testing.T) {
	yaml := `
domain: go.example.com
modules:
  - name: foo
`
	path := writeTempConfig(t, yaml)
	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "repo")
}

func TestLoad_InvalidSubpackageMode(t *testing.T) {
	yaml := `
domain: go.example.com
modules:
  - name: foo
    repo: https://github.com/example/foo
    subpackages:
      mode: invalid
`
	path := writeTempConfig(t, yaml)
	_, err := Load(path)
	require.Error(t, err)
}

func TestLoad_InvalidYAML(t *testing.T) {
	yaml := `
domain: go.example.com
modules:
  - name: [invalid
`
	path := writeTempConfig(t, yaml)
	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing config")
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/.vanity.yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reading config")
}

func TestLoad_OverridesDefaultModules(t *testing.T) {
	// When YAML specifies modules, it should replace the default module list entirely
	yaml := `
domain: go.example.com
modules:
  - name: alpha
    repo: https://github.com/example/alpha
  - name: beta
    repo: https://github.com/example/beta
`
	cfg := loadFromString(t, yaml)

	require.Len(t, cfg.Modules, 2)
	assert.Equal(t, "alpha", cfg.Modules[0].Name)
	assert.Equal(t, "beta", cfg.Modules[1].Name)
}

func TestResolve_InheritsDefaults(t *testing.T) {
	cfg := &Config{
		Domain: "go.example.com",
		Defaults: DefaultsConfig{
			Branch:       "main",
			GoSource:     true,
			RedirectRoot: "https://pkg.go.dev",
		},
		Modules: []Module{
			{Name: "foo", Repo: "https://github.com/example/foo"},
		},
	}

	cfg.Resolve()

	m := cfg.Modules[0]
	assert.Equal(t, "main", m.Branch)
	require.NotNil(t, m.GoSource)
	assert.True(t, *m.GoSource)
	assert.Equal(t, "https://pkg.go.dev/go.example.com/foo", m.Redirect)
}

func TestResolve_ModuleOverridesDefaults(t *testing.T) {
	goSourceFalse := false
	cfg := &Config{
		Domain: "go.example.com",
		Defaults: DefaultsConfig{
			Branch:       "main",
			GoSource:     true,
			RedirectRoot: "https://pkg.go.dev",
		},
		Modules: []Module{
			{
				Name:     "foo",
				Repo:     "https://github.com/example/foo",
				Branch:   "develop",
				GoSource: &goSourceFalse,
				Redirect: "https://custom.example.com/docs",
			},
		},
	}

	cfg.Resolve()

	m := cfg.Modules[0]
	assert.Equal(t, "develop", m.Branch)
	require.NotNil(t, m.GoSource)
	assert.False(t, *m.GoSource)
	assert.Equal(t, "https://custom.example.com/docs", m.Redirect)
}

func TestResolve_SubpackageModeDefaultsToOff(t *testing.T) {
	cfg := &Config{
		Domain:   "go.example.com",
		Defaults: DefaultsConfig{Branch: "main", GoSource: true, RedirectRoot: "https://pkg.go.dev"},
		Modules: []Module{
			{
				Name:        "foo",
				Repo:        "https://github.com/example/foo",
				Subpackages: &SubpackageConfig{
					// Mode is empty string — should default to "off"
				},
			},
		},
	}

	cfg.Resolve()
	assert.Equal(t, SubpackageModeOff, cfg.Modules[0].Subpackages.Mode)
}

func TestImportPath(t *testing.T) {
	m := Module{Name: "vanity"}
	assert.Equal(t, "go.treyburn.dev/vanity", m.ImportPath("go.treyburn.dev"))
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, LogLevelInfo, cfg.Log.Level)
	assert.Equal(t, LogFormatText, cfg.Log.Format)
	assert.True(t, cfg.Log.Color)
	assert.Equal(t, "dist", cfg.Output.Dir)
	assert.True(t, cfg.Output.Clean)
	assert.True(t, cfg.Output.Index)
	assert.True(t, cfg.Output.NotFound)
	assert.True(t, cfg.Output.Robots)
	assert.True(t, cfg.Output.Sitemap)
	assert.Empty(t, cfg.Domain)
	assert.Equal(t, "main", cfg.Defaults.Branch)
	assert.True(t, cfg.Defaults.GoSource)
	assert.Equal(t, "https://pkg.go.dev", cfg.Defaults.RedirectRoot)
	assert.Empty(t, cfg.Modules)
}

func TestExampleConfig(t *testing.T) {
	cfg := ExampleConfig()

	// Should have all the defaults plus placeholder domain and module
	assert.Equal(t, LogLevelInfo, cfg.Log.Level)
	assert.Equal(t, "dist", cfg.Output.Dir)
	assert.Equal(t, "example.com", cfg.Domain)
	assert.Equal(t, "main", cfg.Defaults.Branch)
	require.Len(t, cfg.Modules, 1)
	assert.Equal(t, "my-module", cfg.Modules[0].Name)
	assert.Equal(t, "https://github.com/example/my-module", cfg.Modules[0].Repo)
}

func TestNewLogger_TextFormat(t *testing.T) {
	lc := LogConfig{Level: LogLevelDebug, Format: LogFormatText, Color: false}
	logger, err := lc.NewLogger()
	require.NoError(t, err)
	assert.True(t, logger.Enabled(nil, slog.LevelDebug))
	assert.True(t, logger.Enabled(nil, slog.LevelInfo))
}

func TestNewLogger_JSONFormat(t *testing.T) {
	lc := LogConfig{Level: LogLevelWarn, Format: LogFormatJSON, Color: false}
	logger, err := lc.NewLogger()
	require.NoError(t, err)
	assert.False(t, logger.Enabled(nil, slog.LevelInfo))
	assert.True(t, logger.Enabled(nil, slog.LevelWarn))
	assert.True(t, logger.Enabled(nil, slog.LevelError))
}

func TestNewLogger_ColorOn(t *testing.T) {
	lc := LogConfig{Level: LogLevelInfo, Format: LogFormatText, Color: true}
	logger, err := lc.NewLogger()
	require.NoError(t, err)
	assert.True(t, logger.Enabled(nil, slog.LevelInfo))
}

func TestNewLogger_ErrorLevel(t *testing.T) {
	lc := LogConfig{Level: LogLevelError, Format: LogFormatText}
	logger, err := lc.NewLogger()
	require.NoError(t, err)
	assert.False(t, logger.Enabled(nil, slog.LevelWarn))
	assert.True(t, logger.Enabled(nil, slog.LevelError))
}

func TestNewLogger_UnknownLevel(t *testing.T) {
	lc := LogConfig{Level: "verbose", Format: LogFormatText}
	_, err := lc.NewLogger()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown log level")
}

func TestNewLogger_UnknownFormat(t *testing.T) {
	lc := LogConfig{Level: LogLevelInfo, Format: "yaml"}
	_, err := lc.NewLogger()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown log format")
}

func TestNewLogger_EmptyLevel(t *testing.T) {
	lc := LogConfig{Level: "", Format: LogFormatText}
	_, err := lc.NewLogger()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown log level")
}

// loadFromString is a test helper that writes YAML to a temp file and loads it.
func loadFromString(t *testing.T, content string) *Config {
	t.Helper()
	path := writeTempConfig(t, content)
	cfg, err := Load(path)
	require.NoError(t, err)
	return cfg
}

// writeTempConfig writes content to a temp file and returns the path.
func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, ".vanity.yaml")
	err := os.WriteFile(path, []byte(content), 0o644)
	require.NoError(t, err)
	return path
}
