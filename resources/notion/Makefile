BINARY := notion
BIN_DIR := bin

.PHONY: build clean test

build: clean
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(BINARY) .

clean:
	@rm -f $(BINARY) notion-query $(BIN_DIR)/$(BINARY) $(BIN_DIR)/notion-query

test: build
	@./test.sh
