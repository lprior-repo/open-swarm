#!/usr/bin/env bash
#
# Pre-commit hook for open-swarm
# Fast checks on staged Go files only
#
# Installation:
#   chmod +x scripts/pre-commit.sh
#   ln -sf ../../scripts/pre-commit.sh .git/hooks/pre-commit
#

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Get list of staged Go files
STAGED_GO_FILES=$(git diff --cached --name-only --diff-filter=ACM | grep '\.go$' || true)

if [ -z "$STAGED_GO_FILES" ]; then
    echo -e "${GREEN}✓${NC} No Go files staged, skipping checks"
    exit 0
fi

echo "Running pre-commit checks on staged Go files..."
echo ""

# Track if any check fails
FAILED=0

# 1. Check gofmt
echo -n "1. Checking gofmt... "
UNFORMATTED=$(echo "$STAGED_GO_FILES" | xargs gofmt -l 2>&1 || true)
if [ -n "$UNFORMATTED" ]; then
    echo -e "${RED}✗${NC}"
    echo ""
    echo -e "${RED}Error: The following files are not formatted:${NC}"
    echo "$UNFORMATTED"
    echo ""
    echo "Run: gofmt -w $UNFORMATTED"
    FAILED=1
else
    echo -e "${GREEN}✓${NC}"
fi

# 2. Run go vet on packages containing staged files
echo -n "2. Running go vet... "
# Get unique package paths from staged files
PACKAGES=$(echo "$STAGED_GO_FILES" | xargs -n1 dirname | sort -u | sed 's|^|./|' | tr '\n' ' ')
if ! go vet $PACKAGES >/dev/null 2>&1; then
    echo -e "${RED}✗${NC}"
    echo ""
    echo -e "${RED}go vet found issues:${NC}"
    go vet $PACKAGES 2>&1
    FAILED=1
else
    echo -e "${GREEN}✓${NC}"
fi

# 3. Run golangci-lint on changed files only (fast mode)
echo -n "3. Running golangci-lint (fast)... "
if command -v golangci-lint >/dev/null 2>&1; then
    # Use --new-from-rev to only check changed code
    # Use --fast to skip slow linters
    # Use --timeout to prevent hanging
    if ! echo "$STAGED_GO_FILES" | xargs golangci-lint run --fast --timeout=60s --new 2>&1 | tee /tmp/golangci-lint.out >/dev/null; then
        echo -e "${RED}✗${NC}"
        echo ""
        echo -e "${RED}golangci-lint found issues:${NC}"
        cat /tmp/golangci-lint.out
        rm -f /tmp/golangci-lint.out
        FAILED=1
    else
        echo -e "${GREEN}✓${NC}"
        rm -f /tmp/golangci-lint.out
    fi
else
    echo -e "${YELLOW}⚠${NC} (golangci-lint not installed, skipping)"
fi

# 4. Check go mod tidy
echo -n "4. Checking go mod tidy... "
# Save current go.mod and go.sum
cp go.mod go.mod.pre-commit
cp go.sum go.sum.pre-commit

# Run go mod tidy
go mod tidy >/dev/null 2>&1

# Check if anything changed
if ! diff -q go.mod go.mod.pre-commit >/dev/null 2>&1 || ! diff -q go.sum go.sum.pre-commit >/dev/null 2>&1; then
    echo -e "${RED}✗${NC}"
    echo ""
    echo -e "${RED}Error: go.mod or go.sum is not tidy${NC}"
    echo ""
    echo "Changes needed:"
    diff -u go.mod.pre-commit go.mod || true
    diff -u go.sum.pre-commit go.sum || true
    echo ""
    echo "Run: go mod tidy"
    
    # Restore original files
    mv go.mod.pre-commit go.mod
    mv go.sum.pre-commit go.sum
    FAILED=1
else
    echo -e "${GREEN}✓${NC}"
    rm -f go.mod.pre-commit go.sum.pre-commit
fi

echo ""

# Summary
if [ $FAILED -ne 0 ]; then
    echo -e "${RED}✗ Pre-commit checks failed${NC}"
    echo ""
    echo "Fix the issues above and try again."
    echo "To skip these checks (not recommended), use: git commit --no-verify"
    exit 1
else
    echo -e "${GREEN}✓ All pre-commit checks passed${NC}"
    exit 0
fi
