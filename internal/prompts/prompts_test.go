package prompts

import (
	"strings"
	"testing"
)

func TestArchitectureReviewBuilder(t *testing.T) {
	builder := NewArchitectureReviewBuilder()

	t.Run("GetReviewType", func(t *testing.T) {
		if builder.GetReviewType() != ReviewTypeArchitecture {
			t.Errorf("Expected ReviewTypeArchitecture, got %s", builder.GetReviewType())
		}
	})

	t.Run("ValidRequest", func(t *testing.T) {
		request := ReviewRequest{
			Type:            ReviewTypeArchitecture,
			TaskID:          "TASK-001",
			TaskDescription: "Implement user authentication",
			CodeContext: CodeContext{
				FilePath:    "auth/service.go",
				FileContent: "package auth\n\nfunc Login() error { return nil }",
				Language:    "go",
				PackageName: "auth",
			},
			AcceptanceCriteria: []string{
				"Must follow SOLID principles",
				"Must be extensible",
			},
			AdditionalContext: "This is part of the authentication system",
			RequireVote:       true,
		}

		prompt, err := builder.Build(request)
		if err != nil {
			t.Fatalf("Build failed: %v", err)
		}

		// Verify prompt contains key sections
		assertContains(t, prompt, "senior software architect")
		assertContains(t, prompt, "Architecture and Design Patterns")
		assertContains(t, prompt, "TASK-001")
		assertContains(t, prompt, "Implement user authentication")
		assertContains(t, prompt, "auth/service.go")
		assertContains(t, prompt, "SOLID principles")
		assertContains(t, prompt, "VOTE: APPROVE")
		assertContains(t, prompt, "VOTE: REQUEST_CHANGE")
		assertContains(t, prompt, "VOTE: REJECT")
	})

	t.Run("MissingTaskID", func(t *testing.T) {
		request := ReviewRequest{
			TaskDescription: "Test",
			CodeContext: CodeContext{
				FileContent: "package main",
			},
		}

		_, err := builder.Build(request)
		if err == nil {
			t.Error("Expected error for missing TaskID")
		}
	})

	t.Run("MissingTaskDescription", func(t *testing.T) {
		request := ReviewRequest{
			TaskID: "TASK-001",
			CodeContext: CodeContext{
				FileContent: "package main",
			},
		}

		_, err := builder.Build(request)
		if err == nil {
			t.Error("Expected error for missing TaskDescription")
		}
	})

	t.Run("MissingCodeContent", func(t *testing.T) {
		request := ReviewRequest{
			TaskID:          "TASK-001",
			TaskDescription: "Test",
		}

		_, err := builder.Build(request)
		if err == nil {
			t.Error("Expected error for missing code content")
		}
	})

	t.Run("WithDiff", func(t *testing.T) {
		request := ReviewRequest{
			TaskID:          "TASK-001",
			TaskDescription: "Add logging",
			CodeContext: CodeContext{
				FilePath: "main.go",
				Diff:     "+import \"log\"\n+log.Println(\"test\")",
			},
			RequireVote: false,
		}

		prompt, err := builder.Build(request)
		if err != nil {
			t.Fatalf("Build failed: %v", err)
		}

		assertContains(t, prompt, "Changes (Git Diff)")
		assertContains(t, prompt, "+import \"log\"")
		assertNotContains(t, prompt, "VOTE:")
	})
}

func TestFunctionalReviewBuilder(t *testing.T) {
	builder := NewFunctionalReviewBuilder()

	t.Run("GetReviewType", func(t *testing.T) {
		if builder.GetReviewType() != ReviewTypeFunctional {
			t.Errorf("Expected ReviewTypeFunctional, got %s", builder.GetReviewType())
		}
	})

	t.Run("ValidRequest", func(t *testing.T) {
		request := ReviewRequest{
			Type:            ReviewTypeFunctional,
			TaskID:          "TASK-002",
			TaskDescription: "Calculate user score",
			CodeContext: CodeContext{
				FilePath:    "score/calculator.go",
				FileContent: "package score\n\nfunc Calculate(points int) int { return points * 2 }",
				Language:    "go",
				PackageName: "score",
			},
			AcceptanceCriteria: []string{
				"Must handle negative points",
				"Must not overflow",
			},
			RequireVote: true,
		}

		prompt, err := builder.Build(request)
		if err != nil {
			t.Fatalf("Build failed: %v", err)
		}

		assertContains(t, prompt, "senior software engineer")
		assertContains(t, prompt, "Business Logic and Functional Correctness")
		assertContains(t, prompt, "TASK-002")
		assertContains(t, prompt, "Calculate user score")
		assertContains(t, prompt, "Edge Cases and Error Handling")
		assertContains(t, prompt, "Go Best Practices")
	})

	t.Run("WithSurroundingCode", func(t *testing.T) {
		request := ReviewRequest{
			TaskID:          "TASK-003",
			TaskDescription: "Fix validation",
			CodeContext: CodeContext{
				FileContent:     "func Validate() bool { return true }",
				SurroundingCode: "type User struct { Name string }",
			},
			RequireVote: true,
		}

		prompt, err := builder.Build(request)
		if err != nil {
			t.Fatalf("Build failed: %v", err)
		}

		assertContains(t, prompt, "Related Code Context")
		assertContains(t, prompt, "type User struct")
	})
}

func TestTestingReviewBuilder(t *testing.T) {
	builder := NewTestingReviewBuilder()

	t.Run("GetReviewType", func(t *testing.T) {
		if builder.GetReviewType() != ReviewTypeTesting {
			t.Errorf("Expected ReviewTypeTesting, got %s", builder.GetReviewType())
		}
	})

	t.Run("ValidRequest", func(t *testing.T) {
		request := ReviewRequest{
			Type:            ReviewTypeTesting,
			TaskID:          "TASK-004",
			TaskDescription: "Add tests for payment processing",
			CodeContext: CodeContext{
				FilePath:    "payment/payment_test.go",
				FileContent: "package payment\n\nfunc TestProcess(t *testing.T) {}",
				Language:    "go",
				PackageName: "payment",
			},
			AcceptanceCriteria: []string{
				"Must test edge cases",
				"Must achieve 80% coverage",
			},
			RequireVote: true,
		}

		prompt, err := builder.Build(request)
		if err != nil {
			t.Fatalf("Build failed: %v", err)
		}

		assertContains(t, prompt, "QA engineer and testing specialist")
		assertContains(t, prompt, "Test Coverage and Quality")
		assertContains(t, prompt, "TASK-004")
		assertContains(t, prompt, "payment processing")
		assertContains(t, prompt, "TDD Principles")
		assertContains(t, prompt, "table-driven tests")
	})
}

func TestGetBuilder(t *testing.T) {
	tests := []struct {
		reviewType ReviewType
		wantErr    bool
	}{
		{ReviewTypeArchitecture, false},
		{ReviewTypeFunctional, false},
		{ReviewTypeTesting, false},
		{"invalid", true},
	}

	for _, tt := range tests {
		t.Run(string(tt.reviewType), func(t *testing.T) {
			builder, err := GetBuilder(tt.reviewType)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error for invalid review type")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if builder == nil {
					t.Error("Expected builder, got nil")
				}
				if builder.GetReviewType() != tt.reviewType {
					t.Errorf("Expected %s, got %s", tt.reviewType, builder.GetReviewType())
				}
			}
		})
	}
}

func TestBuildPrompt(t *testing.T) {
	request := ReviewRequest{
		Type:            ReviewTypeFunctional,
		TaskID:          "TASK-005",
		TaskDescription: "Test BuildPrompt",
		CodeContext: CodeContext{
			FileContent: "package main",
		},
		RequireVote: true,
	}

	prompt, err := BuildPrompt(request)
	if err != nil {
		t.Fatalf("BuildPrompt failed: %v", err)
	}

	assertContains(t, prompt, "Business Logic")
	assertContains(t, prompt, "TASK-005")
}

func TestBuildAllReviewPrompts(t *testing.T) {
	request := ReviewRequest{
		TaskID:          "TASK-006",
		TaskDescription: "Multi-review test",
		CodeContext: CodeContext{
			FilePath:    "test.go",
			FileContent: "package main\n\nfunc main() {}",
			Language:    "go",
		},
		RequireVote: true,
	}

	prompts, err := BuildAllReviewPrompts(request)
	if err != nil {
		t.Fatalf("BuildAllReviewPrompts failed: %v", err)
	}

	if len(prompts) != 3 {
		t.Errorf("Expected 3 prompts, got %d", len(prompts))
	}

	// Verify each type
	for _, reviewType := range []ReviewType{
		ReviewTypeArchitecture,
		ReviewTypeFunctional,
		ReviewTypeTesting,
	} {
		prompt, ok := prompts[reviewType]
		if !ok {
			t.Errorf("Missing prompt for %s", reviewType)
			continue
		}
		if prompt == "" {
			t.Errorf("Empty prompt for %s", reviewType)
		}
		assertContains(t, prompt, "TASK-006")
	}

	// Verify each is different
	if prompts[ReviewTypeArchitecture] == prompts[ReviewTypeFunctional] {
		t.Error("Architecture and Functional prompts are identical")
	}
	if prompts[ReviewTypeFunctional] == prompts[ReviewTypeTesting] {
		t.Error("Functional and Testing prompts are identical")
	}
}

func TestConvenienceFunctions(t *testing.T) {
	t.Run("BuildArchitecturePrompt", func(t *testing.T) {
		request := ReviewRequest{
			TaskID:          "TASK-007",
			TaskDescription: "Test",
			CodeContext:     CodeContext{FileContent: "package main"},
		}

		prompt, err := BuildArchitecturePrompt(request)
		if err != nil {
			t.Fatalf("BuildArchitecturePrompt failed: %v", err)
		}
		assertContains(t, prompt, "architect")
	})

	t.Run("BuildFunctionalPrompt", func(t *testing.T) {
		request := ReviewRequest{
			TaskID:          "TASK-008",
			TaskDescription: "Test",
			CodeContext:     CodeContext{FileContent: "package main"},
		}

		prompt, err := BuildFunctionalPrompt(request)
		if err != nil {
			t.Fatalf("BuildFunctionalPrompt failed: %v", err)
		}
		assertContains(t, prompt, "engineer")
	})

	t.Run("BuildTestingPrompt", func(t *testing.T) {
		request := ReviewRequest{
			TaskID:          "TASK-009",
			TaskDescription: "Test",
			CodeContext:     CodeContext{FileContent: "package main"},
		}

		prompt, err := BuildTestingPrompt(request)
		if err != nil {
			t.Fatalf("BuildTestingPrompt failed: %v", err)
		}
		assertContains(t, prompt, "QA engineer")
	})
}

func TestFluentAPI(t *testing.T) {
	request := NewReviewRequest(ReviewTypeArchitecture, "TASK-010", "Test fluent API")
	request = request.
		WithDiff("+func NewFunc() {}").
		WithAcceptanceCriteria("Must be clean", "Must follow patterns").
		WithAdditionalContext("This is important").
		WithoutVote()

	if request.Type != ReviewTypeArchitecture {
		t.Error("Type not set correctly")
	}
	if request.TaskID != "TASK-010" {
		t.Error("TaskID not set correctly")
	}
	if len(request.AcceptanceCriteria) != 2 {
		t.Error("Acceptance criteria not set correctly")
	}
	if request.AdditionalContext != "This is important" {
		t.Error("Additional context not set correctly")
	}
	if request.RequireVote {
		t.Error("Vote should be disabled")
	}
	if request.CodeContext.Diff != "+func NewFunc() {}" {
		t.Error("Diff not set correctly")
	}
}

// Helper functions

func assertContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("Expected to find %q in output", needle)
	}
}

func assertNotContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if strings.Contains(haystack, needle) {
		t.Errorf("Did not expect to find %q in output", needle)
	}
}
