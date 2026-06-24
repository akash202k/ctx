# Quick Setup Guide

## One-Command Installation

### macOS / Linux

```bash
curl -fsSL https://raw.githubusercontent.com/akash202k/ctx/main/scripts/install.sh | sh
```

### Windows (PowerShell)

```powershell
iwr -useb https://raw.githubusercontent.com/akash202k/ctx/main/scripts/install.ps1 | iex
```

## What Gets Installed

The installer:
1. Checks if Go is installed
2. Installs `ctx` via `go install`
3. Verifies installation
4. Provides PATH setup guidance if needed

## Requirements

- **Go 1.21+** (auto-detected, installer guides you if missing)
- Works on:
  - macOS (Intel & Apple Silicon)
  - Linux (x86_64 & ARM64)
  - Windows (x86_64 & ARM64)

## After Installation

Run the interactive wizard:
```bash
ctx
```

Or use commands directly:
```bash
ctx read --exclude vendor --output snapshot.ctx
ctx edit --input changes.ctx
ctx select --task "fix bug" --entry main.go
```

## Troubleshooting

### "ctx: command not found"

Add Go's bin directory to your PATH:

**macOS/Linux:**
```bash
echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.bashrc
source ~/.bashrc
```

**Windows (PowerShell Admin):**
```powershell
$goPath = go env GOPATH
[Environment]::SetEnvironmentVariable('PATH', $env:PATH + ";$goPath\bin", 'User')
```

Then restart your terminal.

### Manual Installation

If the one-liner doesn't work:

```bash
go install github.com/akash202k/ctx/cmd/ctx@latest
```

## Verify Installation

```bash
ctx --version
```

You should see the version number.

## Quick Start

```bash
# Interactive mode (recommended for first use)
ctx

# Generate snapshot excluding vendor and testdata
ctx read --exclude vendor --exclude testdata --output my-project.ctx

# View help
ctx --help
ctx read --help
```

## Uninstall

```bash
rm $(which ctx)
```

Or if installed via `go install`:
```bash
rm $(go env GOPATH)/bin/ctx
```
