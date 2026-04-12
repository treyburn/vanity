package server

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.treyburn.dev/ordered"
)

// Server serves static vanity URL pages.
type Server struct {
	port  int
	quiet bool
}

// New creates a Server with the given options.
func New(port int, quiet bool) *Server {
	return &Server{
		port:  port,
		quiet: quiet,
	}
}

// logCurlHints walks the filesystem for index.html files and logs concrete
// curl examples for each top-level module and up to 3 subpackages per module.
func (s *Server) logCurlHints(baseURL string, content fs.FS) {
	const maxSubpackageHints = 3

	// modules maps top-level name -> list of subpackage paths
	mods := ordered.NewMap[string, []string]()

	if err := fs.WalkDir(content, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || path.Base(p) != "index.html" {
			return nil
		}

		// p is like "vanity/index.html" or "vanity/pkg/cmd/index.html"
		dir := path.Dir(p)
		if dir == "." {
			return nil // root index.html, not a module
		}
		parts := strings.SplitN(dir, "/", 2)
		root := parts[0]
		values, ok := mods.Get(root)
		if !ok {
			mods.Set(root, nil)
		}
		if len(parts) == 2 {
			mods.Set(root, append(values, parts[1]))
		}
		return nil
	}); err != nil {
		slog.Debug("failed to find subpackages", "error", err)
	}

	for key, value := range mods.Entries() {
		slog.Info(fmt.Sprintf("try: curl -sL '%s/%s?go-get=1'", baseURL, key))
		for i, sub := range value {
			if i >= maxSubpackageHints {
				slog.Info("omitting further subpackages...")
				break
			}
			slog.Info(fmt.Sprintf("try: curl -sL '%s/%s/%s?go-get=1'", baseURL, key, sub))
		}
	}
}

// fileHandler builds the chi router for serving static content from the given filesystem.
func fileHandler(content fs.FS) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestLogger(new(slogFormatter)))
	r.Use(middleware.Recoverer)

	fileServer := http.FileServerFS(content)
	r.Handle("/*", fileServer)
	return r
}

// Serve starts the HTTP server, serving files from the given filesystem.
// The filesystem can be an os.DirFS (for serve) or an in-memory fs (for preview).
// Blocks until the context is canceled (e.g., SIGINT/SIGTERM), then shuts down gracefully.
func (s *Server) Serve(ctx context.Context, content fs.FS) error {
	srv := &http.Server{
		Addr:        fmt.Sprintf(":%d", s.port),
		Handler:     fileHandler(content),
		ReadTimeout: 5 * time.Second,
	}

	if !s.quiet {
		url := fmt.Sprintf("http://localhost:%d", s.port)
		slog.Info("server started", "url", url)
		s.logCurlHints(url, content)
		slog.Info("press ctrl+c to stop")
	}

	// Shut down gracefully when context is canceled
	go func() {
		<-ctx.Done()
		slog.Info("shutting down server")
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Error("failed to shutdown server", "error", err)
		}
	}()

	slog.Info("starting server", "port", s.port)
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
