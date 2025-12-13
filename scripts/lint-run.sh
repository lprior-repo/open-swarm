#!/bin/bash

# Easy Lint Runner - Simple interface for common linting workflows
# Combines linting, auto-fixing, and progress tracking in one convenient tool

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
NC='\033[0m' # No Color
BOLD='\033[1m'

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Default values
MODE="check"
AUTOFIX=false
SHOW_PROGRESS=false
CATEGORY=""
VERBOSE=false
WATCH=false

# Help function
show_help() {
    cat << EOF
${BOLD}${CYAN}Easy Lint Runner${NC}
Simple interface for common linting workflows

${BLUE}Usage:${NC}
  $(basename "$0") [MODE] [OPTIONS]

${BLUE}Modes:${NC}
  check               Run linter without changes (default)
  fix                 Auto-fix issues where possible
  autofix             Smart auto-fix for specific categories (thelper, etc.)
  progress            Show linting progress dashboard
  watch               Watch mode - re-run on file changes

${BLUE}Options:${NC}
  --category=<LINTER> Run only specific linter (e.g., errcheck, revive)
  --verbose           Show verbose output
  --help              Show this help message

${BLUE}Common Workflows:${NC}

  ${CYAN}1. Quick check before commit:${NC}
     $(basename "$0") check

  ${CYAN}2. Fix simple issues automatically:${NC}
     $(basename "$0") fix

  ${CYAN}3. Smart auto-fix for test helpers:${NC}
     $(basename "$0") autofix --category=thelper

  ${CYAN}4. See overall linting progress:${NC}
     $(basename "$0") progress

  ${CYAN}5. Watch mode during development:${NC}
     $(basename "$0") watch

  ${CYAN}6. Check specific linter only:${NC}
     $(basename "$0") check --category=errcheck

${BLUE}What Each Mode Does:${NC}

  ${BOLD}check${NC}
    - Runs golangci-lint on the entire codebase
    - Shows all issues without making changes
    - Exit code 0 if no issues, 1 if issues found

  ${BOLD}fix${NC}
    - Runs golangci-lint with --fix flag
    - Automatically fixes issues that can be safely fixed
    - Shows remaining issues that need manual attention

  ${BOLD}autofix${NC}
    - Uses smart pattern-based fixing for specific categories
    - Supports: thelper, unused, errors
    - More targeted than 'fix' mode
    - Example: $(basename "$0") autofix --category=thelper

  ${BOLD}progress${NC}
    - Shows a dashboard of linting errors by category
    - Useful for tracking cleanup progress
    - Color-coded by severity

  ${BOLD}watch${NC}
    - Watches for file changes and re-runs linter
    - Requires fswatch (brew install fswatch)
    - Great for iterative development

${BLUE}Examples:${NC}

  # Before committing
  $(basename "$0") check

  # Fix what can be fixed automatically
  $(basename "$0") fix

  # Fix test helper issues specifically
  $(basename "$0") autofix --category=thelper

  # Monitor linting progress
  $(basename "$0") progress

  # Watch mode with specific linter
  $(basename "$0") watch --category=revive

  # Verbose check
  $(basename "$0") check --verbose

${BLUE}Integration with Makefile:${NC}
  make lint          # Same as: $(basename "$0") check
  make lint-fix      # Same as: $(basename "$0") fix

${BLUE}Tips:${NC}
  - Run 'check' before committing to catch issues early
  - Use 'fix' to auto-fix simple issues like formatting
  - Use 'autofix' for targeted fixes that need pattern matching
  - Use 'progress' to track cleanup of legacy code
  - Use 'watch' during active development for instant feedback

EOF
}

# Log functions
log_info() {
    echo -e "${BLUE}→ ${1}${NC}"
}

log_success() {
    echo -e "${GREEN}✓ ${1}${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠ ${1}${NC}"
}

log_error() {
    echo -e "${RED}✗ ${1}${NC}"
}

log_header() {
    echo ""
    echo -e "${BOLD}${CYAN}=====================================${NC}"
    echo -e "${BOLD}${CYAN}  ${1}${NC}"
    echo -e "${BOLD}${CYAN}=====================================${NC}"
    echo ""
}

# Check if required tools are installed
check_tools() {
    local missing=false

    case $MODE in
        check|fix|watch)
            if ! command -v golangci-lint &> /dev/null; then
                log_error "golangci-lint not found"
                echo "  Install with: make install-tools"
                missing=true
            fi
            ;;
        autofix)
            if [ ! -x "$SCRIPT_DIR/lint-autofix.sh" ]; then
                log_error "lint-autofix.sh not found or not executable"
                missing=true
            fi
            ;;
        progress)
            if [ ! -x "$SCRIPT_DIR/lint-progress.sh" ]; then
                log_error "lint-progress.sh not found or not executable"
                missing=true
            fi
            ;;
    esac

    if [ "$WATCH" = true ] && ! command -v fswatch &> /dev/null; then
        log_error "fswatch not found (required for watch mode)"
        echo "  Install with:"
        echo "    macOS: brew install fswatch"
        echo "    Linux: apt-get install fswatch"
        missing=true
    fi

    if [ "$missing" = true ]; then
        exit 1
    fi
}

# Run check mode
run_check() {
    log_header "Running Lint Check"

    local cmd="$SCRIPT_DIR/lint.sh"
    local args=""

    if [ -n "$CATEGORY" ]; then
        args="$args --category=$CATEGORY"
        log_info "Category filter: $CATEGORY"
    fi

    if [ "$VERBOSE" = true ]; then
        args="$args --verbose"
    fi

    echo ""

    if eval "$cmd $args"; then
        echo ""
        log_success "Linting completed - no issues found!"
        return 0
    else
        echo ""
        log_warning "Linting found issues (see output above)"
        echo ""
        echo "Next steps:"
        echo "  1. Review the issues above"
        echo "  2. Try: $(basename "$0") fix"
        echo "  3. Or manually fix the issues"
        return 1
    fi
}

# Run fix mode
run_fix() {
    log_header "Running Lint Fix"

    local cmd="$SCRIPT_DIR/lint.sh --fix"
    local args=""

    if [ -n "$CATEGORY" ]; then
        args="$args --category=$CATEGORY"
        log_info "Category filter: $CATEGORY"
    fi

    if [ "$VERBOSE" = true ]; then
        args="$args --verbose"
    fi

    echo ""

    if eval "$cmd $args"; then
        echo ""
        log_success "Auto-fix completed successfully!"
        echo ""
        echo "Next steps:"
        echo "  1. Review changes: git diff"
        echo "  2. Run tests: make test"
        echo "  3. Commit if satisfied"
        return 0
    else
        echo ""
        log_warning "Auto-fix applied, but some issues remain"
        echo ""
        echo "Next steps:"
        echo "  1. Review changes: git diff"
        echo "  2. Check remaining issues: $(basename "$0") check"
        echo "  3. Fix remaining issues manually or with autofix"
        return 1
    fi
}

# Run autofix mode
run_autofix() {
    log_header "Running Smart Auto-Fix"

    local cmd="$SCRIPT_DIR/lint-autofix.sh"
    local args=""

    if [ -n "$CATEGORY" ]; then
        args="$args --category=$CATEGORY"
        log_info "Category filter: $CATEGORY"
    else
        log_info "Fixing all categories (thelper, unused, errors)"
    fi

    if [ "$VERBOSE" = true ]; then
        args="$args --verbose"
    fi

    echo ""

    eval "$cmd $args"
    local exit_code=$?

    echo ""
    if [ $exit_code -eq 0 ]; then
        log_success "Auto-fix completed!"
    fi

    echo ""
    echo "Next steps:"
    echo "  1. Review changes: git diff"
    echo "  2. Run lint check: $(basename "$0") check"
    echo "  3. Run tests: make test"

    return $exit_code
}

# Run progress mode
run_progress() {
    log_header "Linting Progress Dashboard"
    echo ""

    eval "$SCRIPT_DIR/lint-progress.sh"
    local exit_code=$?

    echo ""
    if [ $exit_code -eq 0 ]; then
        log_success "No linting errors!"
    else
        echo "Next steps:"
        echo "  1. Fix issues: $(basename "$0") fix"
        echo "  2. Or use: $(basename "$0") autofix --category=<type>"
    fi

    return $exit_code
}

# Run watch mode
run_watch() {
    log_header "Watch Mode"

    local cmd="$SCRIPT_DIR/lint.sh --watch"
    local args=""

    if [ -n "$CATEGORY" ]; then
        args="$args --category=$CATEGORY"
    fi

    if [ "$VERBOSE" = true ]; then
        args="$args --verbose"
    fi

    log_info "Watching for file changes..."
    log_warning "Press Ctrl+C to stop"
    echo ""

    eval "$cmd $args"
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        check|fix|autofix|progress|watch)
            MODE=$1
            if [ "$MODE" = "watch" ]; then
                WATCH=true
            fi
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
            echo -e "${RED}Error: Unknown option or mode '$1'${NC}"
            echo ""
            show_help
            exit 1
            ;;
    esac
done

# Main execution
main() {
    cd "$PROJECT_ROOT"

    # Check for required tools
    check_tools

    # Run the appropriate mode
    case $MODE in
        check)
            run_check
            ;;
        fix)
            run_fix
            ;;
        autofix)
            run_autofix
            ;;
        progress)
            run_progress
            ;;
        watch)
            run_watch
            ;;
        *)
            log_error "Invalid mode: $MODE"
            show_help
            exit 1
            ;;
    esac
}

main "$@"
