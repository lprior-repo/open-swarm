#!/bin/bash

# lint-progress.sh - Display a live dashboard of linting errors by category
# Usage: ./scripts/lint-progress.sh

set -o pipefail

# Color codes
RED='\033[0;31m'
YELLOW='\033[0;33m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
NC='\033[0m' # No Color
BOLD='\033[1m'

# Temporary file for storing results
TEMP_FILE=$(mktemp)
trap "rm -f $TEMP_FILE" EXIT

# Function to print colored header
print_header() {
    echo -e "${BOLD}${CYAN}=====================================${NC}"
    echo -e "${BOLD}${CYAN}   Linting Progress Dashboard${NC}"
    echo -e "${BOLD}${CYAN}=====================================${NC}"
    echo ""
}

# Function to print section
print_section() {
    echo -e "${BOLD}${BLUE}$1${NC}"
}

# Function to print error count
print_error_line() {
    local category="$1"
    local count="$2"
    local color="$3"
    printf "  ${color}%-35s${NC} %4d\n" "$category:" "$count"
}

# Function to extract error type from golangci-lint output
extract_error_type() {
    local line="$1"

    # Extract the error type in parentheses at the end
    # Pattern: something (errortype) at end of line
    local paren_match="${line##*\(}"
    if [[ "$paren_match" != "$line" ]]; then
        paren_match="${paren_match%\)*}"
        if [[ -n "$paren_match" ]] && [[ "$paren_match" != "$line" ]]; then
            # Validate it's a known error type
            case "$paren_match" in
                typecheck|unused|errcheck|staticcheck|gosimple|govet|gocritic)
                    echo "$paren_match"
                    return
                    ;;
            esac
        fi
    fi

    # Try to extract from error message patterns
    if [[ $line =~ "undefined:" ]]; then
        echo "undefined"
    elif [[ $line =~ "assignment mismatch" ]]; then
        echo "assignment"
    elif [[ $line =~ "could not import" ]]; then
        echo "import"
    elif [[ $line =~ "does not implement" ]]; then
        echo "interface"
    elif [[ $line =~ "wrong type" ]]; then
        echo "type"
    elif [[ $line =~ "cannot use" ]]; then
        echo "cannot-use"
    elif [[ $line =~ "not found" ]]; then
        echo "not-found"
    else
        echo "other"
    fi
}

# Function to format error summary
print_error_summary() {
    local -n errors=$1
    local total_errors=0

    # Calculate total
    for count in "${!errors[@]}"; do
        ((total_errors += errors[$count]))
    done

    # Print header
    print_section "Error Categories:"
    echo ""

    # Sort and display errors
    while IFS= read -r category; do
        local count=${errors[$category]}
        local color=$RED

        # Color coding based on count
        if [[ $count -lt 5 ]]; then
            color=$YELLOW
        fi

        print_error_line "$category" "$count" "$color"
    done < <(printf '%s\n' "${!errors[@]}" | sort)

    echo ""
    print_section "Summary:"
    echo ""

    if [[ $total_errors -eq 0 ]]; then
        echo -e "  ${GREEN}✓ No linting errors found!${NC}"
    else
        echo -e "  ${RED}✗ Total errors: ${total_errors}${NC}"
    fi
    echo ""
}

# Main execution
main() {
    print_header

    echo -e "${CYAN}Running: make lint 2>&1${NC}"
    echo ""

    # Run linter and capture output
    local lint_output
    lint_output=$(make lint 2>&1)
    local exit_code=$?

    # Parse output and categorize errors
    declare -A error_counts
    declare -A error_details

    while IFS= read -r line; do
        # Skip empty lines and non-error lines
        if [[ -z "$line" ]] || [[ $line =~ ^Running ]]; then
            continue
        fi

        # Extract error type
        local error_type
        error_type=$(extract_error_type "$line")

        # Increment counter
        if [[ -n "$error_type" ]]; then
            ((error_counts[$error_type]++))
        fi
    done <<< "$lint_output"

    # Display summary
    print_error_summary error_counts

    # Display detailed errors if any
    if [[ ${#error_counts[@]} -gt 0 ]]; then
        print_section "Detailed Error Output:"
        echo ""
        echo "$lint_output" | head -100

        # Show if truncated
        local line_count
        line_count=$(echo "$lint_output" | wc -l)
        if [[ $line_count -gt 100 ]]; then
            echo ""
            echo -e "${YELLOW}... and $(($line_count - 100)) more lines ...${NC}"
        fi
        echo ""
    fi

    # Print footer with exit status
    echo -e "${BOLD}${CYAN}=====================================${NC}"
    if [[ $exit_code -eq 0 ]]; then
        echo -e "${GREEN}✓ Linting passed${NC}"
    else
        echo -e "${RED}✗ Linting failed (exit code: $exit_code)${NC}"
    fi
    echo -e "${BOLD}${CYAN}=====================================${NC}"
    echo ""

    return $exit_code
}

main "$@"
