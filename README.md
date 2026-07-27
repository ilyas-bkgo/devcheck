# devcheck

[![Go Version](https://img.shields.io/github/go-mod/go-version/ilyas-bkgo/devcheck?style=flat-square)](https://golang.org)
[![Latest Release](https://img.shields.io/github/v/tag/ilyas-bkgo/devcheck?label=release&style=flat-square)](https://github.com/ilyas-bkgo/devcheck/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg?style=flat-square)](LICENSE)

> A fast, zero-dependency CLI environment health checker written in Go.

`devcheck` reads a simple YAML configuration manifest to verify that your required CLI tools are installed on your `$PATH`, extracts their version numbers, and confirms that vital files and directories exist.

---

## 💡 Key Features

- **⚡ Zero Runtime Dependencies:** Single binary that executes instantly.
- **🛠️ CLI Diagnostics:** Verifies tool availability on `$PATH` and captures version strings.
- **📁 Path Validation:** Confirms existence of files and directories (with full `~` tilde home path expansion).
- **⚙️ Auto-Initialization:** Run `devcheck init` to generate a sensible starter configuration in seconds.
- **🤖 Script & CI Friendly:** Includes structured JSON output (`--json` / `-j`) and returns non-zero exit codes (`1`) on check failures for dotfile installation scripts.

---

## 📦 Installation

### Via `go install` (Recommended)

Requires Go 1.20+:

```bash
go install [github.com/ilyas-bkgo/devcheck@latest](https://github.com/ilyas-bkgo/devcheck@latest)
Note: Ensure $HOME/go/bin is in your system $PATH:Bashexport PATH=$PATH:$HOME/go/bin
From SourceBashgit clone [https://github.com/ilyas-bkgo/devcheck.git](https://github.com/ilyas-bkgo/devcheck.git)
cd devcheck
go build -o devcheck .
sudo mv devcheck /usr/local/bin/
🚀 Quick StartInitialize a starter configuration:Bashdevcheck init
This creates a devcheck.yaml file in your current directory.Run your environment health check:Bashdevcheck
⚙️ Configuration (devcheck.yaml)Define your development tools and file paths inside devcheck.yaml (or ~/.config/devcheck/config.yaml):YAMLtools:
  - name: Neovim
    cmd: nvim
    flag: --version
  - name: Git
    cmd: git
    flag: --version
  - name: Docker
    cmd: docker
    flag: --version
  - name: Ripgrep
    cmd: rg
    flag: --version
  - name: Tmux
    cmd: tmux
    flag: -V

paths:
  - name: SSH Key Dir
    path: ~/.ssh
  - name: Neovim Config
    path: ~/.config/nvim/init.lua
  - name: Tmux Config
    path: ~/.config/tmux/tmux.conf
📖 CLI Usage & FlagsBashdevcheck [flags]
FlagShorthandDescriptionDefault--config-cSpecify custom path to config filedevcheck.yaml--json-jOutput health results in JSON formatfalse--help-hDisplay help menu—ExamplesRun with a custom config path:Bashdevcheck -c ~/.dotfiles/devcheck.yaml
Pipe JSON output to jq:Bashdevcheck --json | jq '.results[] | select(.passed == false)'
Use in dotfiles bootstrap scripts (install.sh):Bashif ! devcheck; then
  echo "Environment health check failed. Please fix missing dependencies."
  exit 1
fi
📄 LicenseMIT License © Ilyas