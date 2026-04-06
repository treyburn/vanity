package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
)

func TestServe_ModulePage(t *testing.T) {
	moduleHTML := `<!DOCTYPE html>
<html>
<head>
  <meta name="go-import" content="go.example.com/foo git https://github.com/example/foo">
  <meta name="go-source" content="go.example.com/foo https://github.com/example/foo https://github.com/example/foo/tree/main{/dir} https://github.com/example/foo/blob/main{/dir}/{file}#L{line}">
</head>
<body>Redirecting...</body>
</html>`

	fs := fstest.MapFS{
		"foo/index.html": &fstest.MapFile{Data: []byte(moduleHTML)},
	}

	handler := fileHandler(fs)

	req := httptest.NewRequest(http.MethodGet, "/foo/?go-get=1", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, `name="go-import"`)
	assert.Contains(t, body, `content="go.example.com/foo git https://github.com/example/foo"`)
	assert.Contains(t, body, `name="go-source"`)
}

func TestServe_IndexPage(t *testing.T) {
	indexHTML := `<!DOCTYPE html><html><body><h1>Modules</h1></body></html>`

	fs := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte(indexHTML)},
	}

	handler := fileHandler(fs)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "<h1>Modules</h1>")
}

func TestServe_NotFound(t *testing.T) {
	fs := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<html></html>")},
	}

	handler := fileHandler(fs)

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestServe_GracefulShutdown(t *testing.T) {
	fs := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<html></html>")},
	}

	ctx, cancel := context.WithCancel(context.Background())
	srv := New(0, false)

	errCh := make(chan error, 1)
	ready := make(chan struct{}, 1)
	go func() {
		ready <- struct{}{}
		errCh <- srv.Serve(ctx, fs)
	}()

	// synchronize test startup
	<-ready

	// Cancel context to trigger graceful shutdown
	cancel()

	err := <-errCh
	assert.NoError(t, err)
}
