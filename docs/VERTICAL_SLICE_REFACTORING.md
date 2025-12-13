# Vertical Slice Architecture Refactoring Plan

## Overview

This document outlines the refactoring of Open Swarm's temporal package from a **Layered/Technical Architecture** to a **Vertical Slice Architecture** following CUPID principles.

## Current Architecture (Layered/Technical)

The current codebase organizes code by technical concerns:

```
internal/temporal/
├── activities_cell.go          # Cell lifecycle activities
├── activities_enhanced.go      # Enhanced TCR activities
├── activities_locks.go         # File locking activities
├── activities_shell.go         # Shell command activities
├── workflows_tcr.go            # Basic TCR workflow
├── workflows_enhanced.go       # Enhanced TCR workflow
├── workflows_dag.go            # DAG workflow
├── types_enhanced.go           # Type definitions
├── policies.go                 # Retry policies & timeouts
├── metrics.go                  # Metrics collection
├── output_parser.go            # Output parsing
├── test_parser.go              # Test result parsing
├── orchestrator_utils.go       # Orchestration utilities
└── retry_budget.go             # Retry budget management
```

**Problems with this approach:**
- Hard to understand complete features
- Changes require modifying multiple files
- Dependencies scattered across many files
- Difficult to test features in isolation
- Technical layers don't map to business capabilities

## Target Architecture (Vertical Slices)

Organize code by business capability, with each file containing all layers for one feature:

```
internal/temporal/slices/
├── types.go                    # Core domain types (foundation)
├── config.go                   # Configuration and policies
├── observability.go            # Metrics and telemetry
├── test_execution.go           # Test running and validation
├── linting.go                  # Lint validation
├── code_review.go              # Code review orchestration
├── code_generation.go          # Code generation
├── cell_lifecycle.go           # Cell management
├── file_locks.go               # File locking coordination
├── workflow_tcr_basic.go       # Basic TCR workflow
├── workflow_tcr_enhanced.go    # Enhanced TCR workflow
└── workflow_dag.go             # DAG workflow
```

Each slice contains:
- **Domain types** specific to that capability
- **Activities** (data access layer)
- **Business logic** (domain layer)
- **Workflow integration** (orchestration layer)

## CUPID Principles

### C - Composable
- Each slice is self-contained and can be composed with others
- Clear interfaces between slices
- Dependencies explicit through imports

### U - Unix Philosophy
- Each slice does one thing well
- Single responsibility at feature level
- Small, focused files (200-400 lines)

### P - Predictable
- Clear inputs and outputs
- No hidden global state
- Explicit dependencies
- Deterministic behavior

### I - Idiomatic
- Follows Go conventions
- Uses Temporal patterns correctly
- Standard error handling
- Clear naming

### D - Domain-Centric
- Organized by business capability
- Named after what it does, not how
- Reflects user/business language
- Feature-first, not tech-first

## Refactoring Tasks

### Foundation Layer (No Dependencies)

#### 1. types.go - Core Domain Types ✅ COMPLETED
**Status:** Implemented in `/internal/temporal/slices/types.go`

Consolidates all type definitions:
- Cell lifecycle types
- Task execution types
- Workflow state types
- Gate result types
- Test execution types
- Linting types
- Code review types
- File locking types
- DAG workflow types
- TCR workflow types
- Retry budget types

#### 2. config.go - Configuration and Policies ✅ COMPLETED
**Status:** Implemented in `/internal/temporal/slices/config.go`

Contains:
- Timeout configurations
- Retry policies
- Activity options
- Error classification

#### 3. observability.go - Metrics and Telemetry ✅ COMPLETED
**Status:** Implemented in `/internal/temporal/slices/observability.go`

Provides:
- Metrics collection
- Workflow queries
- Progress tracking
- Performance monitoring

### Domain Slices Layer (Depends on Foundation)

#### 4. test_execution.go - Test Running and Validation
**Dependencies:** types.go, config.go

Contains:
- Test execution activities
- Test result parsing
- Test failure analysis
- Red/Green verification gates

Consolidates:
- `test_parser.go` (378 lines)
- Test execution logic from `activities_enhanced.go`
- Test verification workflows

#### 5. linting.go - Lint Validation
**Dependencies:** types.go, config.go

Contains:
- Lint execution activities
- Lint result parsing
- Issue formatting
- Lint gate logic

Consolidates:
- Lint parsing from `orchestrator_utils.go`
- Lint activities from `activities_enhanced.go`

#### 6. code_review.go - Code Review Orchestration
**Dependencies:** types.go, config.go

Contains:
- Multi-reviewer coordination
- Vote aggregation
- Review result parsing
- Review gate logic

Consolidates:
- `ReviewAggregator` from `orchestrator_utils.go`
- `VoteParser` from `orchestrator_utils.go`
- Review activities from `activities_enhanced.go`

#### 7. code_generation.go - Code Generation
**Dependencies:** types.go, config.go, observability.go

Contains:
- Code generation activities
- Implementation generation
- Test generation
- File modification tracking

Consolidates:
- GenTest activity
- GenImpl activity
- Output parsing logic

#### 8. cell_lifecycle.go - Cell Management
**Dependencies:** types.go, config.go

Contains:
- Bootstrap activities
- Teardown activities
- Cell state management
- Port/worktree coordination

Consolidates:
- `activities_cell.go` (185 lines)
- Cell lifecycle logic

#### 9. file_locks.go - File Locking Coordination
**Dependencies:** types.go, config.go

Contains:
- Lock acquisition activities
- Lock release activities
- Conflict detection
- Lock registry integration

Consolidates:
- `activities_locks.go` (261 lines)
- File locking logic

### Workflow Layer (Depends on Slices)

#### 10. workflow_tcr_basic.go - Basic TCR Workflow
**Dependencies:** All domain slices

Contains:
- Basic TCR workflow definition
- Bootstrap → Execute → Test → Commit/Revert → Teardown
- Simple test-commit-revert logic

Consolidates:
- `workflows_tcr.go` (128 lines)

#### 11. workflow_tcr_enhanced.go - Enhanced TCR Workflow
**Dependencies:** All domain slices

Contains:
- 6-Gate Enhanced TCR workflow
- File locking integration
- Retry budget management
- Gate executor pattern

Consolidates:
- `workflows_enhanced.go` (229 lines)
- `activities_enhanced.go` (494 lines)
- Enhanced TCR logic

#### 12. workflow_dag.go - DAG Workflow
**Dependencies:** types.go, config.go, test_execution.go

Contains:
- DAG workflow definition
- Task dependency resolution
- Topological sorting
- Parallel execution

Consolidates:
- `workflows_dag.go` (253 lines)
- DAG execution logic

### Cleanup Layer (Depends on All)

#### 13. Delete Old Files and Update Tests
**Dependencies:** All refactored slices

Tasks:
- Remove old layered files
- Update import paths in tests
- Update main workflow registrations
- Update documentation
- Run full test suite
- Verify no regressions

## Migration Strategy

### Phase 1: Foundation (COMPLETED)
✅ Create `slices/` directory
✅ Implement `types.go`
✅ Implement `config.go`
✅ Implement `observability.go`

### Phase 2: Domain Slices (PENDING)
1. Implement `test_execution.go`
2. Implement `linting.go`
3. Implement `code_review.go`
4. Implement `code_generation.go`
5. Implement `cell_lifecycle.go`
6. Implement `file_locks.go`

### Phase 3: Workflows (PENDING)
1. Implement `workflow_tcr_basic.go`
2. Implement `workflow_tcr_enhanced.go`
3. Implement `workflow_dag.go`

### Phase 4: Cleanup (PENDING)
1. Update all imports
2. Remove old files
3. Update tests
4. Update documentation
5. Run full regression suite

## Benefits of Vertical Slices

### Developer Experience
- **Easier to understand**: All code for a feature in one place
- **Faster changes**: Modify one file instead of many
- **Better testing**: Test complete features, not layers
- **Clear ownership**: Each slice has clear responsibility

### Code Quality
- **Less coupling**: Slices depend on abstractions, not implementations
- **Better cohesion**: Related code lives together
- **Easier refactoring**: Changes localized to single slice
- **Simpler testing**: Mock dependencies, test in isolation

### Team Productivity
- **Parallel development**: Different slices can be worked on simultaneously
- **Clear boundaries**: Less merge conflicts
- **Easier onboarding**: New developers understand features, not layers
- **Better documentation**: Each slice is self-documenting

## Example: Before vs After

### Before (Layered)
To add a new gate to Enhanced TCR:
1. Add types to `types_enhanced.go`
2. Add activity to `activities_enhanced.go`
3. Update workflow in `workflows_enhanced.go`
4. Add parsing to `orchestrator_utils.go`
5. Update metrics in `metrics.go`
6. Update tests across multiple files

**Result:** 6 files modified, scattered logic, hard to review

### After (Vertical Slices)
To add a new gate to Enhanced TCR:
1. Add gate logic to appropriate slice (e.g., `linting.go`)
2. Update workflow in `workflow_tcr_enhanced.go`
3. Update tests in same file

**Result:** 2 files modified, cohesive logic, easy to review

## Testing Strategy

Each slice should have:
- **Unit tests**: Test activities in isolation
- **Integration tests**: Test complete capability
- **Workflow tests**: Test workflow integration

Example for `test_execution.go`:
```go
func TestExecuteTests(t *testing.T) { ... }           // Unit test
func TestTestExecutionSlice(t *testing.T) { ... }    // Integration test
func TestTCRWithTestGate(t *testing.T) { ... }       // Workflow test
```

## Documentation Updates

After refactoring:
1. Update `ARCHITECTURE.md` to reflect vertical slices
2. Update `CONTRIBUTING.md` with new structure
3. Add slice-specific documentation to each file
4. Update workflow diagrams
5. Create migration guide for existing code

## Success Criteria

The refactoring is successful when:
- ✅ All types consolidated in `types.go`
- ✅ All config in `config.go`
- ✅ All observability in `observability.go`
- ⏳ Each domain capability in its own slice
- ⏳ Each workflow in its own file
- ⏳ All tests passing
- ⏳ No duplicate code
- ⏳ Documentation updated
- ⏳ Code review approved

## Risks and Mitigations

### Risk: Breaking Changes
**Mitigation:** Keep old files until all tests pass, then remove

### Risk: Import Cycles
**Mitigation:** Clear dependency hierarchy, foundation → slices → workflows

### Risk: Lost Functionality
**Mitigation:** Comprehensive test suite, line-by-line comparison

### Risk: Team Confusion
**Mitigation:** Clear documentation, migration guide, code examples

## Timeline

- **Phase 1 (Foundation):** ✅ COMPLETED
- **Phase 2 (Domain Slices):** 2-3 days (6 files)
- **Phase 3 (Workflows):** 1-2 days (3 files)
- **Phase 4 (Cleanup):** 1 day (tests, docs, verification)

**Total:** 4-6 days for complete refactoring

## Conclusion

Vertical Slice Architecture will make Open Swarm's codebase:
- Easier to understand
- Faster to modify
- Simpler to test
- Better organized
- More maintainable

The foundation is now in place. Next steps are to implement the domain slices following the same pattern.

---

**Document Version:** 1.0
**Last Updated:** December 13, 2025
**Status:** IN PROGRESS
**Epic:** open-swarm-y1ur
