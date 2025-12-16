# OpenCode Tools

## TDD Guard

Enforces Test-Driven Development workflow for Go code changes.

### Tools

- `validateTDD` - Validates complete TDD workflow (Red → Green → Refactor)
- `checkTestCoverage` - Ensures test coverage meets minimum threshold
- `enforceTestFirst` - Checks git diff to ensure test-first development

### Usage

```typescript
// Validate TDD workflow for a changed file
validateTDD({ filePath: "internal/api/handler.go" })

// Check coverage
checkTestCoverage({ packagePath: "./internal/api", minCoverage: 80 })

// Enforce test-first from git diff
enforceTestFirst({ implFile: "internal/api/handler.go" })
```

### TDD Workflow Validation

1. **Test Exists** - Test file must exist before implementation
2. **Uses Testify** - Test must use `github.com/stretchr/testify`
3. **Atomic Tests** - Tests should be small and focused (≤3 test functions)
4. **RED Phase** - Test must fail initially (validates test correctness)
5. **GREEN Phase** - Test passes with minimal implementation
6. **Full Suite** - All package tests pass


