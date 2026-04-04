package cmd

import (
	"fmt"
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
	defer f.Close()

	if err = config.WriteDefault(f); err != nil {
		return fmt.Errorf("writing %s: %w", configFileName, err)
	}

	fmt.Printf("Created %s\n", configFileName)
	return nil
}
