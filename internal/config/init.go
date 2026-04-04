package config

import (
	"fmt"
	"io"

	"github.com/goccy/go-yaml"
)

// CommentedDefault returns the comment map for init output.
func CommentedDefault() yaml.CommentMap {
	return yaml.CommentMap{
		"$.log":                    {yaml.HeadComment(" CLI behavior (overridable via --flags and VANITY_* env vars)")},
		"$.log.level":              {yaml.LineComment(" Options: debug | info | warn | error")},
		"$.log.format":             {yaml.LineComment(" Options: text (human-friendly) | json (structured)")},
		"$.log.color":              {yaml.LineComment(" Colorize text output (no effect on json)")},
		"$.output":                 {yaml.HeadComment(" Output settings")},
		"$.output.dir":             {yaml.LineComment(" Relative to .vanity.yaml location")},
		"$.output.clean":           {yaml.LineComment(" Remove output dir before generating")},
		"$.output.index":           {yaml.LineComment(" Generate root index.html listing all modules")},
		"$.output.not_found":       {yaml.LineComment(" Generate 404.html redirecting to index")},
		"$.output.robots":          {yaml.LineComment(" Generate robots.txt (permissive, links to sitemap)")},
		"$.output.sitemap":         {yaml.LineComment(" Generate sitemap.xml listing all module URLs")},
		"$.domain":                 {yaml.HeadComment(" Your vanity URL domain"), yaml.LineComment(" REQUIRED")},
		"$.defaults":               {yaml.HeadComment(" Default values applied to all modules (overridable per-module)")},
		"$.defaults.branch":        {yaml.LineComment(" Used in go-source meta tag URL templates")},
		"$.defaults.go_source":     {yaml.LineComment(" Include go-source meta tag for pkg.go.dev source links")},
		"$.defaults.redirect_root": {yaml.LineComment(" Redirect = redirect_root/domain/name")},
		"$.modules":                {yaml.HeadComment(" Module definitions (at least one required)")},
		"$.modules[0].name":        {yaml.LineComment(" REQUIRED: import path becomes {domain}/{name}")},
		"$.modules[0].repo":        {yaml.LineComment(" REQUIRED: full git repository URL")},
	}
}

// WriteDefault writes a fully commented default .vanity.yaml to w.
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
