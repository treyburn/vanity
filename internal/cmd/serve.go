package cmd

import (
	"context"
	"os"

	"go.treyburn.dev/vanity/internal/config"
	"go.treyburn.dev/vanity/internal/server"
)

type ServeCmd struct {
	Port  int  `name:"port" default:"8080" help:"Port for the local HTTP server."`
	Quiet bool `name:"quiet" help:"Suppress the startup banner and curl examples."`
}

func (s *ServeCmd) Run(ctx context.Context, cfg *config.Config) error {
	srv := server.New(s.Port, s.Quiet)
	return srv.Serve(ctx, os.DirFS(cfg.Output.Dir))
}
