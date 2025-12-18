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

// TestDetectBypassEligibility_Documentation verifies documentation-only changes
func TestDetectBypassEligibility_Documentation(t *testing.T) {
	files := []string{
		"README.md",
		"docs/architecture.md",
		"CHANGELOG.md",
	}

	result := DetectBypassEligibility(files)

	if result.BypassType != BypassTypeDocumentation {
		t.Errorf("Expected BypassTypeDocumentation, got %s", result.BypassType)
	}

	if !result.Eligible {
		t.Error("Documentation-only changes should be eligible for bypass")
	}

	if len(result.SkippedGates) != 5 {
		t.Errorf("Expected 5 skipped gates for docs, got %d", len(result.SkippedGates))
	}
}

// TestDetectBypassEligibility_Configuration verifies configuration-only changes
func TestDetectBypassEligibility_Configuration(t *testing.T) {
	files := []string{
		"config/app.yaml",
		".env.example",
		"config/database.json",
	}

	result := DetectBypassEligibility(files)

	if result.BypassType != BypassTypeConfiguration {
		t.Errorf("Expected BypassTypeConfiguration, got %s", result.BypassType)
	}

	if !result.Eligible {
		t.Error("Configuration-only changes should be eligible for bypass")
	}

	if len(result.SkippedGates) != 5 {
		t.Errorf("Expected 5 skipped gates for config, got %d", len(result.SkippedGates))
	}
}

// TestDetectBypassEligibility_MixedChanges verifies mixed changes are not eligible
func TestDetectBypassEligibility_MixedChanges(t *testing.T) {
	files := []string{
		"README.md",
		"internal/api/handler.go",
		"config/app.yaml",
	}

	result := DetectBypassEligibility(files)

	if result.BypassType != BypassTypeNone {
		t.Errorf("Expected BypassTypeNone for mixed changes, got %s", result.BypassType)
	}

	if result.Eligible {
		t.Error("Mixed changes should not be eligible for bypass")
	}

	if len(result.SkippedGates) != 0 {
		t.Errorf("Expected no skipped gates for mixed changes, got %d", len(result.SkippedGates))
	}
}

// TestDetectBypassEligibility_CodeOnly verifies code-only changes are not eligible
func TestDetectBypassEligibility_CodeOnly(t *testing.T) {
	files := []string{
		"internal/api/handler.go",
		"pkg/utils/helper.go",
	}

	result := DetectBypassEligibility(files)

	if result.BypassType != BypassTypeNone {
		t.Errorf("Expected BypassTypeNone for code changes, got %s", result.BypassType)
	}

	if result.Eligible {
		t.Error("Code-only changes should not be eligible for bypass")
	}
}

// TestDetectBypassEligibility_Empty verifies empty file list
func TestDetectBypassEligibility_Empty(t *testing.T) {
	result := DetectBypassEligibility([]string{})

	if result.BypassType != BypassTypeNone {
		t.Errorf("Expected BypassTypeNone for empty list, got %s", result.BypassType)
	}

	if result.Eligible {
		t.Error("Empty file list should not be eligible for bypass")
	}
}
