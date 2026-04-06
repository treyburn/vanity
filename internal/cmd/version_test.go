package cmd

import (
	"bytes"
	"log/slog"
	"runtime/debug"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogVersion_DevBuild(t *testing.T) {
	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
	defer slog.SetDefault(slog.Default())

	logVersion(&debug.BuildInfo{
		Main:      debug.Module{Version: "(devel)"},
		GoVersion: "go1.23.0",
	})

	out := buf.String()
	assert.Contains(t, out, `"version":"dev"`)
	assert.Contains(t, out, `"go":"go1.23.0"`)
}

func TestLogVersion_Release(t *testing.T) {
	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
	defer slog.SetDefault(slog.Default())

	logVersion(&debug.BuildInfo{
		Main:      debug.Module{Version: "v1.2.3"},
		GoVersion: "go1.23.0",
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "abc123def456789"},
			{Key: "vcs.modified", Value: "false"},
		},
	})

	out := buf.String()
	assert.Contains(t, out, `"version":"v1.2.3"`)
	assert.Contains(t, out, `"commit":"abc123def456"`)
	assert.NotContains(t, out, "dirty")
}

func TestLogVersion_DirtyBuild(t *testing.T) {
	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
	defer slog.SetDefault(slog.Default())

	logVersion(&debug.BuildInfo{
		Main:      debug.Module{Version: "v1.0.0"},
		GoVersion: "go1.23.0",
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "abc123def456789"},
			{Key: "vcs.modified", Value: "true"},
		},
	})

	out := buf.String()
	assert.Contains(t, out, `"commit":"abc123def456 (dirty)"`)
}

func TestLogVersion_ShortCommit(t *testing.T) {
	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
	defer slog.SetDefault(slog.Default())

	logVersion(&debug.BuildInfo{
		Main:      debug.Module{Version: "v1.0.0"},
		GoVersion: "go1.23.0",
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "abc"},
		},
	})

	out := buf.String()
	assert.Contains(t, out, `"commit":"abc"`)
}

func TestLogVersion_NoCommit(t *testing.T) {
	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
	defer slog.SetDefault(slog.Default())

	logVersion(&debug.BuildInfo{
		Main:      debug.Module{Version: "v1.0.0"},
		GoVersion: "go1.23.0",
	})

	out := buf.String()
	assert.Contains(t, out, `"commit":""`)
}
