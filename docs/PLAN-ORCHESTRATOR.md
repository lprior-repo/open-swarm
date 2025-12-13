# Plan Orchestrator

The Plan Orchestrator is an automated tool that parses user plans and creates structured Beads issues, enabling rapid project planning and task breakdown.

## Overview

Instead of manually creating individual Beads issues with `bd create`, you can write a high-level plan in natural language and let the orchestrator automatically break it down into actionable tasks.

## Quick Start

```bash
# Create a plan file
cat > my-plan.txt <<EOF
1. Implement user authentication [P0]
2. Add password reset functionality [P1]
3. Create user profile page [P1]
EOF

# Preview what will be created
go run cmd/plan-orchestrator/main.go --file my-plan.txt --dry-run

# Create the issues
go run cmd/plan-orchestrator/main.go --file my-plan.txt --execute

# View created issues
bd list
```

## How It Works

```
┌─────────────────┐
│  User Plan      │  High-level plan in text format
│  (Text File)    │  (numbered lists, bullets, markdown)
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Plan Parser    │  Extracts structured tasks
│  (parser.go)    │  - Title, description, priority
└────────┬────────┘  - Dependencies, labels
         │
         ▼
┌─────────────────┐
│ Beads Creator   │  Generates bd create commands
│ (creator.go)    │  with proper dependencies
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Orchestrator    │  Executes commands and creates
│ (main.go)       │  Beads issues in the database
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Beads Issues    │  Ready for agent execution
│ Created         │  via bead-swarm
└─────────────────┘
```

## Supported Input Formats

### Numbered Lists

```
1. Create database schema
2. Build user model
3. Add authentication
```

### Bullet Points

```
- Setup project structure
- Configure dependencies
- Write tests
```

### With Priorities

Priority levels: P0 (highest) to P5 (lowest)

```
1. Fix critical bug [P0]
2. Add feature [P1]
3. Refactor [P2]
```

### With Dependencies

```
1. Create schema
2. Build model (depends on: 1)
3. Add API (depends on: 2)
```

### Markdown Headers with Descriptions

```markdown
## Task 1: Create authentication endpoint
Description: Build REST API for user login
Acceptance: Returns JWT token on valid credentials

## Task 2: Add password hashing
Description: Use bcrypt for secure storage
```

### Complex Multi-Phase Plans

```
# E-Commerce Feature

## Phase 1: Backend
1. Create cart database schema [P0]
2. Implement cart service [P0] (depends on: 1)
   Description: Core business logic for cart operations

## Phase 2: API
3. Build cart API endpoints [P1] (depends on: 2)
4. Add inventory integration [P1] (depends on: 3)

## Phase 3: Frontend
5. Create cart UI [P2] (depends on: 3)
6. Add cart persistence [P2] (depends on: 2, 5)
```

## CLI Reference

### Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--prefix` | Project prefix for issue IDs | `open-swarm` |
| `--file` | Read plan from file | stdin |
| `--dry-run` | Preview without creating | `false` |
| `--execute` | Actually create issues | `false` |

### Examples

**Preview from file:**
```bash
go run cmd/plan-orchestrator/main.go --file plan.txt --dry-run
```

**Create issues:**
```bash
go run cmd/plan-orchestrator/main.go --file plan.txt --execute
```

**Interactive mode:**
```bash
go run cmd/plan-orchestrator/main.go --execute
# Paste your plan, then Ctrl+D
```

**Custom project prefix:**
```bash
go run cmd/plan-orchestrator/main.go --prefix myapp --file plan.txt --execute
```

## Integration with Bead Swarm

The Plan Orchestrator and Bead Swarm work together to provide a complete automated development workflow:

### Workflow

```
1. Planning Phase (Plan Orchestrator)
   └─> Parse high-level plan
   └─> Create Beads issues with dependencies

2. Execution Phase (Bead Swarm)
   └─> Pick up ready issues
   └─> Execute with AI agents
   └─> Run tests and commit
```

### Example End-to-End Flow

```bash
# Step 1: Create issues from your plan
cat > feature-plan.txt <<EOF
1. Create user registration API [P0]
2. Add email verification [P1] (depends on: 1)
3. Build login UI [P1] (depends on: 1)
4. Add password reset [P2] (depends on: 1, 2)
EOF

go run cmd/plan-orchestrator/main.go --file feature-plan.txt --execute
# Output: Created 4 Beads issues

# Step 2: View the created issues
bd list --status=ready
# Shows all issues ready to work on (no dependencies blocking)

# Step 3: Let the swarm execute them
go run cmd/bead-swarm/main.go --max 3
# Spawns 3 agents to work on ready tasks in parallel

# Step 4: Monitor progress
bd list --status=in_progress
bd list --status=closed

# Step 5: Review and sync
git log
bd sync
```

### Dependency Resolution

The orchestrator creates issues with proper dependencies:

```
Task 1 (No deps) ───┐
                    ├─> Task 2 (depends on: 1) ───┐
Task 3 (No deps) ───┘                             ├─> Task 5 (depends on: 2, 3, 4)
                                                   │
Task 4 (No deps) ──────────────────────────────────┘
```

Bead Swarm respects these dependencies:
- Tasks with no dependencies start immediately
- Dependent tasks wait for parent tasks to complete
- Parallel execution when tasks are independent

## Advanced Features

### Multi-Priority Planning

```
# Critical path items
1. Fix security vulnerability [P0]
2. Patch database leak [P0]

# Important features
3. Add user dashboard [P1]
4. Implement analytics [P1]

# Nice-to-haves
5. Improve UI styling [P2]
6. Add dark mode [P3]
```

Bead Swarm can prioritize based on these priorities using `--priority-filter`.

### Dependency Chains

```
1. Setup infrastructure
2. Create database (depends on: 1)
3. Build API (depends on: 2)
4. Add frontend (depends on: 3)
5. Write e2e tests (depends on: 4)
```

The swarm executes these in order, ensuring each step completes before the next begins.

### Parallel Workstreams

```
# Backend stream
1. Create API endpoints [P0]
2. Add authentication [P1] (depends on: 1)

# Frontend stream
3. Build UI components [P0]
4. Add routing [P1] (depends on: 3)

# Integration
5. Connect frontend to API [P2] (depends on: 2, 4)
```

The swarm can run backend and frontend tasks in parallel, then integrate them.

## Architecture

### Internal Packages

#### `internal/planner/parser.go`
- Parses text input into structured tasks
- Supports multiple formats (numbered, bullets, markdown)
- Extracts priorities and dependencies

#### `internal/planner/creator.go`
- Generates Beads issue creation commands
- Handles dependency relationships
- Creates unique issue IDs

#### `internal/planner/types.go`
- Common data structures
- `ParsedTask`: Structured task representation
- `ExecutionPlan`: Command execution plan

### Command Flow

```go
// 1. Parse input text
parser := planner.NewPlanParser()
tasks, err := parser.Parse(input)

// 2. Create execution plan
creator := planner.NewBeadsCreator("prefix")
plan, err := creator.CreatePlan(tasks)

// 3. Execute commands
for _, cmd := range plan.Commands {
    exec.Command("bd", "create", ...).Run()
}
```

## Testing

### Unit Tests

```bash
go test ./internal/planner/... -v
```

### Integration Test

```bash
# Create a test plan
cat > test-plan.txt <<EOF
1. Test task A [P0]
2. Test task B [P1] (depends on: 1)
EOF

# Test dry-run
go run cmd/plan-orchestrator/main.go --file test-plan.txt --dry-run

# Test execution
go run cmd/plan-orchestrator/main.go --file test-plan.txt --execute

# Verify creation
bd show $(bd list --json | jq -r '.[0].id')

# Clean up
bd close $(bd list --json | jq -r '.[0].id')
bd close $(bd list --json | jq -r '.[1].id')
```

## Example Plans

See `examples/plans/` for complete examples:

- `simple-feature.txt` - Basic numbered list with priorities
- `complex-feature.txt` - Multi-phase plan with dependencies
- `infrastructure.txt` - Infrastructure migration plan

## Troubleshooting

### No tasks parsed

Check your input format:
```bash
# ✓ Correct
1. Task name [P0]
- Task name

# ✗ Incorrect
Task name
a) Task name
```

### Dependencies not working

Ensure task numbers match:
```bash
# ✓ Correct
1. First task
2. Second task (depends on: 1)

# ✗ Incorrect - skipped number
1. First task
3. Third task (depends on: 1)
```

### Command execution failed

Check that `bd` CLI is installed and in PATH:
```bash
which bd
bd --version
```

## See Also

- [Bead Swarm Documentation](../cmd/bead-swarm/README.md)
- [Beads CLI Documentation](../.beads/README.md)
- [Internal Planner Package](../internal/planner/)
