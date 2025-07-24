.PHONY: test test-unit test-integration test-coverage build clean help

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=ntfy-to-slack

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

test-coverage: ## Run tests with coverage
	$(GOTEST) -v -coverprofile=coverage.out ./tests/...
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

lint: ## Run go fmt and go vet
	gofmt -s -w .
	$(GOCMD) vet ./cmd/... ./internal/... ./tests/...

all: clean deps lint test build ## Run full build pipeline