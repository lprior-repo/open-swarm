.PHONY: help build test run-worker run-client docker-up docker-up-monitoring docker-down docker-down-all setup-monitoring clean fmt lint lint-fix test-race test-coverage test-tdd install-tools ci

# Variables
GO := go
DOCKER := docker-compose
BINARY_WORKER := temporal-worker
BINARY_CLIENT := reactor-client
BINARY_MAIN := open-swarm
BINARY_STRESS_TEST := stress-test

# Detect OS for cross-platform support
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Linux)
	OS := linux
endif
ifeq ($(UNAME_S),Darwin)
	OS := darwin
endif

# Tool versions
GOLANGCI_LINT_VERSION := v2.7.2
GOPATH := $(shell go env GOPATH)
GOLANGCI_LINT := $(GOPATH)/bin/golangci-lint

# Default target displays help
help:
	@echo "Open Swarm - Makefile Commands"
	@echo "=============================="
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Build & Development:"
	@echo "  build             - Build all binaries (worker, client, coordinator)"
	@echo "  fmt               - Format code with gofmt"
	@echo "  install-tools     - Install required development tools"
	@echo ""
	@echo "Testing:"
	@echo "  test              - Run all tests"
	@echo "  test-race         - Run tests with race detector"
	@echo "  test-coverage     - Generate HTML coverage report"
	@echo "  test-tdd          - Run tests with TDD Guard reporter"
	@echo ""
	@echo "Linting:"
	@echo "  lint              - Run golangci-lint"
	@echo "  lint-fix          - Run golangci-lint with auto-fix"
	@echo ""
	@echo "CI:"
	@echo "  ci                - Run all CI checks locally (fmt, lint, test)"
	@echo ""
	@echo "Docker:"
	@echo "  docker-up         - Start Docker Compose services (Temporal + PostgreSQL + UI)"
	@echo "  docker-up-monitoring - Start services with Prometheus + Grafana"
	@echo "  docker-down       - Stop Docker Compose services"
	@echo "  docker-down-all   - Stop all services including monitoring"
	@echo "  docker-logs       - View Docker Compose service logs"
	@echo "  setup-monitoring  - Download and install Grafana dashboards"
	@echo ""
	@echo "Runtime:"
	@echo "  run-worker        - Start the Temporal worker (requires docker-up)"
	@echo "  run-client        - Start the reactor client (usage: make run-client TASK=<id> PROMPT='<prompt>')"
	@echo "  run-stress-test   - Run stress test with 100 agents (usage: make run-stress-test STRESS_OPTS='-agents 100')"
	@echo ""
	@echo "Cleanup:"
	@echo "  clean             - Remove built binaries and temporary files"
	@echo ""
	@echo "Examples:"
	@echo "  make install-tools"
	@echo "  make build"
	@echo "  make test"
	@echo "  make lint-fix"
	@echo "  make ci"
	@echo "  make docker-up"
	@echo "  make run-worker"
	@echo ""
	@echo "Detected OS: $(OS)"
	@echo ""

# Build all binaries
build:
	@echo "Building all binaries..."
	@mkdir -p bin
	@$(GO) build -o bin/reactor ./cmd/reactor
	@echo "  ✓ Built ./bin/reactor"
	@$(GO) build -o bin/$(BINARY_WORKER) ./cmd/temporal-worker
	@echo "  ✓ Built ./bin/$(BINARY_WORKER)"
	@$(GO) build -o bin/$(BINARY_CLIENT) ./cmd/reactor-client
	@echo "  ✓ Built ./bin/$(BINARY_CLIENT)"
	@$(GO) build -o bin/single-agent-demo ./cmd/single-agent-demo
	@echo "  ✓ Built ./bin/single-agent-demo"
	@$(GO) build -o bin/workflow-demo ./cmd/workflow-demo
	@echo "  ✓ Built ./bin/workflow-demo"
	@$(GO) build -o bin/$(BINARY_STRESS_TEST) ./cmd/stress-test
	@echo "  ✓ Built ./bin/$(BINARY_STRESS_TEST)"
	@echo ""
	@echo "✓ All binaries built successfully"

# Run all tests
test:
	@echo "Running tests..."
	@$(GO) test -v ./...
	@echo ""
	@echo "✓ Tests completed"

# Start Temporal worker
run-worker: build
	@echo "Starting Temporal worker..."
	@echo "Make sure Docker services are running (run 'make docker-up' first)"
	@echo ""
	./bin/$(BINARY_WORKER)

# Run reactor client with task
run-client: build
	@if [ -z "$(TASK)" ] || [ -z "$(PROMPT)" ]; then \
		echo "Error: TASK and PROMPT are required"; \
		echo "Usage: make run-client TASK=<id> PROMPT='<prompt>'"; \
		exit 1; \
	fi
	@echo "Submitting workflow..."
	@echo "Task ID: $(TASK)"
	@echo "Prompt: $(PROMPT)"
	@echo ""
	./bin/$(BINARY_CLIENT) -task $(TASK) -prompt "$(PROMPT)"

# Run stress test
run-stress-test: build
	@echo "Running stress test..."
	@echo "Make sure Docker services and worker are running:"
	@echo "  1. Terminal 1: make docker-up"
	@echo "  2. Terminal 2: make run-worker"
	@echo "  3. Terminal 3: make run-stress-test"
	@echo ""
	./bin/$(BINARY_STRESS_TEST) $(STRESS_OPTS)

# Start Docker services (Temporal + PostgreSQL + UI)
docker-up:
	@echo "Starting Docker services..."
	@echo "  - PostgreSQL 16"
	@echo "  - Temporal Server v1.29.1"
	@echo "  - Temporal UI v2.43.3"
	@echo ""
	@$(DOCKER) up -d
	@echo ""
	@echo "Waiting for services to be healthy..."
	@sleep 10
	@echo "✓ Services started successfully"
	@echo ""
	@echo "🌐 Service URLs:"
	@echo "  Temporal UI:     http://localhost:8081"
	@echo "  Temporal Server: localhost:7233"
	@echo "  PostgreSQL:      localhost:5433"

# Start Docker services with monitoring
docker-up-monitoring: setup-monitoring
	@echo "Starting Docker services with monitoring..."
	@echo "  - PostgreSQL 16"
	@echo "  - Temporal Server v1.29.1"
	@echo "  - Temporal UI v2.43.3"
	@echo "  - Prometheus v2.54.1"
	@echo "  - Grafana v11.4.0"
	@echo "  - Node Exporter v1.8.2"
	@echo ""
	@$(DOCKER) -f docker-compose.yml -f docker-compose.monitoring.yml up -d
	@echo ""
	@echo "Waiting for services to be healthy..."
	@sleep 15
	@echo "✓ All services started successfully"
	@echo ""
	@echo "🌐 Service URLs:"
	@echo "  Temporal UI:     http://localhost:8081"
	@echo "  Grafana:         http://localhost:3000 (admin/admin)"
	@echo "  Prometheus:      http://localhost:9090"
	@echo "  Temporal Server: localhost:7233"
	@echo "  PostgreSQL:      localhost:5433"
	@echo ""
	@echo "💡 Tip: Visit Grafana → Dashboards → Temporal folder for pre-built dashboards"

# Stop Docker services
docker-down:
	@echo "Stopping Docker services..."
	@$(DOCKER) down
	@echo "✓ Services stopped"

# Stop all services including monitoring
docker-down-all:
	@echo "Stopping all Docker services..."
	@$(DOCKER) -f docker-compose.yml -f docker-compose.monitoring.yml down
	@echo "✓ All services stopped"

# Setup monitoring dashboards
setup-monitoring:
	@echo "Setting up Grafana dashboards..."
	@./scripts/setup-dashboards.sh

# View Docker logs
docker-logs:
	@$(DOCKER) logs -f

# Clean up built binaries and temporary files
clean:
	@echo "Cleaning up..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@$(GO) clean
	@echo "✓ Cleanup complete"

# Format code
fmt:
	@echo "Formatting code..."
	@$(GO) fmt ./...
	@echo "✓ Code formatted"

# Install development tools
install-tools:
	@echo "Installing development tools..."
	@echo "  - golangci-lint $(GOLANGCI_LINT_VERSION)"
	@if [ "$(OS)" = "darwin" ]; then \
		brew install golangci-lint || brew upgrade golangci-lint; \
	elif [ "$(OS)" = "linux" ]; then \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOPATH)/bin $(GOLANGCI_LINT_VERSION); \
	else \
		echo "Unsupported OS. Please install golangci-lint manually."; \
		exit 1; \
	fi
	@echo "  - goimports"
	@$(GO) install golang.org/x/tools/cmd/goimports@latest
	@echo ""
	@echo "✓ Tools installed successfully"
	@echo ""
	@echo "Installed tools:"
	@echo "  golangci-lint: $$($(GOPATH)/bin/golangci-lint --version 2>/dev/null || echo 'not found')"
	@echo "  goimports: $$(test -x $(GOPATH)/bin/goimports && echo '$(GOPATH)/bin/goimports' || echo 'not found')"

# Run linter
lint:
	@echo "Running golangci-lint..."
	@if [ ! -x "$(GOLANGCI_LINT)" ]; then \
		echo "Error: golangci-lint not found at $(GOLANGCI_LINT)"; \
		echo "Run 'make install-tools' first."; \
		exit 1; \
	fi
	@$(GOLANGCI_LINT) run --timeout=5m || (echo "Linting found issues - see output above"; exit 1)
	@echo "✓ Linting completed"

# Run linter with auto-fix
lint-fix:
	@echo "Running golangci-lint with auto-fix..."
	@if [ ! -x "$(GOLANGCI_LINT)" ]; then \
		echo "Error: golangci-lint not found at $(GOLANGCI_LINT)"; \
		echo "Run 'make install-tools' first."; \
		exit 1; \
	fi
	@$(GOLANGCI_LINT) run --fix --timeout=5m || (echo "Linting found issues - see output above"; exit 1)
	@echo "✓ Linting with auto-fix completed"

# Run tests with race detector
test-race:
	@echo "Running tests with race detector..."
	@$(GO) test -v -race ./...
	@echo ""
	@echo "✓ Race detector tests completed"

# Generate coverage report
test-coverage:
	@echo "Running tests with coverage..."
	@$(GO) test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	@echo ""
	@echo "Coverage summary:"
	@$(GO) tool cover -func=coverage.out | tail -1
	@echo ""
	@echo "Generating HTML coverage report..."
	@$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report generated: coverage.html"
	@echo ""
	@echo "Open coverage.html in your browser to view detailed coverage"

# Run tests with TDD Guard reporter
test-tdd:
	@echo "Running tests with TDD Guard reporter..."
	@$(GO) test -json ./... 2>&1 | $(GOPATH)/bin/tdd-guard-go -project-root $(PWD)
	@echo "✓ TDD Guard tests completed"

# Run all CI checks locally
ci: fmt lint test test-race
	@echo ""
	@echo "=============================="
	@echo "✓ All CI checks passed!"
	@echo "=============================="
	@echo ""
	@echo "Summary:"
	@echo "  ✓ Code formatting (gofmt)"
	@echo "  ✓ Linting (golangci-lint)"
	@echo "  ✓ Unit tests"
	@echo "  ✓ Race detector"
	@echo ""
	@echo "Ready to commit and push!"
