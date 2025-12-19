// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package opencode

import (
	"context"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// TestRunner provides a simplified interface for running tests on generated code.
// Agents use this to verify their implementations work correctly.
type TestRunner interface {
	// RunTests executes all tests in a project and returns results
	RunTests(ctx context.Context, workDir string) (*TestRunResult, error)

	// RunTestsWithPattern executes tests matching a pattern (e.g., "TestPokemon*")
	RunTestsWithPattern(ctx context.Context, workDir string, pattern string) (*TestRunResult, error)

	// VerifyTestPass checks that all tests pass
	VerifyTestPass(ctx context.Context, workDir string) (bool, error)

	// VerifyTestFail checks that tests fail (used in TDD RED phase)
	VerifyTestFail(ctx context.Context, workDir string) (bool, error)
}

// TestRunResult contains the output of a test run
type TestRunResult struct {
	Success      bool            // Whether all tests passed
	TotalTests   int             // Total number of tests run
	PassedTests  int             // Number of tests that passed
	FailedTests  int             // Number of tests that failed
	Output       string          // Raw test output
	Duration     time.Duration   // How long the tests took
	FailureTests []FailedTest    // Details of failed tests
}

// FailedTest represents a single test failure
type FailedTest struct {
	Name    string // Test name
	Message string // Failure message
}

// DefaultTestRunner implements TestRunner using actual go test execution
type DefaultTestRunner struct {
	// timeout for test execution
	timeout time.Duration
}

// NewTestRunner creates a new TestRunner with default 30-second timeout
func NewTestRunner() TestRunner {
	return &DefaultTestRunner{
		timeout: 30 * time.Second,
	}
}

// NewTestRunnerWithTimeout creates a TestRunner with custom timeout
func NewTestRunnerWithTimeout(timeout time.Duration) TestRunner {
	return &DefaultTestRunner{
		timeout: timeout,
	}
}

// RunTests executes all tests in a directory using go test
func (r *DefaultTestRunner) RunTests(ctx context.Context, workDir string) (*TestRunResult, error) {
	return r.runGoTest(ctx, workDir, "")
}

// RunTestsWithPattern executes tests matching a pattern
func (r *DefaultTestRunner) RunTestsWithPattern(ctx context.Context, workDir string, pattern string) (*TestRunResult, error) {
	return r.runGoTest(ctx, workDir, pattern)
}

// runGoTest executes go test and parses the output
func (r *DefaultTestRunner) runGoTest(ctx context.Context, workDir string, pattern string) (*TestRunResult, error) {
	startTime := time.Now()

	// Create context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	// Build go test command
	args := []string{"test", "-v"}
	if pattern != "" {
		args = append(args, "-run", pattern)
	}
	args = append(args, "./...")

	cmd := exec.CommandContext(timeoutCtx, "go", args...)
	cmd.Dir = workDir

	// Capture output
	output, err := cmd.CombinedOutput()
	outputStr := string(output)
	duration := time.Since(startTime)

	// Parse test results
	result := &TestRunResult{
		Output:      outputStr,
		Duration:    duration,
		FailureTests: []FailedTest{},
	}

	// Check if tests passed or failed
	if err != nil {
		result.Success = false
	} else {
		result.Success = true
	}

	// Parse test output to count tests
	result.TotalTests = countTests(outputStr)
	result.PassedTests = countPassedTests(outputStr)
	result.FailedTests = result.TotalTests - result.PassedTests

	// Extract failed test details
	if result.FailedTests > 0 {
		result.FailureTests = extractFailedTests(outputStr)
	}

	// If go test command failed but we found some tests, mark as failed
	if err != nil && result.Success {
		result.Success = false
	}

	return result, nil
}

// VerifyTestPass checks that tests pass
func (r *DefaultTestRunner) VerifyTestPass(ctx context.Context, workDir string) (bool, error) {
	result, err := r.RunTests(ctx, workDir)
	if err != nil {
		return false, err
	}
	return result.Success && result.FailedTests == 0, nil
}

// VerifyTestFail checks that tests fail (RED phase)
func (r *DefaultTestRunner) VerifyTestFail(ctx context.Context, workDir string) (bool, error) {
	result, err := r.RunTests(ctx, workDir)
	if err != nil {
		// Command failed, but we need to check if tests exist and failed
		if result != nil && result.FailedTests > 0 {
			return true, nil
		}
		return false, err
	}
	return !result.Success && result.FailedTests > 0, nil
}

// countTests counts the total number of tests in the output
func countTests(output string) int {
	lines := strings.Split(output, "\n")
	count := 0
	for _, line := range lines {
		if strings.Contains(line, "=== RUN") || strings.Contains(line, "--- PASS") || strings.Contains(line, "--- FAIL") {
			count++
		}
	}
	return count
}

// countPassedTests counts the number of passed tests
func countPassedTests(output string) int {
	lines := strings.Split(output, "\n")
	count := 0
	for _, line := range lines {
		if strings.Contains(line, "--- PASS:") {
			count++
		}
	}
	return count
}

// extractFailedTests extracts details about failed tests
func extractFailedTests(output string) []FailedTest {
	var failures []FailedTest
	lines := strings.Split(output, "\n")

	failPattern := regexp.MustCompile(`--- FAIL: (\S+)`)

	for i, line := range lines {
		if matches := failPattern.FindStringSubmatch(line); matches != nil {
			testName := matches[1]
			// Try to find the error message in following lines
			message := ""
			for j := i + 1; j < len(lines) && j < i+10; j++ {
				if strings.HasPrefix(lines[j], "---") {
					break
				}
				if strings.TrimSpace(lines[j]) != "" {
					message += lines[j] + "\n"
				}
			}
			failures = append(failures, FailedTest{
				Name:    testName,
				Message: strings.TrimSpace(message),
			})
		}
	}

	return failures
}

// parseTestRunOutput parses test output into TestRunResult
func parseTestRunOutput(output string) *TestRunResult {
	// Simple parsing - production code should use full parser
	result := &TestRunResult{
		Output:   output,
		Success:  true,
		Duration: 0,
	}

	// Check for test indicators
	if contains(output, "FAIL") {
		result.Success = false
	}
	if contains(output, "PASS") {
		result.Success = true
	}

	return result
}

// contains is a helper to check if a string contains a substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
