# Makefile for FlowGraph - Enforces KISS, YAGNI, SOLID, DRY principles
.PHONY: help clean build test coverage lint principle-check install-tools dev

# Default target
.DEFAULT_GOAL := help

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=flowgraph
BINARY_PATH=./bin/$(BINARY_NAME)

# Build information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u '+%Y-%m-%d_%H:%M:%S')

# Build flags
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildTime=$(BUILD_TIME) -s -w"

## help: Show this help message
help:
	@echo "FlowGraph - Graph-based Workflow Execution in Go"
	@echo ""
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

## dev: Start development with file watching
dev:
	@echo "🚀 Starting development environment..."
	@if command -v air >/dev/null 2>&1; then \
		air -c .air.toml; \
	else \
		echo "❌ air not installed. Run 'make install-tools' first"; \
		exit 1; \
	fi

## install-tools: Install development tools
install-tools:
	@echo "🔧 Installing development tools..."
	@go install github.com/cosmtrek/air@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install honnef.co/go/tools/cmd/staticcheck@latest
	@go install github.com/fzipp/gocyclo/cmd/gocyclo@latest
	@go install github.com/mibk/dupl@latest
	@echo "✅ Development tools installed"

##@ Quality Assurance

## principle-check: Validate architectural principles (KISS, YAGNI, SOLID, DRY)
principle-check:
	@echo "🔍 Validating architectural principles..."
	@./scripts/principle-check.sh

## lint: Run linter with architectural rules
lint:
	@echo "🔍 Running linter with architectural principles..."
	@golangci-lint run --config .golangci.yml --timeout=5m

## test: Run all tests
test:
	@echo "🧪 Running tests..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...

## coverage: Generate and display coverage report
coverage: test
	@echo "📊 Generating coverage report..."
	@$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@$(GOCMD) tool cover -func=coverage.out | grep total | awk '{printf "Total Coverage: %s\n", $$3}'

## coverage-check: Check coverage threshold (95%)
coverage-check: test
	@echo "📊 Checking coverage threshold..."
	@COVERAGE=$$($(GOCMD) tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	if [ "$$(echo "$$COVERAGE < 95" | bc -l)" -eq 1 ]; then \
		echo "❌ Coverage $$COVERAGE% is below 95% threshold"; \
		exit 1; \
	else \
		echo "✅ Coverage $$COVERAGE% meets 95% threshold"; \
	fi

## test-integration: Run integration tests
test-integration:
	@echo "🧪 Running integration tests..."
	$(GOTEST) -v -tags=integration ./test/integration/...

##@ Build and Deploy

## build: Build the binary
build: clean
	@echo "🔨 Building $(BINARY_NAME)..."
	@mkdir -p bin
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_PATH) ./cmd/flowgraph/

## clean: Clean build artifacts
clean:
	@echo "🧹 Cleaning..."
	@$(GOCLEAN)
	@rm -rf bin/
	@rm -f coverage.out coverage.html

## deps: Download and verify dependencies
deps:
	@echo "📦 Downloading dependencies..."
	@$(GOMOD) download
	@$(GOMOD) verify

## tidy: Clean up go.mod and go.sum
tidy:
	@echo "🧹 Tidying go modules..."
	@$(GOMOD) tidy

##@ Docker

## docker-build: Build Docker image
docker-build:
	@echo "🐳 Building Docker image..."
	@docker build -t flowgraph:$(VERSION) .

## docker-run: Run Docker container
docker-run:
	@echo "🐳 Running Docker container..."
	@docker run --rm -it flowgraph:$(VERSION)

##@ Pre-commit Checks

## pre-commit: Run all pre-commit checks (must pass before commit)
pre-commit: deps tidy lint principle-check test coverage-check
	@echo "✅ All pre-commit checks passed!"

## ci: Run CI pipeline locally
ci: pre-commit test-integration
	@echo "✅ CI pipeline completed successfully!"

##@ Security

## security: Run security audit
security:
	@echo "🔒 Running security audit..."
	@$(GOCMD) list -json -m all | nancy sleuth

## vuln-check: Check for known vulnerabilities
vuln-check:
	@echo "🔍 Checking for vulnerabilities..."
	@govulncheck ./...

##@ Maintenance

## update-deps: Update all dependencies
update-deps:
	@echo "⬆️  Updating dependencies..."
	@$(GOGET) -u ./...
	@$(GOMOD) tidy

## vendor: Create vendor directory
vendor:
	@echo "📦 Creating vendor directory..."
	@$(GOMOD) vendor

## format: Format code
format:
	@echo "✨ Formatting code..."
	@$(GOCMD) fmt ./...
	@goimports -w .

## mod-graph: Show module dependency graph
mod-graph:
	@echo "📊 Module dependency graph:"
	@$(GOMOD) graph

##@ Documentation

## docs: Generate documentation
docs:
	@echo "📚 Generating documentation..."
	@$(GOCMD) doc -all ./...

## godoc: Start godoc server
godoc:
	@echo "📚 Starting godoc server on :6060..."
	@godoc -http=:6060

##@ Benchmarks

## benchmark: Run benchmarks
benchmark:
	@echo "⚡ Running benchmarks..."
	@$(GOTEST) -bench=. -benchmem ./...

## profile: Run with profiling
profile:
	@echo "📊 Running with profiling..."
	@$(GOTEST) -cpuprofile=cpu.prof -memprofile=mem.prof -bench=. ./...

##@ Code Analysis

## complexity: Analyze code complexity
complexity:
	@echo "📊 Analyzing code complexity..."
	@gocyclo -over 5 .

## duplicate: Find duplicate code
duplicate:
	@echo "🔍 Finding duplicate code..."
	@dupl -threshold 50 .

## lines: Count lines of code
lines:
	@echo "📊 Counting lines of code..."
	@find . -name "*.go" -not -path "./vendor/*" -not -path "./.git/*" | xargs wc -l | tail -n 1
