package generator

import (
	"bytes"
	"flag"
	html "html/template"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.treyburn.dev/vanity/pkg/config"
)

var update = flag.Bool("update", false, "update golden files")

// minimalConfig returns a resolved config with a single module and all output enabled.
func minimalConfig() *config.Config {
	cfg := &config.Config{
		Log: config.LogConfig{Level: config.LogLevelInfo, Format: config.LogFormatText, Color: true},
		Output: config.OutputConfig{
			Dir:      "dist",
			Clean:    true,
			Index:    true,
			NotFound: true,
			Robots:   true,
			Sitemap:  true,
		},
		Domain: "go.example.com",
		Defaults: config.DefaultsConfig{
			Branch:       "main",
			GoSource:     true,
			RedirectRoot: "https://pkg.go.dev",
		},
		Modules: []config.Module{
			{
				Name:     "foo",
				Repo:     "https://github.com/example/foo",
				Branch:   "main",
				GoSource: new(true),
				Redirect: "https://pkg.go.dev/go.example.com/foo",
			},
		},
	}
	return cfg
}

func TestNew(t *testing.T) {
	gen, err := New()
	require.NoError(t, err)
	assert.NotNil(t, gen.moduleTmpl)
	assert.NotNil(t, gen.submoduleTmpl)
	assert.NotNil(t, gen.indexTmpl)
	assert.NotNil(t, gen.notFoundTmpl)
	assert.NotNil(t, gen.robotsTmpl)
	assert.NotNil(t, gen.sitemapTmpl)
}

func TestGenerate_SingleModule(t *testing.T) {
	cfg := minimalConfig()
	cfg.Output.Dir = filepath.Join(t.TempDir(), "dist")

	gen, err := New()
	require.NoError(t, err)

	err = gen.Generate(cfg, nil)
	require.NoError(t, err)

	assertGoldenFile(t, filepath.Join(cfg.Output.Dir, "foo", "index.html"), "single_module/foo_index.html")
	assertGoldenFile(t, filepath.Join(cfg.Output.Dir, "index.html"), "single_module/index.html")
	assertGoldenFile(t, filepath.Join(cfg.Output.Dir, "404.html"), "single_module/404.html")
	assertGoldenFile(t, filepath.Join(cfg.Output.Dir, "robots.txt"), "single_module/robots.txt")
	assertGoldenFile(t, filepath.Join(cfg.Output.Dir, "sitemap.xml"), "single_module/sitemap.xml")
}

func TestGenerate_GoSourceDisabled(t *testing.T) {
	cfg := minimalConfig()
	cfg.Output.Dir = filepath.Join(t.TempDir(), "dist")
	cfg.Modules[0].GoSource = new(false)

	gen, err := New()
	require.NoError(t, err)

	err = gen.Generate(cfg, nil)
	require.NoError(t, err)

	assertGoldenFile(t, filepath.Join(cfg.Output.Dir, "foo", "index.html"), "go_source_disabled/foo_index.html")
}

func TestGenerate_RedirectOverride(t *testing.T) {
	cfg := minimalConfig()
	cfg.Output.Dir = filepath.Join(t.TempDir(), "dist")
	cfg.Modules[0].Redirect = "https://custom.example.com/docs"

	gen, err := New()
	require.NoError(t, err)

	err = gen.Generate(cfg, nil)
	require.NoError(t, err)

	assertGoldenFile(t, filepath.Join(cfg.Output.Dir, "foo", "index.html"), "redirect_override/foo_index.html")
}

func TestGenerate_WithSubpackages(t *testing.T) {
	cfg := minimalConfig()
	cfg.Output.Dir = filepath.Join(t.TempDir(), "dist")

	subpackages := map[string][]string{
		"foo": {"cmd/tool", "pkg/lib"},
	}

	gen, err := New()
	require.NoError(t, err)

	err = gen.Generate(cfg, subpackages)
	require.NoError(t, err)

	// Module page
	assertGoldenFile(t, filepath.Join(cfg.Output.Dir, "foo", "index.html"), "with_subpackages/foo_index.html")
	// Subpackage pages
	assertGoldenFile(t, filepath.Join(cfg.Output.Dir, "foo", "cmd", "tool", "index.html"), "with_subpackages/foo_cmd_tool_index.html")
	assertGoldenFile(t, filepath.Join(cfg.Output.Dir, "foo", "pkg", "lib", "index.html"), "with_subpackages/foo_pkg_lib_index.html")
	// Sitemap should include subpackages
	assertGoldenFile(t, filepath.Join(cfg.Output.Dir, "sitemap.xml"), "with_subpackages/sitemap.xml")
}

func TestGenerate_MultipleModules(t *testing.T) {
	cfg := minimalConfig()
	cfg.Output.Dir = filepath.Join(t.TempDir(), "dist")
	cfg.Modules = append(cfg.Modules, config.Module{
		Name:     "bar",
		Repo:     "https://github.com/example/bar",
		Branch:   "develop",
		GoSource: new(true),
		Redirect: "https://pkg.go.dev/go.example.com/bar",
	})

	gen, err := New()
	require.NoError(t, err)

	err = gen.Generate(cfg, nil)
	require.NoError(t, err)

	assertGoldenFile(t, filepath.Join(cfg.Output.Dir, "foo", "index.html"), "multiple_modules/foo_index.html")
	assertGoldenFile(t, filepath.Join(cfg.Output.Dir, "bar", "index.html"), "multiple_modules/bar_index.html")
	assertGoldenFile(t, filepath.Join(cfg.Output.Dir, "index.html"), "multiple_modules/index.html")
	assertGoldenFile(t, filepath.Join(cfg.Output.Dir, "sitemap.xml"), "multiple_modules/sitemap.xml")
}

func TestGenerate_OptionalOutputsDisabled(t *testing.T) {
	cfg := minimalConfig()
	cfg.Output.Dir = filepath.Join(t.TempDir(), "dist")
	cfg.Output.Index = false
	cfg.Output.NotFound = false
	cfg.Output.Robots = false
	cfg.Output.Sitemap = false

	gen, err := New()
	require.NoError(t, err)

	err = gen.Generate(cfg, nil)
	require.NoError(t, err)

	// Module page should still exist
	assert.FileExists(t, filepath.Join(cfg.Output.Dir, "foo", "index.html"))

	// Optional files should not exist
	assert.NoFileExists(t, filepath.Join(cfg.Output.Dir, "index.html"))
	assert.NoFileExists(t, filepath.Join(cfg.Output.Dir, "404.html"))
	assert.NoFileExists(t, filepath.Join(cfg.Output.Dir, "robots.txt"))
	assert.NoFileExists(t, filepath.Join(cfg.Output.Dir, "sitemap.xml"))
}

func TestGenerate_CleanRemovesExistingDir(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, "dist")

	// Pre-create a file that should be cleaned
	require.NoError(t, os.MkdirAll(filepath.Join(outputDir, "stale"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(outputDir, "stale", "old.html"), []byte("old"), 0o644))

	cfg := minimalConfig()
	cfg.Output.Dir = outputDir
	cfg.Output.Clean = true

	gen, err := New()
	require.NoError(t, err)

	err = gen.Generate(cfg, nil)
	require.NoError(t, err)

	// Stale file should be gone
	assert.NoFileExists(t, filepath.Join(outputDir, "stale", "old.html"))
	// New files should exist
	assert.FileExists(t, filepath.Join(outputDir, "foo", "index.html"))
}

func TestGenerate_NestedOutputDir(t *testing.T) {
	cfg := minimalConfig()
	cfg.Output.Dir = filepath.Join(t.TempDir(), "some", "dir", "dist")

	gen, err := New()
	require.NoError(t, err)

	err = gen.Generate(cfg, nil)
	require.NoError(t, err)

	assert.FileExists(t, filepath.Join(cfg.Output.Dir, "foo", "index.html"))
	assert.FileExists(t, filepath.Join(cfg.Output.Dir, "index.html"))
	assert.FileExists(t, filepath.Join(cfg.Output.Dir, "404.html"))
}

func TestGenerate_InMemory(t *testing.T) {
	cfg := minimalConfig()

	gen, err := New()
	require.NoError(t, err)

	memFS := make(fstest.MapFS)
	err = gen.Generate(cfg, nil, WithInMemory(memFS))
	require.NoError(t, err)

	// Verify all expected files exist
	assertMemFSFile(t, memFS, "foo/index.html")
	assertMemFSFile(t, memFS, "index.html")
	assertMemFSFile(t, memFS, "404.html")
	assertMemFSFile(t, memFS, "robots.txt")
	assertMemFSFile(t, memFS, "sitemap.xml")
}

func TestGenerate_InMemoryMatchesDisk(t *testing.T) {
	cfg := minimalConfig()
	diskDir := filepath.Join(t.TempDir(), "dist")
	cfg.Output.Dir = diskDir

	gen, err := New()
	require.NoError(t, err)

	// Generate to disk
	err = gen.Generate(cfg, nil)
	require.NoError(t, err)

	// Generate to memory
	memFS := make(fstest.MapFS)
	err = gen.Generate(cfg, nil, WithInMemory(memFS))
	require.NoError(t, err)

	// Compare content of each file
	files := []string{"foo/index.html", "index.html", "404.html", "robots.txt", "sitemap.xml"}
	for _, f := range files {
		diskContent, err := os.ReadFile(filepath.Join(diskDir, f))
		require.NoError(t, err, "reading disk file %s", f)

		memContent, err := fs.ReadFile(memFS, f)
		require.NoError(t, err, "reading memory file %s", f)

		assert.Equal(t, string(diskContent), string(memContent), "content mismatch for %s", f)
	}
}

func TestBuildModuleData_SubpackageImportPath(t *testing.T) {
	cfg := minimalConfig()
	subpackages := map[string][]string{
		"foo": {"cmd/tool"},
	}

	modules := buildModuleData(cfg, subpackages)
	require.Len(t, modules, 1)
	require.Len(t, modules[0].Subpackages, 1)

	// Subpackage should have the parent module's import path (for go-import root-path)
	sub := modules[0].Subpackages[0]
	assert.Equal(t, "foo/cmd/tool", sub.Name)
	assert.Equal(t, "go.example.com/foo", sub.ImportPath, "subpackage go-import must point to module root")
	assert.Equal(t, "https://pkg.go.dev/go.example.com/foo/cmd/tool", sub.Redirect)
}

func TestFuncMap_Available(t *testing.T) {
	// Verify that all expected functions are registered in the FuncMap
	expectedFuncs := []string{
		"upper", "lower", "title", "join", "sprintf",
		"now", "year", "contains", "hasPrefix", "hasSuffix",
		"replace", "trimSpace",
	}
	for _, name := range expectedFuncs {
		assert.Contains(t, funcMap, name, "FuncMap should contain %q", name)
	}
}

func TestBlockFallback_DefaultBodyUsed(t *testing.T) {
	// Without any user overrides, the default block content should be used.
	// This is already covered by all the golden file tests, but let's
	// explicitly verify the body contains expected default content.
	cfg := minimalConfig()
	memFS := make(fstest.MapFS)

	gen, err := New()
	require.NoError(t, err)

	err = gen.Generate(cfg, nil, WithInMemory(memFS))
	require.NoError(t, err)

	// Module page should have the default redirect text
	moduleContent, err := fs.ReadFile(memFS, "foo/index.html")
	require.NoError(t, err)
	assert.Contains(t, string(moduleContent), "Redirecting to")

	// Index page should have the default module listing
	indexContent, err := fs.ReadFile(memFS, "index.html")
	require.NoError(t, err)
	assert.Contains(t, string(indexContent), "<h1>go.example.com</h1>")

	// 404 page should have the default "page not found" text
	notFoundContent, err := fs.ReadFile(memFS, "404.html")
	require.NoError(t, err)
	assert.Contains(t, string(notFoundContent), "Page not found")
}

func TestBlockFallback_HeadBlockEmpty(t *testing.T) {
	// The default head block should produce no extra content in <head>
	cfg := minimalConfig()
	memFS := make(fstest.MapFS)

	gen, err := New()
	require.NoError(t, err)

	err = gen.Generate(cfg, nil, WithInMemory(memFS))
	require.NoError(t, err)

	// The module page head should not have any extra content beyond
	// the go-import/go-source/refresh meta tags
	moduleContent, err := fs.ReadFile(memFS, "foo/index.html")
	require.NoError(t, err)
	content := string(moduleContent)

	// Head block is empty by default — verify no extra blank lines or content
	// between the last meta tag and </head>
	assert.Contains(t, content, `<meta http-equiv="refresh"`)
	assert.Contains(t, content, `</head>`)
}

func TestSubmoduleTemplate_UsedForSubpackages(t *testing.T) {
	// Verify that subpackage pages are rendered with the submodule template
	cfg := minimalConfig()
	memFS := make(fstest.MapFS)

	subpackages := map[string][]string{
		"foo": {"cmd/tool"},
	}

	gen, err := New()
	require.NoError(t, err)

	err = gen.Generate(cfg, subpackages, WithInMemory(memFS))
	require.NoError(t, err)

	// Both module and submodule pages should exist and contain redirect content
	moduleContent, err := fs.ReadFile(memFS, "foo/index.html")
	require.NoError(t, err)
	assert.Contains(t, string(moduleContent), "Redirecting to")

	subContent, err := fs.ReadFile(memFS, "foo/cmd/tool/index.html")
	require.NoError(t, err)
	assert.Contains(t, string(subContent), "Redirecting to")
}

// --- applyUserPartials tests ---

func baseTemplate(t *testing.T) *html.Template {
	t.Helper()
	tmpl, err := html.New("test").Funcs(funcMap).Parse(
		`<head>{{block "head" .}}{{end}}</head><body>{{block "body" .}}default{{end}}</body>`)
	require.NoError(t, err)
	return tmpl
}

func writeFixture(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

func TestApplyUserPartials_NoOverrides(t *testing.T) {
	base := baseTemplate(t)
	result, err := applyUserPartials(base, "", nil)
	require.NoError(t, err)
	// Should return the same pointer — no clone needed
	assert.Equal(t, base, result)
}

func TestApplyUserPartials_BodyOverride(t *testing.T) {
	dir := t.TempDir()
	partial := writeFixture(t, dir, "body.html", `{{define "body"}}custom body{{end}}`)

	base := baseTemplate(t)
	result, err := applyUserPartials(base, partial, nil)
	require.NoError(t, err)

	var buf bytes.Buffer
	require.NoError(t, result.Execute(&buf, nil))
	assert.Contains(t, buf.String(), "custom body")
	assert.NotContains(t, buf.String(), "default")
}

func TestApplyUserPartials_HeadOverride(t *testing.T) {
	dir := t.TempDir()
	partial := writeFixture(t, dir, "head.html", `{{define "head"}}<link rel="stylesheet" href="/css/style.css">{{end}}`)

	base := baseTemplate(t)
	result, err := applyUserPartials(base, partial, nil)
	require.NoError(t, err)

	var buf bytes.Buffer
	require.NoError(t, result.Execute(&buf, nil))
	assert.Contains(t, buf.String(), `<link rel="stylesheet"`)
	// Body should still be the default
	assert.Contains(t, buf.String(), "default")
}

func TestApplyUserPartials_CompositionOnly(t *testing.T) {
	dir := t.TempDir()
	headerPartial := writeFixture(t, dir, "header.html", `{{define "header"}}<nav>header</nav>{{end}}`)
	bodyPartial := writeFixture(t, dir, "body.html", `{{define "body"}}{{template "header" .}}main{{end}}`)

	base := baseTemplate(t)
	result, err := applyUserPartials(base, bodyPartial, []string{headerPartial})
	require.NoError(t, err)

	var buf bytes.Buffer
	require.NoError(t, result.Execute(&buf, nil))
	assert.Contains(t, buf.String(), "<nav>header</nav>")
	assert.Contains(t, buf.String(), "main")
}

func TestApplyUserPartials_DoesNotMutateBase(t *testing.T) {
	dir := t.TempDir()
	partial := writeFixture(t, dir, "body.html", `{{define "body"}}overridden{{end}}`)

	base := baseTemplate(t)

	// Apply override
	_, err := applyUserPartials(base, partial, nil)
	require.NoError(t, err)

	// Original base should still render the default
	var buf bytes.Buffer
	require.NoError(t, base.Execute(&buf, nil))
	assert.Contains(t, buf.String(), "default")
	assert.NotContains(t, buf.String(), "overridden")
}

func TestApplyUserPartials_FileNotFound(t *testing.T) {
	base := baseTemplate(t)
	_, err := applyUserPartials(base, "/nonexistent/body.html", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reading template")
}

func TestApplyUserPartials_InvalidSyntax(t *testing.T) {
	dir := t.TempDir()
	partial := writeFixture(t, dir, "bad.html", `{{define "body"}}`)

	base := baseTemplate(t)
	_, err := applyUserPartials(base, partial, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing template")
}

func TestApplyUserPartials_CompositionPartialNotFound(t *testing.T) {
	base := baseTemplate(t)
	_, err := applyUserPartials(base, "", []string{"/nonexistent/header.html"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reading partial")
}

// --- WithTemplates tests ---

func testGenerator(t *testing.T) *Generator {
	t.Helper()
	base := baseTemplate(t)
	// Clone for each field so they're independent
	module, err := base.Clone()
	require.NoError(t, err)
	submodule, err := base.Clone()
	require.NoError(t, err)
	index, err := base.Clone()
	require.NoError(t, err)
	notFound, err := base.Clone()
	require.NoError(t, err)
	return &Generator{
		moduleTmpl:    module,
		submoduleTmpl: submodule,
		indexTmpl:     index,
		notFoundTmpl:  notFound,
	}
}

func TestWithTemplates_NoTemplates(t *testing.T) {
	gen := testGenerator(t)
	opt := WithTemplates(config.TemplatesConfig{})
	require.NoError(t, opt(gen))
}

func TestWithTemplates_InvalidPath(t *testing.T) {
	gen := testGenerator(t)
	opt := WithTemplates(config.TemplatesConfig{
		Module: "/nonexistent/module.html",
	})
	err := opt(gen)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "applying user module template")
}

func TestWithTemplates_InvalidSubmodulePath(t *testing.T) {
	gen := testGenerator(t)
	opt := WithTemplates(config.TemplatesConfig{
		Submodule: "/nonexistent/submodule.html",
	})
	err := opt(gen)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "applying user submodule template")
}

func TestWithTemplates_InvalidIndexPath(t *testing.T) {
	gen := testGenerator(t)
	opt := WithTemplates(config.TemplatesConfig{
		Index: "/nonexistent/index.html",
	})
	err := opt(gen)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "applying user index template")
}

func TestWithTemplates_InvalidNotFoundPath(t *testing.T) {
	gen := testGenerator(t)
	opt := WithTemplates(config.TemplatesConfig{
		NotFound: "/nonexistent/404.html",
	})
	err := opt(gen)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "applying user not_found template")
}

func TestWithTemplates_SubmoduleFallback(t *testing.T) {
	dir := t.TempDir()
	modulePartial := writeFixture(t, dir, "module.html", `{{define "body"}}module-style{{end}}`)

	gen := testGenerator(t)
	opt := WithTemplates(config.TemplatesConfig{
		Module: modulePartial,
		// Submodule not set — should fall back to Module
	})
	require.NoError(t, opt(gen))

	var buf bytes.Buffer
	require.NoError(t, gen.submoduleTmpl.Execute(&buf, nil))
	assert.Contains(t, buf.String(), "module-style")
}

func TestWithTemplates_AppliesOverrides(t *testing.T) {
	dir := t.TempDir()
	modulePartial := writeFixture(t, dir, "module.html", `{{define "body"}}custom-module{{end}}`)
	indexPartial := writeFixture(t, dir, "index.html", `{{define "body"}}custom-index{{end}}`)
	notFoundPartial := writeFixture(t, dir, "404.html", `{{define "body"}}custom-404{{end}}`)

	gen := testGenerator(t)
	opt := WithTemplates(config.TemplatesConfig{
		Module:   modulePartial,
		Index:    indexPartial,
		NotFound: notFoundPartial,
	})
	require.NoError(t, opt(gen))

	for _, tc := range []struct {
		name     string
		tmpl     *html.Template
		expected string
	}{
		{"module", gen.moduleTmpl, "custom-module"},
		{"index", gen.indexTmpl, "custom-index"},
		{"not_found", gen.notFoundTmpl, "custom-404"},
		{"submodule fallback", gen.submoduleTmpl, "custom-module"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			require.NoError(t, tc.tmpl.Execute(&buf, nil))
			assert.Contains(t, buf.String(), tc.expected)
		})
	}
}

// --- Custom template tests ---

func fixturesDir() string {
	return filepath.Join("testdata", "custom_templates", "fixtures")
}

func TestGenerate_CustomModuleBody(t *testing.T) {
	cfg := minimalConfig()
	cfg.Output.Dir = filepath.Join(t.TempDir(), "dist")

	tmplCfg := config.TemplatesConfig{
		Module: filepath.Join(fixturesDir(), "module_body.html"),
	}

	gen, err := New(WithTemplates(tmplCfg))
	require.NoError(t, err)

	err = gen.Generate(cfg, nil)
	require.NoError(t, err)

	assertGoldenFile(t, filepath.Join(cfg.Output.Dir, "foo", "index.html"), "custom_templates/custom_module_body.html")
}

func TestGenerate_CustomHeadAndBody(t *testing.T) {
	cfg := minimalConfig()
	cfg.Output.Dir = filepath.Join(t.TempDir(), "dist")

	tmplCfg := config.TemplatesConfig{
		Module: filepath.Join(fixturesDir(), "module_head_and_body.html"),
	}

	gen, err := New(WithTemplates(tmplCfg))
	require.NoError(t, err)

	err = gen.Generate(cfg, nil)
	require.NoError(t, err)

	assertGoldenFile(t, filepath.Join(cfg.Output.Dir, "foo", "index.html"), "custom_templates/custom_head_and_body.html")
}

func TestGenerate_CustomIndex(t *testing.T) {
	cfg := minimalConfig()
	cfg.Output.Dir = filepath.Join(t.TempDir(), "dist")

	tmplCfg := config.TemplatesConfig{
		Index: filepath.Join(fixturesDir(), "index_body.html"),
	}

	gen, err := New(WithTemplates(tmplCfg))
	require.NoError(t, err)

	err = gen.Generate(cfg, nil)
	require.NoError(t, err)

	assertGoldenFile(t, filepath.Join(cfg.Output.Dir, "index.html"), "custom_templates/custom_index.html")
}

func TestGenerate_SubmoduleFallbackToModule(t *testing.T) {
	// When only module template is specified, submodules should use it too
	cfg := minimalConfig()
	cfg.Output.Dir = filepath.Join(t.TempDir(), "dist")

	tmplCfg := config.TemplatesConfig{
		Module: filepath.Join(fixturesDir(), "module_body.html"),
	}

	subpackages := map[string][]string{
		"foo": {"cmd/tool"},
	}

	gen, err := New(WithTemplates(tmplCfg))
	require.NoError(t, err)

	err = gen.Generate(cfg, subpackages)
	require.NoError(t, err)

	// Submodule page should use the module template's body override
	assertGoldenFile(t, filepath.Join(cfg.Output.Dir, "foo", "cmd", "tool", "index.html"), "custom_templates/submodule_fallback.html")
}

func TestGenerate_SeparateSubmoduleTemplate(t *testing.T) {
	// When both module and submodule templates are specified, each uses its own
	cfg := minimalConfig()
	cfg.Output.Dir = filepath.Join(t.TempDir(), "dist")

	tmplCfg := config.TemplatesConfig{
		Module:    filepath.Join(fixturesDir(), "module_body.html"),
		Submodule: filepath.Join(fixturesDir(), "submodule_body.html"),
	}

	subpackages := map[string][]string{
		"foo": {"cmd/tool"},
	}

	gen, err := New(WithTemplates(tmplCfg))
	require.NoError(t, err)

	err = gen.Generate(cfg, subpackages)
	require.NoError(t, err)

	// Module uses module template
	assertGoldenFile(t, filepath.Join(cfg.Output.Dir, "foo", "index.html"), "custom_templates/custom_module_body.html")
	// Submodule uses its own template
	assertGoldenFile(t, filepath.Join(cfg.Output.Dir, "foo", "cmd", "tool", "index.html"), "custom_templates/separate_submodule.html")
}

func TestGenerate_CompositionPartials(t *testing.T) {
	cfg := minimalConfig()
	cfg.Output.Dir = filepath.Join(t.TempDir(), "dist")

	tmplCfg := config.TemplatesConfig{
		Module: filepath.Join(fixturesDir(), "module_with_composition.html"),
		Partials: []string{
			filepath.Join(fixturesDir(), "header.html"),
			filepath.Join(fixturesDir(), "footer.html"),
		},
	}

	gen, err := New(WithTemplates(tmplCfg))
	require.NoError(t, err)

	err = gen.Generate(cfg, nil)
	require.NoError(t, err)

	assertGoldenFile(t, filepath.Join(cfg.Output.Dir, "foo", "index.html"), "custom_templates/composition.html")
}

func TestGenerate_CustomTemplateDoesNotAffectDefaults(t *testing.T) {
	// Specifying a module template should not affect index/404/robots/sitemap defaults
	cfg := minimalConfig()
	cfg.Output.Dir = filepath.Join(t.TempDir(), "dist")

	tmplCfg := config.TemplatesConfig{
		Module: filepath.Join(fixturesDir(), "module_body.html"),
	}

	gen, err := New(WithTemplates(tmplCfg))
	require.NoError(t, err)

	err = gen.Generate(cfg, nil)
	require.NoError(t, err)

	// These should match the original golden files (unchanged defaults)
	assertGoldenFile(t, filepath.Join(cfg.Output.Dir, "index.html"), "single_module/index.html")
	assertGoldenFile(t, filepath.Join(cfg.Output.Dir, "404.html"), "single_module/404.html")
	assertGoldenFile(t, filepath.Join(cfg.Output.Dir, "robots.txt"), "single_module/robots.txt")
	assertGoldenFile(t, filepath.Join(cfg.Output.Dir, "sitemap.xml"), "single_module/sitemap.xml")
}

// --- Asset copying tests ---

func TestGenerate_CopiesAssetFile(t *testing.T) {
	dir := t.TempDir()
	cssFile := filepath.Join(dir, "style.css")
	require.NoError(t, os.WriteFile(cssFile, []byte("body { color: red; }"), 0o644))

	cfg := minimalConfig()
	cfg.Output.Dir = filepath.Join(t.TempDir(), "dist")
	cfg.Templates.Assets = []string{cssFile}

	gen, err := New()
	require.NoError(t, err)

	require.NoError(t, gen.Generate(cfg, nil))

	// Asset should be copied preserving its path
	actual, err := os.ReadFile(filepath.Join(cfg.Output.Dir, cssFile))
	require.NoError(t, err)
	assert.Equal(t, "body { color: red; }", string(actual))
}

func TestGenerate_CopiesAssetDirectory(t *testing.T) {
	dir := t.TempDir()
	cssDir := filepath.Join(dir, "css")
	require.NoError(t, os.MkdirAll(cssDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(cssDir, "main.css"), []byte("h1{}"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(cssDir, "reset.css"), []byte("*{}"), 0o644))

	cfg := minimalConfig()
	cfg.Output.Dir = filepath.Join(t.TempDir(), "dist")
	cfg.Templates.Assets = []string{cssDir}

	gen, err := New()
	require.NoError(t, err)

	require.NoError(t, gen.Generate(cfg, nil))

	mainCSS, err := os.ReadFile(filepath.Join(cfg.Output.Dir, cssDir, "main.css"))
	require.NoError(t, err)
	assert.Equal(t, "h1{}", string(mainCSS))

	resetCSS, err := os.ReadFile(filepath.Join(cfg.Output.Dir, cssDir, "reset.css"))
	require.NoError(t, err)
	assert.Equal(t, "*{}", string(resetCSS))
}

func TestGenerate_CopiesAssetsInMemory(t *testing.T) {
	dir := t.TempDir()
	jsFile := filepath.Join(dir, "app.js")
	require.NoError(t, os.WriteFile(jsFile, []byte("console.log('hi')"), 0o644))

	cfg := minimalConfig()
	cfg.Templates.Assets = []string{jsFile}
	memFS := make(fstest.MapFS)

	gen, err := New()
	require.NoError(t, err)

	require.NoError(t, gen.Generate(cfg, nil, WithInMemory(memFS)))

	// In memory mode, assets are stored by their path as-is
	entry, ok := memFS[jsFile]
	require.True(t, ok, "asset should exist in memFS at key %q", jsFile)
	assert.Equal(t, "console.log('hi')", string(entry.Data))
}

func TestGenerate_AssetWithNestedDirs(t *testing.T) {
	dir := t.TempDir()
	nestedDir := filepath.Join(dir, "static", "img", "icons")
	require.NoError(t, os.MkdirAll(nestedDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(nestedDir, "logo.svg"), []byte("<svg/>"), 0o644))

	cfg := minimalConfig()
	cfg.Output.Dir = filepath.Join(t.TempDir(), "dist")
	cfg.Templates.Assets = []string{filepath.Join(dir, "static")}

	gen, err := New()
	require.NoError(t, err)

	require.NoError(t, gen.Generate(cfg, nil))

	logo, err := os.ReadFile(filepath.Join(cfg.Output.Dir, dir, "static", "img", "icons", "logo.svg"))
	require.NoError(t, err)
	assert.Equal(t, "<svg/>", string(logo))
}

func TestGenerate_NoAssetsNoop(t *testing.T) {
	// No assets configured — should work fine (regression guard)
	cfg := minimalConfig()
	memFS := make(fstest.MapFS)

	gen, err := New()
	require.NoError(t, err)

	require.NoError(t, gen.Generate(cfg, nil, WithInMemory(memFS)))
	// Only generated files, no extra entries
	assert.Len(t, memFS, 5) // foo/index.html, index.html, 404.html, robots.txt, sitemap.xml
}

func TestGenerate_MixedFileAndDirAssets(t *testing.T) {
	dir := t.TempDir()

	// A standalone file
	standaloneFile := filepath.Join(dir, "favicon.ico")
	require.NoError(t, os.WriteFile(standaloneFile, []byte("icon"), 0o644))

	// A directory with files
	cssDir := filepath.Join(dir, "css")
	require.NoError(t, os.MkdirAll(cssDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(cssDir, "style.css"), []byte("body{}"), 0o644))

	cfg := minimalConfig()
	cfg.Templates.Assets = []string{standaloneFile, cssDir}
	memFS := make(fstest.MapFS)

	gen, err := New()
	require.NoError(t, err)

	require.NoError(t, gen.Generate(cfg, nil, WithInMemory(memFS)))

	// Both should be in the memFS
	faviconEntry, ok := memFS[standaloneFile]
	require.True(t, ok, "favicon should exist in memFS")
	assert.Equal(t, "icon", string(faviconEntry.Data))

	cssEntry, ok := memFS[filepath.Join(cssDir, "style.css")]
	require.True(t, ok, "style.css should exist in memFS")
	assert.Equal(t, "body{}", string(cssEntry.Data))
}

// assertGoldenFile compares the content of actualPath against a golden file in testdata/.
// When -update is set, it writes the actual content as the new golden file.
func assertGoldenFile(t *testing.T, actualPath, goldenName string) {
	t.Helper()

	actual, err := os.ReadFile(actualPath)
	require.NoError(t, err, "reading actual file %s", actualPath)

	goldenPath := filepath.Join("testdata", goldenName)

	if *update {
		require.NoError(t, os.MkdirAll(filepath.Dir(goldenPath), 0o755))
		require.NoError(t, os.WriteFile(goldenPath, actual, 0o644))
		return
	}

	expected, err := os.ReadFile(goldenPath)
	require.NoError(t, err, "reading golden file %s (run with -update to create)", goldenPath)

	assert.Equal(t, string(expected), string(actual), "mismatch with golden file %s", goldenName)
}

// assertMemFSFile verifies that a file exists and is non-empty in the given fs.FS.
func assertMemFSFile(t *testing.T, memFS fs.FS, path string) {
	t.Helper()
	content, err := fs.ReadFile(memFS, path)
	require.NoError(t, err, "reading %s from memory FS", path)
	assert.NotEmpty(t, content, "%s should not be empty", path)
}
