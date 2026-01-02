.DEFAULT_GOAL := help

VERSION ?= 1.0.0
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
LDFLAGS := -ldflags "-s=false -w=false -X main.version=$(VERSION)"

.PHONY: help
help: ## Show this help message
	@echo "Available targets:"
	@echo ""
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

VERSION ?= dev

.PHONY: build
build: ## Build binary
	@go build $(LDFLAGS) -o ./gha-fetch-digest cmd/main.go

.PHONY: install
install: build ## Install binary to /usr/local/bin/gha-fetch-digest
	@sudo install -m 0755 gha-fetch-digest /usr/local/bin/gha-fetch-digest

.PHONY: test
test: ## Runs vet and test
	@go vet ./...
	@go test -cover -json ./... | go run github.com/mfridman/tparse@ba2512e7be150bfcbd6f6220d517d3741f8f2f75 -all -smallscreen
