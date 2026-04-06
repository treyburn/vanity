package cmd

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServeCmd_MissingOutputDirReturnsError(t *testing.T) {
	cfg := writeAndLoadConfig(t, `
domain: go.example.com
modules:
  - name: foo
    repo: https://github.com/example/foo
`)
	cfg.Output.Dir = filepath.Join(t.TempDir(), "nonexistent")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := &ServeCmd{Port: 8080, Quiet: true}

	errCh := make(chan error, 1)
	go func() {
		errCh <- cmd.Run(ctx, cfg)
	}()

	// Wait for server to be ready, then request a module from the missing dir
	require.Eventually(t, func() bool {
		resp, err := http.Get("http://localhost:8080/foo/")
		if err != nil {
			return false
		}
		assert.NoError(t, resp.Body.Close())
		return true
	}, 2*time.Second, 25*time.Millisecond)

	resp, err := http.Get("http://localhost:8080/foo/?go-get=1")
	require.NoError(t, err)
	defer func() { assert.NoError(t, resp.Body.Close()) }()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	cancel()
	assert.NoError(t, <-errCh)
}

func TestServeCmd_ContentAccessible(t *testing.T) {
	outputDir := filepath.Join(t.TempDir(), "dist")
	require.NoError(t, os.MkdirAll(filepath.Join(outputDir, "foo"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(outputDir, "foo", "index.html"),
		[]byte(`<meta name="go-import" content="go.example.com/foo git https://github.com/example/foo">`),
		0o644,
	))

	cfg := writeAndLoadConfig(t, `
domain: go.example.com
modules:
  - name: foo
    repo: https://github.com/example/foo
`)
	cfg.Output.Dir = outputDir

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := &ServeCmd{Port: 8080, Quiet: true}

	errCh := make(chan error, 1)
	go func() {
		errCh <- cmd.Run(ctx, cfg)
	}()

	// Wait for server to be ready
	require.Eventually(t, func() bool {
		resp, err := http.Get("http://localhost:8080/foo/")
		if err != nil {
			return false
		}
		assert.NoError(t, resp.Body.Close())
		return resp.StatusCode == http.StatusOK
	}, 2*time.Second, 25*time.Millisecond)

	resp, err := http.Get("http://localhost:8080/foo/?go-get=1")
	require.NoError(t, err)
	defer func() { assert.NoError(t, resp.Body.Close()) }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	cancel()
	assert.NoError(t, <-errCh)
}
