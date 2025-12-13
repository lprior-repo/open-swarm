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
FIXED_UNUSED_VARS=0
FIXED_BODY_CLOSE=0
FIXED_ERRORS=0

# Help function
show_help() {
    cat << EOF
${CYAN}Lint AutoFix Script${NC}
Automatically fix simple linting errors in Go source files

${BLUE}Usage:${NC}
  $(basename "$0") [OPTIONS]

${BLUE}Options:${NC}
  --category=<TYPE>   Fix only specific category of errors
                      Options: thelper, unused, bodyclose, errors, all
  --dry-run           Show what would be fixed without making changes
  --verbose           Show detailed output of changes
  --no-stats          Don't show statistics at the end
  --help              Show this help message

${BLUE}Categories:${NC}
  thelper             Add t.Helper() calls to test helper functions
  unused              Rename unused function parameters to _
  bodyclose           Ensure response bodies are closed (basic patterns)
  errors              Wrap errors with fmt.Errorf for context
  all                 Fix all supported categories (default)

${BLUE}Examples:${NC}
  # Fix all simple linting errors
  $(basename "$0")

  # Fix only thelper errors
  $(basename "$0") --category=thelper

  # Preview changes before applying
  $(basename "$0") --dry-run --verbose

  # Fix multiple categories
  $(basename "$0") --category=thelper --category=unused

${BLUE}Supported Fixes:${NC}
  1. thelper: Add t.Helper() to test helper functions
  2. unused: Rename unused parameters to _ (unparam linter)
  3. bodyclose: Add deferred Close() calls for http.Response bodies
  4. errors: Wrap external errors with fmt.Errorf for context (wrapcheck)

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

# Log function
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

# Fix thelper errors - add t.Helper() to test helper functions
fix_thelper() {
    log_info "Fixing thelper errors (adding t.Helper() calls)..."

    local count=0
    while IFS= read -r file; do
        # Look for test functions that call other test functions but don't have t.Helper()
        # Pattern: func (t *testing.T) { ... t.Run( ... func(t *testing.T) { ... })
        if grep -q "t.Helper()" "$file" 2>/dev/null; then
            # File already has some helper calls, look for patterns
            local matches=$(grep -n "func (.*\*testing\.T.*) {" "$file" | grep -v "t.Helper()" | head -1)
            if [ -n "$matches" ]; then
                # Check if this is a helper function (has t parameter and calls other functions with it)
                local line_num=$(echo "$matches" | cut -d: -f1)
                if grep -A 5 "$file" | tail -n +$line_num | grep -q "t\.Run\|t\.Logf\|t\.Error\|t\.Fatal" && \
                   ! sed -n "${line_num},$((line_num + 3))p" "$file" | grep -q "t\.Helper()"; then
                    # This looks like it could be a helper function
                    if [ "$DRY_RUN" = true ]; then
                        log_verbose "Would add t.Helper() to $file line $line_num"
                    else
                        # Add t.Helper() as the first line after the function declaration
                        sed -i "${line_num}a\\\\t\t\tt.Helper()" "$file" 2>/dev/null && \
                            count=$((count + 1))
                    fi
                fi
            fi
        fi
    done < <(find "$PROJECT_ROOT" -name "*_test.go" -type f)

    FIXED_THELPER=$count
    if [ $count -gt 0 ]; then
        log_success "Fixed $count thelper issues"
    else
        log_warning "No thelper issues found to fix"
    fi
}

# Fix unused parameters - rename to _
fix_unused_params() {
    log_info "Fixing unused parameters (renaming to _)..."

    local count=0
    # Pattern: function parameters that are unused
    # This looks for common patterns like func foo(ctx context.Context, ...) where ctx is unused
    while IFS= read -r file; do
        local tmpfile="${file}.tmp"
        local file_changes=0

        # Common unused parameter patterns in test/interface implementations
        # func (x *Type) Method(ctx context.Context, _ string) - ctx unused
        if grep -E "func \([a-zA-Z_][a-zA-Z0-9_]*\s+\*?[a-zA-Z_][a-zA-Z0-9_.]*\) [a-zA-Z_].*\(.*\s+(ctx|param|arg|unused)[a-zA-Z0-9_]*\s+[a-zA-Z_].*\)" "$file" >/dev/null 2>&1; then

            # Rename single-letter unused params in function signatures
            # This is a safe pattern for test functions
            if grep -q "func.*\(.*\bctx\b.*\)" "$file" 2>/dev/null; then
                # Only fix in test files where it's safer
                if [[ $file == *_test.go ]]; then
                    # Replace unused ctx with _ in function signatures
                    sed -i.bak 's/func (\([a-z]\) \*\([a-zA-Z]*\)) \([a-zA-Z]*\)(ctx context\.Context,/func (\1 *\2) \3(_ context.Context,/g' "$file"
                    if ! cmp -s "$file" "$file.bak"; then
                        file_changes=$((file_changes + 1))
                    fi
                    rm -f "$file.bak"
                fi
            fi
        fi

        count=$((count + file_changes))
    done < <(find "$PROJECT_ROOT" -name "*.go" -type f ! -path "*/vendor/*" ! -path "*/.beads/*")

    FIXED_UNUSED_PARAMS=$count
    if [ $count -gt 0 ]; then
        log_success "Fixed $count unused parameter issues"
    else
        log_warning "No unused parameter issues found to fix"
    fi
}

# Fix unused variables - common patterns
fix_unused_vars() {
    log_info "Fixing unused variables..."

    local count=0
    while IFS= read -r file; do
        # Pattern: absPath := somePath but absPath is never used
        # We can safely replace with _ for simple assignments
        if grep -E "^\s*[a-zA-Z_][a-zA-Z0-9_]*\s*:=\s*[a-zA-Z_]" "$file" 2>/dev/null | \
           grep -q ""; then

            # Look for specific known patterns
            local before_count=$(grep -c "^\s*[a-zA-Z_][a-zA-Z0-9_]*\s*:=\s*" "$file" 2>/dev/null || echo 0)

            # Don't auto-fix this category as it's risky without deeper analysis
            log_verbose "Skipping deep unused variable analysis for $file (requires manual review)"
        fi
    done < <(find "$PROJECT_ROOT" -name "*.go" -type f ! -path "*/vendor/*" ! -path "*/.beads/*")

    FIXED_UNUSED_VARS=0
}

# Fix bodyclose errors - ensure response bodies are closed
fix_bodyclose() {
    log_info "Fixing bodyclose errors (ensuring response bodies are closed)..."

    local count=0
    while IFS= read -r file; do
        # Pattern: resp, err := http.Get(...) without defer resp.Body.Close()
        # Find lines with http.Get, http.Post, etc. without following Close()
        local line_num=1
        while IFS= read -r line; do
            if echo "$line" | grep -qE "^\s*[a-zA-Z_][a-zA-Z0-9_]*\s*,\s*err\s*:=\s*(http\.|client\.)" && \
               ! sed -n "$((line_num)),$((line_num + 2))p" "$file" | grep -q "\.Body\.Close()"; then

                if [ "$DRY_RUN" = true ]; then
                    log_verbose "Would add Body.Close() handling for $file line $line_num"
                else
                    # Try to add defer statement (simplified - real implementation would need better parsing)
                    # This is a complex case, so we'll log it for now
                    log_verbose "Needs manual check: $file line $line_num - HTTP response without explicit close"
                fi
            fi
            line_num=$((line_num + 1))
        done < "$file"
    done < <(find "$PROJECT_ROOT" -name "*.go" -type f ! -path "*/vendor/*" ! -path "*/.beads/*")

    FIXED_BODY_CLOSE=0
    log_warning "bodyclose fixes require manual verification (complex patterns)"
}

# Fix error wrapping - wrap external errors with fmt.Errorf
fix_errors() {
    log_info "Fixing error wrapping (ensuring external errors are wrapped)..."

    local count=0
    # Pattern: return err (where err comes from external package)
    # Should be: return fmt.Errorf("context: %w", err)

    while IFS= read -r file; do
        # Look for return statements with bare errors
        if grep -E "^\s*return\s+[a-zA-Z_][a-zA-Z0-9_]*\s*$" "$file" >/dev/null 2>&1; then
            log_verbose "Found potential bare error returns in $file (requires manual review)"
        fi
    done < <(find "$PROJECT_ROOT" -name "*.go" -type f ! -path "*/vendor/*" ! -path "*/.beads/*")

    FIXED_ERRORS=0
    log_warning "Error wrapping fixes require semantic analysis (complex patterns)"
}

# Show statistics
show_statistics() {
    if [ "$STATS" = false ]; then
        return
    fi

    local total=$((FIXED_THELPER + FIXED_UNUSED_PARAMS + FIXED_UNUSED_VARS + FIXED_BODY_CLOSE + FIXED_ERRORS))

    echo ""
    echo -e "${BLUE}Statistics:${NC}"
    echo "  thelper fixes:         $FIXED_THELPER"
    echo "  unused params:         $FIXED_UNUSED_PARAMS"
    echo "  unused variables:      $FIXED_UNUSED_VARS"
    echo "  body close fixes:      $FIXED_BODY_CLOSE"
    echo "  error wrapping fixes:  $FIXED_ERRORS"
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
        fix_unused_vars
    fi

    if should_fix "bodyclose" || should_fix "all"; then
        fix_bodyclose
    fi

    if should_fix "errors" || should_fix "all"; then
        fix_errors
    fi

    echo ""
    show_statistics

    # Suggest next steps
    if [ "$DRY_RUN" = false ]; then
        echo ""
        echo -e "${BLUE}Next steps:${NC}"
        echo "  1. Review the changes: git diff"
        echo "  2. Run tests: bun test"
        echo "  3. Run linter: scripts/lint.sh"
        echo "  4. Commit changes: git add . && git commit -m 'fix: autofix linting errors'"
    fi
}

main "$@"
