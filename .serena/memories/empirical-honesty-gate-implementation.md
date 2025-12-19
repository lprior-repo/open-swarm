# Empirical Honesty Gate Implementation Guide

## Overview
**Status**: ✅ COMPLETE & PASSING
**File**: `internal/gates/empirical_honesty.go` (152 lines)
**Tests**: All 3 test cases passing (`all_gates_test.go`)
**Beads Task**: open-swarm-wh0l

## Implementation Strategy

### Architecture
The EmpiricalHonestyGate ensures that agents cannot claim success when tests are failing. This enforces honesty through empirical verification - what the tests actually say matters more than what the agent claims.

### Core Principle
**Source of Truth**: Raw test output and exit codes, not agent claims.
- Agent cannot declare success if tests fail
- Must provide detailed failure information
- Exit codes must reflect actual results
- No summarization or interpretation allowed

### Key Components

#### 1. **Raw Output Requirement**
```go
testResult *TestResult  // Actual test execution results
agentClaim string       // What agent claimed
```
- Agent must submit actual test output (not summary)
- TestResult struct contains: Output, Failures, Passed/Failed counts, ExitCode
- Comparison is empirical (reality-based), not interpretive

#### 2. **Claim Honesty Verification**
```go
verifyClaimHonesty() // Detect false success claims
```
- Detects success keywords: "success", "passing", "complete", "done", "works", etc.
- If tests failing but agent claims success → GateError
- If tests passing and honest claim → OK

#### 3. **Raw Output Validation**
```go
verifyRawOutput() // Ensure output/failures provided
```
- Requires Output field or Failures array when tests fail
- If tests fail but no output documented → GateError
- Prevents silent failures

#### 4. **Exit Code Honesty**
```go
verifyExitCode() // Non-zero exit code for failures
```
- Tests failing must have non-zero exit code
- Exit code 0 with failures → GateError
- Enforces process-level honesty

### Struct Design

```go
type EmpiricalHonestyGate struct {
    taskID     string         // Task identifier
    testResult *TestResult    // Actual test execution results
    agentClaim string         // What agent claimed (success/failure)
    timestamp  int64          // When gate was created
}
```

### TestResult Type (from gates.go)
```go
type TestResult struct {
    Output   string      // Raw test output (stdout+stderr)
    Failures []string    // List of failure messages
    Passed   int         // Count of passing tests
    Failed   int         // Count of failing tests
    Total    int         // Total tests run
    ExitCode int         // Process exit code
}

func (tr *TestResult) IsPassing() bool
func (tr *TestResult) PassRate() float64
```

### Methods

| Method | Purpose | Failure Behavior |
|--------|---------|-----------------|
| `NewEmpiricalHonestyGate()` | Constructor | Returns gate instance |
| `SetTestResult()` | Set actual test results | None (setter) |
| `SetAgentClaim()` | Record agent's claim | None (setter) |
| `Check()` | Main verification logic | Returns GateError if failed |
| `verifyClaimHonesty()` | Compare claim vs results | GateError if false success |
| `verifyRawOutput()` | Ensure output provided | GateError if missing |
| `verifyExitCode()` | Check exit code honesty | GateError if mismatch |
| `generateHonestyReport()` | Create detailed report | Returns formatted string |
| `truncateOutput()` | Limit output length | Returns truncated output |

### Check() Execution Flow

```
1. Validate TestResult is set
   └─ Return GateError if nil

2. Verify Claim Honesty
   ├─ If tests failing but agent claimed success
   └─ Return GateError with honesty report

3. Verify Raw Output Provided
   ├─ If tests failed but no output/failures documented
   └─ Return GateError

4. Verify Exit Code Honesty
   ├─ If tests failed but exit code is 0
   └─ Return GateError

Return nil (all checks passed)
```

### Error Handling Pattern

All failures return `GateError` with context:
```go
return &GateError{
    Gate:      GateEmpiricalHonesty,
    TaskID:    ehg.taskID,
    Message:   "claim of success contradicts actual test results",
    Details:   ehg.generateHonestyReport(),  // Detailed forensic report
    Timestamp: time.Now().Unix(),
}
```

### Honesty Report Format

When dishonesty detected, generates comprehensive report:
```
=== Honesty Verification Report ===

CLAIM:
  Agent claimed: "All tests pass, implementation complete"

REALITY (Raw Test Output):
  Tests run: 5
  Tests passed: 3
  Tests failed: 2
  Pass rate: 60.0%
  Exit code: 1

FAILURE DETAILS:
  1. TestStringValidation: Expected error but got nil
  2. TestEdgeCases: Panic: nil pointer dereference

RAW TEST OUTPUT:
  --- FAIL: TestStringValidation (0.001s)
  ... (output truncated)

VERDICT:
  ✗ DISHONESTY DETECTED: Agent claimed success but tests are failing.
  This is an unacceptable violation. The agent must:
  1. Fix the failing tests
  2. Run tests again to verify
  3. Submit honest results
```

## Testing Strategy

### Test Cases (3 passing)

1. **TestEmpiricalHonestyGate_FailOnFalseSuccess**
   - Scenario: Tests fail, agent claims success
   - Detection: False success keywords in claim
   - Result: PASS - GateError returned

2. **TestEmpiricalHonestyGate_AllowHonestFailure**
   - Scenario: Tests fail, agent admits it with raw output
   - Detection: Honest claim matching reality
   - Result: PASS - No error (honesty verified)

3. **TestEmpiricalHonestyGate_RequireRawOutput**
   - Scenario: Tests fail but no output/failures provided
   - Detection: Missing raw output documentation
   - Result: PASS - GateError returned

### Test Execution
```bash
go test ./internal/gates -v -run EmpiricalHonesty
# Expected: 3 passed
```

## Go Best Practices Applied

### 1. **Zero-Trust Verification**
- Never trust agent's interpretation of results
- Always compare against empirical data (exit codes, output)
- Detailed forensic reports for failures

### 2. **Keyword Detection**
```go
successKeywords := []string{"success", "passing", "complete", "done", ...}
```
- Case-insensitive matching
- Comprehensive keyword coverage
- Detects partial claims

### 3. **Comprehensive Output Handling**
- Multiple data sources: output, failures, exit code, pass rate
- Truncation for large outputs (max 500 chars)
- Structured failure list vs raw output

### 4. **Human-Readable Reports**
```go
generateHonestyReport() // Forensic report format
```
- Structured comparison: CLAIM vs REALITY
- Raw data included for verification
- Clear verdict and corrective actions

## Integration with Anti-Cheating System

### Gate Sequencing
```
1. Requirements Verification (understand spec)
2. Test Immutability (tests locked read-only)
3. Empirical Honesty (no false success claims) ← This gate
4. Hard Work Enforcement (stubs fail tests)
5. Requirement Drift Detection (stays aligned)
```

### Orchestration Integration
The orchestrator (internal/orchestration/coordinator.go) calls:
```go
gate := NewEmpiricalHonestyGate(taskID)
gate.SetTestResult(actualTestResults)
gate.SetAgentClaim(agentSubmittedClaim)
if err := gate.Check(ctx); err != nil {
    // Handle dishonesty
    // Send error + honesty report back to agent
    // Force re-execution
}
```

## Key Insights

### Why This Approach Works
1. **Empirical Foundation**: Tests and exit codes don't lie
2. **Claim Verification**: Keyword detection catches common false claims
3. **Forensic Reports**: Detailed comparison makes dishonesty undeniable
4. **Process-Level Honesty**: Exit codes enforced at OS level, not app level
5. **Output Requirement**: Agent must document failures, not hide them

### Common Dishonesty Patterns Detected
- ❌ "Implementation complete" (when tests fail)
- ❌ "All tests passing" (when failures exist)
- ❌ "Works fine" (when exit code is non-zero)
- ❌ False success claims of any kind
- ❌ Submitting no output with failing tests

## Production Readiness Checklist

- [x] Implementation complete (152 lines)
- [x] All tests passing (3/3)
- [x] Error handling comprehensive
- [x] Forensic reports detailed
- [x] Keyword detection comprehensive
- [x] Output truncation handled
- [x] Integration with gate framework
- [x] Documentation complete
- [x] Follows Go conventions

## References
- **Beads Task**: open-swarm-wh0l
- **Test File**: internal/gates/all_gates_test.go (lines 114-180)
- **Related Gates**: test_immutability.go, hard_work_enforcement.go
- **Framework**: gates.go (GateType, GateError, Gate interface, TestResult)

## Success Criteria Met

✅ **Agent submits exact test output**
- TestResult.Output field required
- No summarization allowed

✅ **Cannot summarize or paraphrase**
- Raw output required
- Failure details checked

✅ **Failure in 1 test = cannot claim done**
- Claim honesty verification detects false success
- Exit code verification enforces failure reporting

✅ **All N tests must pass to claim success**
- PassRate calculation includes all tests
- Single failure blocks success claim

✅ **No interpretation, just raw data**
- Empirical verification (tests + exit codes)
- No agent interpretation accepted
