#!/usr/bin/env bash
# Start OpenTelemetry Observability Stack
# This script starts the OTEL collector, Jaeger, Prometheus, and Grafana

set -e

echo "🔭 Starting OpenTelemetry Observability Stack..."
echo ""

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$PROJECT_ROOT"

# Check if docker-compose is available
if ! command -v docker-compose &> /dev/null && ! command -v docker &> /dev/null; then
    echo -e "${RED}❌ Docker or docker-compose not found!${NC}"
    echo "Please install Docker Desktop or Docker Engine with Compose plugin"
    exit 1
fi

# Use docker compose or docker-compose
DOCKER_COMPOSE="docker compose"
if ! docker compose version &> /dev/null; then
    DOCKER_COMPOSE="docker-compose"
fi

# Create data directory for OTEL collector
mkdir -p otel-data

echo "📦 Starting containers..."
$DOCKER_COMPOSE -f docker-compose.otel.yml up -d

echo ""
echo "⏳ Waiting for services to be healthy..."
echo ""

# Function to wait for service health
wait_for_service() {
    local service=$1
    local url=$2
    local max_attempts=30
    local attempt=0

    echo -n "   Checking $service... "

    while [ $attempt -lt $max_attempts ]; do
        if curl -f -s "$url" > /dev/null 2>&1; then
            echo -e "${GREEN}✓${NC}"
            return 0
        fi
        attempt=$((attempt + 1))
        sleep 1
    done

    echo -e "${YELLOW}⚠${NC}"
    return 1
}

# Function to wait for container
wait_for_container() {
    local container=$1
    local max_attempts=30
    local attempt=0

    echo -n "   Checking $container... "

    while [ $attempt -lt $max_attempts ]; do
        if docker ps --filter "name=$container" --filter "status=running" | grep -q "$container"; then
            echo -e "${GREEN}✓${NC}"
            return 0
        fi
        attempt=$((attempt + 1))
        sleep 1
    done

    echo -e "${YELLOW}⚠${NC}"
    return 1
}

# Wait for containers to start
sleep 3

# Check services
wait_for_container "open-swarm-otel-collector"
wait_for_container "open-swarm-jaeger"
wait_for_container "open-swarm-prometheus"
wait_for_container "open-swarm-grafana"

echo ""
echo "🔍 Waiting for HTTP endpoints..."
echo ""

wait_for_service "OTEL Collector" "http://localhost:4318"
wait_for_service "Jaeger UI" "http://localhost:16686"
wait_for_service "Prometheus" "http://localhost:9090"
wait_for_service "Grafana" "http://localhost:3001"

echo ""
echo -e "${GREEN}✅ Observability stack started successfully!${NC}"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "🔗 Access URLs:"
echo ""
echo "   📊 Jaeger UI (Traces):     http://localhost:16686"
echo "   📈 Grafana (Dashboards):   http://localhost:3001"
echo "                              User: admin / Pass: admin"
echo "   🔢 Prometheus (Metrics):   http://localhost:9090"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "📝 OTEL Collector endpoints:"
echo "   - OTLP HTTP:  http://localhost:4318"
echo "   - OTLP gRPC:  http://localhost:4317"
echo "   - Metrics:    http://localhost:8888/metrics"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "💡 Quick Commands:"
echo ""
echo "   # View logs"
echo "   docker logs -f open-swarm-otel-collector"
echo "   docker logs -f open-swarm-jaeger"
echo ""
echo "   # View trace file"
echo "   cat otel-data/otel-traces.json | jq '.'"
echo ""
echo "   # Stop observability stack"
echo "   docker-compose -f docker-compose.otel.yml down"
echo ""
echo "   # View all containers"
echo "   docker-compose -f docker-compose.otel.yml ps"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "🚀 Now run your worker with tracing enabled:"
echo ""
echo "   ./bin/temporal-worker"
echo ""
echo "   Or with custom collector URL:"
echo "   OTEL_COLLECTOR_URL=http://localhost:4318 ./bin/temporal-worker"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "📚 Documentation: See TELEMETRY.md for detailed usage guide"
echo ""
