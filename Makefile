.DEFAULT_GOAL := help

SHELL := /bin/sh

.PHONY: help start format lint test

help: ## Show this help
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z0-9_.-]+:.*?##/ {printf "  %-12s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

start: ## Start with air (hot-reload)
	@if ! command -v air >/dev/null 2>&1; then \
		echo "air not found. Install with:"; \
		echo "  go install github.com/air-verse/air@latest"; \
		exit 1; \
	fi
	@air -c .air.toml

format: ## Format Go sources
	@echo "Formatting Go files..."
	@go fmt ./...

lint: ## Lint (golangci-lint if available, else go vet)
	@if command -v golangci-lint >/dev/null 2>&1; then \
		echo "Running golangci-lint..."; \
		golangci-lint run; \
	else \
		echo "golangci-lint not found. Falling back to 'go vet'"; \
		go vet ./...; \
	fi

test: ## Run unit tests with coverage
	@echo "Running tests..."
	@go test ./... -coverprofile=coverage.out -covermode=atomic
