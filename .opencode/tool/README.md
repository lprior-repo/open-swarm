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

## Beads Integration

Beads functionality is provided by the **Beads MCP server** (`beads-mcp`), configured in `opencode.json`.

The previous `.opencode/tool/beads.ts` plugin was redundant and has been removed.

### Using Beads via MCP

All Beads operations use the MCP tools:
- `beads_ready` - Get ready tasks
- `beads_status` - Update task status
- `beads_create` - Create new task
- `beads_close` - Close task with reason
- `beads_list` - List tasks with filters
- `beads_addDependency` - Add task dependencies

### Configuration

See `opencode.json`:
```json
"mcp": {
  "beads": {
    "type": "local",
    "command": ["beads-mcp"],
    "environment": {
      "BEADS_USE_DAEMON": "1",
      "BEADS_ACTOR": "opencode"
    },
    "enabled": true,
    "timeout": 30000
  }
}
```

### Direct CLI Access

For quick operations, `bd` CLI commands still work:
```bash
bd ready --json
bd update bd-xxxx --status in_progress
bd close bd-xxxx --reason "Description"
```
