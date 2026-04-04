package main

import (
	"github.com/alecthomas/kong"

	"go.treyburn.dev/vanity/internal/cmd"
)

func main() {
	ctx := kong.Parse(new(cmd.CLI))
	ctx.FatalIfErrorf(ctx.Run())
}
