.PHONY: all build clean install install-ts install-go

BIN_DIR = bin
INSTALL_DIR = $(HOME)/.local/bin
TS_DIR = standalone-ts

# Go commands (only branch-namer remains)
GO_CMDS = branch-namer

# Standalone TypeScript scripts (bun shebang)
TS_SCRIPTS = autist db-oracle session-hunter

all: build

build:
	@mkdir -p $(BIN_DIR)
	@for cmd in $(GO_CMDS); do \
		echo "Building $$cmd..."; \
		go build -o $(BIN_DIR)/$$cmd ./cmd/$$cmd; \
	done
	@echo "Done! Binaries in $(BIN_DIR)/"

# Install all tools
install: build install-ts install-go

# Install all TS scripts
install-ts:
	@mkdir -p $(INSTALL_DIR)
	@for script in $(TS_SCRIPTS); do \
		echo "Symlinking $$script to $(INSTALL_DIR)/$$script"; \
		rm -f $(INSTALL_DIR)/$$script; \
		ln -s $(CURDIR)/$(TS_DIR)/$$script $(INSTALL_DIR)/$$script; \
	done
	@echo "Done! TS scripts symlinked to $(INSTALL_DIR)/"

# Install all Go binaries
install-go: build
	@mkdir -p $(INSTALL_DIR)
	@for cmd in $(GO_CMDS); do \
		echo "Symlinking $$cmd to $(INSTALL_DIR)/$$cmd"; \
		rm -f $(INSTALL_DIR)/$$cmd; \
		ln -s $(CURDIR)/$(BIN_DIR)/$$cmd $(INSTALL_DIR)/$$cmd; \
	done
	@echo "Done! Go binaries symlinked to $(INSTALL_DIR)/"

# Install a single tool: make install-tool NAME=autist
install-tool:
ifndef NAME
	$(error NAME is required. Usage: make install-tool NAME=autist)
endif
	@mkdir -p $(INSTALL_DIR)
	@if [ -f "$(TS_DIR)/$(NAME)" ]; then \
		echo "Symlinking $(NAME) to $(INSTALL_DIR)/$(NAME)"; \
		rm -f $(INSTALL_DIR)/$(NAME); \
		ln -s $(CURDIR)/$(TS_DIR)/$(NAME) $(INSTALL_DIR)/$(NAME); \
	elif [ -d "cmd/$(NAME)" ]; then \
		echo "Building and symlinking $(NAME)..."; \
		mkdir -p $(BIN_DIR); \
		go build -o $(BIN_DIR)/$(NAME) ./cmd/$(NAME); \
		rm -f $(INSTALL_DIR)/$(NAME); \
		ln -s $(CURDIR)/$(BIN_DIR)/$(NAME) $(INSTALL_DIR)/$(NAME); \
	else \
		echo "Error: Tool '$(NAME)' not found in standalone-ts/ or cmd/"; \
		exit 1; \
	fi
	@echo "Done! $(NAME) installed to $(INSTALL_DIR)/"

clean:
	rm -rf $(BIN_DIR)
