#!/bin/bash
set -e

BINARY=llmsh
INSTALL_PATH="$HOME/.local/bin"
CONFIG_DIR="$HOME/.llmsh"
ZSHRC="$HOME/.zshrc"
REPO="LeslieLeung/llmsh"

# Detect OS and architecture
detect_platform() {
    local os=""
    local arch=""

    # Detect OS
    case "$(uname -s)" in
        Linux*)     os="linux" ;;
        Darwin*)    os="darwin" ;;
        MINGW*|MSYS*|CYGWIN*) os="windows" ;;
        *)
            echo "❌ Error: Unsupported operating system: $(uname -s)"
            exit 1
            ;;
    esac

    # Detect architecture
    case "$(uname -m)" in
        x86_64|amd64)   arch="amd64" ;;
        arm64|aarch64)  arch="arm64" ;;
        *)
            echo "❌ Error: Unsupported architecture: $(uname -m)"
            exit 1
            ;;
    esac

    echo "${os}-${arch}"
}

# Install jq based on OS
install_jq() {
    if command -v jq &>/dev/null; then
        echo "✓ jq is already installed"
        return 0
    fi

    echo "Installing jq..."

    case "$(uname -s)" in
        Darwin*)
            if command -v brew &>/dev/null; then
                brew install jq
            else
                echo "❌ Error: Homebrew is required to install jq on macOS"
                echo "Please install Homebrew from https://brew.sh/"
                echo "Or install jq manually from https://stedolan.github.io/jq/"
                exit 1
            fi
            ;;
        Linux*)
            if [ -f /etc/debian_version ]; then
                # Debian/Ubuntu
                if command -v sudo &>/dev/null; then
                    sudo apt-get update && sudo apt-get install -y jq
                else
                    echo "⚠️  Warning: sudo not available, trying without sudo..."
                    apt-get update && apt-get install -y jq
                fi
            elif [ -f /etc/redhat-release ]; then
                # RHEL/CentOS/Fedora
                if command -v sudo &>/dev/null; then
                    sudo yum install -y jq
                else
                    echo "⚠️  Warning: sudo not available, trying without sudo..."
                    yum install -y jq
                fi
            elif [ -f /etc/arch-release ]; then
                # Arch Linux
                if command -v sudo &>/dev/null; then
                    sudo pacman -S --noconfirm jq
                else
                    echo "⚠️  Warning: sudo not available, trying without sudo..."
                    pacman -S --noconfirm jq
                fi
            else
                echo "❌ Error: Unsupported Linux distribution"
                echo "Please install jq manually: https://stedolan.github.io/jq/"
                exit 1
            fi
            ;;
        *)
            echo "❌ Error: Cannot install jq automatically on this system"
            echo "Please install jq manually from https://stedolan.github.io/jq/"
            exit 1
            ;;
    esac

    echo "✓ jq installed successfully"
}

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

# Install jq if needed
install_jq

# Detect platform
PLATFORM=$(detect_platform)
echo "✓ Detected platform: $PLATFORM"

# Get latest release version
echo ""
echo "Fetching latest release..."
LATEST_VERSION=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST_VERSION" ]; then
    echo "❌ Error: Could not fetch latest release version"
    echo "Please check your internet connection or try again later"
    exit 1
fi

echo "✓ Latest version: $LATEST_VERSION"

# Download binary
echo ""
echo "Downloading llmsh..."
BINARY_NAME="llmsh-${PLATFORM}"
if [[ "$PLATFORM" == windows* ]]; then
    BINARY_NAME="${BINARY_NAME}.exe"
fi

DOWNLOAD_URL="https://github.com/$REPO/releases/download/$LATEST_VERSION/$BINARY_NAME"

# Create install directory
mkdir -p "$INSTALL_PATH"

# Download the binary
if ! curl -L -o "$INSTALL_PATH/$BINARY" "$DOWNLOAD_URL"; then
    echo "❌ Error: Failed to download binary from $DOWNLOAD_URL"
    exit 1
fi

# Make it executable
chmod +x "$INSTALL_PATH/$BINARY"
echo "✓ Downloaded and installed binary to $INSTALL_PATH/$BINARY"

# Create config directory
echo ""
mkdir -p "$CONFIG_DIR"
echo "✓ Created config directory at $CONFIG_DIR"

# Install ZSH plugin
if [ -f "zsh/llmsh.plugin.zsh" ]; then
    install -m 644 zsh/llmsh.plugin.zsh "$CONFIG_DIR/"
    echo "✓ Installed ZSH plugin to $CONFIG_DIR/llmsh.plugin.zsh"
else
    echo "⚠️  Warning: ZSH plugin not found in repository"
    echo "Downloading from GitHub..."
    curl -L -o "$CONFIG_DIR/llmsh.plugin.zsh" "https://github.com/$REPO/raw/$LATEST_VERSION/zsh/llmsh.plugin.zsh"
    echo "✓ Downloaded ZSH plugin to $CONFIG_DIR/llmsh.plugin.zsh"
fi

# Add to .zshrc if not already present
echo ""
if [ -f "$ZSHRC" ]; then
    if ! grep -q "source $CONFIG_DIR/llmsh.plugin.zsh" "$ZSHRC"; then
        echo "Adding llmsh plugin to $ZSHRC..."
        echo "" >> "$ZSHRC"
        echo "# llmsh - AI-powered shell assistant" >> "$ZSHRC"
        echo "source $CONFIG_DIR/llmsh.plugin.zsh" >> "$ZSHRC"
        echo "✓ Added llmsh plugin to $ZSHRC"
    else
        echo "✓ llmsh plugin already in $ZSHRC"
    fi
else
    echo "⚠️  Warning: $ZSHRC not found"
    echo "Please add this line to your .zshrc manually:"
    echo "  source $CONFIG_DIR/llmsh.plugin.zsh"
fi

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

echo "Next steps:"
echo "1. Initialize config:"
echo "   $BINARY config init"
echo ""
echo "2. Set your API key (choose one):"
echo "   export OPENAI_API_KEY=\"your-api-key\""
echo "   or edit $CONFIG_DIR/config.yaml"
echo ""
echo "3. Reload your shell:"
echo "   source ~/.zshrc"
echo ""
echo "For more information, see README.md and USAGE.md"
