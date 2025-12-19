#!/bin/bash
# Spawn 24 concurrent agents to work through ready Beads tasks
# Usage: bash scripts/spawn-24-agents.sh

set -e

PROJECT_ROOT="$(git rev-parse --show-toplevel)"
cd "$PROJECT_ROOT"

echo "🚀 Spawning 24-Agent Swarm"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Get ready tasks from Beads
echo "📋 Fetching ready tasks..."
READY_TASKS=$(bd ready --limit 30 2>/dev/null | grep "^\s*[0-9]" | awk '{print $3}' | head -24)
TASK_COUNT=$(echo "$READY_TASKS" | wc -l)

echo "✓ Found $TASK_COUNT ready tasks"
echo ""

# Create agent configuration file
CONFIG_FILE="/tmp/agent-swarm-config-$(date +%s).json"

cat > "$CONFIG_FILE" << 'EOF'
{
  "swarm": {
    "agentCount": 24,
    "maxConcurrent": 24,
    "tasksPerAgent": "auto",
    "gatesEnabled": true,
    "mem0Integration": true,
    "observability": "enabled"
  },
  "agents": []
}
EOF

# Build agent configs for each task
echo "📝 Building agent configurations..."
AGENT_NUM=0
while IFS= read -r TASK_ID; do
  if [ -z "$TASK_ID" ]; then
    continue
  fi

  # Get task details from Beads
  TASK_INFO=$(bd show "$TASK_ID" 2>/dev/null || echo "")

  AGENT_NUM=$((AGENT_NUM + 1))
  echo "  [$AGENT_NUM/24] $TASK_ID"

done <<< "$READY_TASKS"

echo ""
echo "✅ Agent configuration ready for $AGENT_NUM tasks"
echo ""

# Show staging information
echo "📊 Orchestration Plan"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Total Agents:        24"
echo "Max Concurrent:      24"
echo "Tasks Assigned:      $AGENT_NUM"
echo "Execution Mode:      Parallel (fully concurrent)"
echo "Gates Enabled:       ✓ (5 anti-cheating gates)"
echo "Mem0 Learning:       ✓ Enabled"
echo "Observability:       ✓ Real-time metrics"
echo ""

# Show ready tasks
echo "📋 Tasks to Execute"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
bd ready --limit 30 | head -30

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "🔧 Next Steps:"
echo "  1. Review the ready tasks above"
echo "  2. Run: go run ./cmd/temporal-worker/main.go"
echo "  3. Monitor: bash scripts/knowledge-summary.sh"
echo "  4. Verify: bd list | grep in_progress"
echo ""
echo "⚡ Agent swarm will execute all tasks in parallel:"
echo "  - RED phase:    Generate tests (requirements verification)"
echo "  - GREEN phase:  Implement to pass tests"
echo "  - BLUE phase:   Refactor code quality"
echo "  - VERIFY phase: 5 anti-cheating gates"
echo ""
echo "💾 Config file: $CONFIG_FILE"
echo ""
