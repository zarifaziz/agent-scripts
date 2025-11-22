BIN_DIR?=$(HOME)/.local/bin
BUILD_DIR:=bin
BINARY:=sane
WEBBIN:=sanityweb

.PHONY: build install clean

build:
	@mkdir -p $(BUILD_DIR)
	GO111MODULE=on go build -o $(BUILD_DIR)/$(BINARY) ./cmd/sane
	GO111MODULE=on go build -o $(BUILD_DIR)/$(WEBBIN) ./cmd/sanityweb

install: build
	@mkdir -p $(BIN_DIR)
	ln -sf $(PWD)/$(BUILD_DIR)/$(BINARY) $(BIN_DIR)/$(BINARY)
	ln -sf $(PWD)/$(BUILD_DIR)/$(WEBBIN) $(BIN_DIR)/$(WEBBIN)
	@echo "Installed $(BINARY) -> $(BIN_DIR)/$(BINARY)"
	@echo "Installed $(WEBBIN) -> $(BIN_DIR)/$(WEBBIN)"

clean:
	rm -rf $(BUILD_DIR)/$(BINARY)
