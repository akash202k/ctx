#!/bin/sh
set -e

# ctx installer script
# Usage: curl -fsSL https://raw.githubusercontent.com/akash202k/ctx/main/scripts/install.sh | sh

REPO="akash202k/ctx"
BINARY="ctx"

echo "Installing ctx..."

# Detect OS and architecture
OS="$(uname -s)"
ARCH="$(uname -m)"

case "$OS" in
    Linux*)     OS="linux";;
    Darwin*)    OS="darwin";;
    *)          echo "Unsupported OS: $OS"; exit 1;;
esac

case "$ARCH" in
    x86_64)     ARCH="amd64";;
    aarch64|arm64) ARCH="arm64";;
    *)          echo "Unsupported architecture: $ARCH"; exit 1;;
esac

# Check if Go is installed
if command -v go >/dev/null 2>&1; then
    echo "✓ Go detected, installing via 'go install'..."
    go install github.com/$REPO/cmd/ctx@latest
    
    # Check if installation succeeded
    if command -v ctx >/dev/null 2>&1; then
        echo "✓ ctx installed successfully!"
        echo ""
        echo "Usage:"
        echo "  ctx           - Interactive wizard (default)"
        echo "  ctx read      - Generate repository snapshot"
        echo "  ctx edit      - Apply edits from snapshot"
        echo "  ctx select    - Select relevant files for a task"
        echo ""
        echo "Run 'ctx --help' for more information."
        exit 0
    fi
    
    # If not in PATH, provide guidance
    echo ""
    echo "⚠ ctx installed but not found in PATH"
    echo ""
    echo "Add this to your shell profile (~/.bashrc, ~/.zshrc, etc.):"
    echo "  export PATH=\"\$PATH:\$(go env GOPATH)/bin\""
    echo ""
    echo "Then restart your shell or run: source ~/.bashrc"
    exit 0
fi

echo "⚠ Go not found. Please install Go from https://golang.org/dl/"
echo ""
echo "After installing Go, run:"
echo "  go install github.com/$REPO/cmd/ctx@latest"
echo ""
echo "Then add Go's bin directory to your PATH:"
echo "  export PATH=\"\$PATH:\$(go env GOPATH)/bin\""

exit 1
