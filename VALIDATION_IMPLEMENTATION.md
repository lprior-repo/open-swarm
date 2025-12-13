# Branch Kill Switch Validation Implementation

## Overview

This document describes the comprehensive branch status validation system added to the kill switch operations in the merge queue coordinator.

## Components

### 1. Validation Module (`kill_switch_validation.go`)

Provides a complete validation framework for kill switch operations with the following validators:

#### Core Validators

- **`ValidateBranchExists()`** - Checks if a branch exists in the merge queue
- **`ValidateBranchNotProtected()`** - Prevents killing protected branches (main, master, release/*, hotfix/*, production/*)
- **`ValidateBranchStatus()`** - Ensures branch is in a valid state for killing (pending, testing, failed, passed)
- **`ValidateNoPendingWork()`** - Verifies no in-flight work (active workflows, running containers)
- **`ValidateOwnership()`** - Confirms agent authorization (system agents can kill any branch; regular agents can only kill their own)

#### Composite Validator

- **`ValidateFullKillSwitchPrerequisites()`** - Performs all validations in sequence with clear error reporting

#### Health Reporting

- **`GenerateHealthReport()`** - Creates detailed health reports for debugging and monitoring

### 2. Error Types (`BranchValidationError`)

Provides structured error information with:
- **Code**: Machine-readable error codes (e.g., `BRANCH_PROTECTED`, `PENDING_WORK`)
- **Message**: Human-readable error message
- **Details**: Actionable context information
- **Timestamp**: When validation failed

Error codes defined:
```go
ValidationCodeBranchNotFound      // Branch doesn't exist
ValidationCodeBranchProtected     // Protected branch (main/master)
ValidationCodePendingWork         // Has in-flight work
ValidationCodeInvalidStatus       // Invalid state for killing
ValidationCodeOwnershipMismatch   // Agent doesn't own branch
ValidationCodeValidationTimeout   // Validation took too long
```

### 3. Integration Functions

#### `KillFailedBranchWithValidation()`
- **Signature**: `KillFailedBranchWithValidation(ctx, branchID, reason, requestingAgent) *BranchValidationError`
- **Behavior**: Validates then kills a single branch with comprehensive pre-kill checks
- **Returns**: Nil on success; validation error if any check fails
- **Logging**: Structured logs via slog for all validation and kill events

#### `KillDependentBranchesWithValidation()`
- **Signature**: `KillDependentBranchesWithValidation(ctx, branchID, requestingAgent) (*BranchValidationError, error)`
- **Behavior**: Validates parent then kills all descendants
- **Returns**: Validation error first, then cascade error (allowing callers to distinguish failure types)
- **Protection**: Prevents cascade kills from protected branches

#### `GetBranchHealthReport()`
- **Signature**: `GetBranchHealthReport(branchID) *BranchHealthReport`
- **Behavior**: Generates detailed health status for branch
- **Use Cases**: Debugging, monitoring, UI display, logging context

## Validation Checks

### Protected Branches

Protected branch patterns prevent accidental kills:
- **Exact matches**: `main`, `master`, `develop` (configurable)
- **Prefix patterns**: `release/*`, `hotfix/*`, `production/*`
- **Custom patterns**: Can be added via `AddProtectedBranch()`

### Pending Work Detection

Checks for active resources that prevent killing:
- **Temporal Workflows**: Active workflow IDs indicate running tests
- **Docker Containers**: Active container IDs indicate resource cleanup in progress
- **Test Results**: Pending test results block kill operations
- **Status Checks**: Testing status with active resources prevents kill

### Ownership & Permissions

Fine-grained access control:
- **System Agents**: Can kill any branch (agents: system, admin, coordinator, merge-queue, automated-test)
- **Regular Agents**: Can only kill branches they created
- **Ownership Tracking**: Determined from first change in branch.Changes

### Idempotency

- **Already Killed Branches**: Validation passes; no state change
- **Safe Reruns**: Multiple kill attempts on same branch are safe
- **Metrics Protection**: TotalKills only increments for new kills

## Usage Examples

### Kill with validation

```go
err := coordinator.KillFailedBranchWithValidation(
    ctx, 
    "feature-branch-123", 
    "test timeout: 30s exceeded",
    "agent-id-456",
)

if err != nil {
    switch err.Code {
    case ValidationCodeBranchProtected:
        log.Warnf("Cannot kill protected branch: %s", err.Details)
    case ValidationCodePendingWork:
        log.Warnf("Branch has pending work: %s", err.Details)
    case ValidationCodeOwnershipMismatch:
        log.Errorf("Permission denied: %s", err.Details)
    default:
        log.Errorf("Kill failed: %s", err)
    }
}
```

### Cascade with validation

```go
validationErr, cascadeErr := coordinator.KillDependentBranchesWithValidation(
    ctx,
    "parent-branch-456",
    "agent-id-789",
)

if validationErr != nil {
    log.Errorf("Parent validation failed: %v", validationErr)
    return validationErr
}

if cascadeErr != nil {
    log.Warnf("Cascade had issues: %v (some children may be killed)", cascadeErr)
}
```

### Get health report

```go
report := coordinator.GetBranchHealthReport("branch-id")

if !report.CanBeKilled {
    fmt.Printf("Branch issues:\n")
    for _, issue := range report.ValidationIssues {
        fmt.Printf("  - %s\n", issue)
    }
} else {
    fmt.Printf("Branch is healthy (owner: %s, status: %s)\n", 
        report.Owner, report.Status)
}
```

## Testing

### Test Coverage

- **Unit Tests** (`kill_switch_validation_test.go`): 36 tests covering all validators
- **Integration Tests** (`kill_switch_validated_integration_test.go`): 14 tests with full coordinator setup
- **All validation tests pass**: 100% success rate

### Test Categories

1. **Individual Validators** - Each validation function tested in isolation
2. **Composite Validator** - Full prerequisite check with various failure scenarios
3. **Health Reports** - Report generation for different branch states
4. **Integration Scenarios** - Validation with actual kill operations
5. **Edge Cases** - Nil branches, system agents, already-killed branches

## Integration with Existing Code

### Backward Compatibility

- Original internal functions (`killFailedBranchWithTimeout`, `killDependentBranchesWithTimeout`) unchanged
- New public methods provide validation layer on top
- Existing code continues working without modification

### Complementary Error Handling

Works alongside existing error types:
- **KillSwitchError**: Operational errors during kill
- **CleanupError**: Resource cleanup failures
- **TimeoutError**: Operation timeout tracking
- **BranchValidationError**: Pre-kill validation failures (new)

### Logging Integration

Uses structured logging via `log/slog`:
```go
logger.Warn("Kill switch validation failed",
    "branch_id", branchID,
    "validation_code", err.Code,
    "validation_message", err.Message,
    "requesting_agent", requestingAgent,
)
```

## Benefits

1. **Clear Error Messages**: Users get actionable feedback on why kill failed
2. **Early Validation**: Prevents expensive kill operations on bad inputs
3. **Protection**: Prevents accidental kills of protected branches
4. **Observability**: Detailed logging of all validation decisions
5. **Flexibility**: Configurable protected branches and system agents
6. **Idempotency**: Safe to retry failed kill operations
7. **Testing**: Comprehensive test coverage ensures reliability

## Future Enhancements

Potential improvements for future versions:

1. **Rate Limiting**: Prevent rapid repeated kill attempts on same branch
2. **Audit Trail**: Track all kill attempts (successful and failed)
3. **Advanced Permissions**: Role-based access control
4. **Graceful Warnings**: Pre-kill notifications before cascade
5. **Retention Policy**: Keep killed branch metadata for retention period
6. **Analytics**: Track protection violations and unauthorized kill attempts

## File Locations

- **Validation Logic**: `/home/lewis/src/open-swarm/internal/mergequeue/kill_switch_validation.go`
- **Unit Tests**: `/home/lewis/src/open-swarm/internal/mergequeue/kill_switch_validation_test.go`
- **Integration Tests**: `/home/lewis/src/open-swarm/internal/mergequeue/kill_switch_validated_integration_test.go`
- **Kill Switch Functions**: `/home/lewis/src/open-swarm/internal/mergequeue/kill_switch.go` (appended)

## Deployment Notes

- **Breaking Changes**: None - all changes are backward compatible
- **Database Changes**: None - validation operates on in-memory state
- **Configuration Changes**: Can optionally add protected branches via `AddProtectedBranch()`
- **Performance**: Validation is O(n) where n = number of changes (typically <10)

