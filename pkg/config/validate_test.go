package config

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validConfig() *Config {
	cfg := DefaultConfig()
	cfg.Domain = "go.example.com"
	cfg.Modules = []Module{
		{Name: "foo", Repo: "https://github.com/example/foo"},
	}
	return cfg
}

func TestValidateBasic_ValidConfig(t *testing.T) {
	err := ValidateBasic(validConfig())
	assert.NoError(t, err)
}

func TestValidateBasic_MissingDomain(t *testing.T) {
	cfg := validConfig()
	cfg.Domain = ""

	err := ValidateBasic(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "domain is required")
	assertIsValidationErr(t, err)
}

func TestValidateBasic_NoModules(t *testing.T) {
	cfg := validConfig()
	cfg.Modules = nil

	err := ValidateBasic(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one module is required")
	assertIsValidationErr(t, err)
}

func TestValidateBasic_MissingModuleName(t *testing.T) {
	cfg := validConfig()
	cfg.Modules = []Module{{Name: "", Repo: "https://github.com/example/foo"}}

	err := ValidateBasic(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "modules[0].name is required")
}

func TestValidateBasic_MissingModuleRepo(t *testing.T) {
	cfg := validConfig()
	cfg.Modules = []Module{{Name: "foo", Repo: ""}}

	err := ValidateBasic(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "modules[0].repo is required")
}

func TestValidateBasic_InvalidRepoURL_NotURL(t *testing.T) {
	cfg := validConfig()
	cfg.Modules = []Module{{Name: "foo", Repo: "https://this is not a URL"}}

	err := ValidateBasic(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid URL: parse")
}

func TestValidateBasic_InvalidRepoURL_NoScheme(t *testing.T) {
	cfg := validConfig()
	cfg.Modules = []Module{{Name: "foo", Repo: "github.com/example/foo"}}

	err := ValidateBasic(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing scheme")
}

func TestValidateBasic_InvalidRepoURL_NoHost(t *testing.T) {
	cfg := validConfig()
	cfg.Modules = []Module{{Name: "foo", Repo: "https://"}}

	err := ValidateBasic(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing host")
}

func TestValidateBasic_DuplicateModuleNames(t *testing.T) {
	cfg := validConfig()
	cfg.Modules = []Module{
		{Name: "foo", Repo: "https://github.com/example/foo"},
		{Name: "foo", Repo: "https://github.com/example/foo2"},
	}

	err := ValidateBasic(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `duplicate module name: "foo"`)
}

func TestValidateBasic_DuplicateRepoURLs(t *testing.T) {
	cfg := validConfig()
	cfg.Modules = []Module{
		{Name: "foo", Repo: "https://github.com/example/foo"},
		{Name: "bar", Repo: "https://github.com/example/foo"},
	}

	err := ValidateBasic(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `duplicate repo URL: "https://github.com/example/foo"`)
}

func TestValidateBasic_InvalidSubpackageMode(t *testing.T) {
	cfg := validConfig()
	cfg.Modules = []Module{
		{
			Name: "foo",
			Repo: "https://github.com/example/foo",
			Subpackages: &SubpackageConfig{
				Mode: "invalid",
			},
		},
	}

	err := ValidateBasic(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "subpackages.mode: unknown value")
}

func TestValidateBasic_ExplicitModeRequiresPaths(t *testing.T) {
	cfg := validConfig()
	cfg.Modules = []Module{
		{
			Name: "foo",
			Repo: "https://github.com/example/foo",
			Subpackages: &SubpackageConfig{
				Mode:  SubpackageModeExplicit,
				Paths: nil,
			},
		},
	}

	err := ValidateBasic(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "subpackages.paths is required when mode is explicit")
}

func TestValidateBasic_ExplicitModeWithPaths(t *testing.T) {
	cfg := validConfig()
	cfg.Modules = []Module{
		{
			Name: "foo",
			Repo: "https://github.com/example/foo",
			Subpackages: &SubpackageConfig{
				Mode:  SubpackageModeExplicit,
				Paths: []string{"cmd/tool"},
			},
		},
	}

	err := ValidateBasic(cfg)
	assert.NoError(t, err)
}

func TestValidateBasic_ValidSubpackageModes(t *testing.T) {
	for _, mode := range []SubpackageMode{SubpackageModeOff, SubpackageModeAuto} {
		t.Run(string(mode), func(t *testing.T) {
			cfg := validConfig()
			cfg.Modules = []Module{
				{
					Name:        "foo",
					Repo:        "https://github.com/example/foo",
					Subpackages: &SubpackageConfig{Mode: mode},
				},
			}
			err := ValidateBasic(cfg)
			assert.NoError(t, err)
		})
	}
}

func TestValidateBasic_InvalidLogLevel(t *testing.T) {
	cfg := validConfig()
	cfg.Log.Level = "verbose"

	err := ValidateBasic(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "log.level: unknown value")
}

func TestValidateBasic_InvalidLogFormat(t *testing.T) {
	cfg := validConfig()
	cfg.Log.Format = "yaml"

	err := ValidateBasic(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "log.format: unknown value")
}

func TestValidateBasic_MultipleErrors(t *testing.T) {
	cfg := &Config{
		Log:     LogConfig{Level: "bad", Format: "bad"},
		Modules: []Module{{Name: "", Repo: ""}},
	}

	err := ValidateBasic(cfg)
	require.Error(t, err)

	// Should collect all errors, not stop at the first
	msg := err.Error()
	assert.Contains(t, msg, "log.level: unknown value")
	assert.Contains(t, msg, "log.format: unknown value")
	assert.Contains(t, msg, "domain is required")
	assert.Contains(t, msg, "modules[0].name is required")
	assert.Contains(t, msg, "modules[0].repo is required")
}

func TestValidateBasic_ExitCode(t *testing.T) {
	cfg := validConfig()
	cfg.Domain = ""

	err := ValidateBasic(cfg)
	require.Error(t, err)

	var ve ValidationError
	require.ErrorAs(t, err, &ve)
	assert.Equal(t, 2, ve.ExitCode())
}

func TestValidateFull_BasicInvalidationStillCaught(t *testing.T) {
	cfg := validConfig()
	cfg.Domain = "" // invalid - hit in basic validation pass
	cfg.Modules = []Module{
		{
			Name:      "foo",
			Repo:      "https://github.com/example/foo",
			LocalPath: "/nonexistent/path",
			Subpackages: &SubpackageConfig{
				Mode: SubpackageModeAuto,
			},
		},
	}

	err := ValidateFull(context.Background(), cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "domain is required")
}

func TestValidateFull_LocalPathNotExist(t *testing.T) {
	cfg := validConfig()
	cfg.Modules = []Module{
		{
			Name:      "foo",
			Repo:      "https://github.com/example/foo",
			LocalPath: "/nonexistent/path",
		},
	}

	err := ValidateFull(context.Background(), cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestValidateFull_LocalPathNotDir(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "not-a-dir")
	require.NoError(t, os.WriteFile(filePath, []byte("hello"), 0o644))

	cfg := validConfig()
	cfg.Modules = []Module{
		{
			Name:      "foo",
			Repo:      "https://github.com/example/foo",
			LocalPath: filePath,
		},
	}

	err := ValidateFull(context.Background(), cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is not a directory")
}

func TestValidateFull_LocalPathNotGitRepo(t *testing.T) {
	dir := t.TempDir()

	cfg := validConfig()
	cfg.Modules = []Module{
		{
			Name:      "foo",
			Repo:      "https://github.com/example/foo",
			LocalPath: dir,
		},
	}

	err := ValidateFull(context.Background(), cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is not a git repository")
}

func TestValidateFull_LocalPathValid(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping remote validation in short mode")
	}

	cfg := validConfig()
	cfg.Modules = []Module{
		{
			Name:      "foo",
			Repo:      "https://github.com/treyburn/vanity",
			LocalPath: "../..", // escape from here to this repo's root
		},
	}

	err := ValidateFull(context.Background(), cfg)
	assert.NoError(t, err)
}

func TestValidateFull_UnreachableRepo(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping remote validation in short mode")
	}

	cfg := validConfig()
	cfg.Modules = []Module{
		{
			Name: "foo",
			Repo: "https://github.com/nonexistent-org-12345/nonexistent-repo-67890",
		},
	}

	err := ValidateFull(context.Background(), cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not reachable")
}

// --- Template validation tests ---

func TestValidateBasic_TemplatesFileNotExist(t *testing.T) {
	cfg := validConfig()
	cfg.Templates.Index = "/nonexistent/index.html"

	err := ValidateBasic(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
	assertIsValidationErr(t, err)
}

func TestValidateBasic_TemplatesFileIsDirectory(t *testing.T) {
	dir := t.TempDir()

	cfg := validConfig()
	cfg.Templates.Module = dir

	err := ValidateBasic(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is a directory")
}

func TestValidateBasic_TemplatesBadSyntax(t *testing.T) {
	dir := t.TempDir()
	badTemplate := filepath.Join(dir, "bad.html")
	require.NoError(t, os.WriteFile(badTemplate, []byte(`{{define "body"}}`), 0o644)) // missing {{end}}

	cfg := validConfig()
	cfg.Templates.Index = badTemplate

	err := ValidateBasic(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid syntax")
}

func TestValidateBasic_TemplatesValidFile(t *testing.T) {
	dir := t.TempDir()
	tmplPath := filepath.Join(dir, "index.html")
	require.NoError(t, os.WriteFile(tmplPath, []byte(`{{define "body"}}<h1>hello</h1>{{end}}`), 0o644))

	cfg := validConfig()
	cfg.Templates.Index = tmplPath

	err := ValidateBasic(cfg)
	assert.NoError(t, err)
}

func TestValidateBasic_TemplatesPartials(t *testing.T) {
	dir := t.TempDir()
	headerPath := filepath.Join(dir, "header.html")
	require.NoError(t, os.WriteFile(headerPath, []byte(`{{define "header"}}<nav>hi</nav>{{end}}`), 0o644))

	cfg := validConfig()
	cfg.Templates.Partials = []string{headerPath}

	err := ValidateBasic(cfg)
	assert.NoError(t, err)
}

func TestValidateBasic_TemplatesPartialNotExist(t *testing.T) {
	cfg := validConfig()
	cfg.Templates.Partials = []string{"/nonexistent/header.html"}

	err := ValidateBasic(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestValidateBasic_AssetsNotExist(t *testing.T) {
	cfg := validConfig()
	cfg.Templates.Assets = []string{"/nonexistent/css/"}

	err := ValidateBasic(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "templates.assets")
	assert.Contains(t, err.Error(), "does not exist")
}

func TestValidateBasic_AssetsExist(t *testing.T) {
	dir := t.TempDir()
	cssDir := filepath.Join(dir, "css")
	require.NoError(t, os.MkdirAll(cssDir, 0o750))

	cfg := validConfig()
	cfg.Templates.Assets = []string{cssDir}

	err := ValidateBasic(cfg)
	assert.NoError(t, err)
}

func TestValidateBasic_NoTemplatesNoError(t *testing.T) {
	cfg := validConfig()
	// No templates configured at all
	err := ValidateBasic(cfg)
	assert.NoError(t, err)
}

func TestValidateBasic_MultipleTemplateErrors(t *testing.T) {
	cfg := validConfig()
	cfg.Templates.Index = "/nonexistent/index.html"
	cfg.Templates.Module = "/nonexistent/module.html"
	cfg.Templates.Assets = []string{"/nonexistent/css/"}

	err := ValidateBasic(cfg)
	require.Error(t, err)
	msg := err.Error()
	assert.Contains(t, msg, "index.html")
	assert.Contains(t, msg, "module.html")
	assert.Contains(t, msg, "css")
}

func TestValidateAssetCollisions_WithCollision(t *testing.T) {
	dir := t.TempDir()
	// Create a file called "index.html" as an asset — collides with generated index
	indexPath := filepath.Join(dir, "index.html")
	require.NoError(t, os.WriteFile(indexPath, []byte("<html>asset</html>"), 0o644))

	cfg := validConfig()
	cfg.Templates.Assets = []string{indexPath}

	err := validateAssetCollisions(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "collide")
}

func TestValidateAssetCollisions_NoCollision(t *testing.T) {
	dir := t.TempDir()
	cssDir := filepath.Join(dir, "css")
	require.NoError(t, os.MkdirAll(cssDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(cssDir, "style.css"), []byte("body{}"), 0o644))

	cfg := validConfig()
	cfg.Templates.Assets = []string{cssDir}

	err := validateAssetCollisions(cfg)
	assert.NoError(t, err)
}

func TestValidateAssetCollisions_CollidesWithModuleName(t *testing.T) {
	dir := t.TempDir()
	// Create a file named same as a module
	fooPath := filepath.Join(dir, "foo")
	require.NoError(t, os.WriteFile(fooPath, []byte("asset"), 0o644))

	cfg := validConfig()
	cfg.Templates.Assets = []string{fooPath}

	err := validateAssetCollisions(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "collide")
}

func TestValidateAssetCollisions_DirCollidesWithModuleName(t *testing.T) {
	dir := t.TempDir()
	// Create a directory named same as a module
	fooDir := filepath.Join(dir, "foo")
	require.NoError(t, os.MkdirAll(fooDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(fooDir, "data.json"), []byte("{}"), 0o644))

	cfg := validConfig()
	cfg.Templates.Assets = []string{fooDir}

	err := validateAssetCollisions(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "collide")
}

// assertIsValidationErr verifies that the error tree contains a ValidationError.
func assertIsValidationErr(t *testing.T, err error) {
	t.Helper()
	var ve ValidationError
	assert.True(t, errors.As(err, &ve), "expected error to contain ValidationErr")
}
