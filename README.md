# devcheck

A fast, zero-dependency CLI tool to validate your local development environment and dotfiles.

`devcheck` reads a `devcheck.yaml` manifest to verify that required CLI tools are installed on your `$PATH`, grabs their versions, and checks that essential files or directories exist on your machine.

## Features

- **Zero Runtime Dependencies:** Single binary that starts instantly.
- **CLI Diagnostics:** Checks binary presence and extracts version strings.
- **Path Checks:** Verifies files/folders exist (with automatic `~` home directory expansion).
- **Scripting Friendly:** Includes `-j` / `--json` output and exits with status code `1` on failure.
- **Flexible Config:** Automatically reads `./devcheck.yaml` or `~/.config/devcheck/config.yaml`.

## Quick Start

### 1. Create a `devcheck.yaml`

Place a `devcheck.yaml` in your current directory or in `~/.config/devcheck/config.yaml`:

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
  - name: Neovim Init
    path: ~/.config/nvim/init.lua
  - name: Tmux Config
    path: ~/.config/tmux/tmux.conf
  - name: SSH Directory
    path: ~/.ssh
