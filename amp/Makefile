.PHONY: install install-sandbox uninstall clean

SCRIPT_DIR := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))/
BIN_DIR := $(HOME)/.local/bin
CONFIG_DIR := $(HOME)/.config/amp-permissions
AMP_SETTINGS := $(HOME)/.config/amp/settings.json

CONFIG_FILES := readonly-commands.txt sensitive-paths.txt reject-patterns.txt \
                always-allowed-paths.txt always-allowed-commands.txt \
                always-ask-patterns.txt interpreters.txt

install-sandbox:
	@command -v go >/dev/null 2>&1 || { echo "Error: go is not installed. Install Go first: https://go.dev/dl/"; exit 1; }
	@mkdir -p $(BIN_DIR)
	@echo "Building amps..."
	@go build -o $(BIN_DIR)/amps $(SCRIPT_DIR)amp-sandboxed.go
	@echo "Built: $(BIN_DIR)/amps"
	@if ! echo "$$PATH" | tr ':' '\n' | sed 's:/*$$::' | grep -qx "$(BIN_DIR)"; then \
		echo ""; \
		echo "Warning: $(BIN_DIR) is not in PATH"; \
		echo "Add to your shell rc:"; \
		echo "  export PATH=\"\$$HOME/.local/bin:\$$PATH\""; \
	fi
	@echo ""
	@echo "Done! Run: amps"

install: install-sandbox
	@mkdir -p $(CONFIG_DIR) $(dir $(AMP_SETTINGS))
	@ln -sf $(SCRIPT_DIR)amp-permission-helper $(BIN_DIR)/
	@chmod +x $(SCRIPT_DIR)amp-permission-helper
	@echo "Linked: amp-permission-helper -> $(BIN_DIR)/"
	@for f in $(CONFIG_FILES); do \
		ln -sf $(SCRIPT_DIR)config/$$f $(CONFIG_DIR)/; \
	done
	@echo "Linked: config files -> $(CONFIG_DIR)/"
	@if [ -f "$(AMP_SETTINGS)" ] && [ ! -L "$(AMP_SETTINGS)" ]; then \
		mv "$(AMP_SETTINGS)" "$(AMP_SETTINGS).bak"; \
		echo "Backed up: $(AMP_SETTINGS) -> $(AMP_SETTINGS).bak"; \
	fi
	@ln -sf $(SCRIPT_DIR)config/settings.json $(AMP_SETTINGS)
	@echo "Linked: settings.json -> $(AMP_SETTINGS)"
	@echo ""
	@echo "Done! Commands: amps, amp-permission-helper"

uninstall:
	@echo "Uninstalling..."
	@rm -f $(BIN_DIR)/amps
	@rm -f $(BIN_DIR)/amp-permission-helper
	@for f in $(CONFIG_FILES); do \
		rm -f $(CONFIG_DIR)/$$f; \
	done
	@rm -f $(AMP_SETTINGS)
	@if [ -f "$(AMP_SETTINGS).bak" ]; then \
		mv "$(AMP_SETTINGS).bak" "$(AMP_SETTINGS)"; \
		echo "Restored: $(AMP_SETTINGS).bak -> $(AMP_SETTINGS)"; \
	fi
	@echo "Uninstall complete"

clean:
	@rm -f $(BIN_DIR)/amps
	@echo "Removed $(BIN_DIR)/amps"
