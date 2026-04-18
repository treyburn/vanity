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

	"go.treyburn.dev/vanity/pkg/config"
)

// template is satisfied by both html/template.Template and text/template.Template.
type template interface {
	Execute(wr io.Writer, data any) error
}

//go:embed templates/*
var templateFS embed.FS

// ModuleData holds the template context for a single module page.
type ModuleData struct {
	Domain      string
	Name        string
	ImportPath  string
	Repo        string
	Branch      string
	GoSource    bool
	Redirect    string
	Subpackages []ModuleData
}

// SiteData holds the template context for index, sitemap, robots, etc.
type SiteData struct {
	Domain  string
	Modules []ModuleData
}

// Generator produces static site output from a config.
type Generator struct {
	moduleTmpl    *html.Template
	submoduleTmpl *html.Template
	indexTmpl     *html.Template
	notFoundTmpl  *html.Template
	robotsTmpl    *text.Template
	sitemapTmpl   *text.Template
}

// Option configures a Generator during construction.
type Option func(*Generator) error

// WithTemplates configures user-provided template partials. User-defined
// {{define "head"}} and {{define "body"}} blocks override the built-in defaults.
func WithTemplates(tmplCfg config.TemplatesConfig) Option {
	return func(g *Generator) error {
		if !tmplCfg.HasCustomTemplates() {
			return nil
		}

		var err error

		g.moduleTmpl, err = applyUserPartials(g.moduleTmpl, tmplCfg.Module, tmplCfg.Partials)
		if err != nil {
			return fmt.Errorf("applying user module template: %w", err)
		}

		// Submodule falls back to module partial if not specified
		submodulePartial := tmplCfg.Submodule
		if submodulePartial == "" {
			submodulePartial = tmplCfg.Module
		}
		g.submoduleTmpl, err = applyUserPartials(g.submoduleTmpl, submodulePartial, tmplCfg.Partials)
		if err != nil {
			return fmt.Errorf("applying user submodule template: %w", err)
		}

		g.indexTmpl, err = applyUserPartials(g.indexTmpl, tmplCfg.Index, tmplCfg.Partials)
		if err != nil {
			return fmt.Errorf("applying user index template: %w", err)
		}

		g.notFoundTmpl, err = applyUserPartials(g.notFoundTmpl, tmplCfg.NotFound, tmplCfg.Partials)
		if err != nil {
			return fmt.Errorf("applying user not_found template: %w", err)
		}

		return nil
	}
}

// New parses embedded templates and returns a ready-to-use Generator.
// Use WithTemplates to apply user-provided template partials.
func New(opts ...Option) (*Generator, error) {
	moduleTmpl, err := html.New("module.html.tmpl").Funcs(funcMap).ParseFS(templateFS, "templates/module.html.tmpl")
	if err != nil {
		return nil, fmt.Errorf("parsing module template: %w", err)
	}

	submoduleTmpl, err := html.New("submodule.html.tmpl").Funcs(funcMap).ParseFS(templateFS, "templates/submodule.html.tmpl")
	if err != nil {
		return nil, fmt.Errorf("parsing submodule template: %w", err)
	}

	indexTmpl, err := html.New("index.html.tmpl").Funcs(funcMap).ParseFS(templateFS, "templates/index.html.tmpl")
	if err != nil {
		return nil, fmt.Errorf("parsing index template: %w", err)
	}

	notFoundTmpl, err := html.New("not_found.html.tmpl").Funcs(funcMap).ParseFS(templateFS, "templates/not_found.html.tmpl")
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

	g := &Generator{
		moduleTmpl:    moduleTmpl,
		submoduleTmpl: submoduleTmpl,
		indexTmpl:     indexTmpl,
		notFoundTmpl:  notFoundTmpl,
		robotsTmpl:    robotsTmpl,
		sitemapTmpl:   sitemapTmpl,
	}

	for _, opt := range opts {
		if err = opt(g); err != nil {
			return nil, err
		}
	}

	return g, nil
}

// applyUserPartials composes user template files with a base template.
// It clones the base, parses composition partials first, then the page
// partial. User {{define "head"}} and {{define "body"}} blocks override
// the base template's {{block}} defaults.
//
// If pagePartial is empty, only composition partials are applied.
// If both are empty, the base template is returned unchanged.
func applyUserPartials(base *html.Template, pagePartial string, compositionPartials []string) (*html.Template, error) {
	if pagePartial == "" && len(compositionPartials) == 0 {
		return base, nil
	}

	tmpl, err := base.Clone()
	if err != nil {
		return nil, fmt.Errorf("cloning base template: %w", err)
	}

	// Parse composition partials first so page partials can reference them
	for _, path := range compositionPartials {
		content, err := os.ReadFile(filepath.Clean(path))
		if err != nil {
			return nil, fmt.Errorf("reading partial %q: %w", path, err)
		}
		if _, err := tmpl.Parse(string(content)); err != nil {
			return nil, fmt.Errorf("parsing partial %q: %w", path, err)
		}
	}

	// Parse the page-specific partial (overrides head/body blocks)
	if pagePartial != "" {
		content, err := os.ReadFile(filepath.Clean(pagePartial))
		if err != nil {
			return nil, fmt.Errorf("reading template %q: %w", pagePartial, err)
		}
		if _, err := tmpl.Parse(string(content)); err != nil {
			return nil, fmt.Errorf("parsing template %q: %w", pagePartial, err)
		}
	}

	return tmpl, nil
}

// generateConfig controls how generated output is written.
type generateConfig struct {
	inMemory bool
	memFS    fstest.MapFS
}

// GenerateOption configures Generate behavior.
type GenerateOption func(*generateConfig)

// WithInMemory causes Generate to write to the provided MapFS instead of disk.
func WithInMemory(fs fstest.MapFS) GenerateOption {
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

		// Add subpackage data — each gets the same go-import root-path
		// pointing to the parent module, which is what the Go tool expects.
		for _, subpkg := range subpackages[m.Name] {
			md.Subpackages = append(md.Subpackages, ModuleData{
				Domain:     cfg.Domain,
				Name:       m.Name + "/" + subpkg,
				ImportPath: m.ImportPath(cfg.Domain),
				Repo:       m.Repo,
				Branch:     m.Branch,
				GoSource:   *m.GoSource,
				Redirect:   m.Redirect + "/" + subpkg,
			})
		}

		modules = append(modules, md)
	}

	return modules
}

// Generate renders all pages from the config. By default, it writes to disk
// using cfg.Output.Dir. Use WithInMemory to write to an in-memory filesystem instead.
//
//nolint:gocognit // we'll refactor this later
func (g *Generator) Generate(cfg *config.Config, subpackages map[string][]string, opts ...GenerateOption) error {
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

	// Generate module and subpackage pages
	for _, md := range modules {
		path := filepath.Join(md.Name, "index.html")
		if err := render(path, g.moduleTmpl, md); err != nil {
			return fmt.Errorf("generating module page %s: %w", md.Name, err)
		}
		slog.Debug("generated module page", "module", md.Name)

		for _, sub := range md.Subpackages {
			subPath := filepath.Join(sub.Name, "index.html")
			if err := render(subPath, g.submoduleTmpl, sub); err != nil {
				return fmt.Errorf("generating subpackage page %s: %w", sub.Name, err)
			}
			slog.Debug("generated subpackage page", "subpackage", sub.Name)
		}
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

	// Copy static assets
	if len(cfg.Templates.Assets) > 0 {
		if err := copyAssets(cfg.Templates.Assets, gc, cfg.Output.Dir); err != nil {
			return err
		}
	}

	return nil
}

// copyAssets copies static asset files and directories into the output directory.
// Assets are copied preserving their relative path structure.
func copyAssets(assets []string, gc *generateConfig, outputDir string) error {
	for _, assetPath := range assets {
		info, err := os.Stat(assetPath)
		if err != nil {
			return fmt.Errorf("asset %q: %w", assetPath, err)
		}

		if info.IsDir() {
			if err := copyDir(assetPath, gc, outputDir); err != nil {
				return fmt.Errorf("copying asset directory %q: %w", assetPath, err)
			}
		} else {
			if err := copyFile(assetPath, assetPath, gc, outputDir); err != nil {
				return fmt.Errorf("copying asset file %q: %w", assetPath, err)
			}
		}
	}
	return nil
}

// copyDir recursively copies a directory's contents into the output.
func copyDir(dirPath string, gc *generateConfig, outputDir string) error {
	return filepath.WalkDir(dirPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		return copyFile(path, path, gc, outputDir)
	})
}

// copyFile reads a source file and writes it to the output via generateConfig.writer.
func copyFile(srcPath, outputPath string, gc *generateConfig, outputDir string) error {
	data, err := os.ReadFile(filepath.Clean(srcPath))
	if err != nil {
		return fmt.Errorf("reading %q: %w", srcPath, err)
	}

	w, err := gc.writer(outputDir, outputPath)
	if err != nil {
		return fmt.Errorf("creating output for %q: %w", outputPath, err)
	}
	defer func() {
		if cerr := w.Close(); cerr != nil {
			slog.Debug("failed to close asset file", "error", cerr, "file", outputPath)
		}
	}()

	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("writing %q: %w", outputPath, err)
	}

	slog.Debug("copied asset", "path", outputPath)
	return nil
}
