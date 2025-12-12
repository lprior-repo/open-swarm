.PHONY: help build test run-worker run-client docker-up docker-down clean

# Variables
GO := go
DOCKER := docker-compose
BINARY_WORKER := temporal-worker
BINARY_CLIENT := reactor-client
BINARY_MAIN := open-swarm

# Default target displays help
help:
	@echo "Open Swarm - Makefile Commands"
	@echo "=============================="
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available targets:"
	@echo ""
	@echo "  help              - Show this help message"
	@echo "  build             - Build all binaries (worker, client, coordinator)"
	@echo "  test              - Run all tests with coverage"
	@echo "  run-worker        - Start the Temporal worker (requires docker-up)"
	@echo "  run-client        - Start the reactor client (usage: make run-client TASK=<id> PROMPT='<prompt>')"
	@echo "  docker-up         - Start Docker Compose services (Temporal + PostgreSQL)"
	@echo "  docker-down       - Stop Docker Compose services"
	@echo "  docker-logs       - View Docker Compose service logs"
	@echo "  clean             - Remove built binaries and temporary files"
	@echo ""
	@echo "Examples:"
	@echo "  make build"
	@echo "  make test"
	@echo "  make docker-up"
	@echo "  make run-worker"
	@echo "  make run-client TASK=task-1 PROMPT='Implement feature X'"
	@echo "  make docker-down"
	@echo ""

# Build all binaries
build:
	@echo "Building all binaries..."
	@mkdir -p bin
	@$(GO) build -o bin/$(BINARY_MAIN) ./cmd/open-swarm
	@echo "  ✓ Built ./bin/$(BINARY_MAIN)"
	@$(GO) build -o bin/$(BINARY_WORKER) ./cmd/temporal-worker
	@echo "  ✓ Built ./bin/$(BINARY_WORKER)"
	@$(GO) build -o bin/$(BINARY_CLIENT) ./cmd/reactor-client
	@echo "  ✓ Built ./bin/$(BINARY_CLIENT)"
	@echo ""
	@echo "✓ All binaries built successfully"

# Run tests with coverage
test:
	@echo "Running tests..."
	@$(GO) test -v -race -coverprofile=coverage.out ./...
	@echo ""
	@echo "✓ Tests completed"
	@echo ""
	@echo "Coverage report:"
	@$(GO) tool cover -func=coverage.out | tail -1

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

# Start Docker services (Temporal + PostgreSQL)
docker-up:
	@echo "Starting Docker services..."
	@echo "  - PostgreSQL"
	@echo "  - Temporal Server"
	@echo ""
	@$(DOCKER) up -d
	@echo ""
	@echo "Waiting for services to be healthy..."
	@sleep 10
	@echo "✓ Services started successfully"
	@echo ""
	@echo "Web UI: http://localhost:8233"
	@echo "Temporal server: localhost:7233"
	@echo "PostgreSQL: localhost:5432"

# Stop Docker services
docker-down:
	@echo "Stopping Docker services..."
	@$(DOCKER) down
	@echo "✓ Services stopped"

# View Docker logs
docker-logs:
	@$(DOCKER) logs -f

# Clean up built binaries and temporary files
clean:
	@echo "Cleaning up..."
	@rm -rf bin/
	@rm -f coverage.out
	@$(GO) clean
	@echo "✓ Cleanup complete"

# Additional development targets
.PHONY: fmt lint test-race test-coverage

# Format code
fmt:
	@echo "Formatting code..."
	@$(GO) fmt ./...
	@echo "✓ Code formatted"

# Run tests with race detector
test-race:
	@echo "Running tests with race detector..."
	@$(GO) test -v -race ./...

# Generate coverage report in HTML
test-coverage: test
	@$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report generated: coverage.html"
