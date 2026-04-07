package cmd

import (
	"errors"
	"io/fs"
	"log/slog"
	"os"

	"go.treyburn.dev/vanity/pkg/config"
)

// CleanCmd removes the generated output directory.
type CleanCmd struct{}

// Run removes the output directory if it exists.
func (c *CleanCmd) Run(cfg *config.Config) error {
	dir := cfg.Output.Dir

	if _, err := os.Stat(dir); errors.Is(err, fs.ErrNotExist) {
		slog.Info("output directory does not exist, nothing to clean", "dir", dir)
		return nil
	}

	if err := os.RemoveAll(dir); err != nil {
		return err
	}

	slog.Info("removed output directory", "dir", dir)
	return nil
}
