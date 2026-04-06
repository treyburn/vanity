package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.treyburn.dev/vanity/internal/config"
)

func TestInitCmd_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	cmd := &InitCmd{}
	err := cmd.Run()
	require.NoError(t, err)

	path := filepath.Join(dir, config.ConfigFileName)
	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.False(t, info.IsDir())
	assert.Greater(t, info.Size(), int64(0))
}

func TestInitCmd_ErrorIfExists(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	// Create the file first
	err := os.WriteFile(filepath.Join(dir, config.ConfigFileName), []byte("existing"), 0o644)
	require.NoError(t, err)

	cmd := &InitCmd{}
	err = cmd.Run()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")

	// Verify the existing file was not overwritten
	data, err := os.ReadFile(filepath.Join(dir, config.ConfigFileName))
	require.NoError(t, err)
	assert.Equal(t, "existing", string(data))
}
