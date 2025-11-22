BIN_DIR?=$(HOME)/.local/bin
BUILD_DIR:=bin
BINARY:=sane

.PHONY: build install clean

build:
	@mkdir -p $(BUILD_DIR)
	GO111MODULE=on go build -o $(BUILD_DIR)/$(BINARY) ./cmd/sane

install: build
	@mkdir -p $(BIN_DIR)
	ln -sf $(PWD)/$(BUILD_DIR)/$(BINARY) $(BIN_DIR)/$(BINARY)
	@echo "Installed $(BINARY) -> $(BIN_DIR)/$(BINARY)"

clean:
	rm -rf $(BUILD_DIR)/$(BINARY)
