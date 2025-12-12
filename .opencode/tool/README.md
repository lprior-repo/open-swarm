# OpenCode Tools

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
