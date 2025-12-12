# Linting Guide for Open Swarm

This document explains the golangci-lint configuration for the open-swarm project.

## Overview

The `.golangci.yml` configuration is tailored specifically for:
- **Temporal workflows** - Allows timeouts, retry counts, and context patterns
- **gRPC services** - Handles protobuf generated files appropriately  
- **CLI applications** - Permits global variables and magic numbers in main packages

## Running the Linter

```bash
# Run all checks
golangci-lint run

# Run with auto-fix
golangci-lint run --fix

# Run on specific files/directories
golangci-lint run ./pkg/...
golangci-lint run internal/temporal/

# Run specific linters only
golangci-lint run --disable-all -E errcheck -E gosimple
```

## CI Integration

The linter runs automatically in GitHub Actions CI with a 5-minute timeout:

```yaml
- name: Run golangci-lint
  uses: golangci/golangci-lint-action@v3
  with:
    version: latest
    args: --timeout=5m
```

## Enabled Linters (45 total)

### Critical Linters
- **errcheck** - Ensures all errors are checked
- **govet** - Go's built-in vet tool
- **staticcheck** - Comprehensive static analysis
- **gosec** - Security vulnerability detection

### Code Quality
- **cyclop** - Cyclomatic complexity (max 15)
- **gocognit** - Cognitive complexity (max 20)
- **dupl** - Code duplication detection (min 150 tokens)
- **goconst** - Find repeated strings that should be constants
- **gocritic** - Opinionated linter with multiple checks

### Best Practices
- **errorlint** - Go 1.13+ error wrapping
- **contextcheck** - Context usage validation
- **bodyclose** - HTTP response body closure
- **noctx** - HTTP requests must use context
- **wrapcheck** - External errors must be wrapped

### Style & Formatting
- **gofmt** - Standard Go formatting
- **goimports** - Import statement formatting
- **revive** - Configurable style checker
- **stylecheck** - Golint replacement
- **misspell** - Spell checking

## Disabled Linters (30 total)

The following linters are intentionally disabled to avoid being overly strict:

- **containedctx** - Temporal workflows embed contexts in structs (required pattern)
- **exhaustruct** - Forcing all struct fields is too strict
- **funlen** - We use complexity metrics instead
- **godox** - TODO/FIXME comments are acceptable during development
- **goerr113** - Custom error handling is project-specific
- **lll** - Line length handled by editor settings
- **nonamedreturns** - Named returns improve clarity
- **paralleltest** - Not all tests need t.Parallel()
- **varnamelen** - Short variable names are idiomatic Go

## Special Exclusions

### Test Files (`*_test.go`)
Excluded linters:
- `cyclop`, `gocognit`, `gocyclo` - Tests can be more complex
- `dupl` - Test setup code often duplicates
- `errcheck` - Test errors don't always need checking
- `mnd` - Magic numbers acceptable in tests
- `gosec` - Security checks less critical in tests

### Main Files (`cmd/*/main.go`)
Excluded linters:
- `gochecknoglobals` - Global variables OK for CLI flags
- `mnd` - Magic numbers acceptable for configuration
- `wrapcheck` - Top-level error handling differs

### Temporal Code (`internal/temporal/`)
Excluded linters:
- `mnd` - Timeouts and retry counts are self-documenting
- `wrapcheck` - Temporal SDK handles error wrapping

### Temporal Activities (`internal/temporal/activities_*.go`)
Excluded linters:
- `gosec` - G204 exemption for shell command execution
- `wrapcheck` - Activity errors wrapped by Temporal

### Infrastructure (`internal/infra/`)
Excluded linters:
- `mnd` - Port numbers and timeouts are clear in context

### Generated Files
Completely excluded:
- `*.pb.go` - Protocol buffer generated
- `*.pb.gw.go` - gRPC gateway generated
- `*_gen.go` - Any generated code
- `mock_*.go` - Mock implementations

## Key Configuration Settings

### Cyclomatic Complexity
```yaml
cyclop:
  max-complexity: 15  # Per-function limit
  skip-tests: true
```

### Code Duplication
```yaml
dupl:
  threshold: 150  # Minimum token count
```

### Magic Numbers
```yaml
mnd:
  ignored-numbers:
    - '0'
    - '1'
    - '2'
    - '10'
    - '100'
    - '1000'
  ignored-functions:
    - 'time.*'
    - 'context.WithTimeout'
    - 'make'
```

### Security
```yaml
gosec:
  excludes:
    - G104  # Covered by errcheck
    - G204  # Needed for shell activities
  severity: medium
  confidence: medium
```

### Error Handling
```yaml
errcheck:
  check-type-assertions: true
  exclude-functions:
    - (io.Closer).Close
    - (*database/sql.Rows).Close
```

### GoVet
```yaml
govet:
  enable-all: true
  disable:
    - fieldalignment  # Can break compatibility
    - shadow          # Too strict
```

## Common Issues and Fixes

### "context.Context should be the first parameter"
**Excluded for Temporal workflows** - Temporal patterns sometimes require different parameter orders.

### "Magic number" warnings
Use constants for repeated numbers:
```go
// Bad
time.Sleep(5 * time.Second)
time.Sleep(5 * time.Second)

// Good
const defaultTimeout = 5 * time.Second
time.Sleep(defaultTimeout)
```

### "Error return value is not checked"
Always check errors:
```go
// Bad
file.Close()

// Good
if err := file.Close(); err != nil {
    return fmt.Errorf("close file: %w", err)
}

// Or for defer
defer func() {
    if err := file.Close(); err != nil {
        log.Printf("failed to close file: %v", err)
    }
}()
```

### "Cognitive complexity is too high"
Break down complex functions:
```go
// Bad - all logic in one function
func ProcessUser(user User) error {
    // 50 lines of complex logic
}

// Good - extract helpers
func ProcessUser(user User) error {
    if err := validateUser(user); err != nil {
        return err
    }
    if err := saveUser(user); err != nil {
        return err
    }
    return notifyUser(user)
}
```

### "Duplicate code detected"
Extract common code:
```go
// Bad
func CreateUser() {
    db := connectDB()
    // ... logic
}
func UpdateUser() {
    db := connectDB()
    // ... logic
}

// Good
func withDB(fn func(*DB) error) error {
    db := connectDB()
    defer db.Close()
    return fn(db)
}
```

## Temporal-Specific Patterns

### Activity Options
```go
// Allowed - timeouts are self-documenting in context
ao := workflow.ActivityOptions{
    StartToCloseTimeout: 10 * time.Minute,  // OK
    HeartbeatTimeout:    30 * time.Second,   // OK
    RetryPolicy: &temporal.RetryPolicy{
        MaximumAttempts: 3,  // OK
    },
}
```

### Workflow Contexts
```go
// Allowed - Temporal workflow pattern
type WorkflowState struct {
    ctx workflow.Context  // OK in Temporal workflows
}
```

## IDE Integration

### VS Code
Install the golangci-lint extension:
```json
{
  "go.lintTool": "golangci-lint",
  "go.lintFlags": [
    "--fast"
  ]
}
```

### GoLand/IntelliJ
1. Preferences → Tools → File Watchers
2. Add golangci-lint watcher
3. Arguments: `run --fix $FilePath$`

### Vim/Neovim
Use ale or coc-go:
```vim
let g:ale_linters = {'go': ['golangci-lint']}
let g:ale_go_golangci_lint_options = '--fast'
```

## Performance Tips

```bash
# Fast check (skip some slow linters)
golangci-lint run --fast

# Cache results
golangci-lint cache clean  # Clear if stale
golangci-lint run  # Uses cache automatically

# Parallel execution (enabled by default)
# Controlled by: allow-parallel-runners: true
```

## Updating Configuration

When modifying `.golangci.yml`:

1. Test locally first:
   ```bash
   golangci-lint run
   ```

2. Check specific paths:
   ```bash
   golangci-lint run internal/temporal/
   ```

3. Verify CI compatibility:
   ```bash
   golangci-lint run --timeout=5m
   ```

4. Document changes in this file

## References

- [golangci-lint Documentation](https://golangci-lint.run/)
- [Linter List](https://golangci-lint.run/usage/linters/)
- [Configuration Reference](https://golangci-lint.run/usage/configuration/)
- [GitHub Actions Integration](https://golangci-lint.run/usage/install/#github-actions)

## Troubleshooting

### Linter not installed
```bash
# Install latest version
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
```

### Timeout in CI
Increase timeout in `.golangci.yml`:
```yaml
run:
  timeout: 10m  # Increase if needed
```

### Too many issues
Address incrementally:
```yaml
issues:
  max-issues-per-linter: 50  # Limit per linter
  max-same-issues: 3         # Limit duplicates
```

Or use `new: true` to only check new code:
```yaml
issues:
  new: true
```
