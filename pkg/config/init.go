package config

import (
	"fmt"
	"io"

	"github.com/goccy/go-yaml"
)

// CommentedMinimal returns the comment map for minimal init output.
func CommentedMinimal() yaml.CommentMap {
	return yaml.CommentMap{
		"$.domain":          {yaml.HeadComment(" Your vanity URL domain (may include subdomains and url paths)"), yaml.LineComment(" REQUIRED")},
		"$.modules":         {yaml.HeadComment(" Module definitions (at least one required)")},
		"$.modules[0].name": {yaml.LineComment(" REQUIRED: import path becomes {domain}/{name}")},
		"$.modules[0].repo": {yaml.LineComment(" REQUIRED: full git repository URL")},
	}
}

// WriteMinimal writes a minimal .vanity.yml with only required fields to w.
func WriteMinimal(w io.Writer) error {
	cfg := MinimalConfig()
	cm := CommentedMinimal()

	bytes, err := yaml.MarshalWithOptions(cfg, yaml.WithComment(cm))
	if err != nil {
		return fmt.Errorf("marshaling minimal config: %w", err)
	}

	_, err = w.Write(bytes)
	return err
}

// CommentedDefault returns the comment map for verbose init output.
func CommentedDefault() yaml.CommentMap {
	return yaml.CommentMap{
		"$.log":                            {yaml.HeadComment(" CLI behavior (overridable via --flags and VANITY_* env vars)")},
		"$.log.level":                      {yaml.LineComment(" Options: debug | info | warn | error")},
		"$.log.format":                     {yaml.LineComment(" Options: text (human-friendly) | json (structured)")},
		"$.log.color":                      {yaml.LineComment(" Colorize text output (no effect on json)")},
		"$.output":                         {yaml.HeadComment(" Output settings")},
		"$.output.dir":                     {yaml.LineComment(" Relative to .vanity.yml location")},
		"$.output.clean":                   {yaml.LineComment(" Remove output dir before generating")},
		"$.output.index":                   {yaml.LineComment(" Generate root index.html listing all modules")},
		"$.output.not_found":               {yaml.LineComment(" Generate 404.html redirecting to index")},
		"$.output.robots":                  {yaml.LineComment(" Generate robots.txt (permissive, links to sitemap)")},
		"$.output.sitemap":                 {yaml.LineComment(" Generate sitemap.xml listing all module URLs")},
		"$.domain":                         {yaml.HeadComment(" Your vanity URL domain (may include subdomains and url paths)"), yaml.LineComment(" REQUIRED")},
		"$.defaults":                       {yaml.HeadComment(" Default values applied to all modules (overridable per-module)")},
		"$.defaults.branch":                {yaml.LineComment(" Used in go-source meta tag URL templates (defaults to 'main')")},
		"$.defaults.go_source":             {yaml.LineComment(" Include go-source meta tag for pkg.go.dev source links")},
		"$.defaults.redirect_root":         {yaml.LineComment(" Redirect = redirect_root/domain/name")},
		"$.modules":                        {yaml.HeadComment(" Module definitions (at least one required)")},
		"$.modules[0].name":                {yaml.LineComment(" REQUIRED: import path becomes {domain}/{module.name}")},
		"$.modules[0].repo":                {yaml.LineComment(" REQUIRED: full git repository URL")},
		"$.modules[0].branch":              {yaml.LineComment(" Override defaults.branch for this module")},
		"$.modules[0].go_source":           {yaml.LineComment(" Override defaults.go_source for this module")},
		"$.modules[0].redirect":            {yaml.LineComment(" Override browser redirect URL for this module")},
		"$.modules[0].local_path":          {yaml.LineComment(" Local checkout path (default: in-memory clone from repo remote if not specified)")},
		"$.modules[0].subpackages":         {yaml.HeadComment(" Subpackage discovery settings (enabled by default in 'auto' mode - set subpackages:mode:off to disable)")},
		"$.modules[0].subpackages.mode":    {yaml.LineComment(" Options: off | auto | explicit (defaults to 'auto')")},
		"$.modules[0].subpackages.exclude": {yaml.LineComment(" Directories to skip in auto mode (defaults to [pkg, testdata] in 'auto' mode)")},
		"$.modules[0].subpackages.paths":   {yaml.LineComment(" Allow-list exact subpackage paths (explicit mode only)")},
		"$.templates":                      {yaml.HeadComment(" Custom templates and static assets (all paths relative to .vanity.yml)")},
		"$.templates.index":                {yaml.LineComment(" Custom body partial for the root index page")},
		"$.templates.module":               {yaml.LineComment(" Custom body partial for module pages")},
		"$.templates.submodule":            {yaml.LineComment(" Custom body partial for submodule pages (falls back to module)")},
		"$.templates.not_found":            {yaml.LineComment(" Custom body partial for the 404 page")},
		"$.templates.partials":             {yaml.LineComment(" Reusable template components (referenced via {{template \"name\" .}})")},
		"$.templates.assets":               {yaml.LineComment(" Static files/dirs copied verbatim into the output directory")},
	}
}

// WriteDefault writes a fully commented default .vanity.yml to w.
func WriteDefault(w io.Writer) error {
	cfg := ExampleConfig()
	cm := CommentedDefault()

	bytes, err := yaml.MarshalWithOptions(cfg, yaml.WithComment(cm))
	if err != nil {
		return fmt.Errorf("marshaling default config: %w", err)
	}

	_, err = w.Write(bytes)
	return err
}
