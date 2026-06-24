.PHONY: build install test clean fmt lint help

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -s -w"

help: ## Show this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

build: ## Build the ctx binary
	go build $(LDFLAGS) -o ctx ./cmd/ctx

install: ## Install ctx to GOPATH/bin
	go install $(LDFLAGS) ./cmd/ctx

test: ## Run tests
	go test ./... -v

fmt: ## Format code
	go fmt ./...

lint: ## Run linter (requires golangci-lint)
	golangci-lint run || true

clean: ## Remove built binaries
	rm -f ctx ctxengine ctxengine-mcp ctxengine-server ctxeval

all: fmt build test ## Format, build, and test
