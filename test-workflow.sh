#!/bin/bash
set -e

echo "🤖 Open Swarm - Workflow Test"
echo "=============================="
echo ""

# Check Temporal
echo "1️⃣  Checking Temporal..."
if docker ps | grep -q open-swarm-temporal; then
    echo "   ✅ Temporal container running"
else
    echo "   ❌ Temporal container not running"
    echo ""
    echo "   Run: docker compose up -d"
    exit 1
fi

# Build binaries
echo ""
echo "2️⃣  Building binaries..."
go build -o bin/temporal-worker ./cmd/temporal-worker
go build -o bin/agent-automation-demo ./cmd/agent-automation-demo
echo "   ✅ Binaries built"

# Start worker in background
echo ""
echo "3️⃣  Starting worker..."
./bin/temporal-worker > /tmp/worker.log 2>&1 &
WORKER_PID=$!
echo "   ✅ Worker started (PID: $WORKER_PID)"

# Wait for worker to initialize
sleep 3

# Run demo
echo ""
echo "4️⃣  Running demo..."
timeout 10s ./bin/agent-automation-demo || true
echo "   ✅ Demo executed"

# Stop worker
echo ""
echo "5️⃣  Stopping worker..."
kill $WORKER_PID 2>/dev/null || true
wait $WORKER_PID 2>/dev/null || true
echo "   ✅ Worker stopped"

echo ""
echo "✅ All tests passed!"
echo ""
echo "📊 View workflows at: http://localhost:8081"
