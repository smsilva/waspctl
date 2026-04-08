.PHONY: build docker-build test clean

BINARY_NAME := waspctl
INSTALL_DIR := ~/bin
IMAGE_TAG   := $(BINARY_NAME):dev

build:
	go build -o $(INSTALL_DIR)/$(BINARY_NAME) .

docker-build:
	docker build -t $(IMAGE_TAG) .

docker-run:
	docker run --rm -v $(HOME)/.wasp:/root/.wasp $(IMAGE_TAG) config --set provider aws
	docker run --rm -v $(HOME)/.wasp:/root/.wasp $(IMAGE_TAG) config --list --output json

test:
	go test -c -o /tmp/config.test ./internal/config/ && /tmp/config.test -test.v

clean:
	rm -f $(INSTALL_DIR)/$(BINARY_NAME)
