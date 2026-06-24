#!/bin/sh
set -e

# ctx installer script with binary fallback
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

# Function to install from binary
install_binary() {
    echo "📦 Downloading pre-built binary..."
    
    # Get latest release
    LATEST_RELEASE=$(curl -s https://api.github.com/repos/$REPO/releases/latest | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    
    if [ -z "$LATEST_RELEASE" ]; then
        echo "⚠ No releases found. Attempting to build from source..."
        return 1
    fi
    
    DOWNLOAD_URL="https://github.com/$REPO/releases/download/$LATEST_RELEASE/ctx_${LATEST_RELEASE}_${OS}_${ARCH}.tar.gz"
    
    echo "Downloading: $DOWNLOAD_URL"
    
    # Download to temporary directory
    TMP_DIR=$(mktemp -d)
    cd "$TMP_DIR"
    
    if curl -sL "$DOWNLOAD_URL" | tar xz; then
        # Try installation locations in order of preference
        if [ -w "/usr/local/bin" ]; then
            mv ctx /usr/local/bin/
            echo "✓ ctx installed to /usr/local/bin/ctx"
            cd - > /dev/null
            rm -rf "$TMP_DIR"
            return 0
        elif sudo -n true 2>/dev/null; then
            # Has passwordless sudo
            sudo mv ctx /usr/local/bin/
            echo "✓ ctx installed to /usr/local/bin/ctx"
            cd - > /dev/null
            rm -rf "$TMP_DIR"
            return 0
        else
            # Install to user's home bin
            mkdir -p "$HOME/bin"
            mv ctx "$HOME/bin/"
            echo "✓ ctx installed to $HOME/bin/ctx"
            
            # Auto-add to PATH based on shell
            SHELL_CONFIG=""
            if [ -n "$ZSH_VERSION" ] || [ "$SHELL" = "/bin/zsh" ] || [ "$SHELL" = "/usr/bin/zsh" ]; then
                SHELL_CONFIG="$HOME/.zshrc"
            elif [ -n "$BASH_VERSION" ] || [ "$SHELL" = "/bin/bash" ] || [ "$SHELL" = "/usr/bin/bash" ]; then
                SHELL_CONFIG="$HOME/.bashrc"
            fi
            
            # Check if PATH already contains $HOME/bin
            if ! echo "$PATH" | grep -q "$HOME/bin"; then
                if [ -n "$SHELL_CONFIG" ] && [ -f "$SHELL_CONFIG" ]; then
                    echo "" >> "$SHELL_CONFIG"
                    echo "# Added by ctx installer" >> "$SHELL_CONFIG"
                    echo 'export PATH="$HOME/bin:$PATH"' >> "$SHELL_CONFIG"
                    echo ""
                    echo "✓ Added $HOME/bin to PATH in $SHELL_CONFIG"
                    echo ""
                    echo "Run this to use ctx immediately:"
                    echo "  source $SHELL_CONFIG"
                    echo ""
                    echo "Or restart your terminal"
                else
                    echo ""
                    echo "⚠ Add this to your shell profile (~/.bashrc, ~/.zshrc, etc.):"
                    echo '  export PATH="$HOME/bin:$PATH"'
                    echo ""
                    echo "Then restart your shell or run: source ~/.zshrc"
                fi
            else
                echo "✓ $HOME/bin is already in your PATH"
            fi
            
            cd - > /dev/null
            rm -rf "$TMP_DIR"
            return 0
        fi
    else
        echo "⚠ Failed to download binary"
        cd - > /dev/null
        rm -rf "$TMP_DIR"
        return 1
    fi
}

# Try binary download first (ensures users get latest from main)
echo "Installing ctx..."
if install_binary; then
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

# Binary download failed, try go install as fallback
if command -v go >/dev/null 2>&1; then
    echo "⚠ Binary download failed, trying 'go install'..."
    
    # Try direct install first (bypasses proxy cache issues)
    if GOPROXY=direct go install github.com/$REPO/cmd/ctx@latest 2>/dev/null || \
       go install github.com/$REPO/cmd/ctx@latest 2>/dev/null; then
        
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
fi

# Binary download failed
echo ""
echo "❌ Installation failed. Please try one of these methods:"
echo ""
echo "Method 1: Install Go and use go install"
echo "  1. Install Go from https://golang.org/dl/"
echo "  2. Run: go install github.com/$REPO/cmd/ctx@latest"
echo "  3. Add to PATH: export PATH=\"\$PATH:\$(go env GOPATH)/bin\""
echo ""
echo "Method 2: Build from source"
echo "  git clone https://github.com/$REPO"
echo "  cd ctx"
echo "  make build"
echo "  sudo mv ctx /usr/local/bin/"
echo ""

exit 1
