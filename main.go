package main

import (
	"errors"
	"os"

	"github.com/alecthomas/kong"

	"go.treyburn.dev/vanity/internal/cmd"
)

func main() {
	ctx := kong.Parse(new(cmd.CLI))
	err := ctx.Run()
	if errors.Is(err, cmd.ErrValidationFailed) {
		// TODO - print validation errors here? or elsewhere?
		os.Exit(2)
	}
	ctx.FatalIfErrorf(err) // exit(0) if successful, exit(1) on other errors, exit(80+) on cli misuse
}
