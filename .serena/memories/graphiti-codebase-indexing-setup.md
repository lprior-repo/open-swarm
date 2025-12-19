# Graphiti Codebase Indexing Setup

## Overview
The open-swarm codebase has been comprehensively indexed using Graphiti knowledge graph with 6 structured JSON episodes.

## Group ID
`open-swarm-codebase` - Use this when searching or adding to the graph

## Episodes Created

### 1. Open Swarm Project Architecture
- **Purpose**: Tech stack, package structure, core components
- **Covers**: Go 1.25.4, Temporal SDK, OpenCode SDK, Anthropic SDK, Docker/PostgreSQL
- **Key entities**: Project structure (cmd/, pkg/, internal/), dependencies

### 2. Agent System and Anti-Cheating Gates
- **Purpose**: Agent lifecycle and verification framework
- **Covers**: Agent struct (Name, Program, Model, TaskDescription, LastActive, ProjectKey)
- **5 Gates**: Requirements Verification, Test Immutability, Empirical Honesty, Hard Work Enforcement, Drift Detection
- **Key files**: internal/gates/, pkg/agent/, pkg/coordinator/

### 3. Temporal Workflow Engine and TCR Cycles
- **Purpose**: Workflow orchestration and TDD/TCR methodology
- **Covers**: 4 workflows (Basic, Enhanced, Parallel, Benchmark)
- **TDD/TCR phases**: RED → GREEN → REFACTOR (BLUE) → VERIFY
- **Key files**: internal/temporal/, internal/orchestration/

### 4. Code Patterns and Prompt Engineering System
- **Purpose**: Code generation and AI interaction patterns
- **Covers**: 5 prompt builders, 3 executor types, factory pattern (TestRunner, CodeAnalyzer, CodeGenerator)
- **Key files**: internal/prompts/, internal/opencode/, pkg/dag/

### 5. Package Dependencies and Architecture Relationships
- **Purpose**: Complete dependency map and execution flow
- **Covers**: Package relationships, execution pipeline, data flow
- **Key diagram**: User → Beads → Reader → Spawner → Workflow → Gates → Results → Beads

### 6. Graphiti Query Guide and Best Practices
- **Purpose**: Semantic search guide and agent interaction patterns
- **Covers**: 4 recommended query categories, 4 agent scenarios, maintenance guidelines
- **Best practices**: Specificity, entity relationships, technical terminology, problem context

## Search Examples

### Find Temporal Workflows
```
search_nodes("temporal workflow TCR test commit revert")
```

### Find Anti-Cheating Implementation
```
search_nodes("anti-cheating gates verification")
```

### Understand Agent Spawning
```
search_memory_facts("agent lifecycle ephemeral execution")
```

### Find Code Generation Pattern
```
search_nodes("OpenCode SDK code generation")
```

## Key Architecture Insights

### 5-Gate Verification System
All agents must pass 5 verification gates to claim task completion:
1. Requirements Verification (agent understands spec)
2. Test Immutability (tests can't be modified)
3. Empirical Honesty (raw test output only)
4. Hard Work Enforcement (no stubs allowed)
5. Requirement Drift Detection (stays aligned)

### Ephemeral Agent Lifecycle
```
Spawn → Execute Workflow → Verify Gates → Teardown
```

### TDD/TCR Cycle
```
RED (fail tests) → GREEN (pass tests) → BLUE (refactor) → VERIFY (all pass)
```

### Task Flow
```
Beads Task → Spawner → Agent → Temporal Workflow → Gates → Results → Beads
```

## Package Structure Quick Reference
- **pkg/agent**: Agent identity and management
- **pkg/coordinator**: Multi-agent coordination
- **pkg/dag**: DAG engine for parallel execution
- **internal/temporal**: Workflow definitions and activities
- **internal/opencode**: SDK wrapper and code execution
- **internal/gates**: Verification gates
- **internal/orchestration**: High-level orchestration
- **internal/prompts**: Prompt engineering

## Updating Graphiti Index
When codebase changes:
1. Identify which episode(s) need updating
2. Add new entities/relationships to JSON
3. Re-index via `add_memory` with updated content
4. Maintain group_id: "open-swarm-codebase"

## Related Tools
- **Semantic search**: Use `search_nodes` for entity discovery
- **Fact search**: Use `search_memory_facts` for relationship queries
- **Graph exploration**: Use `get_episodes` to retrieve all indexed content
