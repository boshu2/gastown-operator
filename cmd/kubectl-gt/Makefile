# kubectl-gt Makefile

BINARY_NAME := kubectl-gt
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X github.com/boshu2/gastown-operator/cmd/kubectl-gt/cmd.Version=$(VERSION)"

.PHONY: all build install clean test fmt lint

all: build

build:
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) .

install: build
	cp bin/$(BINARY_NAME) $(shell go env GOPATH)/bin/

# Install as kubectl plugin (symlink)
install-plugin: build
	ln -sf $(shell pwd)/bin/$(BINARY_NAME) $(shell go env GOPATH)/bin/kubectl-gt

clean:
	rm -rf bin/

test:
	go test -v ./...

fmt:
	go fmt ./...

lint:
	golangci-lint run ./...

# Cross-compile for common platforms
release:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-arm64 .
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-arm64 .
