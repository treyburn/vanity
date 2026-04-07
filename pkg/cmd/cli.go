package cmd

// CLI defines the top-level command tree parsed by Kong.
type CLI struct {
	Init     InitCmd     `cmd:"" help:"Generate a minimal .vanity.yml config file (use --verbose for full spec)."`
	Generate GenerateCmd `cmd:"" help:"Generate static HTML files from configuration."`
	Check    CheckCmd    `cmd:"" help:"Validate the .vanity.yml configuration."`
	Preview  PreviewCmd  `cmd:"" help:"Generate in-memory and serve via local HTTP."`
	Serve    ServeCmd    `cmd:"" help:"Serve already-generated output directory."`
	Clean    CleanCmd    `cmd:"" help:"Remove the generated output directory."`
	Version  VersionCmd  `cmd:"" help:"Print version information."`

	// Global flags (env vars override YAML, flags override env vars)
	LogLevel  string `env:"VANITY_LOG_LEVEL"  help:"Override log level (debug, info, warn, error)." name:"log-level"`
	LogFormat string `env:"VANITY_LOG_FORMAT" help:"Override log format (text, json)."              name:"log-format"`
}
