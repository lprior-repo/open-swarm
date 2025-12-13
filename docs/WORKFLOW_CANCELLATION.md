# Temporal Workflow Cancellation Interface

## Overview

The Workflow Cancellation Interface provides a clean, well-defined API for cancelling Temporal workflows in the context of branch kills within the speculative merge queue. It supports both graceful and force cancellation modes, with comprehensive status tracking and cleanup handling.

## Architecture

### Design Principles

1. **Clean API**: Simple, intuitive interface for workflow cancellation
2. **Graceful Degradation**: System continues functioning even if Temporal is unavailable
3. **Multiple Cancellation Modes**: Support both graceful shutdown and force termination
4. **Status Tracking**: Detailed reporting of cancellation results
5. **Resource Cleanup**: Automatic tracking of freed resources
6. **Thread Safety**: Safe concurrent cancellation of multiple workflows
7. **Testability**: Easy to mock and test

### Component Structure

```
┌─────────────────────────────────────────────────────────┐
│         Coordinator (Merge Queue)                       │
└────────────────────┬────────────────────────────────────┘
                     │
                     │ coordinates
                     ▼
┌─────────────────────────────────────────────────────────┐
│      WorkflowCanceller Interface                        │
│  ┌─────────────────────────────────────────────────┐   │
│  │ - CancelWorkflow()                              │   │
│  │ - CancelWorkflowGraceful()                      │   │
│  │ - CancelWorkflowForce()                         │   │
│  │ - CancelWorkflows() [batch]                     │   │
│  │ - GetCancellationStatus()                       │   │
│  │ - HasPendingCancellation()                      │   │
│  │ - Clear()                                       │   │
│  └─────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
         │                              │
         │ implements               implements
         ▼                              ▼
┌──────────────────────┐      ┌──────────────────────┐
│ DefaultWorkflow      │      │ NoOpWorkflow         │
│ Canceller            │      │ Canceller            │
│ (Temporal-backed)    │      │ (Testing/Fallback)   │
└──────────────────────┘      └──────────────────────┘
         │
         │ delegates to
         ▼
   Temporal Client
```

## Core Types

### CancellationMode

Defines how the workflow should be cancelled:

```go
type CancellationMode string

const (
    // Graceful cancellation allows the workflow to clean up
    // Active activities will be signalled to stop
    // Workflow hooks can execute cleanup logic
    CancellationModeGraceful CancellationMode = "graceful"

    // Force cancellation immediately terminates the workflow
    // No cleanup hooks are executed
    // Useful for emergency situations or stuck workflows
    CancellationModeForce CancellationMode = "force"
)
```

### CancellationStatus

Comprehensive result of a cancellation attempt:

```go
type CancellationStatus struct {
    WorkflowID     string        // ID of cancelled workflow
    Success        bool          // Whether cancellation succeeded
    Mode           CancellationMode
    Duration       time.Duration // Time taken to complete cancellation
    Error          error         // Error if cancellation failed
    Message        string        // Human-readable description
    CancelledAt    time.Time     // When cancellation was requested
    CompletedAt    *time.Time    // When cancellation completed
    ResourcesFreed []string      // List of freed resources
}
```

### WorkflowCanceller Interface

The core interface for workflow cancellation:

```go
type WorkflowCanceller interface {
    // Cancel with specified mode
    CancelWorkflow(ctx context.Context, workflowID string, mode CancellationMode) 
        (*CancellationStatus, error)

    // Convenience methods
    CancelWorkflowGraceful(ctx context.Context, workflowID string) 
        (*CancellationStatus, error)
    CancelWorkflowForce(ctx context.Context, workflowID string) 
        (*CancellationStatus, error)

    // Batch cancellation (concurrent, rate-limited)
    CancelWorkflows(ctx context.Context, workflowIDs []string, mode CancellationMode) 
        map[string]*CancellationStatus

    // Query previous cancellation
    GetCancellationStatus(workflowID string) *CancellationStatus

    // Check if cancellation is in progress
    HasPendingCancellation(workflowID string) bool

    // Clear all cached data
    Clear()
}
```

## Implementation Details

### DefaultWorkflowCanceller

Production implementation backed by Temporal client:

```go
canceller := NewDefaultWorkflowCanceller(
    temporalClient,
    gracefulTimeout,  // How long to wait for graceful shutdown
    forceTimeout,     // How long to wait for force termination
)
```

**Features:**
- Thread-safe concurrent cancellations
- Rate limiting (max 10 concurrent by default)
- Status caching for audit/debugging
- Pending cancellation tracking
- Automatic timeout enforcement

**Graceful Flow:**
1. Send CancelWorkflow signal to Temporal
2. Workflow receives cancellation signal
3. Activities can react and clean up
4. Workflow completes execution
5. Status cached with success

**Force Flow:**
1. Send TerminateWorkflow to Temporal
2. Workflow execution immediately terminated
3. No cleanup hooks executed
4. Faster but less graceful
5. Status cached with termination

### NoOpWorkflowCanceller

Test/fallback implementation that does nothing:

```go
canceller := NewNoOpWorkflowCanceller()
```

**Use Cases:**
- Testing (when Temporal unavailable)
- Graceful degradation (don't fail merge queue if Temporal down)
- Development/debugging

## Integration with Branch Kill Switch

### Usage in Kill Operations

When a branch is marked as killed:

```go
// In killFailedBranchWithTimeout()
if branch.WorkflowID != "" {
    // Attempt graceful cancellation first
    status, err := c.CancelWorkflowGraceful(ctx, branch.WorkflowID)
    if err != nil {
        // Fallback to force cancellation
        forceStatus, forceErr := c.CancelWorkflowForce(ctx, branch.WorkflowID)
        if forceErr != nil {
            logger.Error("Failed to cancel workflow", "error", forceErr)
        }
    }
}
```

### Status Tracking for Metrics

Cancellation statuses are cached and can be used for:
- Audit trails
- Debugging failed cancellations
- Metrics collection
- Performance analysis

```go
status := coordinator.GetCancellationStatus(workflowID)
if status != nil && status.Success {
    metrics.RecordCancellationDuration(status.Duration)
}
```

## Usage Examples

### Basic Graceful Cancellation

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

status, err := canceller.CancelWorkflowGraceful(ctx, workflowID)
if err != nil {
    log.Printf("Cancellation failed: %v", err)
    return
}

if status.Success {
    log.Printf("Workflow cancelled gracefully in %v", status.Duration)
} else {
    log.Printf("Cancellation failed: %v", status.Error)
}
```

### Fallback to Force Cancellation

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

// Try graceful first
status, _ := canceller.CancelWorkflowGraceful(ctx, workflowID)
if status != nil && status.Success {
    return status
}

// Fall back to force
forceStatus, err := canceller.CancelWorkflowForce(ctx, workflowID)
if err != nil {
    log.Printf("Force cancellation also failed: %v", err)
}
return forceStatus
```

### Batch Cancellation

```go
ctx := context.Background()
workflowIDs := []string{"wf1", "wf2", "wf3"}

results := canceller.CancelWorkflows(ctx, workflowIDs, CancellationModeGraceful)

for id, status := range results {
    if status.Success {
        log.Printf("Workflow %s cancelled", id)
    } else {
        log.Printf("Workflow %s cancellation failed: %v", id, status.Error)
    }
}
```

### Checking Cancellation Status

```go
// Get status of previous cancellation
status := canceller.GetCancellationStatus(workflowID)
if status != nil {
    log.Printf("Last cancellation: %s (success=%v)", 
        status.Mode, status.Success)
    log.Printf("Duration: %v", status.Duration)
    log.Printf("Freed: %v", status.ResourcesFreed)
}

// Check if still pending
if canceller.HasPendingCancellation(workflowID) {
    log.Printf("Cancellation still in progress")
}
```

### Setting Coordinator Integration

```go
// Create coordinator
coord := NewCoordinator(config)

// Create Temporal client
tc, err := client.Dial(clientOptions)
if err != nil {
    log.Fatal(err)
}

// Set Temporal client (auto-creates default canceller)
coord.SetTemporalClient(tc)

// Or create custom canceller
canceller := NewDefaultWorkflowCanceller(
    tc,
    500*time.Millisecond,   // graceful timeout
    250*time.Millisecond,   // force timeout
)
coord.SetWorkflowCanceller(canceller)
```

## Timeout Behavior

### Graceful Timeout

The graceful timeout controls how long to wait for the workflow to respond to the cancellation signal:

- **Default**: `KillSwitchTimeout` (typically 500ms)
- **Behavior**: If workflow doesn't complete in time, subsequent operations may fail
- **Use when**: Normal/expected cancellations

### Force Timeout

The force timeout controls how long to wait for the workflow termination:

- **Default**: `KillSwitchTimeout / 2` (typically 250ms)
- **Behavior**: If force termination doesn't complete, error is returned
- **Use when**: Emergency cancellation needed

## Graceful Degradation

If no `WorkflowCanceller` is configured:

```go
status, err := c.CancelWorkflowGraceful(ctx, workflowID)
// Returns:
// status: nil
// err: nil
// Effect: System continues normally (no error, just no cancellation)
```

This allows the merge queue to function even if Temporal is unavailable.

## Testing

### Using NoOpWorkflowCanceller

```go
func TestBranchKill(t *testing.T) {
    coord := NewCoordinator(config)
    
    // Use no-op canceller for testing
    coord.SetWorkflowCanceller(NewNoOpWorkflowCanceller())
    
    // Test kill operations without Temporal
    err := coord.killFailedBranch(ctx, branchID, "test failure")
    assert.NoError(t, err)
}
```

### Mocking for Unit Tests

```go
func TestCancellationLogic(t *testing.T) {
    mockClient := &mocks.Client{}
    mockClient.On("CancelWorkflow", mock.Anything, "wf1", "").
        Return(nil)
    
    canceller := NewDefaultWorkflowCanceller(mockClient, 500*time.Millisecond, 250*time.Millisecond)
    
    status, err := canceller.CancelWorkflowGraceful(context.Background(), "wf1")
    assert.NoError(t, err)
    assert.True(t, status.Success)
}
```

## Performance Considerations

### Cancellation Concurrency

The default implementation limits concurrent cancellations to 10:

```go
sem := make(chan struct{}, 10) // Max 10 concurrent
```

This prevents overwhelming Temporal with too many requests.

### Status Cache

Cancellation statuses are kept in memory. For long-running systems, consider:

```go
// Clear old statuses periodically
canceller.Clear()

// Or implement custom cache eviction
// (not currently built-in, but interface supports it)
```

### Timeout Tuning

For high-latency Temporal deployments:

```go
canceller := NewDefaultWorkflowCanceller(
    tc,
    2*time.Second,    // Longer graceful timeout
    1*time.Second,    // Longer force timeout
)
```

## Monitoring and Observability

### Key Metrics to Track

1. **Cancellation Success Rate**: `successful_cancellations / total_cancellations`
2. **Cancellation Duration**: Distribution of `status.Duration`
3. **Cancellation Failures**: Types and counts of failures
4. **Graceful vs Force**: Ratio of graceful to force cancellations
5. **Resource Cleanup**: Track `status.ResourcesFreed`

### Logging Integration

The implementation integrates with `log/slog`:

```go
logger := slog.Default()

logger.Debug("Graceful cancellation initiated",
    "workflow_id", workflowID,
    "branch_id", branchID,
)

logger.Error("Cancellation failed",
    "workflow_id", workflowID,
    "error", err.Error(),
)

logger.Info("Workflow cancelled",
    "workflow_id", workflowID,
    "duration_ms", status.Duration.Milliseconds(),
    "resources_freed", status.ResourcesFreed,
)
```

## Future Enhancements

### Planned Features

1. **Cancellation Hooks**: Allow workflows to register cleanup functions
2. **Cancellation Policies**: Per-workflow-type cancellation strategies
3. **Resource Tracking**: Detailed tracking of freed resources
4. **Retry Logic**: Automatic retry with exponential backoff
5. **Metrics Integration**: Built-in Prometheus metrics
6. **Cancellation Callbacks**: Notify when cancellation completes

### Backward Compatibility

The interface is designed to be stable. Future additions will:
- Add new methods to the interface
- Not change existing method signatures
- Maintain graceful degradation

## See Also

- [KILLSWITCH.md](KILLSWITCH.md) - Kill switch architecture
- [Temporal Workflow Cancellation](https://docs.temporal.io/workflows#cancellation) - Temporal documentation
- `internal/mergequeue/workflow_canceller.go` - Implementation
- `internal/mergequeue/coordinator_temporal.go` - Integration with coordinator
