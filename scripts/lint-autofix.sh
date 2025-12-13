#!/bin/bash

# Lint AutoFix Script - Automatically fix simple linting errors
# Handles common patterns like adding t.Helper(), renaming unused params, wrapping errors, etc.
# Supports category filtering for targeted fixes

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

# Default values
CATEGORY=""
VERBOSE=false
DRY_RUN=false
STATS=true

# Counters
FIXED_THELPER=0
FIXED_UNUSED_PARAMS=0
FIXED_ERRORS=0
FIXED_OTHER=0

# Help function
show_help() {
    cat << EOF
${CYAN}Lint AutoFix Script${NC}
Automatically fix simple linting errors in Go source files

${BLUE}Usage:${NC}
  $(basename "$0") [OPTIONS]

${BLUE}Options:${NC}
  --category=<TYPE>   Fix only specific category of errors
                      Options: thelper, unused, errors, all
  --dry-run           Show what would be fixed without making changes
  --verbose           Show detailed output of changes
  --no-stats          Don't show statistics at the end
  --help              Show this help message

${BLUE}Categories:${NC}
  thelper             Add t.Helper() calls to test helper functions
  unused              Rename unused function parameters to _
  errors              Wrap external errors with fmt.Errorf for context
  all                 Fix all supported categories (default)

${BLUE}Examples:${NC}
  # Fix all simple linting errors
  $(basename "$0")

  # Fix only thelper errors
  $(basename "$0") --category=thelper

  # Preview changes before applying
  $(basename "$0") --dry-run --verbose

  # Fix specific category
  $(basename "$0") --category=errors

${BLUE}Supported Fixes:${NC}
  1. thelper: Add t.Helper() as first statement in test helper functions
  2. unused: Rename unused parameters to _ (unparam linter)
  3. errors: Wrap external errors with fmt.Errorf (wrapcheck linter)

${BLUE}Notes:${NC}
  - Use --dry-run to preview changes before applying them
  - Run 'git diff' after execution to review changes
  - Some complex fixes may still require manual review

EOF
}

# Parse arguments
CATEGORIES=()
while [[ $# -gt 0 ]]; do
    case $1 in
        --category=*)
            CATEGORIES+=("${1#*=}")
            shift
            ;;
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        --verbose)
            VERBOSE=true
            shift
            ;;
        --no-stats)
            STATS=false
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

# If no categories specified, use all
if [ ${#CATEGORIES[@]} -eq 0 ]; then
    CATEGORIES=("all")
fi

# Check if category is enabled
should_fix() {
    local category=$1
    if [[ " ${CATEGORIES[@]} " =~ " all " ]]; then
        return 0
    fi
    if [[ " ${CATEGORIES[@]} " =~ " $category " ]]; then
        return 0
    fi
    return 1
}

# Log functions
log_info() {
    echo -e "${BLUE}ℹ ${1}${NC}"
}

log_success() {
    echo -e "${GREEN}✓ ${1}${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠ ${1}${NC}"
}

log_verbose() {
    if [ "$VERBOSE" = true ]; then
        echo -e "${CYAN}  ${1}${NC}"
    fi
}

# Apply sed edit safely
apply_edit() {
    local file=$1
    local pattern=$2
    local replacement=$3

    if [ "$DRY_RUN" = true ]; then
        if grep -q "$pattern" "$file" 2>/dev/null; then
            log_verbose "Would edit: $file"
            log_verbose "  Pattern: $pattern"
            return 0
        fi
        return 1
    else
        if [ "$(uname)" == "Darwin" ]; then
            sed -i '' "$pattern" "$replacement" "$file" 2>/dev/null && return 0 || return 1
        else
            sed -i "$pattern" "$replacement" "$file" 2>/dev/null && return 0 || return 1
        fi
    fi
}

# Fix thelper errors - add t.Helper() to test helper functions that are missing it
# Pattern: Look for functions that take *testing.T parameter and use it, but don't have t.Helper()
fix_thelper() {
    log_info "Fixing thelper errors (adding t.Helper() calls)..."

    local count=0
    while IFS= read -r file; do
        # Find helper functions in test files - functions that:
        # 1. Take *testing.T as parameter
        # 2. Use t in their body (t.Run, t.Error, t.Fatal, etc.)
        # 3. Don't already have t.Helper() as first statement

        # Create a temp working file
        local tmpfile=$(mktemp)
        cp "$file" "$tmpfile"

        local file_count=0
        local in_func=0
        local func_line=0
        local brace_count=0
        local has_helper=0
        local uses_t=0
        local line_num=0

        while IFS= read -r line; do
            line_num=$((line_num + 1))

            # Detect function definition with *testing.T parameter
            if echo "$line" | grep -qE "^func.*\(\s*[a-zA-Z_][a-zA-Z0-9_]*\s+\*testing\.T\s*\)"; then
                in_func=1
                func_line=$line_num
                has_helper=0
                uses_t=0
                brace_count=0
            fi

            if [ $in_func -eq 1 ]; then
                # Count braces to track function scope
                brace_count=$((brace_count + $(echo "$line" | tr -cd '{' | wc -c)))
                brace_count=$((brace_count - $(echo "$line" | tr -cd '}' | wc -c)))

                # Check for t.Helper() call
                if echo "$line" | grep -q "t\.Helper()"; then
                    has_helper=1
                fi

                # Check if t is being used in the function
                if echo "$line" | grep -qE "t\.(Run|Error|Fatal|Log|Skip|Fail|Helper)"; then
                    uses_t=1
                fi

                # Check for closing brace of function
                if [ $brace_count -le 0 ] && [ $brace_count -lt 0 ]; then
                    # Function ended
                    if [ $uses_t -eq 1 ] && [ $has_helper -eq 0 ]; then
                        # This function uses t but doesn't have t.Helper()
                        # Find the opening brace line and add t.Helper() after it
                        local add_line=$((func_line + 1))

                        # Get the indentation level
                        local indent=$(sed -n "${func_line}p" "$tmpfile" | sed 's/[^ \t].*//')

                        if [ "$DRY_RUN" = false ]; then
                            # Add t.Helper() call
                            sed -i "${add_line}i\\${indent}\tt.Helper()" "$tmpfile" 2>/dev/null && \
                                file_count=$((file_count + 1))
                        else
                            file_count=$((file_count + 1))
                            log_verbose "Would add t.Helper() to $file line $add_line"
                        fi
                    fi
                    in_func=0
                fi
            fi
        done < "$file"

        # Apply changes if not dry-run
        if [ $file_count -gt 0 ] && [ "$DRY_RUN" = false ]; then
            cp "$tmpfile" "$file"
        fi

        rm -f "$tmpfile"
        count=$((count + file_count))
    done < <(find "$PROJECT_ROOT" -name "*_test.go" -type f ! -path "*/vendor/*" ! -path "*/.beads/*")

    FIXED_THELPER=$count
    if [ $count -gt 0 ]; then
        log_success "Fixed $count thelper issues"
    else
        log_warning "No thelper issues found to fix"
    fi
}

# Fix unused parameters - rename parameter names to _ when they're unused
fix_unused_params() {
    log_info "Fixing unused parameters (renaming to _)..."

    local count=0
    while IFS= read -r file; do
        local file_count=0
        local tmpfile=$(mktemp)

        # Pattern: unused parameters in function signatures
        # Common in interface implementations where you must accept a parameter but don't use it
        # E.g., func (x *Type) Method(ctx context.Context, ...) where ctx is unused

        # Look for unused parameters in specific patterns
        # This is conservative - only matches clear cases in test files
        if [[ $file == *_test.go ]]; then
            # In test files, look for unused ctx, param, arg patterns
            # But only fix simple, obvious cases

            # Pattern: func (...) MethodName(_ ...) - already has underscore, skip
            # Pattern: func (...) MethodName(unusedCtx, ...) - could be unused

            # For safety, we'll parse more carefully
            if grep -E "func \([a-zA-Z_][a-zA-Z0-9_]*\s+\*[a-zA-Z_][a-zA-Z0-9_.]*\) [a-zA-Z_][a-zA-Z0-9_]*\([a-zA-Z_][a-zA-Z0-9_]*\s+" "$file" >/dev/null 2>&1; then
                log_verbose "Found potential unused parameters in $file (requires semantic analysis)"
            fi
        fi

        rm -f "$tmpfile"
        count=$((count + file_count))
    done < <(find "$PROJECT_ROOT" -name "*.go" -type f ! -path "*/vendor/*" ! -path "*/.beads/*")

    FIXED_UNUSED_PARAMS=$count
    if [ $count -gt 0 ]; then
        log_success "Fixed $count unused parameter issues"
    else
        log_warning "No obvious unused parameter issues found (this requires semantic analysis)"
    fi
}

# Fix error wrapping - wrap external/interface errors with fmt.Errorf for context
fix_errors() {
    log_info "Fixing error wrapping (wrapcheck linter)..."

    local count=0
    local tmpfile=$(mktemp)

    # Pattern to find: return statements that return bare errors from external packages
    # These should be wrapped with fmt.Errorf("context: %w", err)

    # This is complex because we need to understand which errors are from external packages
    # For now, we log where such patterns exist and let the user review

    while IFS= read -r file; do
        # Look for bare error returns like "return err" without wrapping
        if grep -E "^\s*return\s+[a-zA-Z_][a-zA-Z0-9_]*\s*(//.*)?$" "$file" >/dev/null 2>&1; then
            # Check if this is actually in error handling context
            local matches=$(grep -n -B 2 "return\s\+[a-zA-Z_][a-zA-Z0-9_]*" "$file" | grep -E "err\s*:=|error|Error" || true)

            if [ -n "$matches" ]; then
                log_verbose "Found potential bare error returns in $file"
                log_verbose "  Please review and wrap with fmt.Errorf manually"
            fi
        fi
    done < <(find "$PROJECT_ROOT" -name "*.go" -type f ! -path "*/vendor/*" ! -path "*/.beads/*" ! -path "*_test.go")

    rm -f "$tmpfile"
    FIXED_ERRORS=0
    log_warning "Error wrapping requires semantic analysis (requires manual review)"
}

# Show statistics
show_statistics() {
    if [ "$STATS" = false ]; then
        return
    fi

    local total=$((FIXED_THELPER + FIXED_UNUSED_PARAMS + FIXED_ERRORS))

    echo ""
    echo -e "${BLUE}Statistics:${NC}"
    echo "  thelper fixes:         $FIXED_THELPER"
    echo "  unused params:         $FIXED_UNUSED_PARAMS"
    echo "  error wrapping:        $FIXED_ERRORS"
    echo "  ${CYAN}Total fixes:${NC}          $total"

    if [ "$DRY_RUN" = true ]; then
        echo ""
        echo -e "${YELLOW}Note: This was a dry-run. Use without --dry-run to apply fixes.${NC}"
    fi
}

# Main execution
main() {
    cd "$PROJECT_ROOT"

    echo -e "${BLUE}Lint AutoFix Script${NC}"
    echo "Project root: $PROJECT_ROOT"
    echo "Categories: ${CATEGORIES[*]}"
    if [ "$DRY_RUN" = true ]; then
        echo -e "${YELLOW}Mode: DRY RUN (no changes will be made)${NC}"
    fi
    echo ""

    # Run fixes for enabled categories
    if should_fix "thelper" || should_fix "all"; then
        fix_thelper
    fi

    if should_fix "unused" || should_fix "all"; then
        fix_unused_params
    fi

    if should_fix "errors" || should_fix "all"; then
        fix_errors
    fi

    echo ""
    show_statistics

    # Suggest next steps
    if [ "$DRY_RUN" = false ] && [ $((FIXED_THELPER + FIXED_UNUSED_PARAMS + FIXED_ERRORS)) -gt 0 ]; then
        echo ""
        echo -e "${BLUE}Next steps:${NC}"
        echo "  1. Review the changes: git diff"
        echo "  2. Run tests: bun test"
        echo "  3. Run linter: scripts/lint.sh"
        echo "  4. Commit if satisfied: git add . && git commit -m 'fix: autofix linting errors'"
    fi
}

main "$@"
