.PHONY: install uninstall

SCRIPT := resource-owner
BIN_NAME := redig
INSTALL_DIR := $(HOME)/.local/bin

install:
	@mkdir -p $(INSTALL_DIR)
	@ln -sf $(CURDIR)/$(SCRIPT) $(INSTALL_DIR)/$(BIN_NAME)
	@echo "Installed '$(BIN_NAME)' to $(INSTALL_DIR)/$(BIN_NAME)"

uninstall:
	@rm -f $(INSTALL_DIR)/$(BIN_NAME)
	@echo "Removed $(INSTALL_DIR)/$(BIN_NAME)"
