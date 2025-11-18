.PHONY: build install clean uninstall test deps

BINARY=llmsh
INSTALL_PATH=$(HOME)/.local/bin
CONFIG_DIR=$(HOME)/.llmsh

build:
	@echo "Building $(BINARY)..."
	go build -o $(BINARY) -ldflags="-s -w" .

deps:
	@echo "Installing dependencies..."
	go mod tidy
	go mod download

install: build
	@echo "Installing $(BINARY)..."
	# Create install directory
	@mkdir -p $(INSTALL_PATH)

	# Install binary
	@install -m 755 $(BINARY) $(INSTALL_PATH)/
	@echo "✓ Installed binary to $(INSTALL_PATH)/$(BINARY)"

	# Create config directory
	@mkdir -p $(CONFIG_DIR)
	@echo "✓ Created config directory at $(CONFIG_DIR)"

	# Install ZSH plugin
	@install -m 644 zsh/llmsh.plugin.zsh $(CONFIG_DIR)/
	@echo "✓ Installed ZSH plugin to $(CONFIG_DIR)/llmsh.plugin.zsh"

	@echo ""
	@echo "Installation complete!"
	@echo ""
	@echo "Next steps:"
	@echo "1. Initialize config:"
	@echo "   $(INSTALL_PATH)/$(BINARY) config init"
	@echo ""
	@echo "2. Set your API key (choose one):"
	@echo "   export OPENAI_API_KEY=\"your-api-key\""
	@echo "   or edit $(CONFIG_DIR)/config.yaml"
	@echo ""
	@echo "3. Add to your ~/.zshrc:"
	@echo "   source $(CONFIG_DIR)/llmsh.plugin.zsh"
	@echo ""
	@echo "4. Reload your shell:"
	@echo "   source ~/.zshrc"

clean:
	@echo "Cleaning build artifacts..."
	@rm -f $(BINARY)
	@echo "✓ Cleaned"

uninstall:
	@echo "Uninstalling $(BINARY)..."
	@rm -f $(INSTALL_PATH)/$(BINARY)
	@echo "✓ Removed binary"
	@echo ""
	@echo "To remove all data and ZSH plugin, run:"
	@echo "  rm -rf $(CONFIG_DIR)"
	@echo ""
	@echo "Don't forget to remove this line from ~/.zshrc:"
	@echo "  source $(CONFIG_DIR)/llmsh.plugin.zsh"

test:
	@echo "Running tests..."
	@go test -v ./...
