package cmd

type CLI struct {
	Init  InitCmd  `cmd:"" help:"Generate a .vanity.yaml with default values."`
	Check CheckCmd `cmd:"" help:"Validate the .vanity.yaml configuration."`

	// Global flags (env vars override YAML, flags override env vars)
	LogLevel  string `name:"log-level" env:"VANITY_LOG_LEVEL" help:"Override log level (debug, info, warn, error)."`
	LogFormat string `name:"log-format" env:"VANITY_LOG_FORMAT" help:"Override log format (text, json)."`
}
