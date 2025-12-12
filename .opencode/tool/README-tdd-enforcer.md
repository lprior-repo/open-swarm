# TDD Enforcer Plugin

An OpenCode plugin that enforces Test-Driven Development (TDD) workflow for Go projects.

## Overview

The TDD Enforcer validates that code changes follow proper TDD practices:

1. **Test-First Development**: Test files must exist before implementation
2. **Red Phase**: Tests must fail initially (proving they test the right thing)
3. **Green Phase**: Tests must pass with minimal implementation
4. **Quality Standards**: Tests must use testify and be atomic/focused
5. **Suite Validation**: Full test suite must pass

## Available Tools

### `tdd-enforcer_validateTDD`

Validates the complete TDD workflow for a Go file.

**Arguments:**
- `filePath` (string, required): Path to the Go implementation file (not the test file)
- `skipRedCheck` (boolean, optional): Skip the Red phase check (default: false)

**Usage:**
```typescript
// In OpenCode
await client.tool.execute({
  tool: "tdd-enforcer_validateTDD",
  args: {
    filePath: "internal/api/handler.go",
    skipRedCheck: false
  }
})
```

**Validation Steps:**
1. ✅ Test file exists before implementation
2. ✅ Test uses testify assertions
3. ✅ Test is atomic and focused (≤3 test functions)
4. ✅ Test fails initially (Red phase)
5. ✅ Test passes with implementation (Green phase)
6. ✅ Full test suite passes

### `tdd-enforcer_checkTestCoverage`

Checks test coverage for a Go package and ensures it meets minimum threshold.

**Arguments:**
- `packagePath` (string, required): Package path (e.g., `./internal/api`)
- `minCoverage` (number, optional): Minimum coverage percentage (default: 80)

**Usage:**
```typescript
await client.tool.execute({
  tool: "tdd-enforcer_checkTestCoverage",
  args: {
    packagePath: "./internal/api",
    minCoverage: 80
  }
})
```

**Output:**
- ✅ Coverage percentage and validation status
- ❌ Error message if coverage is below threshold

### `tdd-enforcer_enforceTestFirst`

Validates test-first development by checking git history.

**Arguments:**
- `implFile` (string, required): Implementation file path

**Usage:**
```typescript
await client.tool.execute({
  tool: "tdd-enforcer_enforceTestFirst",
  args: {
    implFile: "internal/api/handler.go"
  }
})
```

**Checks:**
- Test file exists
- Test file was modified alongside implementation
- Test file was committed in git history

## Integration

### OpenCode Configuration

The plugin is automatically enabled via `opencode.json`:

```json
{
  "tools": {
    "tdd-enforcer_*": true
  },
  "permission": {
    "tdd-enforcer_*": "allow"
  }
}
```

### Pre-commit Hook

Create a pre-commit hook to enforce TDD:

```bash
#!/bin/bash
# .git/hooks/pre-commit

# Get modified Go files (not tests)
GO_FILES=$(git diff --cached --name-only --diff-filter=ACM | grep '\.go$' | grep -v '_test\.go$')

for file in $GO_FILES; do
  # Validate TDD workflow
  opencode tool execute tdd-enforcer_validateTDD --filePath "$file" --skipRedCheck true
  if [ $? -ne 0 ]; then
    echo "TDD validation failed for $file"
    exit 1
  fi
done
```

### CI/CD Integration

Add to your CI pipeline:

```yaml
# .github/workflows/ci.yml
- name: Validate TDD
  run: |
    for file in $(find . -name "*.go" -not -name "*_test.go"); do
      opencode tool execute tdd-enforcer_checkTestCoverage \
        --packagePath $(dirname $file) \
        --minCoverage 80
    done
```

## Best Practices

### 1. Use with OpenCode Agents

The enforcer works seamlessly with OpenCode's agent system:

```bash
# In OpenCode TUI
/validate internal/api/handler.go
```

### 2. Incremental Adoption

Start with `skipRedCheck: true` for existing codebases:

```typescript
await validateTDD({
  filePath: "legacy/handler.go",
  skipRedCheck: true  // Skip red phase for existing code
})
```

### 3. Combine with Agent Mail

Use with Agent Mail for multi-agent coordination:

```bash
# Reserve files before development
/reserve internal/api/**/*.go

# Validate TDD before releasing
tdd-enforcer_validateTDD internal/api/handler.go

# Release files
/release
```

## Examples

### Example 1: New Feature Development

```bash
# 1. Write failing test
cat > internal/api/user_test.go <<EOF
package api

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestCreateUser(t *testing.T) {
    result := CreateUser("John")
    assert.NotNil(t, result)
}
EOF

# 2. Validate (should fail - no implementation)
tdd-enforcer_validateTDD internal/api/user.go
# Output: 🔴 RED: Test still failing

# 3. Write minimal implementation
cat > internal/api/user.go <<EOF
package api

func CreateUser(name string) *User {
    return &User{Name: name}
}
EOF

# 4. Validate (should pass)
tdd-enforcer_validateTDD internal/api/user.go
# Output: ✅ TDD WORKFLOW VALIDATED
```

### Example 2: Coverage Check

```bash
# Check coverage for package
tdd-enforcer_checkTestCoverage ./internal/api 80

# Output: ✅ Coverage: 85% (exceeds minimum: 80%)
```

### Example 3: Code Review

```bash
# Validate test-first development in PR
tdd-enforcer_enforceTestFirst internal/api/handler.go

# Output: ✅ Test-first development validated
```

## Troubleshooting

### Test not found

```
❌ TDD VIOLATION: Test file must exist BEFORE implementation.
```

**Solution**: Create the test file first: `internal/api/handler_test.go`

### Test doesn't use testify

```
⚠️  Test should use testify for assertions.
```

**Solution**: Add `import "github.com/stretchr/testify/assert"`

### Too many test functions

```
⚠️  Test file has 5 test functions. Keep tests atomic and focused.
```

**Solution**: Split into multiple focused test files

### Red phase failed

```
❌ TDD VIOLATION: Test passes without implementation (RED phase failed).
```

**Solution**: Ensure test is properly isolated and tests new functionality

## Development

### Running Tests

```bash
# Run TDD enforcer tests
go test ./internal/tdd/... -v

# Run with coverage
go test ./internal/tdd/... -cover
```

### Project Structure

```
.opencode/
  tool/
    tdd-enforcer.ts          # Plugin implementation
    README-tdd-enforcer.md   # This file
internal/
  tdd/
    enforcer_test.go         # Go tests for validation
```

## References

- [OpenCode Custom Tools Documentation](https://opencode.ai/docs/custom-tools)
- [OpenCode Plugin Development](https://opencode.ai/docs/plugins)
- [Effective Go](https://go.dev/doc/effective_go)
- [Testify Documentation](https://github.com/stretchr/testify)

## License

MIT License - See LICENSE file in repository root
