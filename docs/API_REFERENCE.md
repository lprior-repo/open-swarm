# Workflow Cancellation API Reference

## Core Types

### CancellationMode

```go
type CancellationMode string

const (
    // CancellationModeGraceful gracefully cancels the workflow,
    // allowing it to clean up resources and complete pending activities.
    CancellationModeGraceful CancellationMode = "graceful"

    // CancellationModeForce forcefully terminates the workflow,
    // bypassing graceful shutdown mechanisms.
    CancellationModeForce CancellationMode = "force"
)
```

### CancellationStatus

Represents the result of a workflow cancellation attempt.

```go
type CancellationStatus struct {
    // WorkflowID is the ID of the cancelled workflow
    WorkflowID string

    // Success indicates whether the cancellation completed successfully
    Success bool

    // Mode indicates which cancellation mode was used
    Mode CancellationMode

    // Duration is the time taken to complete the cancellation
    Duration time.Duration

    // Error contains any error that occurred during cancellation
    Error error

    // Message provides additional context about the cancellation result
    Message string

    // CancelledAt is the timestamp when the cancellation was requested
    CancelledAt time.Time

    // CompletedAt is the timestamp when the cancellation completed
    CompletedAt *time.Time

    // ResourcesFreed tracks which resources were freed during cancellation
    ResourcesFreed []string
}
```

## WorkflowCanceller Interface

Complete interface for workflow cancellation operations.

### Methods

#### CancelWorkflow

```go
func (wc WorkflowCanceller) CancelWorkflow(
    ctx context.Context,
    workflowID string,
    mode CancellationMode,
) (*CancellationStatus, error)
```

Cancels a workflow with the specified mode.

**Parameters:**
- `ctx`: Context with optional deadline
- `workflowID`: ID of the workflow to cancel
- `mode`: Cancellation mode (graceful or force)

**Returns:**
- `*CancellationStatus`: Detailed status of the cancellation
- `error`: Non-nil if validation fails (e.g., empty workflowID)

**Behavior:**
- Validates workflowID is not empty
- Selects timeout based on mode
- Tracks pending cancellation
- Caches result status
- Returns comprehensive status object

**Example:**
```go
status, err := canceller.CancelWorkflow(
    context.Background(),
    "workflow-123",
    CancellationModeGraceful,
)
if err != nil {
    log.Printf("Validation failed: %v", err)
    return
}
if !status.Success {
    log.Printf("Cancellation failed: %v", status.Error)
}
```

#### CancelWorkflowGraceful

```go
func (wc WorkflowCanceller) CancelWorkflowGraceful(
    ctx context.Context,
    workflowID string,
) (*CancellationStatus, error)
```

Convenience method for graceful cancellation.

**Equivalent to:**
```go
CancelWorkflow(ctx, workflowID, CancellationModeGraceful)
```

**Use when:** You want normal/expected workflow shutdown with cleanup.

**Example:**
```go
status, err := canceller.CancelWorkflowGraceful(ctx, workflowID)
if err != nil {
    log.Fatal(err)
}
if status.Success {
    log.Printf("Workflow cancelled in %v", status.Duration)
}
```

#### CancelWorkflowForce

```go
func (wc WorkflowCanceller) CancelWorkflowForce(
    ctx context.Context,
    workflowID string,
) (*CancellationStatus, error)
```

Convenience method for force cancellation.

**Equivalent to:**
```go
CancelWorkflow(ctx, workflowID, CancellationModeForce)
```

**Use when:** You need immediate termination (emergency, stuck workflow).

**Example:**
```go
status, err := canceller.CancelWorkflowForce(ctx, workflowID)
if err != nil {
    return err
}
```

#### CancelWorkflows

```go
func (wc WorkflowCanceller) CancelWorkflows(
    ctx context.Context,
    workflowIDs []string,
    mode CancellationMode,
) map[string]*CancellationStatus
```

Cancels multiple workflows concurrently with rate limiting.

**Parameters:**
- `ctx`: Context for cancellation signal
- `workflowIDs`: List of workflow IDs to cancel
- `mode`: Cancellation mode applied to all

**Returns:**
- Map of workflowID → CancellationStatus

**Behavior:**
- Limits concurrent requests (default: 10)
- Returns status for each workflow
- Never fails, returns best-effort results
- Can timeout partial workflow cancellations

**Example:**
```go
ids := []string{"wf1", "wf2", "wf3", "wf4", "wf5"}
results := canceller.CancelWorkflows(ctx, ids, CancellationModeGraceful)

for id, status := range results {
    if status.Success {
        log.Printf("Cancelled: %s", id)
    } else {
        log.Printf("Failed: %s - %v", id, status.Error)
    }
}
```

#### GetCancellationStatus

```go
func (wc WorkflowCanceller) GetCancellationStatus(
    workflowID string,
) *CancellationStatus
```

Retrieves status of a previous cancellation attempt.

**Parameters:**
- `workflowID`: ID of the workflow

**Returns:**
- `*CancellationStatus`: Status if found, nil if not

**Use Cases:**
- Audit trails
- Debugging failed cancellations
- Metrics collection
- Post-mortem analysis

**Example:**
```go
status := canceller.GetCancellationStatus(workflowID)
if status != nil {
    log.Printf("Last cancellation: mode=%s, success=%v, duration=%v",
        status.Mode, status.Success, status.Duration)
}
```

#### HasPendingCancellation

```go
func (wc WorkflowCanceller) HasPendingCancellation(
    workflowID string,
) bool
```

Checks if a cancellation is still in progress.

**Parameters:**
- `workflowID`: ID of the workflow

**Returns:**
- `true` if cancellation is being processed
- `false` if not pending or already completed

**Use Cases:**
- Prevent duplicate cancellation requests
- Monitor cancellation progress
- Implement retry logic

**Example:**
```go
if canceller.HasPendingCancellation(workflowID) {
    log.Printf("Cancellation already in progress, skipping")
    return
}

status, _ := canceller.CancelWorkflowGraceful(ctx, workflowID)
```

#### Clear

```go
func (wc WorkflowCanceller) Clear()
```

Removes all stored cancellation statuses and pending operations.

**Use Cases:**
- Shutdown/cleanup
- Reset for testing
- Memory cleanup in long-running systems

**Example:**
```go
defer func() {
    // Clean up on shutdown
    canceller.Clear()
}()
```

## DefaultWorkflowCanceller

Production implementation backed by Temporal client.

### Constructor

```go
func NewDefaultWorkflowCanceller(
    tc client.Client,
    gracefulTimeout time.Duration,
    forceTimeout time.Duration,
) *DefaultWorkflowCanceller
```

**Parameters:**
- `tc`: Temporal client.Client instance
- `gracefulTimeout`: How long to wait for graceful shutdown
- `forceTimeout`: How long to wait for force termination

**Returns:** Configured DefaultWorkflowCanceller instance

**Example:**
```go
canceller := NewDefaultWorkflowCanceller(
    temporalClient,
    500*time.Millisecond,  // graceful: 500ms
    250*time.Millisecond,  // force: 250ms
)
```

### Implementation Details

**Graceful Cancellation:**
1. Sends `CancelWorkflow` signal to Temporal
2. Workflow can react and clean up
3. Waits for completion up to gracefulTimeout
4. Returns success/failure status

**Force Cancellation:**
1. Sends `TerminateWorkflow` to Temporal
2. Workflow immediately terminated
3. Waits for termination up to forceTimeout
4. Returns success/failure status

**Concurrent Limits:**
- Max 10 concurrent cancellations
- Prevents overwhelming Temporal
- Queue model for excess requests

**Status Caching:**
- In-memory cache of recent statuses
- Bounded by number of unique workflows
- Cleared on explicit call or shutdown

## NoOpWorkflowCanceller

Test/fallback implementation that performs no actual cancellation.

### Constructor

```go
func NewNoOpWorkflowCanceller() *NoOpWorkflowCanceller
```

**Returns:** NoOpWorkflowCanceller instance

**Use Cases:**
- Unit testing
- Graceful degradation when Temporal unavailable
- Development/debugging

**Behavior:**
- All operations succeed
- No actual workflow cancellation occurs
- Returns valid but no-op status objects
- Thread-safe (no shared state)

**Example:**
```go
// For testing without Temporal
canceller := NewNoOpWorkflowCanceller()
coordinator.SetWorkflowCanceller(canceller)

// All cancellations succeed without effect
status, err := canceller.CancelWorkflowGraceful(ctx, "any-id")
// Returns: status.Success=true, err=nil
```

## Coordinator Integration

### Methods

#### SetTemporalClient

```go
func (c *Coordinator) SetTemporalClient(tc client.Client)
```

Sets the Temporal client and auto-creates DefaultWorkflowCanceller.

**Behavior:**
- Stores client in thread-safe registry
- Auto-creates default canceller if not already set
- Can be called multiple times (idempotent)

**Example:**
```go
tc, _ := client.Dial(clientOptions)
coordinator.SetTemporalClient(tc)
```

#### SetWorkflowCanceller

```go
func (c *Coordinator) SetWorkflowCanceller(canceller WorkflowCanceller)
```

Sets a custom WorkflowCanceller implementation.

**Use for:**
- Custom cancellation logic
- Testing with mocks
- Specialized implementations

**Example:**
```go
customCanceller := NewCustomCanceller()
coordinator.SetWorkflowCanceller(customCanceller)
```

#### CancelWorkflow

```go
func (c *Coordinator) CancelWorkflow(
    ctx context.Context,
    workflowID string,
    mode CancellationMode,
) (*CancellationStatus, error)
```

Cancels a workflow through the coordinator's canceller.

#### CancelWorkflowGraceful

```go
func (c *Coordinator) CancelWorkflowGraceful(
    ctx context.Context,
    workflowID string,
) (*CancellationStatus, error)
```

Gracefully cancels a workflow.

#### CancelWorkflowForce

```go
func (c *Coordinator) CancelWorkflowForce(
    ctx context.Context,
    workflowID string,
) (*CancellationStatus, error)
```

Forcefully cancels a workflow.

#### GetCancellationStatus

```go
func (c *Coordinator) GetCancellationStatus(
    workflowID string,
) *CancellationStatus
```

Retrieves cached cancellation status.

## Error Handling

### Common Error Scenarios

**Empty WorkflowID:**
```go
status, err := canceller.CancelWorkflow(ctx, "", mode)
// err: "workflowID cannot be empty"
// status: nil
```

**Invalid Mode:**
```go
status, err := canceller.CancelWorkflow(ctx, id, "invalid")
// err: "unknown cancellation mode: invalid"
// status: nil
```

**Timeout (during cancellation):**
```go
status, err := canceller.CancelWorkflowGraceful(ctx, id)
// err: nil
// status.Success: false
// status.Error: <error from Temporal>
```

**No Canceller Configured (Graceful Degradation):**
```go
coord := NewCoordinator(config)
// No SetTemporalClient() or SetWorkflowCanceller() call
status, err := coord.CancelWorkflowGraceful(ctx, id)
// status: nil
// err: nil
// Effect: Operation skipped, system continues normally
```

## Status Interpretation Guide

### Successful Cancellation
```go
status.Success == true
status.Error == nil
status.CompletedAt != nil
status.Duration > 0
```

### Failed Cancellation
```go
status.Success == false
status.Error != nil
status.Message // Contains error details
```

### Pending Cancellation
```go
canceller.HasPendingCancellation(workflowID) == true
// Status not yet available
```

### No Cancellation Attempted
```go
canceller.GetCancellationStatus(workflowID) == nil
// Either never attempted or cleared
```

## Timeout Values

### Recommended Configuration

**Normal/Development:**
```go
gracefulTimeout: 500*time.Millisecond
forceTimeout:    250*time.Millisecond
```

**High-Latency Deployments:**
```go
gracefulTimeout: 2*time.Second
forceTimeout:    1*time.Second
```

**Ultra-Low-Latency:**
```go
gracefulTimeout: 100*time.Millisecond
forceTimeout:    50*time.Millisecond
```

## Thread Safety

All public methods are thread-safe:
- Concurrent CancelWorkflow calls: safe
- Concurrent CancelWorkflows calls: safe
- Concurrent GetCancellationStatus calls: safe
- Concurrent HasPendingCancellation calls: safe
- Clear() during active cancellations: safe

No additional synchronization needed by callers.

## Resource Cleanup

### Status Cache Cleanup

For long-running systems, periodically clear the cache:

```go
// Option 1: Periodic cleanup
ticker := time.NewTicker(1*time.Hour)
go func() {
    for range ticker.C {
        canceller.Clear()
    }
}()

// Option 2: Manual cleanup on shutdown
defer canceller.Clear()
```

### Pending Cancellation Cleanup

Pending operations are automatically cleaned up:
- When cancellation completes (success/failure)
- When context is cancelled
- When Clear() is called

No manual cleanup needed.
