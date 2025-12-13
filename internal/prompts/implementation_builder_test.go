// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package prompts

import (
	"strings"
	"testing"
)

func TestNewImplementationBuilder(t *testing.T) {
	builder := NewImplementationBuilder("Test task")

	if builder == nil {
		t.Error("NewImplementationBuilder should return non-nil instance")
	}
	if builder.taskDescription != "Test task" {
		t.Error("task description mismatch")
	}
	if builder.requestType != RequestTypeInitial {
		t.Error("default request type should be initial")
	}
}

func TestImplementationBuilderInitialType(t *testing.T) {
	testCode := `func TestAdd(t *testing.T) {
	result := Add(2, 3)
	if result != 5 {
		t.Fatalf("expected 5, got %d", result)
	}
}`

	builder := NewImplementationBuilder("Implement Add function")
	builder.WithRequestType(RequestTypeInitial)
	builder.WithOutputPath("math/math.go")
	builder.WithTestContents(testCode)

	prompt := builder.Build()

	// Verify structure
	if !strings.Contains(prompt, "# Implementation Request") {
		t.Error("prompt should contain header")
	}
	if !strings.Contains(prompt, "## Task Description") {
		t.Error("prompt should contain task section")
	}
	if !strings.Contains(prompt, "## Test-Driven Development") {
		t.Error("prompt should contain test section for initial implementation")
	}
	if !strings.Contains(prompt, testCode) {
		t.Error("prompt should contain test code")
	}
	if !strings.Contains(prompt, "math/math.go") {
		t.Error("prompt should contain output path")
	}
	if !strings.Contains(prompt, "## Implementation Instructions") {
		t.Error("prompt should contain implementation instructions")
	}
}

func TestImplementationBuilderRefinementType(t *testing.T) {
	feedback := `The function should handle nil pointers gracefully.
Consider using meaningful variable names instead of x and y.`

	builder := NewImplementationBuilder("Refactor error handling")
	builder.WithRequestType(RequestTypeRefinement)
	builder.WithOutputPath("errors/handler.go")
	builder.WithReviewFeedback(feedback)

	prompt := builder.Build()

	// Verify refinement-specific content
	if !strings.Contains(prompt, "refinement") {
		t.Error("prompt should indicate refinement request type")
	}
	if !strings.Contains(prompt, "## Code Review Feedback") {
		t.Error("prompt should contain review feedback section")
	}
	if !strings.Contains(prompt, feedback) {
		t.Error("prompt should contain review feedback text")
	}
	if !strings.Contains(prompt, "## Refinement Instructions") {
		t.Error("prompt should contain refinement-specific instructions")
	}
}

func TestImplementationBuilderDebugType(t *testing.T) {
	failures := `--- FAIL: TestParse (0.01s)
	parse_test.go:42: parse error: unexpected token`

	builder := NewImplementationBuilder("Fix parsing bug")
	builder.WithRequestType(RequestTypeDebug)
	builder.WithOutputPath("parser/parser.go")
	builder.WithTestFailures(failures)

	prompt := builder.Build()

	// Verify debug-specific content
	if !strings.Contains(prompt, "debug") {
		t.Error("prompt should indicate debug request type")
	}
	if !strings.Contains(prompt, "## Test Failure Details") {
		t.Error("prompt should contain test failure section")
	}
	if !strings.Contains(prompt, failures) {
		t.Error("prompt should contain failure details")
	}
	if !strings.Contains(prompt, "## Bug Fix Instructions") {
		t.Error("prompt should contain bug fix-specific instructions")
	}
}

func TestImplementationBuilderWithContext(t *testing.T) {
	builder := NewImplementationBuilder("Implement component")
	builder.WithContext("Architecture", "Uses dependency injection pattern")
	builder.WithContext("Related Files", "manager.go, factory.go")

	prompt := builder.Build()

	if !strings.Contains(prompt, "## Context & Architecture") {
		t.Error("prompt should contain context section")
	}
	if !strings.Contains(prompt, "Architecture") {
		t.Error("prompt should contain context key")
	}
	if !strings.Contains(prompt, "dependency injection") {
		t.Error("prompt should contain context value")
	}
}

func TestImplementationBuilderWithMetadata(t *testing.T) {
	builder := NewImplementationBuilder("Task")
	builder.WithMetadata("agent", "implementation-bot")
	builder.WithMetadata("priority", "high")

	prompt := builder.Build()

	if !strings.Contains(prompt, "## Metadata") {
		t.Error("prompt should contain metadata section")
	}
	if !strings.Contains(prompt, "agent") {
		t.Error("prompt should contain metadata key")
	}
}

func TestImplementationBuilderString(t *testing.T) {
	builder := NewImplementationBuilder("Test task")
	str := builder.String()

	if str != builder.Build() {
		t.Error("String() should return same as Build()")
	}
}

func TestImplementationBuilderChaining(t *testing.T) {
	// Verify that builder methods return the builder for chaining
	builder := NewImplementationBuilder("Task")
	result := builder.
		WithRequestType(RequestTypeInitial).
		WithOutputPath("file.go").
		WithTestContents("tests").
		WithContext("key", "value").
		WithMetadata("meta", "data")

	if result != builder {
		t.Error("builder methods should return builder for chaining")
	}
}

func TestImplementationBuilderCompleteFlow(t *testing.T) {
	testCode := `func TestFibonacci(t *testing.T) {
	tests := []struct{n, expected int}{
		{0, 0}, {1, 1}, {5, 5}, {10, 55},
	}
	for _, tt := range tests {
		if got := Fibonacci(tt.n); got != tt.expected {
			t.Errorf("Fibonacci(%d) = %d, want %d", tt.n, got, tt.expected)
		}
	}
}`

	builder := NewImplementationBuilder("Implement Fibonacci sequence generator")
	builder.WithRequestType(RequestTypeInitial)
	builder.WithOutputPath("math/fibonacci.go")
	builder.WithTestContents(testCode)
	builder.WithContext("Related Files", "math/math_test.go")
	builder.WithContext("Dependencies", "Standard math library")
	builder.WithMetadata("priority", "high")
	builder.WithMetadata("complexity", "medium")

	prompt := builder.Build()

	// Verify complete prompt structure
	checks := []string{
		"# Implementation Request",
		"## Task Description",
		"## Test-Driven Development",
		"## Context & Architecture",
		"## Metadata",
		"## Implementation Instructions",
		testCode,
		"math/fibonacci.go",
		"Fibonacci",
	}

	for _, check := range checks {
		if !strings.Contains(prompt, check) {
			t.Errorf("prompt missing: %s", check)
		}
	}
}

func TestImplementationBuilderEmptyContent(t *testing.T) {
	builder := NewImplementationBuilder("Task description")
	prompt := builder.Build()

	// Should still generate valid prompt without optional sections
	if !strings.Contains(prompt, "# Implementation Request") {
		t.Error("header missing from empty builder")
	}
	if !strings.Contains(prompt, "Task description") {
		t.Error("task description missing")
	}

	// Optional sections should not be present
	if strings.Contains(prompt, "## Test-Driven Development") {
		t.Error("test section should not be present when no tests provided")
	}
	if strings.Contains(prompt, "## Code Review Feedback") {
		t.Error("feedback section should not be present when no feedback provided")
	}
}

func TestImplementationBuilderRequestTypes(t *testing.T) {
	types := []RequestType{
		RequestTypeInitial,
		RequestTypeRefinement,
		RequestTypeDebug,
	}

	for _, rt := range types {
		builder := NewImplementationBuilder("Task")
		builder.WithRequestType(rt)
		prompt := builder.Build()

		if !strings.Contains(prompt, string(rt)) {
			t.Errorf("prompt should contain request type: %s", rt)
		}
	}
}

func TestBuildImplementationPrompt(t *testing.T) {
	request := &PromptRequest{
		ID:              "test-001",
		TaskDescription: "Implement feature",
		RequestType:     RequestTypeInitial,
		OutputPath:      "feature.go",
		TestContents:    "test code",
		Context:         map[string]string{"key": "value"},
	}

	prompt := BuildImplementationPrompt(request)

	if !strings.Contains(prompt, "# Implementation Request") {
		t.Error("should generate valid prompt")
	}
	if !strings.Contains(prompt, "Implement feature") {
		t.Error("should include task description")
	}
	if !strings.Contains(prompt, "feature.go") {
		t.Error("should include output path")
	}
	if !strings.Contains(prompt, "test code") {
		t.Error("should include test contents")
	}
}

func TestImplementationBuilderNoOptionalFields(t *testing.T) {
	builder := NewImplementationBuilder("Simple task")
	prompt := builder.Build()

	// Verify it still contains required sections
	if !strings.Contains(prompt, "# Implementation Request") {
		t.Error("should have header")
	}
	// Output Expectations section should be skipped when no path
	if strings.Contains(prompt, "## Output Expectations") {
		t.Error("should skip output expectations without path")
	}
}

func TestImplementationBuilderWithAllFields(t *testing.T) {
	testCode := "test code"
	feedback := "review feedback"
	failures := "test failures"

	builder := NewImplementationBuilder("Complete task")
	builder.WithRequestType(RequestTypeInitial)
	builder.WithOutputPath("output.go")
	builder.WithTestContents(testCode)
	builder.WithReviewFeedback(feedback)
	builder.WithTestFailures(failures)

	prompt := builder.Build()

	// All content should be present
	if !strings.Contains(prompt, testCode) {
		t.Error("test contents missing")
	}
	if !strings.Contains(prompt, feedback) {
		t.Error("feedback missing")
	}
	if !strings.Contains(prompt, failures) {
		t.Error("failures missing")
	}
}
