# Linting Guide

This document explains the linting setup for the open-swarm project, how to run linters, fix common errors, and available scripts.

## Overview

The project uses a comprehensive linting setup tailored for Go projects with Temporal workflows, gRPC services, and CLI applications. The linting strategy is production-ready but not overly strict.

### Linters Enabled

- **errcheck** - Detects unchecked errors
- **govet** - Examines Go source code and reports suspicious constructs
- **ineffassign** - Detects when assignments to existing variables are not used
- **staticcheck** - Advanced Go linting with deep analysis
- **unused** - Detects unused variables, constants, functions, and types
- **bodyclose** - Ensures HTTP response bodies are closed
- **contextcheck** - Verifies context usage in functions
- **cyclop** - Checks function complexity
- **dupl** - Detects code duplication
- **durationcheck** - Checks for correct duration usage
- **errname** - Checks if error variable names follow convention
- **errorlint** - Enforces error wrapping and comparison
- **exhaustive** - Ensures switch statements are exhaustive
- **copyloopvar** - Detects loop variable issues
- **gocheckcompilerdirectives** - Validates compiler directives
- **gocognit** - Checks cognitive complexity
- **goconst** - Detects repeated strings that could be constants
- **gocritic** - Opinionated Go code reviewer
- **gocyclo** - Checks cyclomatic complexity
- **mnd** - Detects magic numbers
- **goprintffuncname** - Validates Printf-like function names
- **gosec** - Security scanner for Go
- **misspell** - Finds commonly misspelled words
- **nakedret** - Detects naked returns in long functions
- **nilerr** - Detects nil error returns
- **nilnil** - Detects code returning both nil and nil
- **noctx** - Detects context.Context not passed
- **nolintlint** - Ensures nolint directives are properly formatted
- **prealloc** - Suggests preallocated slices
- **predeclared** - Detects shadowing of predeclared identifiers
- **revive** - Fast linter with customizable rules
- **thelper** - Detects test helper functions
- **tparallel** - Detects unmarked parallel tests
- **unconvert** - Detects unnecessary type conversions
- **unparam** - Detects unused function parameters
- **whitespace** - Detects leading and trailing whitespace
- **wrapcheck** - Ensures errors are properly wrapped

### Formatters Enabled

- **gofmt** - Go code formatter
- **goimports** - Organizes imports and formats code

## Quick Start

### Install Development Tools

```bash
make install-tools
```

This installs:
- `golangci-lint` (v2.7.2)
- `goimports` for import management

### Run All Linters

```bash
make lint
```

Example output:
```
Running golangci-lint...
✓ Linting completed
```

### Auto-Fix Linting Issues

```bash
make lint-fix
```

This runs golangci-lint with the `--fix` flag to automatically correct issues when possible.

### Format Code

```bash
make fmt
```

Applies gofmt to all Go files.

## Common Linting Errors and Fixes

### 1. Unchecked Errors (errcheck)

**Error:**
```
error return value not checked
```

**Problem:** A function returns an error that isn't being checked.

**Fix:**
```go
// WRONG
file, _ := os.Open("config.json")  // Ignoring error

// CORRECT
file, err := os.Open("config.json")
if err != nil {
    return fmt.Errorf("failed to open config: %w", err)
}
```

### 2. Unused Variables (unused)

**Error:**
```
`result` is unused
```

**Problem:** A variable is declared but never used.

**Fix:**
```go
// WRONG
result, err := someFunction()
if err != nil {
    return err
}

// CORRECT (if result is truly not needed)
_, err := someFunction()
if err != nil {
    return err
}

// OR use the variable
value := someFunction()
log.Printf("Result: %v", value)
```

### 3. Unused Function Parameters (unparam)

**Error:**
```
`ctx` is unused
```

**Problem:** A function parameter is declared but never used.

**Fix:**
```go
// WRONG
func ProcessTask(ctx context.Context, task *Task) error {
    return task.Execute()
}

// CORRECT
func ProcessTask(ctx context.Context, task *Task) error {
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
        return task.Execute()
    }
}
```

### 4. Missing Error Wrapping (wrapcheck)

**Error:**
```
error returned by external package function should be wrapped
```

**Problem:** Returning an error directly instead of wrapping it for context.

**Fix:**
```go
// WRONG
return err

// CORRECT
return fmt.Errorf("failed to process workflow: %w", err)
```

### 5. Unclosed Resources (bodyclose)

**Error:**
```
response body must be closed
```

**Problem:** HTTP response body isn't closed, causing resource leaks.

**Fix:**
```go
// WRONG
resp, err := http.Get(url)
if err != nil {
    return err
}
return json.NewDecoder(resp.Body).Decode(&data)

// CORRECT
resp, err := http.Get(url)
if err != nil {
    return err
}
defer resp.Body.Close()
return json.NewDecoder(resp.Body).Decode(&data)
```

### 6. Magic Numbers (mnd)

**Error:**
```
Magic number: 100
```

**Problem:** Using hardcoded numbers instead of named constants.

**Fix:**
```go
// WRONG
if len(items) > 100 {
    return errors.New("too many items")
}

// CORRECT
const maxItems = 100
if len(items) > maxItems {
    return errors.New("too many items")
}
```

### 7. Repeated Strings as Constants (goconst)

**Error:**
```
"worker" has 4 occurrences, but `MinOccurrences` is 3
```

**Problem:** A string appears multiple times and should be a constant.

**Fix:**
```go
// WRONG
if workerType == "worker" {
    // ...
}
startWorker("worker")
logWorker("worker instance started")

// CORRECT
const workerTypeKey = "worker"
if workerType == workerTypeKey {
    // ...
}
startWorker(workerTypeKey)
logWorker(workerTypeKey + " instance started")
```

### 8. Misspellings (misspell)

**Error:**
```
misspelled word: "occured"
```

**Problem:** Typos in code or comments.

**Fix:**
```go
// WRONG
// An error occured during processing
err := process()

// CORRECT
// An error occurred during processing
err := process()
```

### 9. No Context in Function (noctx)

**Error:**
```
context.Context should be the first parameter
```

**Problem:** A function should accept context but doesn't, or context isn't the first parameter.

**Note:** This error is excluded in `internal/temporal/` directory by configuration.

**Fix:**
```go
// WRONG
func ProcessWorkflow(task *Task) error {
    // ...
}

// CORRECT
func ProcessWorkflow(ctx context.Context, task *Task) error {
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
        // process task
    }
}
```

### 10. Empty Error Returns (nilerr)

**Error:**
```
error is nil but returns non-nil error
```

**Problem:** Returning a nil error after nil check.

**Fix:**
```go
// WRONG
if err != nil {
    return err
}
return err

// CORRECT
return err
```

## Linting Configuration Details

### Complexity Thresholds

- **Cyclomatic Complexity (gocyclo):** max 15
- **Cognitive Complexity (gocognit):** min 20
- **Cyclomatic Complexity per function (cyclop):** max 15

Functions exceeding these limits should be refactored into smaller, focused functions.

### Duplication Threshold

- **Code Duplication (dupl):** threshold 150 lines

Code blocks larger than 150 lines should not be duplicated. Extract to shared functions instead.

### String Constants Threshold

- **Min length:** 3 characters
- **Min occurrences:** 3 times

Strings appearing 3+ times with 3+ characters should be extracted to constants.

## Excluded Patterns

The following files/patterns are excluded from certain linter checks:

### Test Files (`*_test.go`)
- cyclop, dupl, errcheck, errorlint, gochecknoglobals, gocognit, goconst, gocritic, gocyclo, mnd, gosec, wrapcheck

### Command Files (`cmd/*/main.go`)
- gochecknoglobals, mnd, wrapcheck

### Temporal Directory (`internal/temporal/`)
- mnd, wrapcheck

### Generated Files
- Protobuf files (`*.pb.go`)
- Protobuf gateway files (`*.pb.gw.go`)
- Generated files (`*_gen.go`)
- Mock files (`mock_*.go`)

## Pre-commit Hooks

The project uses pre-commit hooks to automatically lint and format code before commits.

### Setup Pre-commit Hooks

```bash
pip install pre-commit
pre-commit install
```

### Manual Pre-commit Hook Run

```bash
# Run on staged files only
pre-commit run

# Run on all files
pre-commit run --all-files

# Run markdown linting manually (not in commit hook)
pre-commit run markdownlint --all-files
```

### Available Hooks

- **gofmt** - Formats Go code
- **go vet** - Runs go vet analysis
- **goimports** - Manages and formats imports
- **go mod tidy** - Ensures go.mod and go.sum are tidy
- **golangci-lint (fast)** - Fast linting mode with timeout
- **Trailing whitespace** - Removes trailing whitespace
- **End of file fixer** - Fixes missing newlines
- **YAML syntax check** - Validates YAML files
- **JSON syntax check** - Validates JSON files
- **Merge conflict check** - Detects merge conflicts
- **Large file check** - Prevents committing files > 1MB
- **Line ending check** - Ensures LF line endings
- **Markdownlint** - Lints markdown files (manual stage)

## CI Pipeline

The project's CI pipeline runs the following checks automatically:

1. **Linting** - Full golangci-lint with 5-minute timeout
2. **Testing** - Unit tests with race detector and coverage
3. **Building** - Builds all binaries for Linux and Darwin

### Run Local CI Checks

```bash
make ci
```

This runs:
- Code formatting (`make fmt`)
- Full linting (`make lint`)
- Unit tests (`make test`)
- Race detector tests (`make test-race`)

## Testing Commands

### Run All Tests

```bash
make test
```

### Run Tests with Race Detector

```bash
make test-race
```

Detects potential race conditions in concurrent code.

### Generate Coverage Report

```bash
make test-coverage
```

Generates an HTML coverage report at `coverage.html`.

### Run Tests with TDD Guard

```bash
make test-tdd
```

Requires `tdd-guard-go` to be installed.

## Best Practices

1. **Address Linting Issues Early** - Fix linting errors as you write code, not before commits
2. **Use Auto-Fix** - Use `make lint-fix` for automatic corrections when available
3. **Understand Warnings** - Read and understand linting warnings rather than blindly suppressing them
4. **Proper Error Wrapping** - Always wrap errors with context using `fmt.Errorf("context: %w", err)`
5. **Named Constants** - Extract magic numbers and repeated strings into named constants
6. **Context Usage** - Always pass context as the first parameter to functions that need it
7. **Resource Management** - Always defer close/cleanup operations for resources
8. **Test Organization** - Keep test files properly formatted and organized with helper functions

## Suppressing Linting Errors

When you have a valid reason to suppress a linting error, use `nolint` directives:

### Suppress Single Linter

```go
//nolint:misspell
// An occured error (intentional misspelling for documentation)
```

### Suppress Multiple Linters

```go
//nolint:mnd,wrapcheck
result := complexCalculation() // 42 is a magic number here
```

### Suppress with Explanation

```go
//nolint:wrapcheck // Reviewed and safe to return unwrapped error here
return err
```

The project requires nolint directives to be:
- **Specific** - Target the exact linters being suppressed
- **Explanatory** - Include a comment explaining why

## Linting Performance

- **Timeout:** 5 minutes for full linting, 60 seconds for pre-commit fast mode
- **Parallel Execution:** Enabled for faster linting
- **New Issues Only:** CI checks only for new issues with `--new` flag

## Troubleshooting

### golangci-lint Not Found

**Solution:** Run `make install-tools`

### Linting Timeout

**Solution:** Linting has a 5-minute timeout. If exceeded, consider:
- Breaking down complex functions
- Checking for performance issues in your code
- Running with `--fast` flag for quick feedback

### Pre-commit Hook Failed

**Solution:** Run `pre-commit run --all-files` to see detailed error messages

### Specific Linter Causing Issues

**Solution:** Run golangci-lint with only specific linters:
```bash
golangci-lint run --enable=<linter-name> ./...
```

## Additional Resources

- [golangci-lint Documentation](https://golangci-lint.run/)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Pre-commit Framework](https://pre-commit.com/)
