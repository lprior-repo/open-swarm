# Branch Kill Switch Logging Implementation

## Overview

This document summarizes the comprehensive logging implementation for all branch kill switch events in the Open Swarm merge queue coordinator. The logging uses Go's standard `log/slog` structured logging library for consistent, production-ready logging across all kill switch operations.

## Files Modified

### 1. `internal/mergequeue/kill_switch.go`

#### Import Addition
- Added `"log/slog"` to imports for structured logging capability

#### Function: `killFailedBranchWithTimeout`
**Location:** Lines 82-210

**Logging Events Added:**
- **Kill Initiated (INFO):** Logs when kill switch is triggered
  - Fields: `branch_id`, `reason`, `timeout`

- **Branch Not Found (ERROR):** Logs when branch cannot be located
  - Fields: `branch_id`, `error`

- **Already Killed (DEBUG):** Logs idempotent kill attempts
  - Fields: `branch_id`, `killed_at`, `original_reason`

- **Cleanup Begin (DEBUG):** Logs before cleanup operations
  - Fields: `branch_id`, `current_status`, `changes_count`, `children_count`

- **Kill Success (INFO):** Logs successful kill completion
  - Fields: `branch_id`, `reason`, `killed_at`, `total_kills`

- **Kill Failed (ERROR):** Logs operation failures with metrics
  - Fields: `branch_id`, `reason`, `error`, `duration_ms`

- **Timeout (WARN):** Logs graceful degradation when cleanup times out
  - Fields: `branch_id`, `reason`, `timeout_duration`, `actual_duration_ms`, `total_kills`, `status`

**Log Levels:**
- `INFO`: Critical kill events (initiation, success)
- `WARN`: Timeout with graceful degradation
- `ERROR`: Failures and exceptions
- `DEBUG`: Intermediate steps and idempotent operations

**Performance Metrics:**
- `duration_ms`: Time taken for kill operation
- All timing measurements use `time.Since(startTime)`

---

#### Function: `killDependentBranchesWithTimeout`
**Location:** Lines 270-303

**Logging Events Added:**
- **Cascade Initiated (INFO):** Logs cascade kill start
  - Fields: `branch_id`, `cascade_timeout`, `individual_kill_timeout`

- **Cascade Completed (INFO or ERROR):** Logs final cascade result
  - Fields: `branch_id`, `cascade_timeout`, `error`, `duration_ms`

**Log Levels:**
- `INFO`: Successful cascade completion
- `ERROR`: Cascade completed with errors

---

#### Function: `killDependentBranchesRecursive`
**Location:** Lines 224-281 (with comprehensive error handling types)

**Logging Events Added:**
- **Cascade Timed Out (ERROR):** Logs timeout during cascade
  - Fields: `branch_id`, `error`

- **Branch Not Found (ERROR):** Logs missing branch in cascade
  - Fields: `branch_id`, `error`

- **Cascade Start (INFO):** Logs cascade beginning
  - Fields: `branch_id`, `dependent_branches`, `children_ids`

- **Cascade Timeout During Processing (WARN):** Logs timeout during child processing
  - Fields: `parent_branch_id`, `current_child_id`, `children_processed`, `total_children`

- **Processing Dependent Branch (DEBUG):** Logs each child being processed
  - Fields: `parent_branch_id`, `child_branch_id`, `position`, `total`, `children_remaining`

- **Failed Cascade Kill (ERROR):** Logs descendant cascade failures
  - Fields: `child_branch_id`, `parent_branch_id`, `error`

- **Failed Dependent Kill (ERROR):** Logs individual branch kill failures
  - Fields: `child_branch_id`, `parent_branch_id`, `error`, `kill_reason`

- **Successfully Killed Branch (DEBUG):** Logs each successful kill
  - Fields: `child_branch_id`, `parent_branch_id`, `killed_count`, `total_killed`

- **Cascade Completed (INFO, ERROR, or DEBUG):** Final cascade result
  - Fields: `branch_id`, `total_children`, `successfully_killed`, `failed_kills`, `first_error`, `duration_ms`

---

### 2. `internal/mergequeue/coordinator.go`

#### Import Addition
- Added `"log/slog"` to imports for structured logging capability

#### Function: `killFailedBranch`
**Location:** Lines 335-443

**Logging Events Added:**
- **Kill Initiated (INFO):** Basic kill initiation
  - Fields: `branch_id`, `reason`

- **Branch Not Found (ERROR):** When branch doesn't exist
  - Fields: `branch_id`, `error`

- **Already Killed (DEBUG):** For idempotent attempts
  - Fields: `branch_id`, `previous_reason`

- **Cleanup Begin (DEBUG):** Before worktree cleanup
  - Fields: `branch_id`, `current_status`, `changes_count`, `depth`

- **Cleanup Warning (WARN):** If cleanup has errors
  - Fields: `branch_id`, `error`

- **Cleanup Complete (DEBUG):** Successful worktree cleanup
  - Fields: `branch_id`

- **Kill Completed (INFO):** Main kill completion
  - Fields: `branch_id`, `reason`, `killed_at`, `total_kills`, `duration_ms`

- **Notification Failed (WARN):** If agent notification fails
  - Fields: `branch_id`, `error`, `notification_duration_ms`

- **Notification Success (DEBUG):** Successful agent notification
  - Fields: `branch_id`, `notified_agents`, `notification_duration_ms`

---

#### Function: `killDependentBranches`
**Location:** Lines 446-533

**Logging Events Added:**
- **Cascade Initiated (INFO):** Cascade kill start
  - Fields: `parent_branch_id`

- **Branch Not Found (ERROR):** For missing parent branch
  - Fields: `branch_id`, `error`

- **Processing Dependent (DEBUG):** Each child being processed
  - Fields: `parent_branch_id`, `dependent_branches`, `children_ids`

- **Processing Child (DEBUG):** Individual child processing
  - Fields: `parent_branch_id`, `child_branch_id`, `position`, `total_children`

- **Failed Cascade (ERROR):** Descendant cascade failures
  - Fields: `child_branch_id`, `parent_branch_id`, `error`

- **Failed Kill (ERROR):** Individual branch kill failures
  - Fields: `child_branch_id`, `parent_branch_id`, `error`, `kill_reason`

- **Successfully Killed (DEBUG):** Each successful kill
  - Fields: `child_branch_id`, `parent_branch_id`, `killed_count`, `total_killed`

- **Cascade Complete (INFO, WARN, or DEBUG):** Final result
  - Fields: `parent_branch_id`, `total_children`, `successfully_killed`, `failed_kills`, `duration_ms`

---

## Structured Logging Fields

### Common Fields Across All Functions

| Field | Type | Description |
|-------|------|-------------|
| `branch_id` | string | ID of the branch being killed |
| `parent_branch_id` | string | ID of parent branch (in cascade operations) |
| `child_branch_id` | string | ID of child branch (in cascade operations) |
| `reason` | string | Reason for killing the branch |
| `kill_reason` | string | Specific reason for individual kills in cascade |
| `error` | string | Error message if operation failed |
| `duration_ms` | int64 | Operation duration in milliseconds |
| `timeout` | duration | Kill switch timeout configuration |
| `timeout_duration` | duration | Configured timeout value |
| `actual_duration_ms` | int64 | Actual time taken before timeout |
| `status` | string | Current status (e.g., "marked_as_killed_despite_timeout") |

### Cascade-Specific Fields

| Field | Type | Description |
|-------|------|-------------|
| `dependent_branches` | int | Number of dependent branches |
| `children_ids` | []string | List of child branch IDs |
| `total_children` | int | Total number of children |
| `children_processed` | int | Number of children processed before timeout |
| `children_remaining` | int | Number of children yet to process |
| `successfully_killed` | int | Number of successfully killed branches |
| `failed_kills` | int | Number of failed kills |
| `killed_count` | int | Running count of killed branches |
| `position` | int | Position in processing (1-based) |
| `cascade_timeout` | duration | Timeout for entire cascade |
| `individual_kill_timeout` | duration | Timeout per individual kill |
| `killed_at` | time.Time | Timestamp when branch was killed |
| `original_reason` | string | Original kill reason (for idempotent attempts) |
| `current_status` | BranchStatus | Branch status before kill |
| `changes_count` | int | Number of changes in branch |
| `depth` | int | Branch depth in speculation tree |
| `notified_agents` | int | Number of agents notified |
| `notification_duration_ms` | int64 | Time taken to send notifications |
| `total_kills` | int64 | Running total of all kills |
| `first_error` | string | First error encountered in cascade |

---

## Log Level Strategy

### ERROR (Highest Priority)
- Branch not found
- Kill operation failures
- Cascade failures with no partial progress
- Notification failures (but operation continues)
- Timeout exceptions

### WARN (High Priority)
- Timeout with graceful degradation
- Cleanup errors that don't prevent operation
- Cascade completion with partial failures

### INFO (Medium Priority)
- Kill initiation
- Kill completion
- Cascade completion with success
- Major state transitions

### DEBUG (Lowest Priority)
- Idempotent kill attempts
- Cleanup begin/end
- Individual child processing
- Each kill in cascade
- Successful notifications

---

## Usage Examples

### Monitoring Successful Kills
```
{"level":"info","msg":"Kill switch completed successfully","branch_id":"branch-123","reason":"tests failed: timeout","killed_at":"2025-12-13T10:30:45Z","total_kills":42,"duration_ms":125}
```

### Detecting Timeouts
```
{"level":"warn","msg":"Kill switch timed out (graceful degradation applied)","branch_id":"branch-123","timeout_duration":"500ms","actual_duration_ms":501,"status":"marked_as_killed_despite_timeout"}
```

### Tracking Cascade Operations
```
{"level":"info","msg":"Starting cascade kill for branch","branch_id":"parent-1","dependent_branches":5,"children_ids":["child-1","child-2","child-3","child-4","child-5"]}
{"level":"info","msg":"Cascade kill completed successfully","branch_id":"parent-1","dependent_branches_killed":5,"duration_ms":2150}
```

---

## Performance Metrics

All major operations now include:
- **Timing measurements**: Duration in milliseconds for kill and cascade operations
- **Progress tracking**: Count of successfully killed branches vs. failures
- **Throughput**: Number of branches killed per operation
- **Timeout tracking**: Actual vs. configured timeouts

This enables:
- SLA monitoring for kill operations
- Performance analysis of cascade operations
- Bottleneck identification in cleanup procedures
- Cascade operation depth analysis

---

## Configuration Integration

The logging respects the existing `slog.Default()` configuration, which means:
- Log format can be configured via environment variables
- Supports both JSON and text output formats
- Integrates with standard Go logging ecosystem
- Works with log aggregation systems (ELK, Splunk, etc.)

To enable JSON logs:
```bash
LOG_FORMAT=json go run ./cmd/...
```

---

## Best Practices for Log Analysis

### Identifying Kill Failures
```bash
grep '"level":"error"' logs.json | grep '"msg":"Kill switch failed"'
```

### Analyzing Cascade Operations
```bash
grep '"msg":".*cascade' logs.json | jq '.branch_id, .dependent_branches_killed'
```

### Tracking Timeout Patterns
```bash
grep '"level":"warn"' logs.json | grep "timeout" | jq '.branch_id, .actual_duration_ms'
```

### Performance Monitoring
```bash
grep '"msg":".*completed"' logs.json | jq '.duration_ms' | awk '{sum+=$1; count++} END {print "Average:", sum/count "ms"}'
```

---

## Future Enhancements

1. **Metrics Export**: Integrate with Prometheus for metrics collection
2. **Distributed Tracing**: Add trace IDs for correlation across systems
3. **Alerts**: Configure alerts for repeated failures or timeout patterns
4. **SLA Tracking**: Monitor kill operation latencies against SLAs
5. **Cascade Analysis**: Deep-dive logging for very deep hierarchies

