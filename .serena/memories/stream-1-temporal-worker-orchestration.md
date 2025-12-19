# Stream 1: Temporal Worker Foundation (Slice A)

## Task Summary
**Primary Task**: open-swarm-a1je - Temporal Worker Initialization
**Slice**: A (foundational)
**Agent**: Agent-1-Coordinator
**Status**: IN_PROGRESS

## Objective
Implement `internal/temporal/worker.go` - Temporal client and worker with activity/workflow registration.

## Architecture Design
```go
type TemporalWorker struct {
  client   client.Client
  worker   worker.Worker
  opts     WorkerOptions
}

// Core Methods
func NewTemporalWorker(ctx context.Context, opts WorkerOptions) (*TemporalWorker, error)
func (w *TemporalWorker) Start(ctx context.Context) error
func (w *TemporalWorker) Stop(ctx context.Context) error
func (w *TemporalWorker) RegisterActivity(activity interface{})
func (w *TemporalWorker) RegisterWorkflow(workflow interface{})
```

## Parallel Reviews (5 Lens Review Protocol)
After implementation completes, 5 reviewers examine independently:
- open-swarm-gl09: Lens 1 - Type Safety
- open-swarm-ra8z: Lens 2 - Error Paths
- open-swarm-lo13: Lens 3 - Edge Cases
- open-swarm-o0ig: Lens 4 - Integration
- open-swarm-bnp4: Lens 5 - Simplicity

## TDD/TCR Workflow
1. **RED**: Write failing tests for worker functionality
2. **GREEN**: Minimal implementation to pass tests
3. **BLUE**: Refactor without behavior change
4. **VERIFY**: All 5 lenses pass review

## Success Criteria
✓ Temporal client connects to server
✓ Worker registers successfully
✓ Activity/workflow registration works
✓ Lifecycle methods (Start/Stop) are idempotent
✓ All tests pass
✓ All 5 lenses pass review

## Dependencies
- Temporal SDK (already in go.mod)
- Factory components (testrunner, analyzer, generator - already complete)
- Beads SDK integration

## Next Steps After Completion
1. All 5 reviewers sign off
2. Commit to main
3. Trigger Slice B (Agent Spawning Workflow) - open-swarm-bn6v
