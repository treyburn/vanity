package server

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
)

func TestSlogFormatter_LogsRequest(t *testing.T) {
	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
	defer slog.SetDefault(slog.Default())

	fs := fstest.MapFS{
		"foo/index.html": &fstest.MapFile{Data: []byte("<html>hello</html>")},
	}

	req := httptest.NewRequest(http.MethodGet, "/foo/?go-get=1", nil)
	rec := httptest.NewRecorder()
	fileHandler(fs).ServeHTTP(rec, req)

	out := buf.String()
	assert.Contains(t, out, `"msg":"request"`)
	assert.Contains(t, out, `"method":"GET"`)
	assert.Contains(t, out, `"path":"/foo/?go-get=1"`)
	assert.Contains(t, out, `"status":200`)
}

func TestSlogFormatter_Logs404(t *testing.T) {
	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
	defer slog.SetDefault(slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	rec := httptest.NewRecorder()
	fileHandler(fstest.MapFS{}).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, buf.String(), `"status":404`)
}

func TestSlogFormatter_Panics(t *testing.T) {
	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
	defer slog.SetDefault(slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	rec := httptest.NewRecorder()
	fileHandler(nil).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, buf.String(), "request panic")
}
