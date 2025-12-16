// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

// Package slices provides vertical slice architecture for Open Swarm workflows.
//
// test_execution.go: Complete vertical slice for test execution and validation
// - Test execution via SDK
// - Output parsing and validation
// - RED/GREEN verification
//
// This slice follows CUPID principles:
// - Composable: Self-contained test execution operations
// - Unix philosophy: Does test execution and validation, nothing else
// - Predictable: Clear pass/fail results with parsed output
// - Idiomatic: Go test conventions, Temporal patterns
// - Domain-centric: Organized around test capability
package slices

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/activity"

	"open-swarm/internal/agent"
)

// ============================================================================
// ACTIVITIES
// ============================================================================

// TestExecutionActivities handles all test execution operations
type TestExecutionActivities struct {
	// No external dependencies - uses SDK client from bootstrap output
}

// NewTestExecutionActivities creates a new test execution activities instance
func NewTestExecutionActivities() *TestExecutionActivities {
	return &TestExecutionActivities{}
}

// RunTests executes tests in a cell and returns parsed results
//
// This activity:
// 1. Reconstructs SDK client from bootstrap output
// 2. Executes "go test ./..." via SDK
// 3. Parses output to extract test results
// 4. Returns structured TestResult
//
// Used for both RED (tests should fail) and GREEN (tests should pass) verification.
func (t *TestExecutionActivities) RunTests(ctx context.Context, output BootstrapOutput) (*TestResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Running tests", "cellID", output.CellID)

	activity.RecordHeartbeat(ctx, "executing tests")

	startTime := time.Now()

	// Reconstruct SDK client
	client := ReconstructClient(output)

	// Execute tests via SDK
	result, err := client.ExecutePrompt(ctx, "Run all tests: go test ./...", &agent.PromptOptions{
		Agent: "build",
	})
	if err != nil {
		return &TestResult{
			Passed:   false,
			Output:   err.Error(),
			Duration: time.Since(startTime),
		}, fmt.Errorf("failed to execute tests in cell %q: %w", output.CellID, err)
	}

	duration := time.Since(startTime)

	// Parse test output
	testResult := parseTestOutput(result.GetText())
	testResult.Duration = duration

	logger.Info("Tests completed",
		"cellID", output.CellID,
		"passed", testResult.Passed,
		"total", testResult.TotalTests,
		"failed", testResult.FailedTests,
		"duration", duration)

	return testResult, nil
}

// VerifyRED confirms tests fail as expected (TDD RED state)
//
// In Test-Driven Development, the RED state means:
// - Tests exist
// - Tests fail (because implementation doesn't exist yet)
//
// This activity returns:
// - Success: tests exist and fail (correct RED state)
// - Error: tests pass (implementation already exists) or tests don't exist
func (t *TestExecutionActivities) VerifyRED(ctx context.Context, output BootstrapOutput) (*GateResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Verifying RED state", "cellID", output.CellID)

	startTime := time.Now()

	testResult, err := t.RunTests(ctx, output)
	if err != nil {
		return &GateResult{
			GateName:      "verify_red",
			Passed:        false,
			Duration:      time.Since(startTime),
			Error:         err.Error(),
			TestResult:    testResult,
			RetryAttempts: 0,
		}, err
	}

	// RED verification: tests should FAIL
	if testResult.Passed {
		return &GateResult{
			GateName:   "verify_red",
			Passed:     false,
			Duration:   time.Since(startTime),
			Message:    "Tests passed when they should fail (RED state violated)",
			TestResult: testResult,
		}, fmt.Errorf("RED verification failed: tests passed unexpectedly")
	}

	// Correct RED state: tests exist and fail
	logger.Info("RED state verified", "cellID", output.CellID, "failedTests", testResult.FailedTests)

	return &GateResult{
		GateName:   "verify_red",
		Passed:     true,
		Duration:   time.Since(startTime),
		Message:    fmt.Sprintf("RED state verified: %d tests failing as expected", testResult.FailedTests),
		TestResult: testResult,
	}, nil
}

// VerifyGREEN confirms tests pass as expected (TDD GREEN state)
//
// In Test-Driven Development, the GREEN state means:
// - Tests exist
// - All tests pass (implementation is correct)
//
// This activity returns:
// - Success: all tests pass (correct GREEN state)
// - Error: tests fail (implementation is incomplete or incorrect)
func (t *TestExecutionActivities) VerifyGREEN(ctx context.Context, output BootstrapOutput) (*GateResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Verifying GREEN state", "cellID", output.CellID)

	startTime := time.Now()

	testResult, err := t.RunTests(ctx, output)
	if err != nil {
		return &GateResult{
			GateName:      "verify_green",
			Passed:        false,
			Duration:      time.Since(startTime),
			Error:         err.Error(),
			TestResult:    testResult,
			RetryAttempts: 0,
		}, err
	}

	// GREEN verification: tests should PASS
	if !testResult.Passed {
		return &GateResult{
			GateName:   "verify_green",
			Passed:     false,
			Duration:   time.Since(startTime),
			Message:    fmt.Sprintf("Tests failed when they should pass (GREEN state violated): %d failures", testResult.FailedTests),
			TestResult: testResult,
		}, fmt.Errorf("GREEN verification failed: %d tests failed", testResult.FailedTests)
	}

	// Correct GREEN state: all tests pass
	logger.Info("GREEN state verified", "cellID", output.CellID, "passedTests", testResult.PassedTests)

	return &GateResult{
		GateName:   "verify_green",
		Passed:     true,
		Duration:   time.Since(startTime),
		Message:    fmt.Sprintf("GREEN state verified: %d tests passing", testResult.PassedTests),
		TestResult: testResult,
	}, nil
}

// RunTestsWithRetry executes tests with retry budget for transient failures
//
// Some test failures are transient (network issues, timing, etc.).
// This activity retries test execution up to maxRetries times.
func (t *TestExecutionActivities) RunTestsWithRetry(ctx context.Context, output BootstrapOutput, maxRetries int) (*GateResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Running tests with retry", "cellID", output.CellID, "maxRetries", maxRetries)

	var lastTestResult *TestResult
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			logger.Info("Retrying test execution", "cellID", output.CellID, "attempt", attempt)
			activity.RecordHeartbeat(ctx, fmt.Sprintf("retry attempt %d/%d", attempt, maxRetries))
		}

		startTime := time.Now()
		testResult, err := t.RunTests(ctx, output)
		duration := time.Since(startTime)

		if err != nil {
			lastErr = err
			lastTestResult = testResult
			continue
		}

		if testResult.Passed {
			// Success - tests passed
			return &GateResult{
				GateName:      "test_execution",
				Passed:        true,
				Duration:      duration,
				TestResult:    testResult,
				RetryAttempts: attempt,
			}, nil
		}

		// Tests failed - save result for potential retry
		lastTestResult = testResult
		lastErr = fmt.Errorf("%d tests failed", testResult.FailedTests)
	}

	// All retries exhausted
	logger.Warn("Test execution failed after retries", "cellID", output.CellID, "attempts", maxRetries+1)

	return &GateResult{
		GateName:      "test_execution",
		Passed:        false,
		Duration:      0, // Total duration not tracked across retries
		Error:         lastErr.Error(),
		TestResult:    lastTestResult,
		RetryAttempts: maxRetries,
	}, lastErr
}

// ============================================================================
// BUSINESS LOGIC
// ============================================================================

// parseTestOutput parses Go test output into structured TestResult
//
// Parses output from "go test ./..." to extract:
// - Total tests run
// - Passed tests
// - Failed tests
// - Failure details
//
// This is a simplified parser. Production code should use the full parser
// from internal/temporal/test_parser.go
func parseTestOutput(output string) *TestResult {
	// Simple heuristic parsing
	// TODO: Use full TestParser from internal/temporal/test_parser.go

	passed := true
	failedTests := 0
	passedTests := 0
	totalTests := 0
	var failureTests []string

	lines := splitLines(output)

	for _, line := range lines {
		// Check for FAIL markers
		if containsString(line, "FAIL") && containsString(line, "---") {
			passed = false
			failedTests++
			totalTests++
			// Extract test name (simplified)
			if idx := indexAfter(line, "FAIL:"); idx != -1 && idx < len(line) {
				testName := extractTestName(line[idx:])
				failureTests = append(failureTests, testName)
			}
		}

		// Check for PASS markers
		if containsString(line, "PASS") && containsString(line, "---") {
			passedTests++
			totalTests++
		}

		// Check for overall FAIL
		if containsString(line, "FAIL\t") {
			passed = false
		}
	}

	// If no test markers found but output contains "PASS", assume single test passed
	if totalTests == 0 && containsString(output, "PASS") {
		passedTests = 1
		totalTests = 1
	}

	return &TestResult{
		Passed:       passed,
		TotalTests:   totalTests,
		PassedTests:  passedTests,
		FailedTests:  failedTests,
		Output:       output,
		FailureTests: failureTests,
	}
}

// ============================================================================
// HELPERS
// ============================================================================

// splitLines splits output into lines
func splitLines(s string) []string {
	if s == "" {
		return []string{}
	}
	lines := []string{}
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

// containsString checks if a string contains a substring
func containsString(s, substr string) bool {
	return indexString(s, substr) != -1
}

// indexString finds the index of a substring
func indexString(s, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	if len(s) < len(substr) {
		return -1
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// indexAfter finds the first occurrence after a prefix
func indexAfter(s, prefix string) int {
	idx := indexString(s, prefix)
	if idx == -1 {
		return -1
	}
	return idx + len(prefix)
}

// extractTestName extracts test name from a line
func extractTestName(s string) string {
	// Skip leading whitespace
	start := 0
	for start < len(s) && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	if start >= len(s) {
		return ""
	}

	// Find end (whitespace or parenthesis)
	end := start
	for end < len(s) && s[end] != ' ' && s[end] != '\t' && s[end] != '(' {
		end++
	}

	return s[start:end]
}
