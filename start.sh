#!/bin/bash
set -e

echo "🚀 Open Swarm - Complete Startup"
echo "=================================="
echo ""

# Start Docker services
echo "1️⃣  Starting Docker services..."
docker compose up -d

# Wait for health
echo "2️⃣  Waiting for services to be healthy (20s)..."
sleep 20

# Check status
echo "3️⃣  Service status:"
docker compose ps

echo ""
echo "✅ Open Swarm is running!"
echo ""
echo "🌐 Services:"
echo "  Temporal UI:  http://localhost:8081"
echo "  Temporal RPC: localhost:7233"
echo "  PostgreSQL:   localhost:5433"
echo ""
echo "🔧 Next steps:"
echo "  make run-worker    # Start Temporal worker"
echo "  bd list            # View Beads tasks"
echo "  ./test-workflow.sh # Run end-to-end test"
echo ""
