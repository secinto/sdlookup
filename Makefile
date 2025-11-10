.PHONY: build test bench clean install run fmt vet lint help

# Variables
BINARY_NAME=sdlookup
CMD_DIR=./cmd/sdlookup
BUILD_DIR=./build
VERSION?=2.0.0
LDFLAGS=-ldflags "-X main.appVersion=$(VERSION) -s -w"

# Colors for output
CYAN=\033[0;36m
NC=\033[0m # No Color

help: ## Show this help message
	@echo "$(CYAN)sdlookup - Makefile commands:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(CYAN)%-15s$(NC) %s\n", $$1, $$2}'

build: ## Build the binary
	@echo "$(CYAN)Building $(BINARY_NAME)...$(NC)"
	@mkdir -p $(BUILD_DIR)
	@go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "$(CYAN)✓ Build complete: $(BUILD_DIR)/$(BINARY_NAME)$(NC)"

build-all: ## Build for all platforms
	@echo "$(CYAN)Building for all platforms...$(NC)"
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_DIR)
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(CMD_DIR)
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_DIR)
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_DIR)
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_DIR)
	@echo "$(CYAN)✓ Multi-platform build complete$(NC)"

install: ## Install the binary to $GOPATH/bin
	@echo "$(CYAN)Installing $(BINARY_NAME)...$(NC)"
	@go install $(LDFLAGS) $(CMD_DIR)
	@echo "$(CYAN)✓ Installed to $(shell go env GOPATH)/bin/$(BINARY_NAME)$(NC)"

test: ## Run tests
	@echo "$(CYAN)Running tests...$(NC)"
	@go test -v -race -coverprofile=coverage.out ./...
	@echo "$(CYAN)✓ Tests complete$(NC)"

test-coverage: test ## Run tests with coverage report
	@echo "$(CYAN)Generating coverage report...$(NC)"
	@go tool cover -html=coverage.out -o coverage.html
	@echo "$(CYAN)✓ Coverage report: coverage.html$(NC)"

bench: ## Run benchmarks
	@echo "$(CYAN)Running benchmarks...$(NC)"
	@go test -bench=. -benchmem ./...
	@echo "$(CYAN)✓ Benchmarks complete$(NC)"

fmt: ## Format code
	@echo "$(CYAN)Formatting code...$(NC)"
	@go fmt ./...
	@echo "$(CYAN)✓ Code formatted$(NC)"

vet: ## Run go vet
	@echo "$(CYAN)Running go vet...$(NC)"
	@go vet ./...
	@echo "$(CYAN)✓ Vet complete$(NC)"

lint: ## Run golangci-lint (if installed)
	@echo "$(CYAN)Running linter...$(NC)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
		echo "$(CYAN)✓ Lint complete$(NC)"; \
	else \
		echo "$(CYAN)golangci-lint not installed, skipping...$(NC)"; \
	fi

clean: ## Clean build artifacts
	@echo "$(CYAN)Cleaning...$(NC)"
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@go clean
	@echo "$(CYAN)✓ Clean complete$(NC)"

run: build ## Build and run
	@echo "$(CYAN)Running $(BINARY_NAME)...$(NC)"
	@$(BUILD_DIR)/$(BINARY_NAME)

deps: ## Download dependencies
	@echo "$(CYAN)Downloading dependencies...$(NC)"
	@go mod download
	@go mod tidy
	@echo "$(CYAN)✓ Dependencies updated$(NC)"

docker-build: ## Build Docker image
	@echo "$(CYAN)Building Docker image...$(NC)"
	@docker build -t $(BINARY_NAME):$(VERSION) .
	@docker tag $(BINARY_NAME):$(VERSION) $(BINARY_NAME):latest
	@echo "$(CYAN)✓ Docker image built$(NC)"

docker-run: ## Run Docker container
	@echo "$(CYAN)Running Docker container...$(NC)"
	@docker run --rm -i $(BINARY_NAME):latest

ci: fmt vet test ## Run CI checks (format, vet, test)
	@echo "$(CYAN)✓ All CI checks passed$(NC)"

all: clean deps fmt vet test build ## Run all build steps
	@echo "$(CYAN)✓ All tasks complete$(NC)"

.DEFAULT_GOAL := help
