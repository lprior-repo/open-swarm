# Beads Plugin - Quick Reference

## Installation Check

```bash
# Verify all dependencies
which bd                          # Should show: /home/lewis/.local/bin/bd
which bun                         # Should show: /usr/bin/bun
which typescript-language-server  # Should show: ~/.local/share/mise/.../typescript-language-server
```

## Function Reference

### 📋 Get Ready Tasks
```typescript
ready({ format: "json" | "text" })
```

### 📝 Update Task Status
```typescript
status({ 
  taskId: "open-swarm-axu.1.1",
  status: "ready" | "in_progress" | "blocked" | "done"
})
```

### ✅ Close Task
```typescript
close({ 
  taskId: "open-swarm-axu.1.1",
  reason: "Completed successfully"
})
```

### ➕ Create Task
```typescript
create({ 
  title: "Task title",
  type: "feature" | "bug" | "chore" | "doc" | "task",    // optional
  priority: "low" | "normal" | "high" | "urgent",         // optional
  parent: "parent-task-id"                                // optional
})
```

### 📊 List Tasks
```typescript
list({ 
  status: "ready" | "in_progress" | "blocked" | "done" | "closed",  // optional
  tag: "tag-name",                                                   // optional
  format: "json" | "text"                                            // optional
})
```

### 🔗 Add Dependency
```typescript
addDependency({ 
  childId: "task-1",
  parentId: "task-2",
  type: "blocks" | "related" | "discovered-from" | "parent-child"  // optional
})
```

## Common Workflows

### Start Working on a Task
```typescript
// 1. Check ready tasks
await ready.execute({ format: "text" })

// 2. Update status
await status.execute({ taskId: "bd-xxxx", status: "in_progress" })
```

### Create Task with Dependency
```typescript
// 1. Create parent task
const parent = await create.execute({ 
  title: "Build API",
  type: "feature",
  priority: "high"
})

// 2. Create child task
const child = await create.execute({ 
  title: "Write tests",
  type: "task",
  parent: parent.id
})

// 3. Add blocking dependency
await addDependency.execute({
  childId: child.id,
  parentId: parent.id,
  type: "blocks"
})
```

### Complete a Task
```typescript
// 1. Update status
await status.execute({ taskId: "bd-xxxx", status: "done" })

// 2. Close with reason
await close.execute({ 
  taskId: "bd-xxxx",
  reason: "Implemented and tested successfully"
})
```

## Error Handling

All functions throw descriptive errors:

```typescript
try {
  await status.execute({ taskId: "invalid", status: "done" })
} catch (error) {
  console.error(error.message)
  // Output: "Failed to update task invalid: Task not found"
}
```

## Testing

```bash
# Run structural tests
cd .opencode
bun tool/beads.test.ts

# Build verification
bun build tool/beads.ts --target=bun --outfile=/tmp/test.js

# Check CLI integration
bd ready --json
bd list --json
```

## Troubleshooting

### Plugin not loading
```bash
# Check Bun runtime
bun --version

# Check OpenCode configuration
cat opencode.json | grep -A 5 '"lsp"'

# Rebuild node_modules
cd .opencode
rm -rf node_modules
bun install
```

### TypeScript errors
```bash
# Install types
cd .opencode
npm install --save-dev @types/bun

# Verify tsconfig
cat tsconfig.json
```

### Beads CLI issues
```bash
# Check Beads installation
bd --version

# Verify database
bd list --json

# Re-initialize if needed
bd init
```

## Configuration Files

| File | Purpose |
|------|---------|
| `opencode.json` | LSP configuration |
| `tsconfig.json` | TypeScript settings |
| `bunfig.toml` | Bun runtime config |
| `package.json` | Dependencies |

## Documentation

- **Full Documentation:** `tool/README.md`
- **Validation Report:** `VALIDATION_REPORT.md`
- **This Reference:** `QUICK_REFERENCE.md`

## Support

For issues or questions:
1. Check `VALIDATION_REPORT.md` for known issues
2. Review `README.md` for detailed examples
3. Verify all dependencies are installed
4. Test with `bun tool/beads.test.ts`
