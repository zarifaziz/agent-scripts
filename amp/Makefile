.PHONY: install install-strict install-lenient uninstall clean test

SCRIPT_DIR := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))
BIN_DIR := $(SCRIPT_DIR)/bin
LOCAL_BIN := $(HOME)/.local/bin
CONFIG_DIR := $(HOME)/.config/amp-permissions
AMP_SETTINGS := $(HOME)/.config/amp/settings.json
PARSER_DIR := $(SCRIPT_DIR)/amp-permission-parser

CONFIG_FILES := sensitive-paths.txt reject-patterns.txt \
                always-allowed-paths.txt always-ask-patterns.txt \
                interpreters.txt write-commands.txt sandbox-blocked-paths.txt

# Build strict sandbox: samp (security-first)
install-strict:
	@command -v go >/dev/null 2>&1 || { echo "Error: go is not installed"; exit 1; }
	@mkdir -p $(BIN_DIR) $(LOCAL_BIN)
	@echo "Building samp (strict sandbox)..."
	@go build -o $(BIN_DIR)/samp $(SCRIPT_DIR)/amp-sandboxed.go
	@ln -sf $(BIN_DIR)/samp $(LOCAL_BIN)/samp
	@echo "Installed: samp -> $(LOCAL_BIN)/samp"

# Build lenient sandbox + Go parser
install-lenient:
	@command -v go >/dev/null 2>&1 || { echo "Error: go is not installed"; exit 1; }
	@mkdir -p $(BIN_DIR) $(LOCAL_BIN)
	@echo "Building amp-permission-parser..."
	@cd $(PARSER_DIR) && go build -o $(BIN_DIR)/amp-permission-parser
	@echo "Building amps (lenient sandbox)..."
	@go build -o $(BIN_DIR)/amps $(SCRIPT_DIR)/amp-sandbox-lenient.go
	@ln -sf $(BIN_DIR)/amps $(LOCAL_BIN)/amps
	@ln -sf $(BIN_DIR)/amp-permission-parser $(LOCAL_BIN)/amp-permission-parser
	@echo "Installed: amps -> $(LOCAL_BIN)/amps"

# Full install: lenient sandbox + permission helper + configs
install: install-lenient
	@mkdir -p $(CONFIG_DIR) $(dir $(AMP_SETTINGS))
	@# Symlink bash scripts directly
	@ln -sf $(SCRIPT_DIR)/amp-permission-eval $(LOCAL_BIN)/amp-permission-eval
	@ln -sf $(SCRIPT_DIR)/amp-prompt-handler $(LOCAL_BIN)/amp-prompt-handler
	@chmod +x $(SCRIPT_DIR)/amp-permission-eval $(SCRIPT_DIR)/amp-prompt-handler
	@echo "Linked: amp-permission-eval, amp-prompt-handler"
	@# Symlink config files
	@for f in $(CONFIG_FILES); do \
		ln -sf $(SCRIPT_DIR)/config/$$f $(CONFIG_DIR)/; \
	done
	@echo "Linked: config files -> $(CONFIG_DIR)/"
	@# Settings
	@if [ -f "$(AMP_SETTINGS)" ] && [ ! -L "$(AMP_SETTINGS)" ]; then \
		mv "$(AMP_SETTINGS)" "$(AMP_SETTINGS).bak"; \
		echo "Backed up: $(AMP_SETTINGS)"; \
	fi
	@ln -sf $(SCRIPT_DIR)/config/settings.json $(AMP_SETTINGS)
	@echo ""
	@echo "=== Installed ==="
	@echo "  amps  - Lenient sandbox (blocklist, only blocks sensitive paths)"
	@echo "  amp   - No sandbox (uses permission helper via settings.json)"
	@echo ""
	@echo "Run 'make install-strict' for samp (strict allowlist sandbox)"

# Test parser and evaluator
test:
	@echo "=== Parser test ===" && echo 'rm -rf ~ && brew install foo' | $(BIN_DIR)/amp-permission-parser | head -20
	@echo ""
	@echo "=== Eval test ===" && $(SCRIPT_DIR)/amp-permission-eval --parse 'kubectl delete ns prod'

uninstall:
	@echo "Uninstalling..."
	@rm -f $(LOCAL_BIN)/samp $(LOCAL_BIN)/amps
	@rm -f $(LOCAL_BIN)/amp-permission-eval $(LOCAL_BIN)/amp-prompt-handler
	@rm -f $(LOCAL_BIN)/amp-permission-parser
	@for f in $(CONFIG_FILES); do rm -f $(CONFIG_DIR)/$$f; done
	@rm -f $(AMP_SETTINGS)
	@[ -f "$(AMP_SETTINGS).bak" ] && mv "$(AMP_SETTINGS).bak" "$(AMP_SETTINGS)" || true
	@echo "Done"

clean:
	@rm -rf $(BIN_DIR)/*
	@echo "Cleaned bin/"
