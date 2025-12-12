# Open Swarm Example Workflows

This directory contains example workflow definitions demonstrating different patterns and capabilities of the Open Swarm multi-agent coordination framework.

## Overview

Workflows in Open Swarm are JSON-based definitions that describe how agents coordinate to accomplish tasks. They can range from simple single-agent implementations to complex multi-stage DAGs with parallel execution.

## Workflow Types

Open Swarm supports two primary workflow types:

### 1. TCR Workflows (Test-Commit-Revert)

A sequential pattern where an agent:
1. Receives a task prompt
2. Implements changes
3. Runs tests
4. Commits if tests pass, reverts if they fail

**Best for:** Single-agent feature implementation with safety nets

### 2. DAG Workflows (Directed Acyclic Graph)

Workflows with explicit dependency management allowing:
- Sequential stages where each stage depends on previous stages
- Parallel execution of independent stages or tasks
- Complex branching and conditional logic
- Multiple agents working simultaneously

**Best for:** Build pipelines, complex features, CI/CD orchestration

## Example Workflows

### 1. Simple TCR: `simple-tcr.json`

**Complexity:** Beginner
**Type:** Single-agent TCR pattern
**Use Case:** Safe feature implementation with automatic rollback

#### Workflow Stages

```
Bootstrap Agent → Execute Prompt → Run Tests → Commit/Revert → Cleanup
```

#### Key Features

- Single agent implementation
- File reservation for exclusive access
- Test-driven validation
- Automatic rollback on test failure
- Clean session management

#### Quick Start

```bash
# View the workflow definition
cat simple-tcr.json

# Key sections:
# - "steps": Linear execution stages
# - "agents": Single implementer role
# - "failure_handling": Revert on test failure strategy
```

#### Customization

To adapt this workflow for your use case:

1. **Change the prompt**: Edit the `prompt_template` in the `prompt_execution` step
2. **Modify file pattern**: Update `pattern` in the `reserve_files` action
3. **Adjust timeouts**: Change `timeout_seconds` values as needed
4. **Update test command**: Modify the `command` in the `run_tests` step

Example: Implementing a new API endpoint

```json
{
  "prompt_template": "Implement a POST /api/users endpoint in pkg/api/handlers.go that validates JSON input and creates a user in the database"
}
```

### 2. Build-Test DAG: `build-test-dag.json`

**Complexity:** Intermediate
**Type:** Sequential DAG with two agents
**Use Case:** Standard CI/CD pipeline (build → test → report)

#### Workflow Stages

```
Prepare → Build → Test → Report
   ↓       ↓      ↓       ↓
Checkout  Build  Unit    Coverage
Deps      Lint   Tests   Report
                 Coverage Summary
```

#### Key Features

- Sequential stages with clear dependencies
- Parallel execution within stages (where applicable)
- Build and lint quality checks
- Code coverage validation
- Automated test execution
- Report generation

#### Agent Roles

- **Builder Agent**: Executes prepare, build, and test stages
- **Reporter Agent**: Generates final reports and artifacts

#### Configuration

```json
{
  "stages": [
    {
      "stage_id": "prepare",
      "tasks": [...]
    },
    {
      "stage_id": "build",
      "depends_on": ["prepare"],
      "tasks": [...]
    }
  ]
}
```

#### Expected Outputs

- Compiled binary: `bin/open-swarm`
- Coverage report: `coverage.html`
- Build logs: `build.log`, `test.log`

#### Integration Example

```bash
# Execute the workflow
open-swarm run ./examples/build-test-dag.json

# Check results
ls -la bin/open-swarm
open coverage.html
```

### 3. Multi-Stage DAG: `multi-stage-dag.json`

**Complexity:** Advanced
**Type:** Complex DAG with parallel stages and 4 agents
**Use Case:** Full feature development lifecycle with comprehensive QA

#### Workflow Stages

```
Setup
  ↓
Development (parallel: backend_api, data_layer, service_logic)
  ↓
Integration
  ↓
Quality Assurance (parallel: unit_tests, lint, security, benchmark)
  ↓
Documentation
  ↓
Deployment Preparation
  ↓
Reporting
```

#### Key Features

- Multiple parallel development tasks
- Comprehensive quality assurance
  - Unit testing with coverage threshold (80%)
  - Code linting and formatting
  - Security vulnerability scanning
  - Performance benchmarking
- Automated documentation generation
- Version management and changelog generation
- Release artifact creation
- Detailed final reporting

#### Agent Roles

- **Backend Dev**: Implements API and data layer
- **Service Dev**: Implements business logic
- **QA Agent**: Runs comprehensive quality checks
- **Release Agent**: Manages documentation and deployment prep

#### Advanced Features

##### Parallel Execution

The development stage allows three agents to work simultaneously:

```json
{
  "stage_id": "development",
  "type": "parallel",
  "tasks": [
    { "task_id": "backend_api", ... },
    { "task_id": "data_layer", ... },
    { "task_id": "service_logic", ... }
  ]
}
```

Estimated time: 20 minutes (vs 60 minutes sequentially)

##### Quality Gates

Each task includes success criteria:

```json
{
  "type": "test",
  "config": {
    "coverage_threshold": 80,
    "expected_result": "PASS"
  }
}
```

##### Conditional Execution

Tasks can have conditional behavior:

```json
{
  "type": "release",
  "config": {
    "version_strategy": "semantic",
    "only_if": "all_tests_passed"
  }
}
```

#### Using This Workflow

```bash
# Full feature development with 4 parallel agents
open-swarm run ./examples/multi-stage-dag.json \
  --parallel \
  --feature "New Feature Name"

# Watch progress
open-swarm monitor

# Check final report
cat build-report.md
```

#### Customization for Your Project

1. **Add/remove development tasks**:
   ```json
   {
     "task_id": "custom_component",
     "name": "Implement Custom Component",
     "type": "feature",
     "config": {
       "prompt": "Your custom prompt here",
       "file_pattern": "pkg/custom/**/*.go"
     }
   }
   ```

2. **Adjust quality thresholds**:
   ```json
   {
     "task_id": "unit_tests",
     "config": {
       "coverage_threshold": 85
     }
   }
   ```

3. **Add deployment stage**:
   ```json
   {
     "stage_id": "deploy",
     "stage_name": "Deployment",
     "depends_on": ["reporting"],
     "tasks": [
       {
         "task_id": "deploy_staging",
         "command": "kubectl apply -f k8s/staging/"
       }
     ]
   }
   ```

## Common Workflow Patterns

### Pattern 1: Single Feature Implementation

Use the **Simple TCR** workflow:

```bash
cd open-swarm
open-swarm run examples/simple-tcr.json \
  --task "Add user authentication" \
  --agent "FeatureDev"
```

### Pattern 2: Standard Build & Test

Use the **Build-Test DAG** workflow:

```bash
open-swarm run examples/build-test-dag.json \
  --branch "develop"
```

### Pattern 3: Complete Release Pipeline

Use the **Multi-Stage DAG** workflow:

```bash
open-swarm run examples/multi-stage-dag.json \
  --release-version "1.0.0" \
  --parallel
```

## Workflow Anatomy

Every workflow JSON file has this structure:

```json
{
  "id": "unique-workflow-id",
  "name": "Human-Readable Name",
  "description": "What this workflow does",
  "version": "1.0",
  "type": "tcr" | "dag",
  "metadata": { ... },
  "workflow": {
    "steps" | "stages": [ ... ]
  },
  "agents": [ ... ],
  "expected_artifacts": { ... }
}
```

### Key Sections

#### `metadata`
- **author**: Who created the workflow
- **created**: When it was created
- **tags**: Searchable keywords
- **complexity**: "beginner", "intermediate", "advanced"

#### `workflow`
Contains the execution logic:
- **steps** (TCR): Linear execution sequence
- **stages** (DAG): Stages with dependencies
- **failure_handling**: What to do when things fail

#### `agents`
Defines agent roles and resource requirements:

```json
{
  "id": "unique-agent-id",
  "role": "what_they_do",
  "stages": ["stage1", "stage2"],
  "resources": {
    "memory_mb": 1024,
    "cpu_cores": 2,
    "timeout_minutes": 15
  }
}
```

#### `expected_artifacts`
Documents what the workflow produces:
- Binaries
- Reports
- Source code changes
- Test outputs

## Creating Your Own Workflow

### Step 1: Choose a Type

```
TCR → Simple, single-agent, sequential work
DAG → Complex, multi-agent, parallel work
```

### Step 2: Define Stages/Steps

```json
{
  "stages": [
    {
      "stage_id": "my_stage",
      "stage_name": "My Stage",
      "tasks": [
        { "task_id": "my_task", ... }
      ]
    }
  ]
}
```

### Step 3: Define Agents

```json
{
  "agents": [
    {
      "id": "my_agent",
      "role": "what_they_do",
      "stages": ["my_stage"],
      "resources": { ... }
    }
  ]
}
```

### Step 4: Add Dependencies

For DAG workflows:

```json
{
  "stages": [
    { "stage_id": "setup", "depends_on": [] },
    { "stage_id": "build", "depends_on": ["setup"] },
    { "stage_id": "test", "depends_on": ["build"] }
  ]
}
```

### Step 5: Test Your Workflow

```bash
# Validate JSON
python -m json.tool your-workflow.json

# Dry run (doesn't execute)
open-swarm validate examples/your-workflow.json

# Execute
open-swarm run examples/your-workflow.json --dry-run
```

## Troubleshooting

### "Workflow failed at stage X"

1. Check the error logs
2. Verify dependencies are met
3. Review timeout values
4. Check agent resource allocation

### "Test coverage below threshold"

Edit the workflow to:
1. Lower the threshold temporarily: `"coverage_threshold": 75`
2. Or add more test coverage before running

### "Parallel tasks not executing"

Ensure:
1. `"type": "parallel"` is set on the stage
2. Tasks don't have circular dependencies
3. Sufficient system resources available

## Best Practices

### Naming

- Use descriptive IDs: `backend_api_implementation`, not `task1`
- Use clear stage names: "Feature Development", not "Dev"
- Document the purpose in the description

### Dependencies

- Keep stage dependencies simple and clear
- Document with comments why dependencies exist
- Use parallel execution to reduce total time

### Resources

- Allocate realistic timeout values
- Add 20% buffer to estimated times
- Monitor actual usage and adjust

### Error Handling

- Define failure strategies for each stage
- Use fast-fail for critical paths
- Collect detailed logs for debugging

### Testing

- Always include test stages before deployment
- Set appropriate coverage thresholds
- Add security and performance checks

## Next Steps

1. **Understand Your Workflow**: Read the JSON definition carefully
2. **Customize for Your Needs**: Edit prompts, stages, and agents
3. **Test Locally**: Use `--dry-run` before executing
4. **Monitor Execution**: Check logs and reports
5. **Iterate**: Refine based on actual execution results

## References

- [Open Swarm README](../README.md) - Architecture and overview
- [AGENTS.md](../AGENTS.md) - Agent coordination guidelines
- [REACTOR.md](../REACTOR.md) - Reactor orchestrator documentation
- [QUICKSTART.md](../QUICKSTART.md) - Getting started guide

## Support

For questions about workflows:

1. Check this README for common patterns
2. Review the example workflow JSON files
3. See AGENTS.md for coordination patterns
4. File an issue: `bd create "Workflow question" --tag documentation`

---

**Happy orchestrating!** Create amazing workflows with Open Swarm.
