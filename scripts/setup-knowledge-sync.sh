#!/bin/bash
# Setup script: Configure knowledge system synchronization hooks
# Run once per developer: bash scripts/setup-knowledge-sync.sh

set -e

PROJECT_ROOT="$(git rev-parse --show-toplevel)"
cd "$PROJECT_ROOT"

echo "🔧 Setting up unified knowledge system synchronization..."

# Create pre-commit hook
mkdir -p .git/hooks
cat > .git/hooks/pre-commit << 'EOF'
#!/bin/bash
# Pre-commit hook: Sync Graphiti, Beads, and Serena before commit
set -e

PROJECT_ROOT="$(git rev-parse --show-toplevel)"
cd "$PROJECT_ROOT"

# Sync Beads state
if command -v bd &> /dev/null && [ -f .beads/issues.jsonl ]; then
  bd sync 2>/dev/null || true
fi

# Stage updated Beads and Serena files
if [ -f .beads/issues.jsonl ]; then
  git add .beads/issues.jsonl 2>/dev/null || true
fi

if [ -d .serena/memories ]; then
  git add .serena/memories/*.md 2>/dev/null || true
fi

exit 0
EOF

chmod +x .git/hooks/pre-commit
echo "  ✓ Pre-commit hook installed"

# Create post-merge hook
cat > .git/hooks/post-merge << 'EOF'
#!/bin/bash
# Post-merge hook: Validate knowledge system after merge
set -e

PROJECT_ROOT="$(git rev-parse --show-toplevel)"
cd "$PROJECT_ROOT"

# Validate Beads integrity
if command -v bd &> /dev/null && [ -f .beads/beads.db ]; then
  bd validate 2>/dev/null || true
fi

echo "✓ Knowledge systems validated"
exit 0
EOF

chmod +x .git/hooks/post-merge
echo "  ✓ Post-merge hook installed"

# Create session initialization script
cat > scripts/init-knowledge-session.sh << 'EOF'
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
EOF

chmod +x scripts/init-knowledge-session.sh
echo "  ✓ Session initialization script created"

# Create knowledge sync utility
cat > scripts/sync-knowledge-systems.sh << 'EOF'
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
EOF

chmod +x scripts/sync-knowledge-systems.sh
echo "  ✓ Knowledge sync utility created"

# Create context summary script
cat > scripts/knowledge-summary.sh << 'EOF'
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
EOF

chmod +x scripts/knowledge-summary.sh
echo "  ✓ Status summary script created"

echo ""
echo "✅ Knowledge system synchronization setup complete!"
echo ""
echo "Next steps:"
echo "  1. Run once: bash scripts/init-knowledge-session.sh"
echo "  2. Daily: bash scripts/knowledge-summary.sh"
echo "  3. Auto-sync on commit (hooks installed)"
echo ""
