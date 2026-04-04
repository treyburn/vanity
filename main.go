package main

import "github.com/alecthomas/kong"

var cli struct{}

func main() {
	ctx := kong.Parse(&cli)
	ctx.FatalIfErrorf(ctx.Run())
}
