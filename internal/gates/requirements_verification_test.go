package gates

import (
	"context"
	"errors"
	"testing"
)

func TestRequirementsVerificationGate_Type(t *testing.T) {
	gate := NewRequirementsVerificationGate("task-123", &Requirement{})

	if gate.Type() != GateRequirements {
		t.Errorf("Type() = %q, want %q", gate.Type(), GateRequirements)
	}
}

func TestRequirementsVerificationGate_Name(t *testing.T) {
	gate := NewRequirementsVerificationGate("task-123", &Requirement{})

	if gate.Name() != "Requirements Verification" {
		t.Errorf("Name() = %q, want %q", gate.Name(), "Requirements Verification")
	}
}

func TestRequirementsVerificationGate_Check_NoRequirement(t *testing.T) {
	gate := NewRequirementsVerificationGate("task-123", nil)
	gate.SetGeneratedTests([]string{"TestFoo"})

	err := gate.Check(context.Background())

	if err == nil {
		t.Fatal("expected error when requirement is nil")
	}

	var gateErr *GateError
	if !errors.As(err, &gateErr) {
		t.Fatalf("expected *GateError, got %T", err)
	}

	if gateErr.Gate != GateRequirements {
		t.Errorf("Gate = %q, want %q", gateErr.Gate, GateRequirements)
	}
}

func TestRequirementsVerificationGate_Check_NoTests(t *testing.T) {
	req := &Requirement{
		TaskID:    "task-123",
		Title:     "Test Task",
		Scenarios: []string{"Happy path"},
	}
	gate := NewRequirementsVerificationGate("task-123", req)

	err := gate.Check(context.Background())

	if err == nil {
		t.Fatal("expected error when no tests generated")
	}

	var gateErr *GateError
	if !errors.As(err, &gateErr) {
		t.Fatalf("expected *GateError, got %T", err)
	}

	if gateErr.Message != "no tests generated" {
		t.Errorf("Message = %q, want %q", gateErr.Message, "no tests generated")
	}
}

func TestRequirementsVerificationGate_Check_InsufficientCoverage(t *testing.T) {
	req := &Requirement{
		TaskID: "task-123",
		Title:  "String Validation",
		Scenarios: []string{
			"Valid input accepted",
			"Empty string rejected",
			"Special characters handled",
			"Unicode supported",
			"Max length enforced",
		},
	}
	gate := NewRequirementsVerificationGate("task-123", req)
	gate.SetCoverageThreshold(0.90) // Need 90%+

	// Only cover 2 out of 5 scenarios (40%)
	gate.SetGeneratedTests([]string{
		"TestValidInputAccepted",
		"TestEmptyStringRejected",
	})

	err := gate.Check(context.Background())

	if err == nil {
		t.Fatal("expected error for insufficient coverage")
	}

	var gateErr *GateError
	if !errors.As(err, &gateErr) {
		t.Fatalf("expected *GateError, got %T", err)
	}

	if gateErr.Gate != GateRequirements {
		t.Errorf("Gate = %q, want %q", gateErr.Gate, GateRequirements)
	}
}

func TestRequirementsVerificationGate_Check_SufficientCoverage(t *testing.T) {
	req := &Requirement{
		TaskID: "task-123",
		Title:  "String Validation",
		Scenarios: []string{
			"Valid input accepted",
			"Empty string rejected",
			"Special characters handled",
		},
	}
	gate := NewRequirementsVerificationGate("task-123", req)
	gate.SetCoverageThreshold(0.90)

	// Cover all 3 scenarios
	gate.SetGeneratedTests([]string{
		"TestValidInputAccepted",
		"TestEmptyStringRejected",
		"TestSpecialCharactersHandled",
	})

	err := gate.Check(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRequirementsVerificationGate_Check_VagueLanguage(t *testing.T) {
	req := &Requirement{
		TaskID:    "task-123",
		Title:     "Test Task",
		Scenarios: []string{"Valid input handled properly"},
	}
	gate := NewRequirementsVerificationGate("task-123", req)

	// Test with vague language
	gate.SetGeneratedTests([]string{
		"TestValidInputHandledProperly",
	})

	err := gate.Check(context.Background())

	if err == nil {
		t.Fatal("expected error for vague language")
	}

	gateErr, ok := err.(*GateError)
	if !ok {
		t.Fatalf("expected *GateError, got %T", err)
	}

	if gateErr.Message != "tests contain ambiguous language" {
		t.Errorf("Message = %q, want %q", gateErr.Message, "tests contain ambiguous language")
	}
}

func TestRequirementsVerificationGate_SetCoverageThreshold(t *testing.T) {
	tests := []struct {
		name      string
		threshold float64
		expected  float64
	}{
		{"valid threshold 0.8", 0.8, 0.8},
		{"valid threshold 0.95", 0.95, 0.95},
		{"invalid threshold 1.5", 1.5, 0.90},   // Should keep default
		{"invalid threshold 0", 0, 0.90},       // Should keep default
		{"invalid threshold -0.5", -0.5, 0.90}, // Should keep default
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gate := NewRequirementsVerificationGate("task-123", &Requirement{})
			gate.SetCoverageThreshold(tt.threshold)

			if gate.coverageThreshold != tt.expected {
				t.Errorf("coverageThreshold = %v, want %v", gate.coverageThreshold, tt.expected)
			}
		})
	}
}

func TestRequirementsVerificationGate_CalculateCoverage(t *testing.T) {
	tests := []struct {
		name      string
		scenarios []string
		generated []string
		expected  float64
	}{
		{
			name:      "no scenarios",
			scenarios: []string{},
			generated: []string{"TestFoo"},
			expected:  1.0,
		},
		{
			name:      "all covered",
			scenarios: []string{"Foo", "Bar"},
			generated: []string{"TestFoo", "TestBar"},
			expected:  1.0,
		},
		{
			name:      "partial coverage",
			scenarios: []string{"Foo", "Bar", "Baz"},
			generated: []string{"TestFoo", "TestBar"},
			expected:  2.0 / 3.0,
		},
		{
			name:      "no coverage",
			scenarios: []string{"Foo", "Bar"},
			generated: []string{"TestSomethingElse"},
			expected:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &Requirement{Scenarios: tt.scenarios}
			gate := NewRequirementsVerificationGate("task-123", req)
			gate.SetGeneratedTests(tt.generated)

			coverage := gate.calculateCoverage()

			if coverage != tt.expected {
				t.Errorf("calculateCoverage() = %v, want %v", coverage, tt.expected)
			}
		})
	}
}

func TestRequirementsVerificationGate_GateErrorFormat(t *testing.T) {
	gate := NewRequirementsVerificationGate("my-task-id", nil)

	err := gate.Check(context.Background())
	_ = err.(*GateError) // Type assertion to ensure it's a GateError

	// Check error string format
	errStr := err.Error()
	if !contains(errStr, "requirements_verification") {
		t.Errorf("Error string missing gate type: %q", errStr)
	}
	if !contains(errStr, "my-task-id") {
		t.Errorf("Error string missing task ID: %q", errStr)
	}
}

// contains checks if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func BenchmarkRequirementsVerificationGate_Check(b *testing.B) {
	req := &Requirement{
		TaskID: "task-123",
		Title:  "String Validation",
		Scenarios: []string{
			"Valid input accepted",
			"Empty string rejected",
			"Special characters handled",
		},
	}
	gate := NewRequirementsVerificationGate("task-123", req)
	gate.SetGeneratedTests([]string{
		"TestValidInputAccepted",
		"TestEmptyStringRejected",
		"TestSpecialCharactersHandled",
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = gate.Check(context.Background())
	}
}
