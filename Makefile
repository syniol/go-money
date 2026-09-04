.PHONY: all build test test-race test-cover vet fmt lint clean help generate bench bench-compare fuzz install-tools deps check

# Default Go settings
GO := go
GOFLAGS := -v
TIMEOUT := 10m
RACE_TIMEOUT := 15m

# Project settings
BINARY_NAME := go-money
PKG := ./...

# Default target
all: check test build

# Build the project
build:
	@echo "Building..."
	$(GO) build $(GOFLAGS) $(PKG)

# Build with optimizations (for release)
build-release:
	@echo "Building release..."
	CGO_ENABLED=0 $(GO) build -ldflags="-s -w" -a -installsuffix cgo $(PKG)

# Run tests
test:
	@echo "Running tests..."
	$(GO) test $(GOFLAGS) -timeout $(TIMEOUT) $(PKG)

# Run tests with race detection
test-race:
	@echo "Running tests with race detection..."
	$(GO) test $(GOFLAGS) -race -timeout $(RACE_TIMEOUT) $(PKG)

# Run tests with coverage
test-cover:
	@echo "Running tests with coverage..."
	$(GO) test $(GOFLAGS) -timeout $(TIMEOUT) -coverprofile=coverage.out -covermode=atomic $(PKG)
	$(GO) tool cover -html=coverage.out -o coverage.html

# Run tests with coverage and race detection
test-cover-race:
	@echo "Running tests with coverage and race detection..."
	$(GO) test $(GOFLAGS) -race -timeout $(RACE_TIMEOUT) -coverprofile=coverage.out -covermode=atomic $(PKG)
	$(GO) tool cover -html=coverage.out -o coverage.html

# Run go vet
vet:
	@echo "Running go vet..."
	$(GO) vet $(PKG)

# Format code
fmt:
	@echo "Formatting code..."
	$(GO) fmt $(PKG)
	goimports -w -local github.com/syniol/go-money .

# Check formatting
fmt-check:
	@echo "Checking formatting..."
	@test -z "$$(gofmt -l . | grep -v vendor)" || (echo "Code not formatted correctly" && exit 1)
	@test -z "$$(goimports -l -local github.com/syniol/go-money . | grep -v vendor)" || (echo "Imports not formatted correctly" && exit 1)

# Run golangci-lint
lint:
	@echo "Running golangci-lint..."
	golangci-lint run

# Fix linting issues automatically where possible
lint-fix:
	@echo "Running golangci-lint with auto-fix..."
	golangci-lint run --fix

# Generate code (currencies)
generate:
	@echo "Generating code..."
	$(GO) generate $(PKG)

# Run in-repo benchmarks
bench:
	@echo "Running benchmarks..."
	$(GO) test $(GOFLAGS) -bench=. -benchmem $(PKG)

# Run cross-library comparison benchmarks (Rhymond, bojanz, leekchan).
# See BENCHMARK.md for the current table and interpretation.
bench-compare:
	@echo "Running comparison benchmarks..."
	cd benchmarks && $(GO) test -bench=. -benchmem -run=^$$ -benchtime=3s -count=1

# Run fuzz tests
fuzz:
	@echo "Running fuzz tests..."
	$(GO) test -fuzz=FuzzNewFromString -fuzztime=30s

# Install development tools
install-tools:
	@echo "Installing development tools..."
	$(GO) install golang.org/x/tools/cmd/goimports@latest
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GO) mod download
	$(GO) mod verify

# Tidy dependencies
deps-tidy:
	@echo "Tidying dependencies..."
	$(GO) mod tidy

# Update dependencies
deps-update:
	@echo "Updating dependencies..."
	$(GO) get -u $(PKG)
	$(GO) mod tidy

# Run security check
security:
	@echo "Running security checks..."
	gosec $(PKG)

# Install gosec if not present and run security check
security-install:
	@command -v gosec >/dev/null 2>&1 || $(GO) install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	@$(MAKE) security

# Comprehensive check (what CI typically runs)
check: deps fmt-check vet lint test-race

# Quick check (faster feedback loop)
quick-check: fmt vet test

# Clean build artifacts and caches
clean:
	@echo "Cleaning..."
	$(GO) clean $(PKG)
	rm -f coverage.out coverage.html
	$(GO) clean -cache
	$(GO) clean -testcache
	$(GO) clean -modcache

# Run CI pipeline locally
ci: clean install-tools check test-cover build-release

# Development workflow
dev: fmt vet test

# Docker build (if needed)
docker-build:
	@echo "Building Docker image..."
	docker build -t $(BINARY_NAME) .

# Show help
help:
	@echo "Available targets:"
	@echo "  all          - Run check, test, and build"
	@echo "  build        - Build the project"
	@echo "  build-release- Build optimized release binary"
	@echo "  test         - Run tests"
	@echo "  test-race    - Run tests with race detection"
	@echo "  test-cover   - Run tests with coverage report"
	@echo "  test-cover-race - Run tests with coverage and race detection"
	@echo "  vet          - Run go vet"
	@echo "  fmt          - Format code"
	@echo "  fmt-check    - Check code formatting"
	@echo "  lint         - Run golangci-lint"
	@echo "  lint-fix     - Run golangci-lint with auto-fix"
	@echo "  generate     - Generate code"
	@echo "  bench        - Run benchmarks"
	@echo "  fuzz         - Run fuzz tests"
	@echo "  install-tools- Install development tools"
	@echo "  deps         - Download dependencies"
	@echo "  deps-tidy    - Tidy dependencies"
	@echo "  deps-update  - Update dependencies"
	@echo "  security     - Run security checks"
	@echo "  check        - Run comprehensive checks (CI-like)"
	@echo "  quick-check  - Run quick development checks"
	@echo "  clean        - Clean build artifacts and caches"
	@echo "  ci           - Run full CI pipeline locally"
	@echo "  dev          - Quick development workflow"
	@echo "  help         - Show this help message"

# Targets for specific environments
.PHONY: ci-build ci-test ci-lint

ci-build:
	@echo "CI: Building..."
	$(GO) build $(GOFLAGS) $(PKG)

ci-test:
	@echo "CI: Running tests with race detection and coverage..."
	$(GO) test $(GOFLAGS) -race -timeout $(RACE_TIMEOUT) -coverprofile=coverage.out -covermode=atomic $(PKG)

ci-lint:
	@echo "CI: Running linter..."
	golangci-lint run --timeout $(TIMEOUT)