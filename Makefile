# ECR Prometheus Exporter Makefile

.PHONY: help build test lint clean run dev deps security

# Default target
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Build targets
build: ## Build the binary
	@echo "Building ECR exporter..."
	CGO_ENABLED=0 go build -a -installsuffix cgo -o ecr-exporter .

build-all: ## Build binaries for all platforms
	@echo "Building for all platforms..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o dist/ecr-exporter-linux-amd64 .
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -a -installsuffix cgo -o dist/ecr-exporter-darwin-amd64 .
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -a -installsuffix cgo -o dist/ecr-exporter-darwin-arm64 .
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -a -installsuffix cgo -o dist/ecr-exporter-windows-amd64.exe .

# Development targets
deps: ## Download and verify dependencies
	go mod download
	go mod verify
	go mod tidy

test: ## Run tests
	go test -v -race -coverprofile=coverage.out ./...

test-coverage: test ## Run tests and show coverage
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

lint: ## Run linter
	golangci-lint run

lint-fix: ## Run linter with auto-fix
	golangci-lint run --fix

security: ## Run security scan
	gosec ./...

# Development targets
run: ## Run the application locally
	LOG_LEVEL=info go run .

dev: ## Run the application with debug logging
	LOG_LEVEL=debug go run .

# Utility targets
clean: ## Clean build artifacts
	rm -f ecr-exporter ecr-exporter-* *.exe
	rm -rf dist/
	rm -f coverage.out coverage.html

fmt: ## Format code
	go fmt ./...
	goimports -w .

check: deps lint test security ## Run all checks (deps, lint, test, security)

ci: check build ## Run CI pipeline locally

# Install development tools
install-tools: ## Install development tools
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	go install golang.org/x/tools/cmd/goimports@latest