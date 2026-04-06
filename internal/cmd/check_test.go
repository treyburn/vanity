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

func TestCheckCmd_BasicValid(t *testing.T) {
	cfg := writeAndLoadConfig(t, `
domain: go.example.com
modules:
  - name: foo
    repo: https://github.com/example/foo
`)
	cmd := &CheckCmd{SkipRepoValidation: true}
	err := cmd.Run(context.Background(), cfg)
	assert.NoError(t, err)
}

func TestCheckCmd_BasicInvalid(t *testing.T) {
	// Build config directly — config.Load would reject this before we get here.
	cfg := config.DefaultConfig()
	cfg.Domain = ""
	cfg.Modules = []config.Module{
		{Name: "foo", Repo: "https://github.com/example/foo"},
	}

	cmd := &CheckCmd{SkipRepoValidation: true}
	err := cmd.Run(context.Background(), cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "domain is required")

	var ve config.ValidationError
	assert.ErrorAs(t, err, &ve)
	assert.Equal(t, 2, ve.ExitCode())
}

func TestCheckCmd_FullValid(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping remote validation in short mode")
	}

	cfg := writeAndLoadConfig(t, `
domain: go.example.com
modules:
  - name: foo
    repo: https://github.com/treyburn/vanity
`)
	cmd := &CheckCmd{SkipRepoValidation: false}
	err := cmd.Run(context.Background(), cfg)
	assert.NoError(t, err)
}

func TestCheckCmd_FullUnreachableRepo(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping remote validation in short mode")
	}

	cfg := writeAndLoadConfig(t, `
domain: go.example.com
modules:
  - name: foo
    repo: https://github.com/nonexistent-org-12345/nonexistent-repo-67890
`)
	cmd := &CheckCmd{SkipRepoValidation: false}
	err := cmd.Run(context.Background(), cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not reachable")
}

// writeAndLoadConfig writes YAML to a temp file and loads it via config.Load.
func writeAndLoadConfig(t *testing.T, yaml string) *config.Config {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, config.ConfigFileName)
	require.NoError(t, os.WriteFile(path, []byte(yaml), 0o644))
	cfg, err := config.Load(path)
	require.NoError(t, err)
	return cfg
}
