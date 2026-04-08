.PHONY: build test clean

BINARY_NAME := waspctl
INSTALL_DIR := ~/bin

build:
	go build -o $(INSTALL_DIR)/$(BINARY_NAME) .

test:
	go test -c -o /tmp/config.test ./internal/config/ && /tmp/config.test -test.v

clean:
	rm -f $(INSTALL_DIR)/$(BINARY_NAME)
