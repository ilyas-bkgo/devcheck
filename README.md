# devcheck

[![Go Version](https://img.shields.io/github/go-mod/go-version/ilyas-bkgo/devcheck?style=flat-square)](https://golang.org)
[![Latest Release](https://img.shields.io/github/v/release/ilyas-bkgo/devcheck?style=flat-square)](https://github.com/ilyas-bkgo/devcheck/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg?style=flat-square)](LICENSE)

> A fast, concurrent, zero-dependency CLI environment health checker written in Go.

Current release: `v0.2.0` · Requires Go `1.25.0`

`devcheck` reads a simple YAML configuration manifest to verify CLI binaries on your `$PATH`, inspect environment variables, validate file/directory paths, and check TCP or HTTP services.

---

## 💡 Key Features

- **⚡ Fast & Concurrent:** Uses goroutines to run all checks in parallel, providing near-instant results.
- **⚡ Zero Runtime Dependencies:** Single static binary.
- **🛠️ CLI Diagnostics:** Verifies tool availability on `$PATH`, supports regex-matching on version outputs, and respects execution timeouts.
- **📁 Path & Env Validation:** Confirms existence of files/directories (with home directory `~` expansion) and verifies exported environment variables.
- **🌐 Service Checks:** Tests TCP ports and HTTP endpoints, including expected HTTP status codes.
- **⚙️ Templates & Auto-Initialization:** Run `devcheck init --template node|python|go` to generate a configurable project starter in seconds.
- **🩺 Self-Diagnostics:** `devcheck version` identifies the running binary; `devcheck doctor` validates the selected configuration.
- **✅ Strict Config Validation:** Rejects unknown YAML fields and invalid checks before they reach developers or CI.
- **🏢 Team-ready Configs:** Share baseline checks with `include` and add role-specific checks with `profiles`.
- **🔧 Interactive Fixes:** Run `devcheck --fix` to choose and execute repair commands defined in failed checks.
- **🤖 Script & CI Friendly:** Supports structured JSON output (`--json`), quiet mode (`--quiet`), and returns non-zero exit code `1` on check failures.

---

## 📦 Installation

### Shell Installer (Linux & macOS / WSL)

Install the latest pre-compiled binary instantly:

```bash
curl -fsSL https://raw.githubusercontent.com/ilyas-bkgo/devcheck/main/install.sh | sh
```

### Via `go install`

Requires Go 1.25.0:

```bash
go install github.com/ilyas-bkgo/devcheck@v0.2.0
export PATH=$PATH:$HOME/go/bin
```

### From Source

```bash
git clone https://github.com/ilyas-bkgo/devcheck.git
cd devcheck
go build -o devcheck ./cmd/devcheck
sudo mv devcheck /usr/local/bin/
```

---

## 🚀 Quick Start

1. **Initialize a starter configuration:**
   ```bash
   devcheck init
   ```
   This creates a generic `devcheck.yaml` file in your current working directory. For a project-specific starter, choose a template:
   ```bash
   devcheck init --template node
   # or: python, go
   ```

2. **Run your environment health check:**
   ```bash
   devcheck
   ```

3. **Confirm your installation and configuration when troubleshooting:**
   ```bash
   devcheck version
   devcheck doctor
   ```

## 📚 Ready-to-use examples

The [`examples/`](examples) directory contains copy-ready configurations:

| Template | Checks |
| :--- | :--- |
| [`web-dev-node.yaml`](examples/web-dev-node.yaml) | Node.js, pnpm, Docker, PostgreSQL on port 5432 |
| [`python-backend.yaml`](examples/python-backend.yaml) | Python, Poetry, Redis, an HTTP health endpoint |
| [`go-microservices.yaml`](examples/go-microservices.yaml) | Go, `protoc`, `golangci-lint`, Docker, an HTTP health endpoint |

Copy and tailor one for your repository:

```bash
cp examples/web-dev-node.yaml devcheck.yaml
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
    pattern: 'go1\.25\.'

paths:
  - name: SSH Key Dir
    path: ~/.ssh

env:
  - name: Default Editor
    var: EDITOR
    pattern: 'nvim|vim'

services:
  - name: Local API
    url: http://localhost:8080/health
    expected_status: 200
    timeout: 3s

  - name: PostgreSQL
    host: 127.0.0.1
    port: 5432
    timeout: 2s
```

Each check can include a `fix` command. It can be a string or an operating-system-specific map; with `--fix`, devcheck prompts before running the selected failed-check commands.

Tool checks support either the original single `flag` value or an `args` list for multi-argument commands:

```yaml
tools:
  - name: npm registry
    cmd: npm
    args: ["config", "get", "registry"]
```

	## 🏢 Team configuration
	
	Keep a shared baseline and extend it for each repository or role. You can use local paths or remote URLs:
	
	```yaml
	# devcheck.yaml
	include:
	  - https://raw.githubusercontent.com/ilyas-bkgo/devcheck-standards/main/baseline-common.yaml
	  - .devcheck/team-secrets.yaml
	
	tools:
	  - name: pnpm
	    cmd: pnpm
	    flag: --version
	```

profiles:
  frontend:
    services:
      - name: Local web app
        url: http://127.0.0.1:3000/health
        expected_status: 200
```

Run the shared baseline alone, or add a profile:

```bash
devcheck
devcheck --profile frontend
```

Validate configuration in a pull request before asking developers to use it:

```bash
devcheck validate
devcheck validate --profile frontend
```

Environment-variable values are never included in terminal or JSON results. Use a `pattern` to verify a value without exposing it.

---

## 📖 CLI Usage & Flags

```bash
devcheck [command] [flags]
```

| Command | Description |
| :--- | :--- |
| `init` | Create `devcheck.yaml`; supports `--template default\|node\|python\|go` |
| `version` | Print version, build metadata, and the executable path |
| `doctor` | Show binary details and validate the resolved configuration; supports `--config` / `-c` |
| `validate` | Strictly validate the resolved configuration; supports `--config`, `-c`, and `--profile` |

| Flag | Shorthand | Description | Default |
| :--- | :--- | :--- | :--- |
| `--config` | `-c` | Path to custom config file | `devcheck.yaml` |
| `--json` | `-j` | Output results in JSON format | `false` |
| `--quiet` | `-q` | Only print failing checks | `false` |
| `--fix` | `-f` | Prompt to run configured fixes for failed checks | `false` |
| `--fix-dry-run` | — | Show selected fix commands without executing them | `false` |
| `--fix-only` | — | Restrict fixes to named checks, comma-separated | — |
| `--profile` | — | Add checks from a named configuration profile | — |
| `--format` | — | Output `terminal`, `json`, `junit`, or `github-actions` | `terminal` |
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

**Use in GitHub Actions:**
```bash
devcheck --config devcheck.ci.yaml --format github-actions
```

**Publish a JUnit report:**
```bash
devcheck --format junit > devcheck-report.xml
```

`--fix` executes commands from your configuration through the system shell. Review configuration changes like code, use `--fix-dry-run` first, and do not enable fixes in unattended CI.

---

## 📄 License
MIT License © Ilyas
