.PHONY: install uninstall

SCRIPT_DIR := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))/
BIN_DIR := $(HOME)/.local/bin
CONFIG_DIR := $(HOME)/.config/amp-permissions
AMP_SETTINGS := $(HOME)/.config/amp/settings.json

CONFIG_FILES := readonly-commands.txt sensitive-paths.txt reject-patterns.txt \
                always-allowed-paths.txt always-allowed-commands.txt \
                always-ask-patterns.txt interpreters.txt

install:
	@echo "Installing amp sandbox tools..."
	@mkdir -p $(BIN_DIR) $(CONFIG_DIR) $(dir $(AMP_SETTINGS))
	@ln -sf $(SCRIPT_DIR)amp-permission-helper $(BIN_DIR)/
	@ln -sf $(SCRIPT_DIR)amp-sandboxed $(BIN_DIR)/
	@ln -sf $(SCRIPT_DIR)amp.sb $(BIN_DIR)/
	@chmod +x $(SCRIPT_DIR)amp-permission-helper $(SCRIPT_DIR)amp-sandboxed
	@echo "Linked: amp-permission-helper, amp-sandboxed, amp.sb -> $(BIN_DIR)/"
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
	@echo "Done! Regular amp now uses permission-helper by default."
	@echo "For stricter sandbox mode: amp-sandboxed"
	@echo "Or add alias: alias amps='amp-sandboxed'"

uninstall:
	@echo "Uninstalling amp sandbox tools..."
	@rm -f $(BIN_DIR)/amp-permission-helper
	@rm -f $(BIN_DIR)/amp-sandboxed
	@rm -f $(BIN_DIR)/amp.sb
	@for f in $(CONFIG_FILES); do \
		rm -f $(CONFIG_DIR)/$$f; \
	done
	@rm -f $(AMP_SETTINGS)
	@if [ -f "$(AMP_SETTINGS).bak" ]; then \
		mv "$(AMP_SETTINGS).bak" "$(AMP_SETTINGS)"; \
		echo "Restored: $(AMP_SETTINGS).bak -> $(AMP_SETTINGS)"; \
	fi
	@echo "Uninstall complete"
