# devcheck

[![Go Version](https://img.shields.io/github/go-mod/go-version/ilyas-bkgo/devcheck?style=flat-square)](https://golang.org)
[![Latest Release](https://img.shields.io/github/v/release/ilyas-bkgo/devcheck?style=flat-square)](https://github.com/ilyas-bkgo/devcheck/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg?style=flat-square)](LICENSE)

> A fast, concurrent, zero-dependency CLI environment health checker written in Go.

`devcheck` reads a simple YAML configuration manifest to verify CLI binaries on your `$PATH`, inspect environment variables, and validate file/directory paths.

---

## 💡 Key Features

- **⚡ Fast & Concurrent:** Uses goroutines to run all checks in parallel, providing near-instant results.
- **⚡ Zero Runtime Dependencies:** Single static binary.
- **🛠️ CLI Diagnostics:** Verifies tool availability on `$PATH`, supports regex-matching on version outputs, and respects execution timeouts.
- **📁 Path & Env Validation:** Confirms existence of files/directories (with home directory `~` expansion) and verifies exported environment variables.
- **⚙️ Auto-Initialization:** Run `devcheck init` to generate a sensible starter configuration in seconds.
- **🤖 Script & CI Friendly:** Supports structured JSON output (`--json`), quiet mode (`--quiet`), and returns non-zero exit code `1` on check failures.

---

## 📦 Installation

### Shell Installer (Linux & macOS / WSL)

Install the latest pre-compiled binary instantly:

```bash
curl -fsSL https://raw.githubusercontent.com/ilyas-bkgo/devcheck/main/install.sh | sh
```

### Via `go install`

Requires Go 1.25+:

```bash
go install github.com/ilyas-bkgo/devcheck@latest
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

1. **Initialize a starter configuration:**
   ```bash
   devcheck init
   ```
   This creates a `devcheck.yaml` file in your current working directory.

2. **Run your environment health check:**
   ```bash
   devcheck
   ```

---

## ⚙️ Configuration (`devcheck.yaml`)

Define your development tools, environment variables, and paths inside `devcheck.yaml` (or `~/.config/devcheck/config.yaml`):

```yaml
tools:
  - name: Go Compiler
    cmd: go
    flag: version
    pattern: 'go1\.(2[2-9]|[3-9][0-9])'

paths:
  - name: SSH Key Dir
    path: ~/.ssh

env:
  - name: Default Editor
    var: EDITOR
    pattern: 'nvim|vim'
```

---

## 📖 CLI Usage & Flags

```bash
devcheck [flags]
```

| Flag | Shorthand | Description | Default |
| :--- | :--- | :--- | :--- |
| `--config` | `-c` | Path to custom config file | `devcheck.yaml` |
| `--json` | `-j` | Output results in JSON format | `false` |
| `--quiet` | `-q` | Only print failing checks | `false` |
| `--help` | `-h` | Display help menu | — |

### Examples

**Pipe JSON output to `jq`:**
```bash
devcheck --json | jq '.results[] | select(.passed == false)'
```

**Use in dotfiles bootstrap scripts:**
```bash
if ! devcheck; then
  echo "Environment health check failed."
  exit 1
fi
```

---

## 📄 License
MIT License © Ilyas
