package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"go.treyburn.dev/vanity/internal/config"
)

type InitCmd struct{}

func (c *InitCmd) Run() error {
	if _, err := os.Stat(config.ConfigFileName); err == nil {
		return fmt.Errorf("%s already exists", config.ConfigFileName)
	}

	f, err := os.Create(config.ConfigFileName)
	if err != nil {
		return fmt.Errorf("creating %s: %w", config.ConfigFileName, err)
	}
	defer func() {
		cErr := f.Close()
		if cErr != nil {
			slog.Debug("failed to close file", "error", cErr)
		}
	}()

	if err = config.WriteDefault(f); err != nil {
		return fmt.Errorf("writing %s: %w", config.ConfigFileName, err)
	}

	slog.Info("created config file", "path", config.ConfigFileName)
	return nil
}
