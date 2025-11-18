#!/bin/bash
set -e

BINARY=llmsh
INSTALL_PATH="$HOME/.local/bin"
CONFIG_DIR="$HOME/.llmsh"
ZSHRC="$HOME/.zshrc"

# Uninstall function
uninstall() {
    echo "========================================="
    echo "  Uninstalling llmsh"
    echo "========================================="
    echo ""

    # Remove binary
    if [ -f "$INSTALL_PATH/$BINARY" ]; then
        echo "Removing binary from $INSTALL_PATH/$BINARY..."
        rm -f "$INSTALL_PATH/$BINARY"
        echo "✓ Binary removed"
    else
        echo "ℹ️  Binary not found at $INSTALL_PATH/$BINARY"
    fi

    # Remove config directory
    echo ""
    if [ -d "$CONFIG_DIR" ]; then
        echo "Config directory found at $CONFIG_DIR"
        echo "This contains your configuration, cache, and token data."
        read -p "Remove config directory? (y/n) " -n 1 -r
        echo ""
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            rm -rf "$CONFIG_DIR"
            echo "✓ Config directory removed"
        else
            echo "ℹ️  Config directory kept"
        fi
    else
        echo "ℹ️  Config directory not found"
    fi

    # Remove ZSH plugin configuration
    echo ""
    if [ -f "$ZSHRC" ]; then
        if grep -q "llmsh.plugin.zsh" "$ZSHRC"; then
            echo "ZSH plugin configuration found in $ZSHRC"
            read -p "Remove llmsh plugin from ~/.zshrc? (y/n) " -n 1 -r
            echo ""
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                # Create backup
                cp "$ZSHRC" "$ZSHRC.bak"
                echo "✓ Created backup at $ZSHRC.bak"

                # Remove lines containing llmsh.plugin.zsh
                grep -v "llmsh.plugin.zsh" "$ZSHRC.bak" > "$ZSHRC"
                echo "✓ Removed llmsh plugin from $ZSHRC"
                echo ""
                echo "Please reload your shell:"
                echo "  source ~/.zshrc"
            else
                echo "ℹ️  ZSH plugin configuration kept"
                echo ""
                echo "To manually remove, delete this line from ~/.zshrc:"
                echo "  source .../zsh/llmsh.plugin.zsh"
            fi
        else
            echo "ℹ️  No llmsh plugin configuration found in $ZSHRC"
        fi
    else
        echo "ℹ️  ~/.zshrc not found"
    fi

    echo ""
    echo "========================================="
    echo "  Uninstallation Complete!"
    echo "========================================="
    echo ""
    exit 0
}

# Check if uninstall is requested
if [ "$1" = "uninstall" ]; then
    uninstall
fi

echo "========================================="
echo "  Installing llmsh"
echo "========================================="
echo ""

# Check for Go
if ! command -v go &>/dev/null; then
    echo "❌ Error: Go is not installed"
    echo "Please install Go from https://golang.org/dl/"
    exit 1
fi

echo "✓ Go found: $(go version)"

# Check for zsh
if ! command -v zsh &>/dev/null; then
    echo "⚠️  Warning: zsh is not installed"
    echo "zsh is required for the ZSH plugin to work."
    echo ""
    echo "Install zsh:"
    echo "  macOS: brew install zsh"
    echo "  Linux (Debian/Ubuntu): sudo apt-get install zsh"
    echo "  Linux (RHEL/CentOS): sudo yum install zsh"
    echo ""
    read -p "Continue anyway? (y/n) " -n 1 -r
    echo ""
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
else
    echo "✓ zsh found: $(zsh --version)"
fi

# Check for jq
if ! command -v jq &>/dev/null; then
    echo "⚠️  Warning: jq is not installed"
    echo "jq is required for the ZSH plugin to work."
    echo ""
    echo "Install jq:"
    echo "  macOS: brew install jq"
    echo "  Linux (Debian/Ubuntu): sudo apt-get install jq"
    echo "  Linux (RHEL/CentOS): sudo yum install jq"
    echo ""
    read -p "Continue anyway? (y/n) " -n 1 -r
    echo ""
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
else
    echo "✓ jq found"
fi

# Build and install
echo ""
echo "Building llmsh..."
make deps
make build

echo ""
echo "Installing llmsh..."
make install

echo ""
echo "========================================="
echo "  Installation Complete!"
echo "========================================="
echo ""

# Check for zsh-autosuggestions
if [ -f "$ZSHRC" ]; then
    if grep -q "zsh-autosuggestions" "$ZSHRC" 2>/dev/null; then
        echo "✓ zsh-autosuggestions detected (enhanced experience enabled)"
        echo ""
    else
        echo "ℹ️  Optional: Install zsh-autosuggestions for better suggestion display"
        echo "  https://github.com/zsh-users/zsh-autosuggestions"
        echo "  brew install zsh-autosuggestions (macOS)"
        echo ""
    fi
fi

echo "For more information, see README.md and USAGE.md"
