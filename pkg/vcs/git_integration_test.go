package vcs

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testRepo = "https://github.com/treyburn/vanity" // dogfooding our own repo for the integration tests

func TestIntegration_ValidateRemote(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	err := ValidateRemote(context.Background(), testRepo)
	require.NoError(t, err)
}

func TestIntegration_ValidateRemote_Unreachable(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	err := ValidateRemote(context.Background(), "https://github.com/treyburn/nonexistent-repo-abc123")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not reachable")
}

func TestIntegration_ValidateLocalRepo(t *testing.T) {
	// Use the current repo (two levels up from pkg/vcs)
	err := ValidateLocalRepo("../..")
	require.NoError(t, err)
}

func TestIntegration_ValidateLocalRepo_NotARepo(t *testing.T) {
	dir := t.TempDir()
	err := ValidateLocalRepo(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is not a git repository")
}

func TestIntegration_DiscoverSubpackages_Local(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pkgs, err := DiscoverSubpackages(context.Background(), testRepo, nil, WithLocalPath("../.."))
	require.NoError(t, err)

	// Light assertion: the repo has at least one subpackage (pkg/cmd, pkg/config, etc.)
	assert.NotEmpty(t, pkgs)
	assert.Contains(t, pkgs, "pkg/cmd")
	assert.Contains(t, pkgs, "pkg/config")

	// Root package (main.go) should not appear — it's the module itself
	assert.NotContains(t, pkgs, ".")

	// Verify none of the skipped dirs leaked through
	for _, pkg := range pkgs {
		assert.NotContains(t, pkg, "vendor")
		assert.NotContains(t, pkg, "testdata")
		assert.NotContains(t, pkg, "internal")
		assert.NotContains(t, pkg, ".git")
		assert.NotContains(t, pkg, ".github")
	}
}

func TestIntegration_DiscoverSubpackages_WithExclude(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pkgs, err := DiscoverSubpackages(context.Background(), testRepo, []string{"pkg/cmd"}, WithLocalPath("../.."))
	require.NoError(t, err)

	assert.NotEmpty(t, pkgs)
	assert.NotContains(t, pkgs, "pkg/cmd")
	assert.Contains(t, pkgs, "pkg/config")
}

func TestIntegration_DiscoverSubpackages_LocalPath(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Use the current repo as the local path (two levels up from pkg/vcs)
	pkgs, err := DiscoverSubpackages(context.Background(), testRepo, nil, WithLocalPath("../.."))
	require.NoError(t, err)

	assert.NotEmpty(t, pkgs)
	assert.Contains(t, pkgs, "pkg/cmd")
	assert.Contains(t, pkgs, "pkg/config")
	assert.NotContains(t, pkgs, ".")

	for _, pkg := range pkgs {
		assert.NotContains(t, pkg, "vendor")
		assert.NotContains(t, pkg, "testdata")
		assert.NotContains(t, pkg, "internal")
		assert.NotContains(t, pkg, ".git")
		assert.NotContains(t, pkg, ".github")
	}
}
