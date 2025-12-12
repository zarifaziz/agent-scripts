SETTINGS_FILE := $(HOME)/.config/amp/settings.json
MCP_FILE := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))/mcp.json

.PHONY: install uninstall

install:
	@if [ ! -e "$(SETTINGS_FILE)" ]; then \
		echo "Error: $(SETTINGS_FILE) not found"; \
		exit 1; \
	fi
	@jq --slurpfile mcp "$(MCP_FILE)" '.["amp.mcpServers"] = ((.["amp.mcpServers"] // {}) * $$mcp[0])' \
		"$(SETTINGS_FILE)" > "$(SETTINGS_FILE).tmp" && \
		mv "$(SETTINGS_FILE).tmp" "$(SETTINGS_FILE)"
	@echo "Installed playwriter MCP to $(SETTINGS_FILE)"

uninstall:
	@if [ ! -f "$(SETTINGS_FILE)" ]; then \
		echo "Error: $(SETTINGS_FILE) not found"; \
		exit 1; \
	fi
	@jq 'del(.["amp.mcpServers"].playwriter)' "$(SETTINGS_FILE)" > "$(SETTINGS_FILE).tmp" && \
		mv "$(SETTINGS_FILE).tmp" "$(SETTINGS_FILE)"
	@echo "Removed playwriter MCP from $(SETTINGS_FILE)"
