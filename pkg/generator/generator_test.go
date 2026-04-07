package generator

import (
	"flag"
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
