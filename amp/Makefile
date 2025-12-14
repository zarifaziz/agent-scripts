.PHONY: install install-strict uninstall clean test

SCRIPT_DIR := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))
BIN_DIR := $(SCRIPT_DIR)/bin
LOCAL_BIN := $(HOME)/.local/bin
CONFIG_DIR := $(HOME)/.config/amp-permissions
AMP_SETTINGS := $(HOME)/.config/amp/settings.json
PERMISSION_DIR := $(SCRIPT_DIR)/amp-permission

# Config file with amp.permissions and amp.guardedFiles.allowlist
AMP_INJECT_CONFIG := $(SCRIPT_DIR)/config/amp-settings.json

# Build strict sandbox: samp (security-first, allowlist)
install-strict:
	@command -v go >/dev/null 2>&1 || { echo "Error: go is not installed"; exit 1; }
	@mkdir -p $(BIN_DIR) $(LOCAL_BIN)
	@echo "Building samp (strict sandbox)..."
	@go build -o $(BIN_DIR)/samp $(SCRIPT_DIR)/amp-sandboxed.go
	@ln -sf $(BIN_DIR)/samp $(LOCAL_BIN)/samp
	@echo "Installed: samp -> $(LOCAL_BIN)/samp"

# Full install: permission system + lenient sandbox
install:
	@command -v go >/dev/null 2>&1 || { echo "Error: go is not installed"; exit 1; }
	@command -v jq >/dev/null 2>&1 || { echo "Error: jq is not installed (brew install jq)"; exit 1; }
	@mkdir -p $(BIN_DIR) $(LOCAL_BIN) $(CONFIG_DIR)
	@# Check user's settings.json exists
	@if [ ! -f "$(AMP_SETTINGS)" ]; then \
		echo "Error: $(AMP_SETTINGS) not found"; \
		echo "Please run 'amp' at least once to create it"; \
		exit 1; \
	fi
	@echo "Building amp-permission..."
	@cd $(PERMISSION_DIR) && go build -o $(BIN_DIR)/amp-permission
	@echo "Building amps (lenient sandbox)..."
	@go build -o $(BIN_DIR)/amps $(SCRIPT_DIR)/amp-sandbox-lenient.go
	@# Install binaries
	@ln -sf $(BIN_DIR)/amps $(LOCAL_BIN)/amps
	@ln -sf $(BIN_DIR)/amp-permission $(LOCAL_BIN)/amp-permission
	@# Prompt handler (bash script)
	@ln -sf $(SCRIPT_DIR)/amp-prompt-handler $(LOCAL_BIN)/amp-prompt-handler
	@chmod +x $(SCRIPT_DIR)/amp-prompt-handler
	@# Config
	@ln -sf $(SCRIPT_DIR)/config/permissions.yaml $(CONFIG_DIR)/
	@# Update user's settings.json: remove dangerouslyAllowAll, merge our config
	@echo "Updating $(AMP_SETTINGS)..."
	@cp "$(AMP_SETTINGS)" "$(AMP_SETTINGS).bak"
	@HOME_ESCAPED=$$(echo "$(HOME)" | sed 's/\//\\\//g'); \
	sed "s/\$$HOME/$$HOME_ESCAPED/g" "$(AMP_INJECT_CONFIG)" > /tmp/amp-inject.json
	@jq -s '.[0] * .[1] | del(.["amp.dangerouslyAllowAll"])' \
		"$(AMP_SETTINGS).bak" /tmp/amp-inject.json > "$(AMP_SETTINGS)"
	@rm -f /tmp/amp-inject.json
	@echo ""
	@echo "=== Installed ==="
	@echo "  amp-permission     - Go binary (Amp calls this via settings.json)"
	@echo "  amp-prompt-handler - Bash prompt UI (osascript/tmux)"
	@echo "  amps               - Lenient sandbox wrapper"
	@echo ""
	@echo "Config: $(CONFIG_DIR)/permissions.yaml"
	@echo "Settings: $(AMP_SETTINGS) (backup: .bak)"
	@echo ""
	@echo "Override prompt handler: AMP_PERMISSION_PROMPT_HANDLER=/path/to/handler"
	@echo ""

test:
	@tmp=$$(mktemp); \
	if $(SCRIPT_DIR)/test-permissions.sh > "$$tmp" 2>&1; then \
		printf "  ✓ permission tests\n"; \
	else \
		cat "$$tmp"; \
		exit 1; \
	fi; \
	rm -f "$$tmp"

uninstall:
	@echo "Uninstalling..."
	@rm -f $(LOCAL_BIN)/samp $(LOCAL_BIN)/amps
	@rm -f $(LOCAL_BIN)/amp-permission
	@rm -f $(LOCAL_BIN)/amp-prompt-handler
	@rm -f $(CONFIG_DIR)/permissions.yaml
	@# Restore settings.json backup if exists
	@if [ -f "$(AMP_SETTINGS).bak" ]; then \
		mv "$(AMP_SETTINGS).bak" "$(AMP_SETTINGS)"; \
		echo "Restored: $(AMP_SETTINGS)"; \
	fi
	@echo "Done"

clean:
	@rm -rf $(BIN_DIR)/*
	@echo "Cleaned bin/"
