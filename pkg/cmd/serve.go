package cmd

import (
	"context"
	"os"

	"go.treyburn.dev/vanity/pkg/config"
	"go.treyburn.dev/vanity/pkg/server"
)

// ServeCmd serves an already-generated output directory.
type ServeCmd struct {
	Port  int  `default:"8080"                                        help:"Port for the local HTTP server." name:"port"`
	Quiet bool `help:"Suppress the startup banner and curl examples." name:"quiet"`
}

// Run starts the HTTP file server.
func (s *ServeCmd) Run(ctx context.Context, cfg *config.Config) error {
	srv := server.New(s.Port, s.Quiet)
	return srv.Serve(ctx, os.DirFS(cfg.Output.Dir))
}
