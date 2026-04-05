package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.treyburn.dev/vanity/internal/config"
)

func TestGenerateCmd_SingleModule(t *testing.T) {
	outputDir := filepath.Join(t.TempDir(), "dist")
	cfg := writeAndLoadConfig(t, `
domain: go.example.com
modules:
  - name: foo
    repo: https://github.com/example/foo
`)
	cfg.Output.Dir = outputDir

	cmd := &GenerateCmd{}
	err := cmd.Run(context.Background(), cfg)
	require.NoError(t, err)

	assert.FileExists(t, filepath.Join(outputDir, "foo", "index.html"))
	assert.FileExists(t, filepath.Join(outputDir, "index.html"))
	assert.FileExists(t, filepath.Join(outputDir, "404.html"))
	assert.FileExists(t, filepath.Join(outputDir, "robots.txt"))
	assert.FileExists(t, filepath.Join(outputDir, "sitemap.xml"))

	// Verify module page contains correct go-import meta tag
	content, err := os.ReadFile(filepath.Join(outputDir, "foo", "index.html"))
	require.NoError(t, err)
	assert.Contains(t, string(content), `content="go.example.com/foo git https://github.com/example/foo"`)
}

func TestGenerateCmd_MultipleModules(t *testing.T) {
	outputDir := filepath.Join(t.TempDir(), "dist")
	cfg := writeAndLoadConfig(t, `
domain: go.example.com
modules:
  - name: foo
    repo: https://github.com/example/foo
  - name: bar
    repo: https://github.com/example/bar
`)
	cfg.Output.Dir = outputDir

	cmd := &GenerateCmd{}
	err := cmd.Run(context.Background(), cfg)
	require.NoError(t, err)

	assert.FileExists(t, filepath.Join(outputDir, "foo", "index.html"))
	assert.FileExists(t, filepath.Join(outputDir, "bar", "index.html"))
}

func TestGenerateCmd_CleanRemovesStaleFiles(t *testing.T) {
	outputDir := filepath.Join(t.TempDir(), "dist")

	// Pre-create stale content
	require.NoError(t, os.MkdirAll(filepath.Join(outputDir, "stale"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(outputDir, "stale", "old.html"), []byte("old"), 0o644))

	cfg := writeAndLoadConfig(t, `
domain: go.example.com
output:
  clean: true
modules:
  - name: foo
    repo: https://github.com/example/foo
`)
	cfg.Output.Dir = outputDir

	cmd := &GenerateCmd{}
	err := cmd.Run(context.Background(), cfg)
	require.NoError(t, err)

	assert.NoFileExists(t, filepath.Join(outputDir, "stale", "old.html"))
	assert.FileExists(t, filepath.Join(outputDir, "foo", "index.html"))
}

func TestGenerateCmd_OptionalOutputsDisabled(t *testing.T) {
	outputDir := filepath.Join(t.TempDir(), "dist")
	cfg := writeAndLoadConfig(t, `
domain: go.example.com
output:
  index: false
  not_found: false
  robots: false
  sitemap: false
modules:
  - name: foo
    repo: https://github.com/example/foo
`)
	cfg.Output.Dir = outputDir

	cmd := &GenerateCmd{}
	err := cmd.Run(context.Background(), cfg)
	require.NoError(t, err)

	assert.FileExists(t, filepath.Join(outputDir, "foo", "index.html"))
	assert.NoFileExists(t, filepath.Join(outputDir, "index.html"))
	assert.NoFileExists(t, filepath.Join(outputDir, "404.html"))
	assert.NoFileExists(t, filepath.Join(outputDir, "robots.txt"))
	assert.NoFileExists(t, filepath.Join(outputDir, "sitemap.xml"))
}

func TestGenerateCmd_ExplicitSubpackages(t *testing.T) {
	outputDir := filepath.Join(t.TempDir(), "dist")
	cfg := writeAndLoadConfig(t, `
domain: go.example.com
modules:
  - name: foo
    repo: https://github.com/example/foo
    subpackages:
      mode: explicit
      paths:
        - cmd/tool
        - pkg/lib
`)
	cfg.Output.Dir = outputDir

	cmd := &GenerateCmd{}
	err := cmd.Run(context.Background(), cfg)
	require.NoError(t, err)

	assert.FileExists(t, filepath.Join(outputDir, "foo", "index.html"))
	assert.FileExists(t, filepath.Join(outputDir, "foo", "cmd", "tool", "index.html"))
	assert.FileExists(t, filepath.Join(outputDir, "foo", "pkg", "lib", "index.html"))

	// Subpackage page should reference parent module in go-import
	content, err := os.ReadFile(filepath.Join(outputDir, "foo", "cmd", "tool", "index.html"))
	require.NoError(t, err)
	assert.Contains(t, string(content), `content="go.example.com/foo git https://github.com/example/foo"`)
}

func TestDiscoverSubpackages_NoSubpackages(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Domain = "go.example.com"
	cfg.Modules = []config.Module{
		{Name: "foo", Repo: "https://github.com/example/foo"},
	}

	result, err := discoverSubpackages(context.Background(), cfg)
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestDiscoverSubpackages_ModeOff(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Domain = "go.example.com"
	cfg.Modules = []config.Module{
		{
			Name: "foo",
			Repo: "https://github.com/example/foo",
			Subpackages: &config.SubpackageConfig{
				Mode: config.SubpackageModeOff,
			},
		},
	}

	result, err := discoverSubpackages(context.Background(), cfg)
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestDiscoverSubpackages_Explicit(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Domain = "go.example.com"
	cfg.Modules = []config.Module{
		{
			Name: "foo",
			Repo: "https://github.com/example/foo",
			Subpackages: &config.SubpackageConfig{
				Mode:  config.SubpackageModeExplicit,
				Paths: []string{"cmd/tool", "pkg/lib"},
			},
		},
	}

	result, err := discoverSubpackages(context.Background(), cfg)
	require.NoError(t, err)
	assert.Equal(t, []string{"cmd/tool", "pkg/lib"}, result["foo"])
}
