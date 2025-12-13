# Plan Orchestrator

Automated tool that parses user plans and creates Beads issues.

## Overview

The Plan Orchestrator takes a high-level plan (in various text formats) and automatically breaks it down into structured Beads issues. This enables quick project planning and task creation without manually running `bd create` for each task.

## Features

- **Multiple Input Formats**: Supports numbered lists, bullet points, and markdown headers
- **Dependency Tracking**: Automatically handles task dependencies
- **Priority Support**: Set task priorities inline with `[P0]` to `[P5]`
- **Dry Run Mode**: Preview what will be created before executing
- **File or Stdin Input**: Read plans from files or paste them interactively

## Usage

### Basic Usage (Interactive)

```bash
go run cmd/plan-orchestrator/main.go --execute
# Then paste your plan and press Ctrl+D
```

### From File

```bash
go run cmd/plan-orchestrator/main.go --file plan.txt --execute
```

### Dry Run (Preview)

```bash
go run cmd/plan-orchestrator/main.go --file plan.txt --dry-run
```

### With Custom Project Prefix

```bash
go run cmd/plan-orchestrator/main.go --prefix myproject --file plan.txt --execute
```

## Input Formats

### Numbered List

```
1. Create database schema
2. Build user model
3. Add authentication middleware
```

### Bullet Points

```
- Setup project structure
- Configure dependencies
- Write initial tests
```

### With Priorities

```
1. Fix critical security bug [P0]
2. Add new feature [P1]
3. Refactor code [P2]
```

### With Dependencies

```
1. Create database schema
2. Build user model (depends on: 1)
3. Add authentication middleware (depends on: 2)
```

### Markdown Headers

```markdown
# User Authentication System

## Task 1: Create user authentication endpoint
Description: Build REST API endpoint for user login
Acceptance: Returns JWT token on valid credentials

## Task 2: Add password hashing
Description: Use bcrypt for secure password storage
```

### Complex Example

```
Project Plan: API Development

1. Setup infrastructure [P0]
2. Create database schema (depends on: 1)
3. Build API endpoints [P1]
   Description: RESTful endpoints for CRUD operations
4. Add authentication [P1] (depends on: 2, 3)
5. Write integration tests [P2] (depends on: 4)
```

## Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--prefix` | Project prefix for issue IDs | `open-swarm` |
| `--file` | Read plan from file instead of stdin | (stdin) |
| `--dry-run` | Show what would be created without executing | `false` |
| `--execute` | Actually execute bd commands to create issues | `false` |

## Examples

### Create Issues from a Plan File

```bash
# 1. Create a plan file
cat > my-plan.txt <<EOF
1. Implement user authentication [P0]
2. Add password reset functionality [P1] (depends on: 1)
3. Create user profile page [P1] (depends on: 1)
4. Add email verification [P2] (depends on: 1)
EOF

# 2. Preview what will be created
go run cmd/plan-orchestrator/main.go --file my-plan.txt --dry-run

# 3. Create the issues
go run cmd/plan-orchestrator/main.go --file my-plan.txt --execute

# 4. View created issues
bd list
```

### Interactive Mode

```bash
go run cmd/plan-orchestrator/main.go --execute

# Then paste:
1. Setup CI/CD pipeline [P0]
2. Configure automated testing [P1]
3. Add deployment automation [P1] (depends on: 1)
# Press Ctrl+D

# Issues are created automatically
```

## Integration with Bead Swarm

The Plan Orchestrator works seamlessly with the existing bead-swarm system:

1. **Plan Orchestrator**: Creates structured Beads issues from high-level plans
2. **Bead Swarm**: Picks up ready issues and executes them with AI agents

```bash
# Step 1: Create issues from plan
go run cmd/plan-orchestrator/main.go --file feature-plan.txt --execute

# Step 2: Let the swarm work on them
go run cmd/bead-swarm/main.go --max 5
```

## Architecture

```
User Plan Text
     ↓
Parser (internal/planner/parser.go)
     ↓
Parsed Tasks (structured data)
     ↓
Creator (internal/planner/creator.go)
     ↓
Execution Plan (bd create commands)
     ↓
Command Executor (this CLI)
     ↓
Beads Issues Created
```

## Error Handling

The orchestrator validates:
- Plan format and syntax
- Task dependencies (no circular dependencies)
- Command execution (bd CLI must be available)

If any command fails, execution stops and reports which tasks were created successfully.

## Development

### Run Tests

```bash
go test ./internal/planner/...
```

### Build Binary

```bash
go build -o bin/plan-orchestrator cmd/plan-orchestrator/main.go
```

### Add to PATH

```bash
# Add to your shell profile
export PATH="$PATH:$HOME/src/open-swarm/bin"

# Then use it directly
plan-orchestrator --file plan.txt --execute
```

## See Also

- [Beads Documentation](../../.beads/README.md)
- [Bead Swarm](../bead-swarm/main.go)
- [Internal Planner Package](../../internal/planner/)
