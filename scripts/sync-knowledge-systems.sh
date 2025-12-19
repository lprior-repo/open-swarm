#!/bin/bash
# Sync all knowledge systems: Graphiti, Beads, Serena

set -e

PROJECT_ROOT="$(git rev-parse --show-toplevel)"
cd "$PROJECT_ROOT"

echo "🔄 Syncing knowledge systems..."

# Sync Beads
if command -v bd &> /dev/null; then
  echo "  ✓ Syncing Beads..."
  bd sync 2>/dev/null || echo "    (offline)"
fi

# Verify Serena memories
if [ -d .serena/memories ]; then
  echo "  ✓ Serena memories: $(ls -1 .serena/memories/*.md 2>/dev/null | wc -l) files"
fi

# Stage changes
echo "  ✓ Staging changes..."
git add .beads/issues.jsonl 2>/dev/null || true
git add .serena/memories/*.md 2>/dev/null || true

echo "✅ Knowledge systems synchronized"
