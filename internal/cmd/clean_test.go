package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.treyburn.dev/vanity/internal/config"
)

func TestCleanCmd_RemovesOutputDir(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, "dist")

	// Create some files in the output directory
	require.NoError(t, os.MkdirAll(filepath.Join(outputDir, "foo"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(outputDir, "foo", "index.html"), []byte("test"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(outputDir, "index.html"), []byte("test"), 0o644))

	cfg := config.DefaultConfig()
	cfg.Output.Dir = outputDir

	cmd := &CleanCmd{}
	err := cmd.Run(cfg)
	require.NoError(t, err)

	assert.NoDirExists(t, outputDir)
}

func TestCleanCmd_NoOpIfMissing(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Output.Dir = filepath.Join(t.TempDir(), "nonexistent")

	cmd := &CleanCmd{}
	err := cmd.Run(cfg)
	assert.NoError(t, err)
}
