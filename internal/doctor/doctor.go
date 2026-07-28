// Package doctor diagnoses common devcheck installation and configuration issues.
package doctor

import (
	"fmt"
	"io"

	"github.com/ilyas-bkgo/devcheck/internal/config"
	"github.com/ilyas-bkgo/devcheck/internal/version"
)

// Run prints diagnostics and returns true when the selected configuration is usable.
func Run(writer io.Writer, configPath, profile string) bool {
	info := version.Current()
	fmt.Fprintln(writer, "🩺 devcheck doctor")
	fmt.Fprintf(writer, "Version:    %s (%s, %s)\n", info.Version, info.Commit, info.Date)
	fmt.Fprintf(writer, "Executable: %s\n", info.Executable)
	fmt.Fprintf(writer, "Runtime:    %s %s/%s\n", info.GoVersion, info.OS, info.Arch)

	resolvedPath := config.ResolvePath(configPath)
	fmt.Fprintf(writer, "Config:     %s\n", resolvedPath)
	cfg, err := config.LoadProfile(resolvedPath, profile)
	if err != nil {
		fmt.Fprintf(writer, "❌ Configuration is not usable: %v\n", err)
		return false
	}

	if err := config.Validate(cfg); err != nil {
		fmt.Fprintf(writer, "❌ Configuration is not usable: %v\n", err)
		return false
	}
	count := len(cfg.Tools) + len(cfg.Paths) + len(cfg.Env) + len(cfg.Services)
	fmt.Fprintf(writer, "✔ Configuration is valid (%d checks).\n", count)
	return true
}
