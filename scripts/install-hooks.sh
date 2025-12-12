#!/usr/bin/env bash
#
# Install Git hooks for open-swarm
#
# Usage:
#   ./scripts/install-hooks.sh
#

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
HOOKS_DIR="$PROJECT_ROOT/.git/hooks"

echo "Installing Git hooks for open-swarm..."
echo ""

# Check if .git directory exists
if [ ! -d "$PROJECT_ROOT/.git" ]; then
    echo "Error: .git directory not found. Are you in a Git repository?"
    exit 1
fi

# Install pre-commit hook
echo "Installing pre-commit hook..."
ln -sf "../../scripts/pre-commit.sh" "$HOOKS_DIR/pre-commit"
echo "✓ Linked $HOOKS_DIR/pre-commit -> scripts/pre-commit.sh"

# Make sure the hook is executable
chmod +x "$SCRIPT_DIR/pre-commit.sh"

echo ""
echo "✓ Git hooks installed successfully!"
echo ""
echo "The following hook is now active:"
echo "  - pre-commit: Runs gofmt, go vet, golangci-lint, and go mod tidy checks"
echo ""
echo "To bypass hooks (not recommended):"
echo "  git commit --no-verify"
echo ""
echo "To uninstall:"
echo "  rm .git/hooks/pre-commit"
echo ""
