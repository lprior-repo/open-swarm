// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package temporal

import (
	"strings"
	"testing"
)

// TestExecuteGenImpl_WithRetryFeedback verifies that ExecuteGenImpl includes
// parsed test failures in the prompt when testFailureOutput is provided
func TestExecuteGenImpl_WithRetryFeedback(t *testing.T) {
	// Sample test failure output
	testOutput := `=== RUN   TestCalculator
--- FAIL: TestCalculator (0.00s)
    calculator_test.go:15: Expected 4, got 0
FAIL
FAIL    example.com/calculator  0.001s`

	// Parse the output using TestParser
	parser := NewTestParser()
	parseResult := parser.ParseTestOutput(testOutput)

	// Verify parsing worked
	if !parseResult.HasFailures {
		t.Fatal("Expected test failures to be detected")
	}

	if len(parseResult.Failures) != 1 {
		t.Fatalf("Expected 1 failure, got %d", len(parseResult.Failures))
	}

	failure := parseResult.Failures[0]
	if failure.TestName != "TestCalculator" {
		t.Errorf("Expected TestCalculator, got %s", failure.TestName)
	}

	// Get failure summary
	summary := parser.GetFailureSummary(parseResult)

	// Verify summary contains key information
	if !strings.Contains(summary, "TestCalculator") {
		t.Error("Summary should contain test name")
	}

	if !strings.Contains(summary, "Expected 4, got 0") {
		t.Error("Summary should contain error message")
	}

	// Verify the summary would be useful for retry
	if !strings.Contains(summary, "Test Failures:") {
		t.Error("Summary should have clear header")
	}
}

// TestExecuteGenImpl_NoRetryFeedback verifies that ExecuteGenImpl works
// correctly when no testFailureOutput is provided (first attempt)
func TestExecuteGenImpl_NoRetryFeedback(t *testing.T) {
	// Empty test failure output should not cause issues
	parser := NewTestParser()
	parseResult := parser.ParseTestOutput("")

	if parseResult.HasFailures {
		t.Error("Empty output should not have failures")
	}

	summary := parser.GetFailureSummary(parseResult)
	if summary != "All tests passed" {
		t.Errorf("Expected 'All tests passed', got: %s", summary)
	}
}

// TestExecuteGenImpl_MultipleFailures verifies parsing multiple test failures
func TestExecuteGenImpl_MultipleFailures(t *testing.T) {
	testOutput := `=== RUN   TestAdd
--- FAIL: TestAdd (0.00s)
    math_test.go:10: Expected 5, got 0
=== RUN   TestSubtract
--- FAIL: TestSubtract (0.00s)
    math_test.go:20: Expected 3, got 0
FAIL
FAIL    example.com/math  0.002s`

	parser := NewTestParser()
	parseResult := parser.ParseTestOutput(testOutput)

	if !parseResult.HasFailures {
		t.Fatal("Expected test failures to be detected")
	}

	if len(parseResult.Failures) != 2 {
		t.Fatalf("Expected 2 failures, got %d", len(parseResult.Failures))
	}

	// Verify both test names are captured
	testNames := make(map[string]bool)
	for _, failure := range parseResult.Failures {
		testNames[failure.TestName] = true
	}

	if !testNames["TestAdd"] {
		t.Error("TestAdd should be in failures")
	}

	if !testNames["TestSubtract"] {
		t.Error("TestSubtract should be in failures")
	}

	// Get summary and verify it includes both failures
	summary := parser.GetFailureSummary(parseResult)
	if !strings.Contains(summary, "TestAdd") {
		t.Error("Summary should contain TestAdd")
	}

	if !strings.Contains(summary, "TestSubtract") {
		t.Error("Summary should contain TestSubtract")
	}
}
