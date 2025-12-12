# Beads Plugin for OpenCode

This plugin provides integration between OpenCode and the Beads issue tracking system.

## Features

- ✅ **Type-safe**: Enum validation for all status and type parameters
- ✅ **Error handling**: Comprehensive try/catch blocks with descriptive errors
- ✅ **Security**: Command injection prevention using proper argument passing
- ✅ **Dual output**: JSON and human-readable text formats
- ✅ **Full CRUD**: Create, read, update, and manage Beads tasks

## Functions

### `ready`
Get ready (unblocked) tasks from Beads issue tracker.

**Arguments:**
- `format`: `"json" | "text"` (optional, default: `"json"`)

**Example:**
```typescript
await ready.execute({ format: "text" })
```

### `status`
Update status of a Beads task.

**Arguments:**
- `taskId`: string - Task ID (e.g., "bd-a1b2")
- `status`: `"ready" | "in_progress" | "blocked" | "done"`

**Example:**
```typescript
await status.execute({ 
  taskId: "open-swarm-axu.1.1", 
  status: "in_progress" 
})
```

### `close`
Close a Beads task with a completion reason.

**Arguments:**
- `taskId`: string - Task ID to close
- `reason`: string - Reason for completion

**Example:**
```typescript
await close.execute({ 
  taskId: "open-swarm-axu.1.1", 
  reason: "Implemented and tested" 
})
```

### `create`
Create a new Beads task.

**Arguments:**
- `title`: string - Task title (required)
- `type`: `"feature" | "bug" | "chore" | "doc" | "task"` (optional)
- `priority`: `"low" | "normal" | "high" | "urgent"` (optional)
- `parent`: string - Parent task ID (optional)

**Example:**
```typescript
await create.execute({ 
  title: "Add user authentication",
  type: "feature",
  priority: "high"
})
```

### `list`
List Beads tasks with optional filters.

**Arguments:**
- `status`: `"ready" | "in_progress" | "blocked" | "done" | "closed"` (optional)
- `tag`: string (optional)
- `format`: `"json" | "text"` (optional, default: `"json"`)

**Example:**
```typescript
await list.execute({ 
  status: "in_progress",
  format: "json" 
})
```

### `addDependency`
Add a dependency between two Beads tasks.

**Arguments:**
- `childId`: string - Child task ID
- `parentId`: string - Parent task ID
- `type`: `"blocks" | "related" | "discovered-from" | "parent-child"` (optional, default: `"blocks"`)

**Example:**
```typescript
await addDependency.execute({ 
  childId: "open-swarm-axu.1.2",
  parentId: "open-swarm-axu.1.1",
  type: "blocks"
})
```

## Security Improvements

### Command Injection Prevention

The `create` and `list` functions now use `Bun.spawn()` with proper argument arrays instead of string concatenation:

**Before:**
```typescript
let cmd = `bd create "${args.title}"`  // ❌ Vulnerable
const result = await $`${cmd}`.text()
```

**After:**
```typescript
const cmdArgs = ["bd", "create", args.title]  // ✅ Safe
const proc = Bun.spawn(cmdArgs, { stdout: "pipe" })
```

### Input Validation

All parameters are validated with TypeScript enums:

```typescript
status: tool.schema.enum(["ready", "in_progress", "blocked", "done"])
type: tool.schema.enum(["feature", "bug", "chore", "doc", "task"])
```

### Error Handling

All functions wrapped in try/catch with descriptive error messages:

```typescript
try {
  await $`bd update ${args.taskId} --status ${args.status}`
  return `Task ${args.taskId} updated to status: ${args.status}`
} catch (error) {
  throw new Error(`Failed to update task ${args.taskId}: ${error.message}`)
}
```

## Testing

Run tests with:

```bash
cd .opencode
bun tool/beads.test.ts
```

Build for validation:

```bash
bun build tool/beads.ts --target=bun --outfile=/tmp/beads.test.js
```

## Requirements

- **Bun** runtime: `bun --version`
- **Beads CLI**: `bd --version`
- **@opencode-ai/plugin**: Installed in node_modules

## Integration with OpenCode

This plugin is automatically loaded by OpenCode when placed in `.opencode/tool/`.

The tools are available in OpenCode workflows and can be called from agents.

## Development

### Setup TypeScript LSP

TypeScript Language Server is configured in `opencode.json`:

```json
"lsp": {
  "typescript": {
    "command": ["typescript-language-server", "--stdio"],
    "filetypes": [".ts", ".tsx", ".js", ".jsx"],
    "rootPatterns": ["package.json", "tsconfig.json"]
  }
}
```

### Type Definitions

Install development dependencies:

```bash
npm install --save-dev @types/bun
```

### Configuration Files

- `tsconfig.json` - TypeScript compiler configuration
- `bunfig.toml` - Bun runtime configuration
- `package.json` - Node dependencies

## Changelog

### v2.0.0 (Current)
- ✅ Added enum validation for all status/type parameters
- ✅ Added comprehensive error handling with try/catch
- ✅ Fixed command injection vulnerability in `create` and `list`
- ✅ Added proper Bun.spawn() usage for secure command execution
- ✅ Added TypeScript LSP support
- ✅ Added bunfig.toml configuration
- ✅ Added comprehensive test suite

### v1.0.0
- Initial implementation with basic functionality
