.PHONY: test test-verbose test-coverage lint clean fmt install help

# Variables
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

# Default target
.DEFAULT_GOAL := help

# Run tests
test:
	go test ./...

# Run tests with verbose output
test-verbose:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run linting checks (same as CI)
lint:
	@echo "Running linters..."
	@echo "Checking formatting..."
	@UNFORMATTED=$$(gofmt -l .); \
	if [ -n "$$UNFORMATTED" ]; then \
		echo "Code is not formatted. Run 'make fmt' to fix:"; \
		echo "$$UNFORMATTED"; \
		exit 1; \
	fi
	@go mod tidy
	@if ! git diff --quiet go.mod go.sum; then echo "go.mod or go.sum is not tidy, run 'go mod tidy'"; git diff go.mod go.sum; exit 1; fi
	@if ! command -v golangci-lint &> /dev/null; then echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; fi
	@golangci-lint run --timeout=5m

# Format code
fmt:
	go fmt ./...
	gofmt -s -w .

# Clean build artifacts
clean:
	rm -f coverage.out coverage.html
	go clean -cache -testcache

# Install as local module (for testing)
install:
	go mod download
	go mod tidy

# Check for security vulnerabilities
vuln:
	@if ! command -v govulncheck &> /dev/null; then echo "Installing govulncheck..." && go install golang.org/x/vuln/cmd/govulncheck@latest; fi
	govulncheck ./...

# Help target
help:
	@echo "oauth-mcp-proxy Makefile targets:"
	@echo ""
	@echo "  make test            Run tests"
	@echo "  make test-verbose    Run tests with verbose output"
	@echo "  make test-coverage   Run tests with coverage report"
	@echo "  make lint            Run linters (same as CI)"
	@echo "  make fmt             Format code"
	@echo "  make clean           Clean build artifacts"
	@echo "  make install         Download dependencies"
	@echo "  make vuln            Check for security vulnerabilities"
	@echo "  make help            Show this help message"
	@echo ""
	@echo "Version: $(VERSION)"
