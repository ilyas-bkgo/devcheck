# devcheck

[![Go Version](https://img.shields.io/github/go-mod/go-version/ilyas-bkgo/devcheck?style=flat-square)](https://golang.org)
[![Latest Release](https://img.shields.io/github/v/release/ilyas-bkgo/devcheck?style=flat-square)](https://github.com/ilyas-bkgo/devcheck/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg?style=flat-square)](LICENSE)

> A fast, zero-dependency CLI environment health checker written in Go.

`devcheck` reads a simple YAML configuration manifest to verify that your required CLI tools are installed on your `$PATH`, extracts their version numbers, and confirms that vital files and directories exist.

---

## 💡 Key Features

- **⚡ Zero Runtime Dependencies:** Single static binary that executes instantly.
- **🛠️ CLI Diagnostics:** Verifies tool availability on `$PATH` and captures version strings.
- **📁 Path Validation:** Confirms existence of files and directories (with full `~` tilde home path expansion).
- **⚙️ Auto-Initialization:** Run `devcheck init` to generate a sensible starter configuration in seconds.
- **🤖 Script & CI Friendly:** Includes structured JSON output (`--json` / `-j`) and returns non-zero exit codes (`1`) on check failures for dotfile installation scripts.

---

## 📦 Installation

### Shell Installer (Linux & macOS)

Install the latest pre-compiled binary instantly, no Go required:

```bash
curl -fsSL https://raw.githubusercontent.com/ilyas-bkgo/devcheck/main/install.sh | sh
```

### Via `go install`

Requires Go 1.20+:

```bash
go install github.com/ilyas-bkgo/devcheck@latest
```

Note: Ensure `$HOME/go/bin` is in your system `$PATH`:

```bash
export PATH=$PATH:$HOME/go/bin
```

### From Source

```bash
git clone https://github.com/ilyas-bkgo/devcheck.git
cd devcheck
go build -o devcheck .
sudo mv devcheck /usr/local/bin/
```

---

## 🚀 Quick Start

Initialize a starter configuration:

```bash
devcheck init
```

This creates a `devcheck.yaml` file in your current working directory.

Run your environment health check:

```bash
devcheck
```

---

## ⚙️ Configuration (`devcheck.yaml`)

Define your development tools and file paths inside `devcheck.yaml` (or `~/.config/devcheck/config.yaml`):

```yaml
tools:
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
```

---

## 📖 CLI Usage & Flags

```bash
devcheck [command] [flags]
```

### Commands

| Command | Description |
|---|---|
| `init` | Creates a starter `devcheck.yaml` configuration in the current directory |

### Flags

| Flag | Shorthand | Description | Default |
|---|---|---|---|
| `--config` | `-c` | Path to custom config file | `devcheck.yaml` |
| `--json` | `-j` | Output results in JSON format | `false` |
| `--help` | `-h` | Display help menu | — |

### Examples

Run with a custom config path:

```bash
devcheck -c ~/.dotfiles/devcheck.yaml
```

Pipe JSON output to `jq`:

```bash
devcheck --json | jq '.results[] | select(.passed == false)'
```

Use in dotfiles bootstrap scripts:

```bash
if ! devcheck; then
  echo "Environment health check failed. Please fix missing dependencies."
  exit 1
fi
```

---

## 📄 License

MIT License © Ilyas