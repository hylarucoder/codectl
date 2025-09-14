.DEFAULT_GOAL := help

SHELL := /bin/sh

.PHONY: help start format lint test docker-test docker-coverage docker-test-run build local-install

# Extra flags for tests when running in Docker build (e.g. -v, -run TestName)
GO_TEST_FLAGS ?= -v
GO_VET_FLAGS  ?=

# Binary/install settings
APP_NAME      ?= codectl
BINDIR        ?= $(HOME)/.local/bin

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

build: ## Build $(APP_NAME) binary
	@echo "Building $(APP_NAME)..."
	@go build -o $(APP_NAME)

test: ## Run unit tests with coverage
	@echo "Running tests..."
	@go test ./... -coverprofile=coverage.out -covermode=atomic

docker-test: ## Run tests inside Docker (build image and export coverage.out). Use GO_TEST_FLAGS to customize (default: -v)
	@if ! command -v docker >/dev/null 2>&1; then \
		echo "docker not found. Please install Docker Desktop or CLI."; \
		exit 1; \
	fi
	@echo "Building test image (Dockerfile.test) with flags: $(GO_TEST_FLAGS)"
	@DOCKER_BUILDKIT=1 docker build \
		--progress=plain \
		--no-cache \
		--build-arg GO_TEST_FLAGS="$(GO_TEST_FLAGS)" \
		--build-arg GO_VET_FLAGS="$(GO_VET_FLAGS)" \
		-f Dockerfile.test -t codectl-tests .
	@echo "Exporting coverage.out from coverage stage..."
	@DOCKER_BUILDKIT=1 docker build \
		--progress=plain \
		--build-arg GO_TEST_FLAGS="$(GO_TEST_FLAGS)" \
		--build-arg GO_VET_FLAGS="$(GO_VET_FLAGS)" \
		-f Dockerfile.test --target coverage -o . .
	@echo "Coverage written to ./coverage.out"

docker-coverage: docker-test ## Alias of docker-test (kept for convenience)

DOCKER_GO_IMAGE ?= golang:1.25.1-bookworm

docker-test-run: ## Run tests in a throwaway container via docker run (streams logs). Use GO_TEST_FLAGS to customize (default: -v)
	@if ! command -v docker >/dev/null 2>&1; then \
		echo "docker not found. Please install Docker Desktop or CLI."; \
		exit 1; \
	fi
	@echo "Running tests in container ($(DOCKER_GO_IMAGE)) with flags: $(GO_TEST_FLAGS)"
	@docker run --rm \
		-v "$(PWD)":/src \
		-w /src \
		-e CGO_ENABLED=0 \
		-e GOFLAGS=-buildvcs=false \
		-v codectl-go-mod-cache:/go/pkg/mod \
		-v codectl-go-build-cache:/root/.cache/go-build \
		$(DOCKER_GO_IMAGE) bash -c 'go mod download && go vet $(GO_VET_FLAGS) ./... && go test $(GO_TEST_FLAGS) ./... -coverprofile=coverage.out -covermode=atomic'
	@echo "Coverage written to ./coverage.out"

staging: build ## Install $(APP_NAME) to local bin (~/.local/bin by default; override with BINDIR=/path)
	@set -e; \
		dest="$(BINDIR)"; \
		bin_name="$(APP_NAME)"; \
		echo "Installing ./$$bin_name to $$dest/$$bin_name"; \
		mkdir -p "$$dest"; \
		install -m 0755 "./$$bin_name" "$$dest/$$bin_name"; \
		case ":$$PATH:" in *":$$dest:"*) \
			echo "Installed to $$dest (in PATH)" ;; \
		*) \
			echo "Installed to $$dest"; \
			echo "Note: $$dest is not in PATH. Add it, e.g.:"; \
			echo "  export PATH=\"$$dest:\$$PATH\""; \
		esac
