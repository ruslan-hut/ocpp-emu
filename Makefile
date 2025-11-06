.PHONY: help build run test clean fmt vet docker-up docker-down docker-build lint

# Default target
.DEFAULT_GOAL := help

# Build variables
BINARY_NAME=ocpp-emu
BUILD_DIR=bin
MAIN_PATH=cmd/server/main.go
VERSION?=0.1.0
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')

# Go variables
GOBASE=$(shell pwd)
GOBIN=$(GOBASE)/$(BUILD_DIR)

help: ## Display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

build: ## Build the application
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build -ldflags="-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILD_TIME)" \
		-o $(GOBIN)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Build complete: $(GOBIN)/$(BINARY_NAME)"

run: ## Run the application
	@echo "Running $(BINARY_NAME)..."
	@go run $(MAIN_PATH)

dev: ## Run with auto-reload (requires air: go install github.com/air-verse/air@latest)
	@which air > /dev/null || (echo "Installing air..." && go install github.com/air-verse/air@latest)
	@air

test: ## Run tests
	@echo "Running tests..."
	@go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...

test-coverage: test ## Run tests with coverage report
	@echo "Generating coverage report..."
	@go tool cover -html=coverage.txt -o coverage.html
	@echo "Coverage report generated: coverage.html"

fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...
	@gofmt -s -w .

vet: ## Run go vet
	@echo "Running go vet..."
	@go vet ./...

lint: ## Run golangci-lint (requires golangci-lint)
	@which golangci-lint > /dev/null || (echo "golangci-lint not found. Install from https://golangci-lint.run/usage/install/" && exit 1)
	@echo "Running golangci-lint..."
	@golangci-lint run ./...

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.txt coverage.html
	@go clean

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

docker-up: ## Start Docker services
	@echo "Starting Docker services..."
	@docker-compose up -d

docker-down: ## Stop Docker services
	@echo "Stopping Docker services..."
	@docker-compose down

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@docker-compose build

docker-logs: ## Show Docker logs
	@docker-compose logs -f

mongo-shell: ## Connect to MongoDB shell
	@docker-compose exec mongodb mongosh ocpp_emu

install-tools: ## Install development tools
	@echo "Installing development tools..."
	@go install github.com/air-verse/air@latest
	@echo "Tools installed successfully"

.PHONY: all
all: clean deps fmt vet test build ## Run all checks and build
