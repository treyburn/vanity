package config

import (
	"bytes"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteDefault(t *testing.T) {
	var buf bytes.Buffer
	err := WriteDefault(&buf)
	require.NoError(t, err)

	output := buf.String()
	assert.NotEmpty(t, output)

	// Output should contain key comments
	assert.Contains(t, output, "CLI behavior")
	assert.Contains(t, output, "Output settings")
	assert.Contains(t, output, "REQUIRED")
	assert.Contains(t, output, "Module definitions")
}

func TestWriteDefault_RoundTrips(t *testing.T) {
	var buf bytes.Buffer
	err := WriteDefault(&buf)
	require.NoError(t, err)

	// The output should be valid YAML that parses back into a Config
	var cfg Config
	err = yaml.Unmarshal(buf.Bytes(), &cfg)
	require.NoError(t, err)

	// Verify the round-tripped values match ExampleConfig
	example := ExampleConfig()
	assert.Equal(t, example.Domain, cfg.Domain)
	assert.Equal(t, example.Log.Level, cfg.Log.Level)
	assert.Equal(t, example.Log.Format, cfg.Log.Format)
	assert.Equal(t, example.Log.Color, cfg.Log.Color)
	assert.Equal(t, example.Output.Dir, cfg.Output.Dir)
	assert.Equal(t, example.Output.Clean, cfg.Output.Clean)
	assert.Equal(t, example.Output.Index, cfg.Output.Index)
	assert.Equal(t, example.Output.NotFound, cfg.Output.NotFound)
	assert.Equal(t, example.Output.Robots, cfg.Output.Robots)
	assert.Equal(t, example.Output.Sitemap, cfg.Output.Sitemap)
	assert.Equal(t, example.Defaults.Branch, cfg.Defaults.Branch)
	assert.Equal(t, example.Defaults.GoSource, cfg.Defaults.GoSource)
	assert.Equal(t, example.Defaults.RedirectRoot, cfg.Defaults.RedirectRoot)
	require.Len(t, cfg.Modules, 1)
	assert.Equal(t, example.Modules[0].Name, cfg.Modules[0].Name)
	assert.Equal(t, example.Modules[0].Repo, cfg.Modules[0].Repo)
}

func TestCommentedDefault_AllFieldsCovered(t *testing.T) {
	cm := CommentedDefault()

	expectedKeys := []string{
		"$.log",
		"$.log.level",
		"$.log.format",
		"$.log.color",
		"$.output",
		"$.output.dir",
		"$.output.clean",
		"$.output.index",
		"$.output.not_found",
		"$.output.robots",
		"$.output.sitemap",
		"$.domain",
		"$.defaults",
		"$.defaults.branch",
		"$.defaults.go_source",
		"$.defaults.redirect_root",
		"$.modules",
		"$.modules[0].name",
		"$.modules[0].repo",
	}

	for _, key := range expectedKeys {
		_, ok := cm[key]
		assert.True(t, ok, "missing comment for %s", key)
	}
}
