package generator

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing/fstest"

	html "html/template"
	text "text/template"

	"go.treyburn.dev/vanity/internal/config"
)

// template is satisfied by both html/template.Template and text/template.Template.
type template interface {
	Execute(wr io.Writer, data any) error
}

//go:embed templates/*
var templateFS embed.FS

// ModuleData holds the template context for a single module page.
type ModuleData struct {
	Domain     string
	Name       string
	ImportPath string
	Repo       string
	Branch     string
	GoSource   bool
	Redirect   string
}

// SiteData holds the template context for index, sitemap, robots, etc.
type SiteData struct {
	Domain  string
	Modules []ModuleData
}

// Generator produces static site output from a config.
type Generator struct {
	moduleTmpl   *html.Template
	indexTmpl    *html.Template
	notFoundTmpl *html.Template
	robotsTmpl   *text.Template
	sitemapTmpl  *text.Template
}

// New parses embedded templates and returns a ready-to-use Generator.
func New() (*Generator, error) {
	moduleTmpl, err := html.ParseFS(templateFS, "templates/module.html.tmpl")
	if err != nil {
		return nil, fmt.Errorf("parsing module template: %w", err)
	}

	indexTmpl, err := html.ParseFS(templateFS, "templates/index.html.tmpl")
	if err != nil {
		return nil, fmt.Errorf("parsing index template: %w", err)
	}

	notFoundTmpl, err := html.ParseFS(templateFS, "templates/not_found.html.tmpl")
	if err != nil {
		return nil, fmt.Errorf("parsing not_found template: %w", err)
	}

	robotsTmpl, err := text.ParseFS(templateFS, "templates/robots.txt.tmpl")
	if err != nil {
		return nil, fmt.Errorf("parsing robots template: %w", err)
	}

	sitemapTmpl, err := text.ParseFS(templateFS, "templates/sitemap.xml.tmpl")
	if err != nil {
		return nil, fmt.Errorf("parsing sitemap template: %w", err)
	}

	return &Generator{
		moduleTmpl:   moduleTmpl,
		indexTmpl:    indexTmpl,
		notFoundTmpl: notFoundTmpl,
		robotsTmpl:   robotsTmpl,
		sitemapTmpl:  sitemapTmpl,
	}, nil
}

// generateConfig controls how generated output is written.
type generateConfig struct {
	inMemory bool
	memFS    fstest.MapFS
}

// Option configures Generate behavior.
type Option func(*generateConfig)

// WithInMemory causes Generate to write to the provided MapFS instead of disk.
func WithInMemory(fs fstest.MapFS) Option {
	return func(c *generateConfig) {
		c.inMemory = true
		c.memFS = fs
	}
}

// writer returns an io.WriteCloser for the given path.
// In disk mode, it creates parent directories and returns an *os.File.
// In memory mode, it returns a memWriter that stores bytes into the MapFS on Close.
func (gc *generateConfig) writer(outputDir, path string) (io.WriteCloser, error) {
	if gc.inMemory {
		return &memWriter{path: path, fs: gc.memFS}, nil
	}
	fullPath := filepath.Join(outputDir, path)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o750); err != nil {
		return nil, err
	}
	return os.Create(filepath.Clean(fullPath))
}

// memWriter is an io.WriteCloser that accumulates bytes and stores them
// into a MapFS entry on Close.
type memWriter struct {
	bytes.Buffer

	path string
	fs   fstest.MapFS
}

func (m *memWriter) Close() error {
	m.fs[m.path] = &fstest.MapFile{Data: m.Bytes()}
	return nil
}

// buildModuleData converts config modules into template data.
// subpackages is a map from module name to discovered subpackage paths.
func buildModuleData(cfg *config.Config, subpackages map[string][]string) []ModuleData {
	var modules []ModuleData

	for _, m := range cfg.Modules {
		md := ModuleData{
			Domain:     cfg.Domain,
			Name:       m.Name,
			ImportPath: m.ImportPath(cfg.Domain),
			Repo:       m.Repo,
			Branch:     m.Branch,
			GoSource:   *m.GoSource,
			Redirect:   m.Redirect,
		}
		modules = append(modules, md)

		// Add subpackage pages — each gets the same go-import root-path
		// pointing to the parent module, which is what the Go tool expects.
		for _, subpkg := range subpackages[m.Name] {
			subName := m.Name + "/" + subpkg
			modules = append(modules, ModuleData{
				Domain:     cfg.Domain,
				Name:       subName,
				ImportPath: m.ImportPath(cfg.Domain),
				Repo:       m.Repo,
				Branch:     m.Branch,
				GoSource:   *m.GoSource,
				Redirect:   m.Redirect + "/" + subpkg,
			})
		}
	}

	return modules
}

// Generate renders all pages from the config. By default it writes to disk
// using cfg.Output.Dir. Use WithInMemory to write to an in-memory filesystem instead.
func (g *Generator) Generate(cfg *config.Config, subpackages map[string][]string, opts ...Option) error {
	gc := &generateConfig{}
	for _, opt := range opts {
		opt(gc)
	}

	if !gc.inMemory && cfg.Output.Clean {
		slog.Debug("cleaning output directory", "dir", cfg.Output.Dir)
		if err := os.RemoveAll(cfg.Output.Dir); err != nil {
			return fmt.Errorf("cleaning output dir: %w", err)
		}
	}

	modules := buildModuleData(cfg, subpackages)
	siteData := SiteData{Domain: cfg.Domain, Modules: modules}

	// render gets a writer from the config and renders the template into it.
	render := func(path string, tmpl template, data any) error {
		w, err := gc.writer(cfg.Output.Dir, path)
		if err != nil {
			return err
		}
		defer func() {
			if err = w.Close(); err != nil {
				slog.Debug("failed to close file", "error", err, "file", path)
			}
		}()
		return tmpl.Execute(w, data)
	}

	// Generate module pages
	for _, md := range modules {
		path := filepath.Join(md.Name, "index.html")
		if err := render(path, g.moduleTmpl, md); err != nil {
			return fmt.Errorf("generating module page %s: %w", md.Name, err)
		}
		slog.Debug("generated module page", "module", md.Name)
	}

	// Conditional site-level pages
	if cfg.Output.Index {
		if err := render("index.html", g.indexTmpl, siteData); err != nil {
			return fmt.Errorf("generating index: %w", err)
		}
		slog.Debug("generated index page")
	}

	if cfg.Output.NotFound {
		if err := render("404.html", g.notFoundTmpl, nil); err != nil {
			return fmt.Errorf("generating 404 page: %w", err)
		}
		slog.Debug("generated 404 page")
	}

	if cfg.Output.Robots {
		if err := render("robots.txt", g.robotsTmpl, siteData); err != nil {
			return fmt.Errorf("generating robots.txt: %w", err)
		}
		slog.Debug("generated robots.txt")
	}

	if cfg.Output.Sitemap {
		if err := render("sitemap.xml", g.sitemapTmpl, siteData); err != nil {
			return fmt.Errorf("generating sitemap.xml: %w", err)
		}
		slog.Debug("generated sitemap.xml")
	}

	return nil
}
