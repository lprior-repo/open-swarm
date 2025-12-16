// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package temporal

import (
	"strings"
	"testing"
)

func TestParseTestOutput_AllPassing(t *testing.T) {
	parser := NewTestParser()
	output := `=== RUN   TestFoo
--- PASS: TestFoo (0.00s)
=== RUN   TestBar
--- PASS: TestBar (0.01s)
PASS
ok  	github.com/example/pkg	0.123s`

	result := parser.ParseTestOutput(output)

	if result.HasFailures {
		t.Errorf("Expected no failures, but HasFailures = true")
	}

	if len(result.Failures) != 0 {
		t.Errorf("Expected 0 failures, got %d", len(result.Failures))
	}

	if result.RawFailureOutput != "" {
		t.Errorf("Expected empty raw failure output, got: %s", result.RawFailureOutput)
	}
}

func TestParseTestOutput_SingleFailure(t *testing.T) {
	parser := NewTestParser()
	output := `=== RUN   TestFoo
--- FAIL: TestFoo (0.00s)
    foo_test.go:10: Expected 1, got 2
FAIL
FAIL	github.com/example/pkg	0.123s`

	result := parser.ParseTestOutput(output)

	if !result.HasFailures {
		t.Errorf("Expected failures, but HasFailures = false")
	}

	if len(result.Failures) != 1 {
		t.Fatalf("Expected 1 failure, got %d", len(result.Failures))
	}

	failure := result.Failures[0]
	if failure.TestName != "TestFoo" {
		t.Errorf("Expected test name 'TestFoo', got '%s'", failure.TestName)
	}

	if failure.FileName != "foo_test.go" {
		t.Errorf("Expected filename 'foo_test.go', got '%s'", failure.FileName)
	}

	if failure.LineNumber != "10" {
		t.Errorf("Expected line number '10', got '%s'", failure.LineNumber)
	}

	if !strings.Contains(failure.ErrorMessage, "Expected 1, got 2") {
		t.Errorf("Expected error message to contain 'Expected 1, got 2', got: %s", failure.ErrorMessage)
	}

	// Check that timing lines are excluded from raw output
	if strings.Contains(result.RawFailureOutput, "0.123s") {
		t.Errorf("Raw failure output should not contain timing: %s", result.RawFailureOutput)
	}

	// Check that PASS lines are excluded
	if strings.Contains(result.RawFailureOutput, "PASS") {
		t.Errorf("Raw failure output should not contain PASS: %s", result.RawFailureOutput)
	}
}

func TestParseTestOutput_Panic(t *testing.T) {
	parser := NewTestParser()
	output := `=== RUN   TestPanic
panic: runtime error: index out of range

goroutine 6 [running]:
github.com/example/pkg.TestPanic(0xc0000a6000)
	/path/to/test.go:15 +0x123
--- FAIL: TestPanic (0.00s)
FAIL
FAIL	github.com/example/pkg	0.456s`

	result := parser.ParseTestOutput(output)

	if !result.HasFailures {
		t.Errorf("Expected failures, but HasFailures = false")
	}

	if len(result.Failures) != 1 {
		t.Fatalf("Expected 1 failure, got %d", len(result.Failures))
	}

	failure := result.Failures[0]
	if !failure.IsPanic {
		t.Errorf("Expected IsPanic = true, got false")
	}

	if !strings.Contains(result.RawFailureOutput, "panic:") {
		t.Errorf("Expected raw output to contain panic, got: %s", result.RawFailureOutput)
	}
}

func TestParseTestOutput_BuildFailure(t *testing.T) {
	parser := NewTestParser()
	output := `# github.com/example/pkg [build failed]
./main.go:10:2: undefined: Foo
FAIL	github.com/example/pkg [build failed]`

	result := parser.ParseTestOutput(output)

	if !result.HasFailures {
		t.Errorf("Expected failures, but HasFailures = false")
	}

	if len(result.Failures) != 1 {
		t.Fatalf("Expected 1 failure, got %d", len(result.Failures))
	}

	failure := result.Failures[0]
	if failure.TestName != "BUILD" {
		t.Errorf("Expected test name 'BUILD', got '%s'", failure.TestName)
	}

	if failure.Package != "github.com/example/pkg" {
		t.Errorf("Expected package 'github.com/example/pkg', got '%s'", failure.Package)
	}
}

func TestParseTestOutput_MultipleFailures(t *testing.T) {
	parser := NewTestParser()
	output := `=== RUN   TestFoo
--- FAIL: TestFoo (0.00s)
    foo_test.go:10: Expected 1, got 2
=== RUN   TestBar
--- FAIL: TestBar (0.01s)
    bar_test.go:20: Expected true, got false
=== RUN   TestBaz
--- PASS: TestBaz (0.00s)
FAIL
FAIL	github.com/example/pkg	0.123s`

	result := parser.ParseTestOutput(output)

	if !result.HasFailures {
		t.Errorf("Expected failures, but HasFailures = false")
	}

	if len(result.Failures) != 2 {
		t.Errorf("Expected 2 failures, got %d", len(result.Failures))
	}

	// Verify PASS lines are not in raw output
	if strings.Contains(result.RawFailureOutput, "PASS: TestBaz") {
		t.Errorf("Raw failure output should not contain PASS lines")
	}

	// Verify both failures are captured
	testNames := []string{}
	for _, f := range result.Failures {
		testNames = append(testNames, f.TestName)
	}

	if !testContains(testNames, "TestFoo") {
		t.Errorf("Expected to find TestFoo in failures")
	}

	if !testContains(testNames, "TestBar") {
		t.Errorf("Expected to find TestBar in failures")
	}
}

func TestParseTestOutput_ExcludeTimingAndCoverage(t *testing.T) {
	parser := NewTestParser()
	output := `=== RUN   TestFoo
--- FAIL: TestFoo (0.00s)
    foo_test.go:10: Failed
FAIL
coverage: 45.5% of statements
FAIL	github.com/example/pkg	0.123s`

	result := parser.ParseTestOutput(output)

	// Verify timing lines are excluded
	if strings.Contains(result.RawFailureOutput, "0.123s") {
		t.Errorf("Raw output should not contain timing")
	}

	// Verify coverage lines are excluded
	if strings.Contains(result.RawFailureOutput, "coverage:") {
		t.Errorf("Raw output should not contain coverage lines")
	}

	// But should contain the actual failure
	if !strings.Contains(result.RawFailureOutput, "FAIL: TestFoo") {
		t.Errorf("Raw output should contain the failure")
	}
}

func TestParseTestOutput_ComplexErrorMessage(t *testing.T) {
	parser := NewTestParser()
	output := `=== RUN   TestCompare
--- FAIL: TestCompare (0.00s)
    compare_test.go:25:
        Error Trace:	compare_test.go:25
        Error:      	Not equal:
                        expected: []string{"a", "b", "c"}
                        actual  : []string{"a", "b"}
        Test:       	TestCompare
FAIL
FAIL	github.com/example/pkg	0.089s`

	result := parser.ParseTestOutput(output)

	if len(result.Failures) != 1 {
		t.Fatalf("Expected 1 failure, got %d", len(result.Failures))
	}

	failure := result.Failures[0]
	if !strings.Contains(failure.ErrorMessage, "Not equal") {
		t.Errorf("Expected error message to contain 'Not equal', got: %s", failure.ErrorMessage)
	}

	if !strings.Contains(failure.ErrorMessage, "expected:") {
		t.Errorf("Expected error message to contain 'expected:', got: %s", failure.ErrorMessage)
	}
}

func TestParseTestOutput_NoTestOutput(t *testing.T) {
	parser := NewTestParser()
	output := `?   	github.com/example/pkg	[no test files]`

	result := parser.ParseTestOutput(output)

	if result.HasFailures {
		t.Errorf("Expected no failures for 'no test files' output")
	}

	if len(result.Failures) != 0 {
		t.Errorf("Expected 0 failures, got %d", len(result.Failures))
	}
}

func TestGetFailureSummary_NoFailures(t *testing.T) {
	parser := NewTestParser()
	result := &TestParseResult{
		HasFailures: false,
		Failures:    []TestFailure{},
	}

	summary := parser.GetFailureSummary(result)

	if !strings.Contains(summary, "All tests passed") {
		t.Errorf("Expected summary to say 'All tests passed', got: %s", summary)
	}
}

func TestGetFailureSummary_WithFailures(t *testing.T) {
	parser := NewTestParser()
	result := &TestParseResult{
		HasFailures: true,
		Failures: []TestFailure{
			{
				TestName:     "TestFoo",
				Package:      "github.com/example/pkg",
				ErrorMessage: "Expected 1, got 2",
				FileName:     "foo_test.go",
				LineNumber:   "10",
			},
			{
				TestName: "TestBar",
				IsPanic:  true,
			},
		},
		FailedTests: 2,
	}

	summary := parser.GetFailureSummary(result)

	if !strings.Contains(summary, "TestFoo") {
		t.Errorf("Expected summary to contain TestFoo")
	}

	if !strings.Contains(summary, "TestBar") {
		t.Errorf("Expected summary to contain TestBar")
	}

	if !strings.Contains(summary, "[PANIC]") {
		t.Errorf("Expected summary to indicate panic")
	}

	if !strings.Contains(summary, "foo_test.go:10") {
		t.Errorf("Expected summary to contain location")
	}

	if !strings.Contains(summary, "Total failed: 2") {
		t.Errorf("Expected summary to contain failure count")
	}
}

func TestGetFailureSummary_BuildFailure(t *testing.T) {
	parser := NewTestParser()
	result := &TestParseResult{
		HasFailures: true,
		Failures: []TestFailure{
			{
				TestName: "BUILD",
				Package:  "github.com/example/pkg",
			},
		},
	}

	summary := parser.GetFailureSummary(result)

	if !strings.Contains(summary, "Build failed") {
		t.Errorf("Expected summary to mention build failure, got: %s", summary)
	}

	if !strings.Contains(summary, "github.com/example/pkg") {
		t.Errorf("Expected summary to mention package, got: %s", summary)
	}
}

func TestGetRawFailures(t *testing.T) {
	parser := NewTestParser()
	rawOutput := "--- FAIL: TestFoo\n    error message\nFAIL\tpackage"
	result := &TestParseResult{
		RawFailureOutput: rawOutput,
	}

	raw := parser.GetRawFailures(result)

	if raw != rawOutput {
		t.Errorf("Expected raw output to match, got: %s", raw)
	}
}

// Helper function
func testContains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
