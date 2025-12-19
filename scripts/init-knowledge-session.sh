#!/bin/bash
# Session initialization: Load context and verify knowledge systems

PROJECT_ROOT="$(git rev-parse --show-toplevel)"
cd "$PROJECT_ROOT"

echo "🚀 Initializing knowledge systems session..."

# Verify Beads is active
if command -v bd &> /dev/null; then
  echo "  ✓ Beads ready"
  bd stats 2>/dev/null | head -1 || true
fi

# Verify Serena memory access
if [ -d .serena/memories ]; then
  COUNT=$(ls -1 .serena/memories/*.md 2>/dev/null | wc -l)
  echo "  ✓ Serena memories ($COUNT files)"
fi

# Verify Graphiti status (if available)
if command -v curl &> /dev/null; then
  STATUS=$(curl -s http://localhost:6000/status 2>/dev/null | grep -q ok && echo "connected" || echo "offline")
  echo "  ✓ Graphiti: $STATUS"
fi

echo ""
echo "📚 Knowledge system ready. Use these queries:"
echo "   - bd ready        : See available tasks"
echo "   - bd list         : View all issues"
echo "   - bd show <id>    : View task details"
echo "   - Serena: read_memory('graphiti-codebase-indexing-setup')"
echo ""
