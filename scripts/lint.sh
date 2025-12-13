#!/bin/bash

# Lint Runner Script - Wrapper for golangci-lint with enhanced features
# Provides better output formatting and convenient options for common tasks

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
GOPATH="${GOPATH:-$(go env GOPATH)}"
GOLANGCI_LINT="${GOPATH}/bin/golangci-lint"

# Default values
FIX=false
WATCH=false
CATEGORY=""
VERBOSE=false

# Help function
show_help() {
    cat << EOF
${CYAN}Lint Runner Script${NC}
A convenience wrapper for golangci-lint with enhanced features

${BLUE}Usage:${NC}
  $(basename "$0") [OPTIONS]

${BLUE}Options:${NC}
  --fix                 Auto-fix issues where possible
  --watch               Watch mode - re-run linter on file changes (requires fswatch)
  --category=<LINTER>   Run only specific linter (e.g., errcheck, govet, staticcheck)
  --verbose             Show verbose output
  --help                Show this help message

${BLUE}Examples:${NC}
  # Run all linters
  $(basename "$0")

  # Auto-fix all issues
  $(basename "$0") --fix

  # Check only errcheck linter
  $(basename "$0") --category=errcheck

  # Watch mode - re-run on file changes
  $(basename "$0") --watch

  # Combine options
  $(basename "$0") --fix --category=govet

${BLUE}Available Linters:${NC}
  errcheck, govet, ineffassign, staticcheck, unused,
  bodyclose, contextcheck, cyclop, dupl, durationcheck,
  errname, errorlint, exhaustive, copyloopvar,
  gocheckcompilerdirectives, gocognit, goconst, gocritic,
  gocyclo, mnd, goprintffuncname, gosec, misspell,
  namedret, nilerr, nilnil, noctx, nolintlint, prealloc,
  predeclared, revive, thelper, tparallel, unconvert,
  unparam, whitespace, wrapcheck

EOF
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --fix)
            FIX=true
            shift
            ;;
        --watch)
            WATCH=true
            shift
            ;;
        --category=*)
            CATEGORY="${1#*=}"
            shift
            ;;
        --verbose)
            VERBOSE=true
            shift
            ;;
        --help|-h)
            show_help
            exit 0
            ;;
        *)
            echo -e "${RED}Error: Unknown option '$1'${NC}"
            show_help
            exit 1
            ;;
    esac
done

# Check if golangci-lint is installed
if [ ! -x "$GOLANGCI_LINT" ]; then
    echo -e "${RED}Error: golangci-lint not found at $GOLANGCI_LINT${NC}"
    echo -e "${YELLOW}Run 'make install-tools' first.${NC}"
    exit 1
fi

# Build golangci-lint command
build_lint_command() {
    local cmd="$GOLANGCI_LINT run"

    if [ "$FIX" = true ]; then
        cmd="$cmd --fix"
    fi

    if [ -n "$CATEGORY" ]; then
        cmd="$cmd --enable-only=$CATEGORY"
    fi

    if [ "$VERBOSE" = true ]; then
        cmd="$cmd --verbose"
    fi

    # Add timeout
    cmd="$cmd --timeout=5m"

    echo "$cmd"
}

# Run linter once
run_lint() {
    local cmd=$(build_lint_command)

    echo -e "${BLUE}Running lint check...${NC}"
    if [ -n "$CATEGORY" ]; then
        echo -e "${CYAN}Category: $CATEGORY${NC}"
    fi
    if [ "$FIX" = true ]; then
        echo -e "${CYAN}Mode: Auto-fix enabled${NC}"
    fi
    echo ""

    cd "$PROJECT_ROOT"

    # Execute the lint command
    if eval "$cmd"; then
        echo ""
        echo -e "${GREEN}✓ Linting completed successfully${NC}"
        return 0
    else
        echo ""
        echo -e "${YELLOW}⚠ Linting found issues${NC}"
        return 1
    fi
}

# Watch mode implementation
watch_mode() {
    echo -e "${BLUE}Watch mode enabled${NC}"
    echo -e "${CYAN}Watching for Go file changes...${NC}"
    echo -e "${YELLOW}Press Ctrl+C to stop${NC}"
    echo ""

    # Check if fswatch is available
    if ! command -v fswatch &> /dev/null; then
        echo -e "${RED}Error: fswatch not found${NC}"
        echo "Install it with:"
        echo "  - macOS: brew install fswatch"
        echo "  - Linux: apt-get install fswatch (or equivalent)"
        exit 1
    fi

    cd "$PROJECT_ROOT"

    # Initial run
    run_lint || true

    # Watch for changes
    fswatch -r --exclude='.beads|.git|vendor|bin|coverage' -e '\.go$' . | while read -r file; do
        clear
        echo -e "${BLUE}[$(date '+%H:%M:%S')] File changed: $file${NC}"
        echo ""
        run_lint || true
        echo ""
    done
}

# Main execution
if [ "$WATCH" = true ]; then
    watch_mode
else
    run_lint
fi
