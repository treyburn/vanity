package cmd

import (
	"fmt"
	"log/slog"
	"runtime/debug"
)

type VersionCmd struct{}

func (v *VersionCmd) Run() error {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return fmt.Errorf("failed to read build info")
	}

	logVersion(info)
	return nil
}

func logVersion(info *debug.BuildInfo) {
	version := info.Main.Version
	if version == "" || version == "(devel)" {
		version = "dev"
	}

	var commit, dirty string
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			commit = s.Value
		case "vcs.modified":
			if s.Value == "true" {
				dirty = " (dirty)"
			}
		}
	}

	if commit != "" {
		// truncate to the github style short commit which is just 7 chars
		commit = commit[:min(7, len(commit))] + dirty
	}

	slog.Info("vanity", "version", version, "commit", commit, "go", info.GoVersion)
}
