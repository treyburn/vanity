package vcs

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	gitconfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/storage/memory"
)

// ValidateRemote checks that a git remote is reachable.
func ValidateRemote(ctx context.Context, repoURL string) error {
	// TODO - consider adding retry's here?
	_, err := git.NewRemote(memory.NewStorage(), &gitconfig.RemoteConfig{
		Name: "origin",
		URLs: []string{repoURL},
	}).ListContext(ctx, &git.ListOptions{})
	if err != nil {
		return fmt.Errorf("remote %s is not reachable: %w", repoURL, err)
	}
	return nil
}

type discoverConfig struct {
	localPath string
}

// Option configures DiscoverSubpackages behavior.
type Option func(*discoverConfig)

// WithLocalPath reads subpackages from a local checkout instead of cloning.
func WithLocalPath(path string) Option {
	return func(c *discoverConfig) {
		c.localPath = path
	}
}

// DiscoverSubpackages returns Go package directory paths relative to the repo root.
// By default, it performs a shallow in-memory clone of repoURL. Use WithLocalPath to read from a local checkout instead.
func DiscoverSubpackages(ctx context.Context, repoURL string, exclude []string, opts ...Option) ([]string, error) {
	cfg := &discoverConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	var bfs billy.Filesystem
	if cfg.localPath != "" {
		bfs = osfs.New(cfg.localPath)
	} else {
		bfs = memfs.New()
		// TODO - is it possible to just use the `ListContext` method and walk the dir that way? that would avoid an expensive clone...
		_, err := git.CloneContext(ctx, memory.NewStorage(), bfs, &git.CloneOptions{
			URL:   repoURL,
			Depth: 1,
		})
		if err != nil {
			return nil, fmt.Errorf("cloning %s: %w", repoURL, err)
		}
	}

	return findGoPackages(bfs, exclude)
}

// findGoPackages walks a billy.Filesystem and returns directories containing
// .go source files, filtered by exclude patterns.
func findGoPackages(bfs billy.Filesystem, exclude []string) ([]string, error) {
	seen := make(map[string]bool)

	if err := walkDir(bfs, ".", seen); err != nil {
		return nil, fmt.Errorf("walking filesystem: %w", err)
	}

	return filterExcluded(seen, exclude), nil
}

func walkDir(bfs billy.Filesystem, dir string, seen map[string]bool) error {
	entries, err := bfs.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())

		if entry.IsDir() {
			if shouldSkipDir(path) {
				continue
			}
			if err := walkDir(bfs, path, seen); err != nil {
				return err
			}
			continue
		}

		if isGoSourceFile(path) {
			parent := filepath.Dir(path)
			if parent == "." {
				continue // root package is the module itself
			}
			seen[parent] = true
		}
	}

	return nil
}

// shouldSkipDir returns true for directories that should never be scanned.
func shouldSkipDir(path string) bool {
	name := filepath.Base(path)
	switch name {
	case ".", "..":
		return false
	case "vendor", "testdata", ".git":
		return true
	}
	return strings.HasPrefix(name, ".")
}

// isGoSourceFile returns true for .go files that are not test files.
func isGoSourceFile(path string) bool {
	return strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go")
}

// filterExcluded removes paths matching any exclude glob pattern and returns
// the remaining paths in sorted order.
func filterExcluded(seen map[string]bool, exclude []string) []string {
	var result []string
	for dir := range seen {
		if matchesAny(dir, exclude) {
			continue
		}
		result = append(result, dir)
	}
	sort.Strings(result)
	return result
}

// matchesAny returns true if path matches any of the glob patterns.
func matchesAny(path string, patterns []string) bool {
	for _, pattern := range patterns {
		if matched, _ := filepath.Match(pattern, path); matched {
			return true
		}
	}
	return false
}
