.PHONY: test test-unit test-integration test-coverage build clean help fmt fmt-check lint lint-golangci

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=ntfy-to-slack

# Keep in step with .github/workflows/test.yml so local runs match CI.
GOLANGCI_LINT_VERSION=v2.12.2
GOLANGCI_LINT_ARGS=--disable=errcheck --enable=govet,ineffassign,staticcheck --timeout=5m

help: ## Show this help message
	@echo 'Usage:'
	@echo '  make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

test: ## Run all tests
	$(GOTEST) -v ./tests/...

test-unit: ## Run unit tests only
	$(GOTEST) -v ./tests/unit/...

test-integration: ## Run integration tests only
	$(GOTEST) -v ./tests/integration/...

test-coverage: ## Run tests with coverage (matches CI's -coverpkg so local numbers agree with the Coverage badge)
	$(GOTEST) -v -coverprofile=coverage.out -coverpkg=./... ./tests/...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-short: ## Run tests in short mode
	$(GOTEST) -v -short ./tests/...

build: ## Build the binary
	$(GOBUILD) -o $(BINARY_NAME) -v ./cmd/ntfy-to-slack

clean: ## Clean build artifacts and test files
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html

deps: ## Download dependencies
	$(GOMOD) download
	$(GOMOD) tidy

verify: ## Verify dependencies and run tests
	$(GOMOD) verify
	make test

fmt: ## Format code with go fmt
	gofmt -s -w .

fmt-check: ## Check if code is properly formatted
	@if [ "$$(gofmt -s -l . | wc -l)" -gt 0 ]; then \
		echo "Code is not properly formatted:"; \
		gofmt -s -l .; \
		exit 1; \
	fi

lint: fmt-check ## Run formatting check and go vet
	$(GOCMD) vet ./cmd/... ./internal/... ./tests/...

lint-golangci: ## Run golangci-lint with the same version and flags as CI
	@command -v golangci-lint >/dev/null 2>&1 || { \
		echo "golangci-lint not found. Install the official $(GOLANGCI_LINT_VERSION) release binary:"; \
		echo "  https://github.com/golangci/golangci-lint/releases/tag/$(GOLANGCI_LINT_VERSION)"; \
		echo ""; \
		echo "Do not use 'go install': it builds golangci-lint with the Go toolchain"; \
		echo "named in golangci-lint's own go.mod, which is older than the one this"; \
		echo "module targets, and the resulting binary refuses to lint it."; \
		exit 1; \
	}
	@installed=$$(golangci-lint version --short 2>/dev/null || echo unknown); \
	if [ "v$$installed" != "$(GOLANGCI_LINT_VERSION)" ]; then \
		echo "warning: golangci-lint v$$installed installed, CI uses $(GOLANGCI_LINT_VERSION)"; \
	fi
	golangci-lint run $(GOLANGCI_LINT_ARGS)

all: clean deps lint test build ## Run full build pipeline