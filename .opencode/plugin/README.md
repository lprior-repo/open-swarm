# OpenCode Plugins for Open Swarm

This directory contains OpenCode plugins that enforce development practices and workflows for the Open Swarm project.

## TDD Enforcer Plugin

**File:** `tdd-enforcer.ts`

### Purpose

Enforces Test-Driven Development (TDD) practices by monitoring file edits and ensuring:

1. **Test files exist BEFORE implementation files** - Prevents writing implementation code without tests
2. **Tests use testify** - Ensures all tests use `github.com/stretchr/testify` for assertions
3. **TDD workflow tracking** - Logs statistics on test-first development

### How It Works

The plugin listens to `file.edited` events and:

- **When a test file (`*_test.go`) is edited:**
  - Tracks it as a test file
  - Checks if it uses testify assertions
  - Shows success message if implementation doesn't exist yet (true TDD!)
  
- **When an implementation file (`*.go`) is edited:**
  - **BLOCKS** the edit if corresponding test file doesn't exist
  - Shows violation if test is missing
  - Warns if test doesn't use testify
  - Logs all violations for review

### Configuration

The plugin is enabled in `opencode.json`:

```json
{
  "plugin": ["tdd-enforcer"]
}
```

### Features

#### Real-time Enforcement
- Immediate feedback when TDD rules are violated
- Toast notifications in the OpenCode UI
- Console logging for debugging

#### Session Statistics
- Tracks tests written vs implementations written
- Reports violations and warnings
- Shows summary at session end

#### Notifications
- 🔴 **Violations**: Test file missing before implementation
- ⚠️ **Warnings**: Test doesn't use testify assertions
- ✅ **Success**: Tests written first (proper TDD!)

### Example Workflow

```bash
# 1. Write test first (encouraged!)
# Edit: pkg/calculator/calculator_test.go
[TDD Enforcer] Test file edited: pkg/calculator/calculator_test.go
✅ TDD: Test written first - following TDD! ✨

# 2. Write implementation
# Edit: pkg/calculator/calculator.go
[TDD Enforcer] Implementation file edited: pkg/calculator/calculator.go
[TDD Enforcer] ✅ Test-first development - good job!

# 3. Session complete
[TDD Enforcer] Session complete:
  - Total test files: 1
  - RED phase passed: 1
  - GREEN phase passed: 1
TDD Stats: 1/1 complete (RED→GREEN)
```

### Violation Example

```bash
# Bad: Writing implementation first
# Edit: pkg/newfeature/handler.go (without test)
🔴 TDD Violation: pkg/newfeature/handler_test.go does not exist - write tests FIRST!
[TDD Enforcer] VIOLATION: Implementation pkg/newfeature/handler.go created before test
```

### Benefits

1. **Enforces TDD discipline** - Can't accidentally skip tests
2. **Consistent test quality** - Ensures testify usage across codebase
3. **Visibility** - Session stats show TDD adherence
4. **Learning tool** - Teaches proper TDD workflow

### Compliance with AGENTS.md

This plugin directly enforces the following rules from `AGENTS.md`:

> ### 🔴 RULE #4: TDD IS MANDATORY
> **ALL Go code changes follow Test-Driven Development.**
> - Test file must exist BEFORE implementation
> - Test must fail first (RED)
> - Minimal implementation makes test pass (GREEN)
> - Use testify for assertions
> - Tests must be atomic, small, deterministic

## Installation

The plugin is automatically loaded from `.opencode/plugin/` when OpenCode starts, provided:

1. Plugin file exists: `.opencode/plugin/tdd-enforcer.ts`
2. Plugin is enabled in `opencode.json`
3. `@opencode-ai/plugin` types are installed: `npm install @opencode-ai/plugin`

## Development

To modify the plugin:

1. Edit `.opencode/plugin/tdd-enforcer.ts`
2. Restart OpenCode to reload the plugin
3. Test with: `opencode`

TypeScript is supported natively via Bun.

## Troubleshooting

### Plugin not loading
- Check `opencode.json` has `"plugin": ["tdd-enforcer"]`
- Verify file exists at `.opencode/plugin/tdd-enforcer.ts`
- Check console for plugin initialization: `[TDD Enforcer] Plugin initialized`

### Violations not showing
- Ensure you're editing Go files (`.go`)
- Check that OpenCode is running and monitoring file edits
- Look for event logs: `[TDD Enforcer] Implementation file edited: ...`

### False positives
- If a test file already exists from a previous session, the plugin should detect it
- Plugin tracks state within a session; state clears on `session.idle`
