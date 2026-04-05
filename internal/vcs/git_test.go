package vcs

import (
	"path/filepath"
	"testing"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindGoPackages_Basic(t *testing.T) {
	bfs := memfs.New()
	createFile(t, bfs, "main.go")
	createFile(t, bfs, "cmd/tool/main.go")
	createFile(t, bfs, "pkg/lib/lib.go")
	createFile(t, bfs, "pkg/lib/lib_test.go")

	pkgs, err := findGoPackages(bfs, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"cmd/tool", "pkg/lib"}, pkgs)
}

func TestFindGoPackages_SkipsRootPackage(t *testing.T) {
	bfs := memfs.New()
	createFile(t, bfs, "main.go")

	pkgs, err := findGoPackages(bfs, nil)
	require.NoError(t, err)
	assert.Empty(t, pkgs)
}

func TestFindGoPackages_SkipsTestOnlyPackages(t *testing.T) {
	bfs := memfs.New()
	createFile(t, bfs, "main.go")
	createFile(t, bfs, "testpkg/foo_test.go")

	pkgs, err := findGoPackages(bfs, nil)
	require.NoError(t, err)
	assert.Empty(t, pkgs)
}

func TestFindGoPackages_SkipsVendor(t *testing.T) {
	bfs := memfs.New()
	createFile(t, bfs, "main.go")
	createFile(t, bfs, "vendor/dep/dep.go")
	createFile(t, bfs, "cmd/tool/main.go")

	pkgs, err := findGoPackages(bfs, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"cmd/tool"}, pkgs)
}

func TestFindGoPackages_SkipsTestdata(t *testing.T) {
	bfs := memfs.New()
	createFile(t, bfs, "main.go")
	createFile(t, bfs, "testdata/fixture/fix.go")
	createFile(t, bfs, "pkg/lib/lib.go")

	pkgs, err := findGoPackages(bfs, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"pkg/lib"}, pkgs)
}

func TestFindGoPackages_SkipsDotDirs(t *testing.T) {
	bfs := memfs.New()
	createFile(t, bfs, "main.go")
	createFile(t, bfs, ".hidden/secret.go")
	createFile(t, bfs, "pkg/lib/lib.go")

	pkgs, err := findGoPackages(bfs, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"pkg/lib"}, pkgs)
}

func TestFindGoPackages_ExcludePatterns(t *testing.T) {
	bfs := memfs.New()
	createFile(t, bfs, "main.go")
	createFile(t, bfs, "internal/core/core.go")
	createFile(t, bfs, "cmd/tool/main.go")
	createFile(t, bfs, "pkg/lib/lib.go")

	pkgs, err := findGoPackages(bfs, []string{"internal/*"})
	require.NoError(t, err)
	assert.Equal(t, []string{"cmd/tool", "pkg/lib"}, pkgs)
}

func TestFindGoPackages_MultipleExcludePatterns(t *testing.T) {
	bfs := memfs.New()
	createFile(t, bfs, "main.go")
	createFile(t, bfs, "internal/core/core.go")
	createFile(t, bfs, "cmd/tool/main.go")
	createFile(t, bfs, "pkg/lib/lib.go")

	pkgs, err := findGoPackages(bfs, []string{"internal/*", "cmd/*"})
	require.NoError(t, err)
	assert.Equal(t, []string{"pkg/lib"}, pkgs)
}

func TestFindGoPackages_NonGoFilesIgnored(t *testing.T) {
	bfs := memfs.New()
	createFile(t, bfs, "main.go")
	createFile(t, bfs, "docs/readme.md")
	createFile(t, bfs, "assets/style.css")

	pkgs, err := findGoPackages(bfs, nil)
	require.NoError(t, err)
	assert.Empty(t, pkgs)
}

func TestMatchesAny(t *testing.T) {
	assert.True(t, matchesAny("internal/core", []string{"internal/*"}))
	assert.False(t, matchesAny("cmd/tool", []string{"internal/*"}))
	assert.True(t, matchesAny("cmd/tool", []string{"cmd/*", "internal/*"}))
	assert.False(t, matchesAny("pkg/lib", []string{"internal/*"}))
}

func TestShouldSkipDir(t *testing.T) {
	assert.True(t, shouldSkipDir("vendor"))
	assert.True(t, shouldSkipDir("testdata"))
	assert.True(t, shouldSkipDir(".git"))
	assert.True(t, shouldSkipDir(".hidden"))
	assert.False(t, shouldSkipDir("cmd"))
	assert.False(t, shouldSkipDir("internal"))
	assert.False(t, shouldSkipDir("pkg"))
	assert.True(t, shouldSkipDir("."))
	assert.True(t, shouldSkipDir(".."))
}

func TestIsGoSourceFile(t *testing.T) {
	assert.True(t, isGoSourceFile("main.go"))
	assert.True(t, isGoSourceFile("pkg/lib/lib.go"))
	assert.False(t, isGoSourceFile("pkg/lib/lib_test.go"))
	assert.False(t, isGoSourceFile("readme.md"))
	assert.False(t, isGoSourceFile("style.css"))
}

// createFile creates a file in a billy filesystem, creating parent dirs as needed.
func createFile(t *testing.T, bfs billy.Filesystem, path string) {
	t.Helper()
	dir := filepath.Dir(path)
	if dir != "." {
		require.NoError(t, bfs.MkdirAll(dir, 0o755))
	}
	f, err := bfs.Create(path)
	require.NoError(t, err)
	require.NoError(t, f.Close())
}
