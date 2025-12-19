// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package opencode

import (
	"context"
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

// DefaultTestRunner implements TestRunner using the SDK client
type DefaultTestRunner struct {
	// Could integrate with executor for actual test running
	// For now, this is a stub for agents to call
}

// NewTestRunner creates a new TestRunner
func NewTestRunner() TestRunner {
	return &DefaultTestRunner{}
}

// RunTests executes all tests
func (r *DefaultTestRunner) RunTests(ctx context.Context, workDir string) (*TestRunResult, error) {
	// Stub implementation - agents would integrate with executor
	// This would call "go test ./..." via executor
	result := &TestRunResult{
		Success:     true,
		TotalTests:  0,
		PassedTests: 0,
		FailedTests: 0,
		Duration:    0,
	}
	return result, nil
}

// RunTestsWithPattern executes tests matching a pattern
func (r *DefaultTestRunner) RunTestsWithPattern(ctx context.Context, workDir string, pattern string) (*TestRunResult, error) {
	// Stub implementation
	result := &TestRunResult{
		Success:     true,
		TotalTests:  0,
		PassedTests: 0,
		FailedTests: 0,
		Duration:    0,
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
		return false, err
	}
	return !result.Success && result.FailedTests > 0, nil
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
