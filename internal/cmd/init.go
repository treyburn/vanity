package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"go.treyburn.dev/vanity/internal/config"
)

type InitCmd struct {
	Verbose bool `help:"Include all configuration options with comments." short:"v"`
}

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

	if c.Verbose {
		if err = config.WriteDefault(f); err != nil {
			return fmt.Errorf("writing %s: %w", config.ConfigFileName, err)
		}
	} else {
		if err = config.WriteMinimal(f); err != nil {
			return fmt.Errorf("writing %s: %w", config.ConfigFileName, err)
		}
	}

	slog.Info("created config file", "path", config.ConfigFileName)
	return nil
}
