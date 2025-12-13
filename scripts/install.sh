#!/bin/bash
set -e

# Open Swarm - Complete Installation Script
# Bundles everything: Docker setup, monitoring, binaries, and tests

VERSION="1.0.0"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
CONFIG_DIR="${CONFIG_DIR:-$HOME/.open-swarm}"

echo "🚀 Open Swarm Installation Script v${VERSION}"
echo "=============================================="
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
info() { echo -e "${BLUE}ℹ${NC}  $1"; }
success() { echo -e "${GREEN}✓${NC}  $1"; }
warn() { echo -e "${YELLOW}⚠${NC}  $1"; }
error() { echo -e "${RED}✗${NC}  $1"; }

# Check prerequisites
check_prerequisites() {
    info "Checking prerequisites..."
    
    local missing=()
    
    if ! command -v docker &> /dev/null; then
        missing+=("docker")
    fi
    
    if ! command -v docker compose &> /dev/null && ! command -v docker-compose &> /dev/null; then
        missing+=("docker-compose")
    fi
    
    if ! command -v go &> /dev/null; then
        missing+=("go")
    fi
    
    if ! command -v git &> /dev/null; then
        missing+=("git")
    fi
    
    if [ ${#missing[@]} -gt 0 ]; then
        error "Missing prerequisites: ${missing[*]}"
        echo ""
        echo "Please install:"
        for tool in "${missing[@]}"; do
            echo "  - $tool"
        done
        exit 1
    fi
    
    success "All prerequisites installed"
}

# Build binaries
build_binaries() {
    info "Building Open Swarm binaries..."
    
    make build
    
    success "Binaries built successfully"
}

# Install binaries (optional)
install_binaries() {
    if [ "$1" != "--install-binaries" ]; then
        return 0
    fi
    
    info "Installing binaries to ${INSTALL_DIR}..."
    
    sudo cp bin/temporal-worker "${INSTALL_DIR}/open-swarm-worker" 2>/dev/null || \
        cp bin/temporal-worker "${INSTALL_DIR}/open-swarm-worker"
    
    sudo cp bin/agent-automation-demo "${INSTALL_DIR}/open-swarm-demo" 2>/dev/null || \
        cp bin/agent-automation-demo "${INSTALL_DIR}/open-swarm-demo"
        
    sudo cp bin/reactor-client "${INSTALL_DIR}/open-swarm-client" 2>/dev/null || \
        cp bin/reactor-client "${INSTALL_DIR}/open-swarm-client"
    
    success "Binaries installed to ${INSTALL_DIR}"
}

# Setup monitoring
setup_monitoring() {
    info "Setting up Grafana dashboards..."
    
    ./scripts/setup-dashboards.sh
    
    success "Monitoring configured"
}

# Start services
start_services() {
    local mode="$1"
    
    if [ "$mode" == "monitoring" ]; then
        info "Starting all services (including monitoring)..."
        make docker-up-monitoring
    else
        info "Starting core services..."
        make docker-up
    fi
    
    success "Services started"
}

# Verify installation
verify_installation() {
    info "Verifying installation..."
    
    # Wait for services
    sleep 10
    
    # Check Docker containers
    if ! docker compose ps | grep -q "healthy"; then
        error "Some services are not healthy"
        docker compose ps
        exit 1
    fi
    
    # Test Temporal server
    if ! curl -s http://localhost:7233 > /dev/null 2>&1; then
        warn "Temporal server not responding yet (may still be starting)"
    else
        success "Temporal server running"
    fi
    
    # Test UI
    if curl -s http://localhost:8081 > /dev/null 2>&1; then
        success "Temporal UI running"
    else
        warn "Temporal UI not accessible yet"
    fi
    
    success "Installation verified"
}

# Create config directory
setup_config() {
    info "Setting up configuration..."
    
    mkdir -p "$CONFIG_DIR"
    
    cat > "$CONFIG_DIR/env" << 'ENVEOF'
# Open Swarm Configuration
TEMPORAL_ADDRESS=localhost:7233
TEMPORAL_UI=http://localhost:8081
TEMPORAL_NAMESPACE=default

# Monitoring (if enabled)
GRAFANA_URL=http://localhost:3000
PROMETHEUS_URL=http://localhost:9090

# PostgreSQL
POSTGRES_HOST=localhost
POSTGRES_PORT=5433
POSTGRES_USER=temporal
POSTGRES_DB=temporal
ENVEOF

    success "Configuration saved to $CONFIG_DIR/env"
}

# Run end-to-end test
run_e2e_test() {
    if [ "$1" != "--test" ]; then
        return 0
    fi
    
    info "Running end-to-end test..."
    
    # Build and run test
    ./test-workflow.sh
    
    success "End-to-end test passed"
}

# Print summary
print_summary() {
    echo ""
    echo "=============================================="
    echo -e "${GREEN}✓ Installation Complete!${NC}"
    echo "=============================================="
    echo ""
    echo "🌐 Service URLs:"
    echo "   Temporal UI:  http://localhost:8081"
    echo "   Temporal RPC: localhost:7233"
    
    if [ "$1" == "monitoring" ]; then
        echo "   Grafana:      http://localhost:3000 (admin/admin)"
        echo "   Prometheus:   http://localhost:9090"
    fi
    
    echo ""
    echo "📦 Installed Binaries:"
    echo "   $(which open-swarm-worker 2>/dev/null || echo 'bin/temporal-worker')"
    echo "   $(which open-swarm-demo 2>/dev/null || echo 'bin/agent-automation-demo')"
    echo "   $(which open-swarm-client 2>/dev/null || echo 'bin/reactor-client')"
    echo ""
    echo "🚀 Quick Start:"
    echo "   # Start worker"
    echo "   make run-worker"
    echo ""
    echo "   # Or use installed binary"
    echo "   open-swarm-worker"
    echo ""
    echo "   # Run demo"
    echo "   open-swarm-demo"
    echo ""
    echo "🔧 Management:"
    echo "   make docker-down        # Stop services"
    echo "   make docker-up          # Start services"
    echo "   make docker-logs        # View logs"
    echo ""
    echo "📚 Documentation:"
    echo "   README.md               # Project overview"
    echo "   UPGRADE.md              # Upgrade guide"
    echo "   AGENTS.md               # Agent workflows"
    echo ""
}

# Main installation flow
main() {
    local mode="core"
    local install_bins=false
    local run_test=false
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --monitoring)
                mode="monitoring"
                shift
                ;;
            --install-binaries)
                install_bins=true
                shift
                ;;
            --test)
                run_test=true
                shift
                ;;
            --help)
                echo "Usage: $0 [OPTIONS]"
                echo ""
                echo "Options:"
                echo "  --monitoring         Install with Grafana + Prometheus"
                echo "  --install-binaries   Install binaries to /usr/local/bin"
                echo "  --test               Run end-to-end tests after installation"
                echo "  --help               Show this help message"
                echo ""
                echo "Examples:"
                echo "  $0                                    # Core installation"
                echo "  $0 --monitoring                       # With monitoring"
                echo "  $0 --monitoring --install-binaries    # Full installation"
                echo "  $0 --test                             # Install and test"
                exit 0
                ;;
            *)
                error "Unknown option: $1"
                echo "Run with --help for usage"
                exit 1
                ;;
        esac
    done
    
    # Run installation steps
    check_prerequisites
    build_binaries
    
    if [ "$install_bins" = true ]; then
        install_binaries --install-binaries
    fi
    
    if [ "$mode" == "monitoring" ]; then
        setup_monitoring
    fi
    
    setup_config
    start_services "$mode"
    verify_installation
    
    if [ "$run_test" = true ]; then
        run_e2e_test --test
    fi
    
    print_summary "$mode"
}

# Run main
main "$@"
