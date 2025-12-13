#!/bin/bash
# Open Swarm - Quick Demo Runner
# Shows cooperative agent conflict resolution in action

set -e

echo "======================================"
echo "   OPEN SWARM - LOGGING DEMO"
echo "======================================"
echo ""
echo "This demo shows how agents cooperatively"
echo "resolve conflicts without hostile behavior."
echo ""
echo "You'll see:"
echo "  ✓ Agent registration"
echo "  ✓ Coordination sync"
echo "  ✓ Conflict detection"
echo "  ✓ Cooperative resolution (negotiate, wait, force-release)"
echo ""
echo "======================================"
echo ""

# Check if binary exists
if [ ! -f "./bin/logging-demo" ]; then
    echo "Building logging demo..."
    go build -o bin/logging-demo ./cmd/logging-demo
    echo "✓ Built"
    echo ""
fi

# Run the demo
echo "Running demo..."
echo ""
./bin/logging-demo

echo ""
echo "======================================"
echo ""
echo "💡 Key Takeaways:"
echo ""
echo "1. Agents use NEGOTIATION as default resolution"
echo "   → 'Contact holders via Agent Mail to coordinate access'"
echo ""
echo "2. Agents WAIT politely for expiring reservations"
echo "   → 'Wait for reservations to expire (within 5 minutes)'"
echo ""
echo "3. Force-release ONLY for stale (expired) locks"
echo "   → 'Use force_release_file_reservation for stale reservations'"
echo ""
echo "4. NO aggressive behavior:"
echo "   ✗ No forced takeovers"
echo "   ✗ No kill commands"
echo "   ✗ No blame assignment"
echo "   ✗ No competitive retries"
echo ""
echo "======================================"
echo ""
echo "Want JSON logs? Run:"
echo "  LOG_FORMAT=json ./bin/logging-demo"
echo ""
echo "Want to see the code? Check:"
echo "  internal/conflict/analyzer.go"
echo "  pkg/agent/manager.go"
echo "  cmd/logging-demo/main.go"
echo ""
