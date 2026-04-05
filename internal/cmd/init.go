package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"go.treyburn.dev/vanity/internal/config"
)

const configFileName = ".vanity.yaml"

type InitCmd struct{}

func (c *InitCmd) Run() error {
	if _, err := os.Stat(configFileName); err == nil {
		return fmt.Errorf("%s already exists", configFileName)
	}

	f, err := os.Create(configFileName)
	if err != nil {
		return fmt.Errorf("creating %s: %w", configFileName, err)
	}
	defer func() {
		cErr := f.Close()
		if cErr != nil {
			slog.With(slog.Any("error", err)).Debug("Failed to close file")
		}
	}()

	if err = config.WriteDefault(f); err != nil {
		return fmt.Errorf("writing %s: %w", configFileName, err)
	}

	slog.Info(fmt.Sprintf("Created %s", configFileName))
	return nil
}
