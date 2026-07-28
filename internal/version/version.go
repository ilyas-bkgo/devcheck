// Package version exposes build metadata for the devcheck command.
package version

import (
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
)

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

type Info struct {
	Version    string
	Commit     string
	Date       string
	Executable string
	GoVersion  string
	OS         string
	Arch       string
}

// Current returns the executable and build details for the running command.
func Current() Info {
	executable, err := os.Executable()
	if err == nil {
		if resolved, resolveErr := filepath.EvalSymlinks(executable); resolveErr == nil {
			executable = resolved
		}
	}
	buildVersion, buildCommit, buildDate := Version, Commit, Date
	if buildInfo, ok := debug.ReadBuildInfo(); ok {
		if buildVersion == "dev" && buildInfo.Main.Version != "" && buildInfo.Main.Version != "(devel)" {
			buildVersion = buildInfo.Main.Version
		}
		for _, setting := range buildInfo.Settings {
			switch setting.Key {
			case "vcs.revision":
				if buildCommit == "none" {
					buildCommit = setting.Value
				}
			case "vcs.time":
				if buildDate == "unknown" {
					buildDate = setting.Value
				}
			}
		}
	}
	return Info{
		Version: buildVersion, Commit: buildCommit, Date: buildDate, Executable: executable,
		GoVersion: runtime.Version(), OS: runtime.GOOS, Arch: runtime.GOARCH,
	}
}
