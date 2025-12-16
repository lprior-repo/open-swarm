#!/usr/bin/env bash
# Open Swarm Infrastructure Management
#
# Comprehensive script for managing all infrastructure:
# - Cleanup zombie/leftover resources
# - Stand up fresh infrastructure with latest Docker images
# - Verify everything is healthy
# - Run benchmarks
#
# Usage:
#   ./scripts/infra.sh setup              # Clean + setup all infra
#   ./scripts/infra.sh cleanup            # Only cleanup resources
#   ./scripts/infra.sh verify             # Only verify health
#   ./scripts/infra.sh benchmark          # Run benchmarks
#   ./scripts/infra.sh full               # Full cycle: cleanup + setup + verify + benchmark
#   ./scripts/infra.sh update-versions    # Update Docker images to latest
#   ./scripts/infra.sh --help             # Show help

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Docker container names
CONTAINERS=(
  "open-swarm-temporal-ui"
  "open-swarm-temporal"
  "open-swarm-postgresql"
  "open-swarm-temporal-admin-tools"
  "open-swarm-otel-collector"
  "open-swarm-jaeger"
  "open-swarm-prometheus"
  "open-swarm-grafana"
  "open-swarm-node-exporter"
)

NETWORKS=("open-swarm-network")
VOLUMES=("postgresql-data" "prometheus-data" "grafana-data")

# Docker image versions (latest as of 2025)
POSTGRES_VERSION="16-alpine"                                    # PostgreSQL 16 (stable)
TEMPORAL_VERSION="1.29.1"                                       # Latest Temporal Server
TEMPORAL_UI_VERSION="2.43.3"                                    # Latest Temporal UI
OTEL_COLLECTOR_VERSION="0.115.1"                                # Latest OTEL Collector
JAEGER_VERSION="1.52"                                           # Latest Jaeger
PROMETHEUS_VERSION="v3.0.1"                                     # Latest Prometheus
GRAFANA_VERSION="11.4.0"                                        # Latest Grafana
NODE_EXPORTER_VERSION="v1.8.2"                                  # Latest Node Exporter

# Options
REMOVE_VOLUMES=false
INCLUDE_OBSERVABILITY=true
RUN_BENCHMARK=false
BENCHMARK_STRATEGY="basic"
BENCHMARK_RUNS=3
BENCHMARK_PROMPT="Implement a simple hello world function"

# Logging helpers
log_info() {
  echo -e "${BLUE}ℹ${NC} $1"
}

log_success() {
  echo -e "${GREEN}✓${NC} $1"
}

log_error() {
  echo -e "${RED}✗${NC} $1"
}

log_warn() {
  echo -e "${YELLOW}⚠${NC} $1"
}

log_section() {
  echo ""
  echo -e "${CYAN}============================================================"
  echo -e "$1"
  echo -e "============================================================${NC}"
  echo ""
}

log_step() {
  echo -e "${MAGENTA}▶${NC} $1"
}

# Utility: Check if Docker is available
check_docker() {
  if ! command -v docker &> /dev/null; then
    log_error "Docker not found! Please install Docker."
    exit 1
  fi

  if ! docker info &> /dev/null; then
    log_error "Docker daemon not running! Please start Docker."
    exit 1
  fi

  # Detect docker compose command
  if docker compose version &> /dev/null; then
    DOCKER_COMPOSE="docker compose"
  elif command -v docker-compose &> /dev/null; then
    DOCKER_COMPOSE="docker-compose"
  else
    log_error "Docker Compose not found! Please install Docker Compose."
    exit 1
  fi
}

# Utility: Wait for URL to be accessible
wait_for_url() {
  local url=$1
  local name=$2
  local max_attempts=${3:-60}
  local attempt=0

  echo -n "  Checking $name... "

  while [ $attempt -lt $max_attempts ]; do
    if curl -f -s "$url" > /dev/null 2>&1; then
      echo -e "${GREEN}✓${NC}"
      return 0
    fi
    attempt=$((attempt + 1))
    sleep 1
  done

  echo -e "${RED}✗${NC}"
  return 1
}

# Utility: Wait for container to be running
wait_for_container() {
  local container=$1
  local max_attempts=${2:-60}
  local attempt=0

  echo -n "  Checking $container... "

  while [ $attempt -lt $max_attempts ]; do
    if docker ps --filter "name=$container" --filter "status=running" | grep -q "$container"; then
      echo -e "${GREEN}✓${NC}"
      return 0
    fi
    attempt=$((attempt + 1))
    sleep 1
  done

  echo -e "${RED}✗${NC}"
  return 1
}

# Cleanup: Stop and remove all containers
cleanup_containers() {
  log_step "Stopping and removing all Open Swarm containers..."

  for container in "${CONTAINERS[@]}"; do
    if docker ps -a --filter "name=$container" --format '{{.Names}}' | grep -q "^${container}$"; then
      log_info "  Removing container: $container"
      docker rm -f "$container" > /dev/null 2>&1 || log_warn "  Could not remove $container"
      log_success "  Removed: $container"
    fi
  done
}

# Cleanup: Remove networks
cleanup_networks() {
  log_step "Removing Open Swarm networks..."

  for network in "${NETWORKS[@]}"; do
    if docker network ls --filter "name=$network" --format '{{.Name}}' | grep -q "^${network}$"; then
      log_info "  Removing network: $network"
      docker network rm "$network" > /dev/null 2>&1 || log_warn "  Could not remove $network"
      log_success "  Removed: $network"
    fi
  done
}

# Cleanup: Remove volumes
cleanup_volumes() {
  if [ "$REMOVE_VOLUMES" = false ]; then
    log_info "Preserving Docker volumes (use --remove-volumes to delete)"
    return
  fi

  log_step "Removing Open Swarm volumes..."

  for volume in "${VOLUMES[@]}"; do
    if docker volume ls --filter "name=$volume" --format '{{.Name}}' | grep -q "$volume"; then
      log_info "  Removing volume: $volume"
      docker volume rm "$volume" > /dev/null 2>&1 || log_warn "  Could not remove $volume"
      log_success "  Removed: $volume"
    fi
  done
}

# Cleanup: Prune Docker resources
prune_docker() {
  log_step "Pruning unused Docker resources..."
  docker system prune -f > /dev/null 2>&1
  log_success "Docker resources pruned"
}

# Cleanup: Remove temporary files
cleanup_temp_files() {
  log_step "Cleaning up temporary files..."

  cd "$PROJECT_ROOT"

  # Remove OTEL data directory
  if [ -d "otel-data" ]; then
    rm -rf otel-data
    log_success "Removed otel-data"
  fi

  # Clean up any stray worktrees
  if [ -d "worktrees" ]; then
    find worktrees -mindepth 1 -delete 2>/dev/null || true
    log_success "Cleaned worktrees directory"
  fi
}

# Full cleanup
cleanup() {
  log_section "🧹 CLEANUP: Removing leftover resources"

  cleanup_containers
  cleanup_networks
  cleanup_volumes
  prune_docker
  cleanup_temp_files

  log_success "Cleanup completed!"
}

# Setup: Start core infrastructure
setup_core() {
  log_step "Starting core infrastructure (Temporal + PostgreSQL + UI)..."

  cd "$PROJECT_ROOT"
  $DOCKER_COMPOSE up -d

  log_success "Core services started"
  log_info "Waiting for services to start..."
  sleep 15
}

# Setup: Start observability stack
setup_observability() {
  log_step "Starting observability stack (OTEL + Jaeger + Prometheus + Grafana)..."

  cd "$PROJECT_ROOT"

  # Create OTEL data directory
  mkdir -p otel-data

  $DOCKER_COMPOSE -f docker-compose.otel.yml up -d

  log_success "Observability services started"
  log_info "Waiting for observability services to start..."
  sleep 20
}

# Setup: Full infrastructure
setup() {
  log_section "🚀 SETUP: Standing up infrastructure"

  check_docker

  setup_core

  if [ "$INCLUDE_OBSERVABILITY" = true ]; then
    setup_observability
  fi

  log_success "Setup completed!"
}

# Verify: All services
verify() {
  log_section "🔍 VERIFY: Checking service health"

  local all_healthy=true

  # Check containers
  log_info "Checking Docker containers..."
  wait_for_container "open-swarm-postgresql" || all_healthy=false
  wait_for_container "open-swarm-temporal" || all_healthy=false
  wait_for_container "open-swarm-temporal-ui" || all_healthy=false

  if [ "$INCLUDE_OBSERVABILITY" = true ]; then
    wait_for_container "open-swarm-otel-collector" || all_healthy=false
    wait_for_container "open-swarm-jaeger" || all_healthy=false
    wait_for_container "open-swarm-prometheus" || all_healthy=false
    wait_for_container "open-swarm-grafana" || all_healthy=false
  fi

  echo ""
  log_info "Checking HTTP endpoints..."

  # Check URLs
  wait_for_url "http://localhost:8081" "Temporal UI" || all_healthy=false

  if [ "$INCLUDE_OBSERVABILITY" = true ]; then
    # Note: OTEL Collector returns 404 on root, but that means it's running
    # We already checked the container is running above
    wait_for_url "http://localhost:16686" "Jaeger UI" || all_healthy=false
    wait_for_url "http://localhost:9090" "Prometheus" || all_healthy=false
    wait_for_url "http://localhost:3001" "Grafana" || all_healthy=false
  fi

  echo ""

  if [ "$all_healthy" = true ]; then
    log_success "All services are healthy! ✨"
    print_service_urls
    return 0
  else
    log_error "Some services are not healthy"
    return 1
  fi
}

# Print service URLs
print_service_urls() {
  echo ""
  echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
  echo ""
  echo -e "${GREEN}🔗 Service URLs:${NC}"
  echo ""
  echo "  📊 Temporal UI:        http://localhost:8081"
  echo "  🔭 Jaeger (Traces):    http://localhost:16686"
  echo "  📈 Grafana:            http://localhost:3001 (admin/admin)"
  echo "  🔢 Prometheus:         http://localhost:9090"
  echo "  🔌 OTEL HTTP:          http://localhost:4318"
  echo "  🔌 OTEL gRPC:          http://localhost:4317"
  echo "  🗄️  PostgreSQL:         localhost:5433"
  echo "  ⚡ Temporal Server:    localhost:7233"
  echo ""
  echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
  echo ""
}

# Build binaries
build() {
  log_step "Building binaries..."

  cd "$PROJECT_ROOT"
  make build

  log_success "Binaries built"
}

# Run benchmark
benchmark() {
  log_section "📊 BENCHMARK: Running $BENCHMARK_STRATEGY strategy"

  log_info "Strategy: $BENCHMARK_STRATEGY"
  log_info "Runs: $BENCHMARK_RUNS"
  log_info "Prompt: $BENCHMARK_PROMPT"
  echo ""

  cd "$PROJECT_ROOT"
  build

  ./bin/benchmark-tcr \
    -strategy "$BENCHMARK_STRATEGY" \
    -runs "$BENCHMARK_RUNS" \
    -prompt "$BENCHMARK_PROMPT"

  log_success "Benchmark completed"
}

# Update Docker image versions in compose files
update_versions() {
  log_section "🔄 UPDATE: Updating Docker image versions"

  cd "$PROJECT_ROOT"

  log_info "Current versions:"
  echo "  PostgreSQL:       $POSTGRES_VERSION"
  echo "  Temporal Server:  $TEMPORAL_VERSION"
  echo "  Temporal UI:      $TEMPORAL_UI_VERSION"
  echo "  OTEL Collector:   $OTEL_COLLECTOR_VERSION"
  echo "  Jaeger:           $JAEGER_VERSION"
  echo "  Prometheus:       $PROMETHEUS_VERSION"
  echo "  Grafana:          $GRAFANA_VERSION"
  echo ""

  log_step "Updating docker-compose.yml..."
  sed -i.bak "s|postgres:[0-9]\+-alpine|postgres:$POSTGRES_VERSION|g" docker-compose.yml
  sed -i.bak "s|temporalio/auto-setup:[0-9.]\+|temporalio/auto-setup:$TEMPORAL_VERSION|g" docker-compose.yml
  sed -i.bak "s|temporalio/ui:[0-9.]\+|temporalio/ui:$TEMPORAL_UI_VERSION|g" docker-compose.yml
  sed -i.bak "s|temporalio/admin-tools:[0-9.]\+|temporalio/admin-tools:$TEMPORAL_VERSION|g" docker-compose.yml

  log_step "Updating docker-compose.otel.yml..."
  sed -i.bak "s|otel/opentelemetry-collector-contrib:[0-9.]\+|otel/opentelemetry-collector-contrib:$OTEL_COLLECTOR_VERSION|g" docker-compose.otel.yml
  sed -i.bak "s|jaegertracing/all-in-one:[0-9.]\+|jaegertracing/all-in-one:$JAEGER_VERSION|g" docker-compose.otel.yml
  sed -i.bak "s|prom/prometheus:v[0-9.]\+|prom/prometheus:$PROMETHEUS_VERSION|g" docker-compose.otel.yml
  sed -i.bak "s|grafana/grafana:[0-9.]\+|grafana/grafana:$GRAFANA_VERSION|g" docker-compose.otel.yml

  log_step "Updating docker-compose.monitoring.yml..."
  sed -i.bak "s|prom/prometheus:v[0-9.]\+|prom/prometheus:$PROMETHEUS_VERSION|g" docker-compose.monitoring.yml
  sed -i.bak "s|grafana/grafana:[0-9.]\+|grafana/grafana:$GRAFANA_VERSION|g" docker-compose.monitoring.yml
  sed -i.bak "s|prom/node-exporter:v[0-9.]\+|prom/node-exporter:$NODE_EXPORTER_VERSION|g" docker-compose.monitoring.yml

  # Remove backup files
  rm -f docker-compose.yml.bak docker-compose.otel.yml.bak docker-compose.monitoring.yml.bak

  log_success "Docker Compose files updated to latest versions"
  log_info "Review changes with: git diff docker-compose*.yml"
}

# Full cycle
full_cycle() {
  log_section "🔄 FULL CYCLE: Complete infrastructure refresh"

  cleanup
  setup
  verify || {
    log_error "Infrastructure is not healthy. Aborting."
    exit 1
  }

  if [ "$RUN_BENCHMARK" = true ]; then
    benchmark
  fi

  log_section "✨ COMPLETE: Infrastructure is ready!"
}

# Show help
show_help() {
  cat << EOF
Open Swarm Infrastructure Management

Usage:
  ./scripts/infra.sh <command> [options]

Commands:
  setup              Clean and setup all infrastructure
  cleanup            Only cleanup resources
  verify             Only verify health
  benchmark          Run benchmarks
  full               Full cycle: cleanup + setup + verify + benchmark
  update-versions    Update Docker images to latest versions
  help               Show this help

Options:
  --remove-volumes           Remove Docker volumes (deletes data)
  --no-observability         Skip observability stack (OTEL, Jaeger, etc.)
  --benchmark                Run benchmark after setup
  --strategy <type>          Benchmark strategy: 'basic' or 'enhanced' (default: basic)
  --runs <number>            Number of benchmark runs (default: 3)
  --prompt <text>            Benchmark prompt (default: "Implement a simple hello world function")

Examples:
  ./scripts/infra.sh setup
  ./scripts/infra.sh cleanup --remove-volumes
  ./scripts/infra.sh full --benchmark --strategy enhanced --runs 5
  ./scripts/infra.sh benchmark --strategy basic --runs 10 --prompt "Add LRU cache"
  ./scripts/infra.sh update-versions

Environment Variables:
  POSTGRES_VERSION           PostgreSQL version (default: $POSTGRES_VERSION)
  TEMPORAL_VERSION           Temporal Server version (default: $TEMPORAL_VERSION)
  TEMPORAL_UI_VERSION        Temporal UI version (default: $TEMPORAL_UI_VERSION)
  OTEL_COLLECTOR_VERSION     OTEL Collector version (default: $OTEL_COLLECTOR_VERSION)
  JAEGER_VERSION             Jaeger version (default: $JAEGER_VERSION)
  PROMETHEUS_VERSION         Prometheus version (default: $PROMETHEUS_VERSION)
  GRAFANA_VERSION            Grafana version (default: $GRAFANA_VERSION)

EOF
}

# Parse arguments
parse_args() {
  while [[ $# -gt 0 ]]; do
    case $1 in
      --remove-volumes)
        REMOVE_VOLUMES=true
        shift
        ;;
      --no-observability)
        INCLUDE_OBSERVABILITY=false
        shift
        ;;
      --benchmark)
        RUN_BENCHMARK=true
        shift
        ;;
      --strategy)
        BENCHMARK_STRATEGY="$2"
        shift 2
        ;;
      --runs)
        BENCHMARK_RUNS="$2"
        shift 2
        ;;
      --prompt)
        BENCHMARK_PROMPT="$2"
        shift 2
        ;;
      --help|-h)
        show_help
        exit 0
        ;;
      *)
        COMMAND="$1"
        shift
        ;;
    esac
  done
}

# Main
main() {
  parse_args "$@"

  local command="${COMMAND:-setup}"

  case "$command" in
    cleanup)
      check_docker
      cleanup
      ;;
    setup)
      check_docker
      cleanup
      setup
      verify
      ;;
    verify)
      check_docker
      verify
      ;;
    benchmark)
      check_docker
      benchmark
      ;;
    full)
      check_docker
      full_cycle
      ;;
    update-versions)
      update_versions
      ;;
    help|--help|-h)
      show_help
      ;;
    *)
      log_error "Unknown command: $command"
      log_info "Run './scripts/infra.sh --help' for usage"
      exit 1
      ;;
  esac
}

main "$@"
