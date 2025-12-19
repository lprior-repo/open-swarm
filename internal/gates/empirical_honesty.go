package gates

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// EmpiricalHonestyGate ensures agents cannot claim success if tests are failing.
// Agents must submit raw test output and accept empirical results.
type EmpiricalHonestyGate struct {
	taskID     string
	testResult *TestResult
	agentClaim string // What the agent claimed (success/failure)
	timestamp  int64
}

// NewEmpiricalHonestyGate creates a new empirical honesty gate.
func NewEmpiricalHonestyGate(taskID string) *EmpiricalHonestyGate {
	return &EmpiricalHonestyGate{
		taskID:    taskID,
		timestamp: time.Now().Unix(),
	}
}

// SetTestResult sets the actual test execution results.
func (ehg *EmpiricalHonestyGate) SetTestResult(result *TestResult) {
	ehg.testResult = result
}

// SetAgentClaim sets what the agent claimed about test results.
func (ehg *EmpiricalHonestyGate) SetAgentClaim(claim string) {
	ehg.agentClaim = claim
}

// Type returns the gate type.
func (ehg *EmpiricalHonestyGate) Type() GateType {
	return GateEmpiricalHonesty
}

// Name returns the human-readable name.
func (ehg *EmpiricalHonestyGate) Name() string {
	return "Empirical Honesty"
}

// Check verifies that the agent's claims match the actual test results.
func (ehg *EmpiricalHonestyGate) Check(_ context.Context) error {
	// Validate inputs
	if ehg.testResult == nil {
		return &GateError{
			Gate:      ehg.Type(),
			TaskID:    ehg.taskID,
			Message:   "test result not set",
			Details:   "Cannot verify honesty without actual test results",
			Timestamp: time.Now().Unix(),
		}
	}

	// Check 1: Cannot claim success if tests are failing
	if err := ehg.verifyClaimHonesty(); err != nil {
		return err
	}

	// Check 2: Verify raw output is provided (not just summary)
	if err := ehg.verifyRawOutput(); err != nil {
		return err
	}

	// Check 3: Verify exit code honesty (non-zero for failures)
	if err := ehg.verifyExitCode(); err != nil {
		return err
	}

	return nil
}

// verifyClaimHonesty checks if the agent's claim matches actual test results.
func (ehg *EmpiricalHonestyGate) verifyClaimHonesty() error {
	if ehg.testResult.IsPassing() {
		return nil // Success claim is fine if tests pass
	}

	successKeywords := []string{"success", "passing", "complete", "done", "finished", "all tests pass", "works", "implemented"}
	claimLower := strings.ToLower(ehg.agentClaim)

	for _, keyword := range successKeywords {
		if strings.Contains(claimLower, keyword) {
			return &GateError{
				Gate:      ehg.Type(),
				TaskID:    ehg.taskID,
				Message:   "claim of success contradicts actual test results",
				Details:   ehg.generateHonestyReport(),
				Timestamp: time.Now().Unix(),
			}
		}
	}
	return nil
}

// verifyRawOutput ensures test output is provided, not just a summary.
func (ehg *EmpiricalHonestyGate) verifyRawOutput() error {
	hasOutput := ehg.testResult.Output != ""
	hasFailures := len(ehg.testResult.Failures) > 0

	// If tests failed but no output provided
	if !ehg.testResult.IsPassing() && !hasOutput && !hasFailures {
		return &GateError{
			Gate:      ehg.Type(),
			TaskID:    ehg.taskID,
			Message:   "test failures not documented",
			Details:   "When tests fail, failure messages and stack traces must be provided.",
			Timestamp: time.Now().Unix(),
		}
	}

	// If no output at all, require at least failure details
	if !hasOutput && !hasFailures {
		return &GateError{
			Gate:      ehg.Type(),
			TaskID:    ehg.taskID,
			Message:   "raw test output not provided",
			Details:   "Agent must submit actual test output, not just summary. Output and failure details are required.",
			Timestamp: time.Now().Unix(),
		}
	}

	return nil
}

// verifyExitCode checks that exit code reflects test results.
func (ehg *EmpiricalHonestyGate) verifyExitCode() error {
	if ehg.testResult.IsPassing() {
		return nil // Success exit code is fine if tests pass
	}

	if ehg.testResult.ExitCode == 0 {
		return &GateError{
			Gate:      ehg.Type(),
			TaskID:    ehg.taskID,
			Message:   "exit code does not reflect test failures",
			Details:   "Test process exited with code 0 but tests failed. Exit code must be non-zero for failures.",
			Timestamp: time.Now().Unix(),
		}
	}

	return nil
}

// generateHonestyReport provides detailed report of what claim vs. reality shows.
func (ehg *EmpiricalHonestyGate) generateHonestyReport() string {
	var report strings.Builder

	report.WriteString("=== Honesty Verification Report ===\n\n")

	report.WriteString("CLAIM:\n")
	report.WriteString(fmt.Sprintf("  Agent claimed: %q\n\n", ehg.agentClaim))

	report.WriteString("REALITY (Raw Test Output):\n")
	report.WriteString(fmt.Sprintf("  Tests run: %d\n", ehg.testResult.Total))
	report.WriteString(fmt.Sprintf("  Tests passed: %d\n", ehg.testResult.Passed))
	report.WriteString(fmt.Sprintf("  Tests failed: %d\n", ehg.testResult.Failed))
	report.WriteString(fmt.Sprintf("  Pass rate: %.1f%%\n", ehg.testResult.PassRate()))
	report.WriteString(fmt.Sprintf("  Exit code: %d\n\n", ehg.testResult.ExitCode))

	if len(ehg.testResult.Failures) > 0 {
		report.WriteString("FAILURE DETAILS:\n")
		for i, failure := range ehg.testResult.Failures {
			report.WriteString(fmt.Sprintf("  %d. %s\n", i+1, failure))
		}
		report.WriteString("\n")
	}

	if ehg.testResult.Output != "" {
		report.WriteString("RAW TEST OUTPUT:\n")
		report.WriteString(ehg.truncateOutput(ehg.testResult.Output, 500))
		report.WriteString("\n\n")
	}

	report.WriteString("VERDICT:\n")
	if !ehg.testResult.IsPassing() {
		report.WriteString("  ✗ DISHONESTY DETECTED: Agent claimed success but tests are failing.\n")
		report.WriteString("  This is an unacceptable violation. The agent must:\n")
		report.WriteString("  1. Fix the failing tests\n")
		report.WriteString("  2. Run tests again to verify\n")
		report.WriteString("  3. Submit honest results\n")
	} else {
		report.WriteString("  ✓ Honesty verified: Claims match actual results.\n")
	}

	return report.String()
}

// truncateOutput limits output to a maximum length.
func (ehg *EmpiricalHonestyGate) truncateOutput(output string, maxLen int) string {
	if len(output) <= maxLen {
		return output
	}
	return output[:maxLen] + "\n... (output truncated)"
}
