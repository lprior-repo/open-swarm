#!/bin/bash
# Display knowledge system summary for current session

PROJECT_ROOT="$(git rev-parse --show-toplevel)"
cd "$PROJECT_ROOT"

echo "📊 Knowledge System Status"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Beads summary
if command -v bd &> /dev/null && [ -f .beads/beads.db ]; then
  STATS=$(bd stats 2>/dev/null || echo "")
  echo ""
  echo "🐝 Beads (Work Tracking)"
  echo "$STATS" | head -6 || echo "  (offline)"
fi

# Serena summary
echo ""
echo "🧠 Serena Memories"
if [ -d .serena/memories ]; then
  ls -1 .serena/memories/*.md 2>/dev/null | sed 's|.*/||; s|\.md$||' | sed 's/^/  • /'
fi

# Graphiti summary
echo ""
echo "🔗 Graphiti (Knowledge Graph)"
echo "  Group ID: open-swarm-codebase"
echo "  Episodes: 6 (architecture, agents, workflows, patterns, dependencies, queries)"
echo "  Status: Use queries to explore architecture"

echo ""
echo "✨ All systems ready"
