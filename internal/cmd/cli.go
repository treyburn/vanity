package cmd

import (
	"errors"
)

type CLI struct {
	Init InitCmd `cmd:"" help:"Generate a .vanity.yaml with default values."`

	// Global flags (env vars override YAML, flags override env vars)
	LogLevel  string `name:"log-level" default:"" env:"VANITY_LOG_LEVEL" help:"Override log level (debug, info, warn, error)."`
	LogFormat string `name:"log-format" default:"" env:"VANITY_LOG_FORMAT" help:"Override log format (text, json)."`
}

// TODO - make this a custom type which implements https://github.com/square/exit
var ErrValidationFailed = errors.New("validation failed")
