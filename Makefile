# Makefile for KubeAnalysis - Kubernetes Analysis with RAG + Temporal

# Build variables
BINARY_NAME := kubeanalysis
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "0.1.0")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.gitCommit=$(GIT_COMMIT)"

# Go related variables
GOBASE := $(shell pwd)
GOBIN := $(GOBASE)/bin
GOFILES := $(shell find . -type f -name '*.go' -not -path "./vendor/*")
GOPACKAGES := $(shell go list ./... | grep -v /vendor/)

# Docker variables
DOCKER_REGISTRY ?= ghcr.io
DOCKER_IMAGE := $(DOCKER_REGISTRY)/kubeanalysis/kubeanalysis
DOCKER_TAG ?= $(VERSION)

# Test variables
COVERAGE_FILE := coverage.out
COVERAGE_HTML := coverage.html

# Colors for terminal output
COLOR_RESET := \033[0m
COLOR_CYAN := \033[36m
COLOR_GREEN := \033[32m
COLOR_YELLOW := \033[33m

.PHONY: all build clean test coverage lint fmt vet help
.PHONY: docker-build docker-push docker-run
.PHONY: deps tidy vendor
.PHONY: install run worker analyze init-config
.PHONY: proto generate mocks

# Default target
all: deps lint test build

## Build targets
build: ## Build the binary
	@echo "$(COLOR_CYAN)Building $(BINARY_NAME)...$(COLOR_RESET)"
	@go build $(LDFLAGS) -o $(GOBIN)/$(BINARY_NAME) ./cmd/kubeanalysis
	@echo "$(COLOR_GREEN)Build complete: $(GOBIN)/$(BINARY_NAME)$(COLOR_RESET)"

build-linux: ## Build for Linux
	@echo "$(COLOR_CYAN)Building $(BINARY_NAME) for Linux...$(COLOR_RESET)"
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(GOBIN)/$(BINARY_NAME)-linux-amd64 ./cmd/kubeanalysis

build-darwin: ## Build for macOS
	@echo "$(COLOR_CYAN)Building $(BINARY_NAME) for macOS...$(COLOR_RESET)"
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(GOBIN)/$(BINARY_NAME)-darwin-amd64 ./cmd/kubeanalysis
	@GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(GOBIN)/$(BINARY_NAME)-darwin-arm64 ./cmd/kubeanalysis

build-windows: ## Build for Windows
	@echo "$(COLOR_CYAN)Building $(BINARY_NAME) for Windows...$(COLOR_RESET)"
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(GOBIN)/$(BINARY_NAME)-windows-amd64.exe ./cmd/kubeanalysis

build-all: build-linux build-darwin build-windows ## Build for all platforms
	@echo "$(COLOR_GREEN)All builds complete!$(COLOR_RESET)"

install: ## Install the binary to GOPATH/bin
	@echo "$(COLOR_CYAN)Installing $(BINARY_NAME)...$(COLOR_RESET)"
	@go install $(LDFLAGS) ./cmd/kubeanalysis
	@echo "$(COLOR_GREEN)Installed to $(shell go env GOPATH)/bin/$(BINARY_NAME)$(COLOR_RESET)"

## Test targets
test: ## Run tests
	@echo "$(COLOR_CYAN)Running tests...$(COLOR_RESET)"
	@go test -race -v $(GOPACKAGES)

test-short: ## Run tests (short mode)
	@echo "$(COLOR_CYAN)Running tests (short)...$(COLOR_RESET)"
	@go test -short -v $(GOPACKAGES)

test-unit: ## Run unit tests only
	@echo "$(COLOR_CYAN)Running unit tests...$(COLOR_RESET)"
	@go test -race -v -short $(GOPACKAGES)

test-integration: ## Run integration tests
	@echo "$(COLOR_CYAN)Running integration tests...$(COLOR_RESET)"
	@go test -race -v -run Integration $(GOPACKAGES)

coverage: ## Run tests with coverage
	@echo "$(COLOR_CYAN)Running tests with coverage...$(COLOR_RESET)"
	@go test -race -coverprofile=$(COVERAGE_FILE) -covermode=atomic $(GOPACKAGES)
	@go tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "$(COLOR_GREEN)Coverage report: $(COVERAGE_HTML)$(COLOR_RESET)"
	@go tool cover -func=$(COVERAGE_FILE) | tail -1

coverage-report: coverage ## Generate and display coverage report
	@go tool cover -func=$(COVERAGE_FILE)

benchmark: ## Run benchmarks
	@echo "$(COLOR_CYAN)Running benchmarks...$(COLOR_RESET)"
	@go test -bench=. -benchmem $(GOPACKAGES)

## Code quality targets
lint: ## Run linter
	@echo "$(COLOR_CYAN)Running linter...$(COLOR_RESET)"
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run --timeout 5m ./...; \
	else \
		echo "$(COLOR_YELLOW)golangci-lint not installed, running go vet only$(COLOR_RESET)"; \
		go vet $(GOPACKAGES); \
	fi

fmt: ## Format code
	@echo "$(COLOR_CYAN)Formatting code...$(COLOR_RESET)"
	@gofmt -s -w $(GOFILES)
	@echo "$(COLOR_GREEN)Formatting complete$(COLOR_RESET)"

fmt-check: ## Check code formatting
	@echo "$(COLOR_CYAN)Checking code formatting...$(COLOR_RESET)"
	@diff=$$(gofmt -s -d $(GOFILES)); \
	if [ -n "$$diff" ]; then \
		echo "$$diff"; \
		exit 1; \
	fi
	@echo "$(COLOR_GREEN)Formatting check passed$(COLOR_RESET)"

vet: ## Run go vet
	@echo "$(COLOR_CYAN)Running go vet...$(COLOR_RESET)"
	@go vet $(GOPACKAGES)

## Dependency targets
deps: ## Download dependencies
	@echo "$(COLOR_CYAN)Downloading dependencies...$(COLOR_RESET)"
	@go mod download

tidy: ## Tidy go modules
	@echo "$(COLOR_CYAN)Tidying go modules...$(COLOR_RESET)"
	@go mod tidy

vendor: ## Vendor dependencies
	@echo "$(COLOR_CYAN)Vendoring dependencies...$(COLOR_RESET)"
	@go mod vendor

verify-deps: ## Verify dependencies
	@echo "$(COLOR_CYAN)Verifying dependencies...$(COLOR_RESET)"
	@go mod verify

## Docker targets
docker-build: ## Build Docker image
	@echo "$(COLOR_CYAN)Building Docker image...$(COLOR_RESET)"
	@docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	@docker tag $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_IMAGE):latest
	@echo "$(COLOR_GREEN)Built $(DOCKER_IMAGE):$(DOCKER_TAG)$(COLOR_RESET)"

docker-push: docker-build ## Push Docker image
	@echo "$(COLOR_CYAN)Pushing Docker image...$(COLOR_RESET)"
	@docker push $(DOCKER_IMAGE):$(DOCKER_TAG)
	@docker push $(DOCKER_IMAGE):latest

docker-run: ## Run Docker container
	@echo "$(COLOR_CYAN)Running Docker container...$(COLOR_RESET)"
	@docker run --rm -it \
		-v $(HOME)/.kube:/root/.kube:ro \
		-v $(PWD)/config.yaml:/app/config.yaml:ro \
		$(DOCKER_IMAGE):$(DOCKER_TAG)

## Run targets
run: build ## Build and run the binary
	@echo "$(COLOR_CYAN)Running $(BINARY_NAME)...$(COLOR_RESET)"
	@$(GOBIN)/$(BINARY_NAME) $(ARGS)

worker: build ## Run the Temporal worker
	@echo "$(COLOR_CYAN)Starting Temporal worker...$(COLOR_RESET)"
	@$(GOBIN)/$(BINARY_NAME) worker

analyze: build ## Run cluster analysis
	@echo "$(COLOR_CYAN)Running cluster analysis...$(COLOR_RESET)"
	@$(GOBIN)/$(BINARY_NAME) analyze -t full

init-config: build ## Initialize configuration file
	@echo "$(COLOR_CYAN)Initializing configuration...$(COLOR_RESET)"
	@$(GOBIN)/$(BINARY_NAME) init

## Development targets
dev-setup: ## Set up development environment
	@echo "$(COLOR_CYAN)Setting up development environment...$(COLOR_RESET)"
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/golang/mock/mockgen@latest
	@go install github.com/temporalio/temporal/cmd/temporal@latest
	@echo "$(COLOR_GREEN)Development environment ready$(COLOR_RESET)"

temporal-start: ## Start Temporal dev server
	@echo "$(COLOR_CYAN)Starting Temporal development server...$(COLOR_RESET)"
	@temporal server start-dev --namespace kubeanalysis

mocks: ## Generate mocks
	@echo "$(COLOR_CYAN)Generating mocks...$(COLOR_RESET)"
	@go generate ./...

## Clean targets
clean: ## Clean build artifacts
	@echo "$(COLOR_CYAN)Cleaning build artifacts...$(COLOR_RESET)"
	@rm -rf $(GOBIN)
	@rm -f $(COVERAGE_FILE) $(COVERAGE_HTML)
	@go clean -cache -testcache
	@echo "$(COLOR_GREEN)Clean complete$(COLOR_RESET)"

clean-all: clean ## Clean all artifacts including vendor
	@echo "$(COLOR_CYAN)Cleaning all artifacts...$(COLOR_RESET)"
	@rm -rf vendor/

## Release targets
release-check: lint test ## Run all checks before release
	@echo "$(COLOR_GREEN)All release checks passed!$(COLOR_RESET)"

release-notes: ## Generate release notes
	@echo "$(COLOR_CYAN)Generating release notes...$(COLOR_RESET)"
	@git log --oneline $(shell git describe --tags --abbrev=0 2>/dev/null || echo "HEAD~10")..HEAD

## Help
help: ## Display this help message
	@echo ""
	@echo "KubeAnalysis - Kubernetes Analysis with RAG + Temporal"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(COLOR_CYAN)%-20s$(COLOR_RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "Variables:"
	@echo "  VERSION=$(VERSION)"
	@echo "  DOCKER_IMAGE=$(DOCKER_IMAGE)"
	@echo "  DOCKER_TAG=$(DOCKER_TAG)"
	@echo ""
