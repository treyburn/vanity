package tmpl

import (
	"embed"
	"io"
)

// Templates stores our base, minimal HTML templates
//
//go:embed templates/*
var Templates embed.FS

// Template is a templater abstraction satisfied by both html/template and text/template.
type Template interface {
	Execute(wr io.Writer, data any) error
}
