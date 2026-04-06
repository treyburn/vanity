package server

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// slogFormatter adapts chi's request logging to use slog.
type slogFormatter struct{}

func (s *slogFormatter) NewLogEntry(r *http.Request) middleware.LogEntry {
	return &slogEntry{
		method: r.Method,
		path:   r.RequestURI,
		from:   r.RemoteAddr,
	}
}

type slogEntry struct {
	method string
	path   string
	from   string
}

func (e *slogEntry) Write(status, bytes int, header http.Header, elapsed time.Duration, _ interface{}) {
	slog.Info("request",
		"method", e.method,
		"path", e.path,
		"from", e.from,
		"status", status,
		"bytes", bytes,
		"elapsed", elapsed.Round(time.Millisecond).String(),
	)
}

func (e *slogEntry) Panic(v interface{}, stack []byte) {
	slog.Error("request panic",
		"method", e.method,
		"path", e.path,
		"error", fmt.Sprint(v),
		"stack", string(stack),
	)
}
