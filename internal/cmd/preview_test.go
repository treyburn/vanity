package cmd

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPreviewCmd_GeneratesAndServes(t *testing.T) {
	cfg := writeAndLoadConfig(t, `
domain: go.example.com
modules:
  - name: foo
    repo: https://github.com/example/foo
`)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := &PreviewCmd{Port: 8080, Quiet: true}

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

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, string(body), `content="go.example.com/foo git https://github.com/example/foo"`)

	cancel()
	assert.NoError(t, <-errCh)
}

func TestPreviewCmd_ValidationFailure(t *testing.T) {
	cfg := writeAndLoadConfig(t, `
domain: go.example.com
modules:
  - name: foo
    repo: https://github.com/example/foo
`)
	// Break the config to trigger ValidateBasic failure
	cfg.Domain = ""

	cmd := &PreviewCmd{Port: 0, Quiet: true}
	err := cmd.Run(context.Background(), cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "domain")
}
