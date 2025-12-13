package prompts

import (
	"strings"
	"testing"
)

func TestNewArchitectureReviewBuilder(t *testing.T) {
	builder := NewArchitectureReviewBuilder()
	if builder == nil {
		t.Fatal("NewArchitectureReviewBuilder should return a non-nil builder")
	}
}

func TestArchitectureReviewBuilder_GetReviewType(t *testing.T) {
	builder := NewArchitectureReviewBuilder()
	reviewType := builder.GetReviewType()
	if reviewType != ReviewTypeArchitecture {
		t.Errorf("Expected ReviewTypeArchitecture, got %v", reviewType)
	}
}

func TestArchitectureReviewBuilder_Build_MinimalRequest(t *testing.T) {
	request := ReviewRequest{
		TaskID:          "TASK-123",
		TaskDescription: "Review user authentication module",
		CodeContext: CodeContext{
			FileContent: "package auth\n\nfunc Login() {}",
			Language:    "go",
		},
	}

	builder := NewArchitectureReviewBuilder()
	prompt, err := builder.Build(request)

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Check essential sections
	if !strings.Contains(prompt, "senior software architect") {
		t.Error("Prompt should contain role description")
	}
	if !strings.Contains(prompt, "Review Focus: Architecture") {
		t.Error("Prompt should contain review focus")
	}
	if !strings.Contains(prompt, "TASK-123") {
		t.Error("Prompt should contain task ID")
	}
	if !strings.Contains(prompt, "Review user authentication module") {
		t.Error("Prompt should contain task description")
	}
	if !strings.Contains(prompt, "package auth") {
		t.Error("Prompt should contain code content")
	}
}

func TestArchitectureReviewBuilder_Build_WithAcceptanceCriteria(t *testing.T) {
	request := ReviewRequest{
		TaskID:          "TASK-456",
		TaskDescription: "Add caching layer",
		AcceptanceCriteria: []string{
			"Must support cache invalidation",
			"Should handle concurrent access",
			"Must be testable",
		},
		CodeContext: CodeContext{
			FileContent: "package cache\n\ntype Cache struct {}",
			Language:    "go",
		},
	}

	builder := NewArchitectureReviewBuilder()
	prompt, err := builder.Build(request)

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if !strings.Contains(prompt, "## Acceptance Criteria") {
		t.Error("Prompt should contain acceptance criteria section")
	}
	if !strings.Contains(prompt, "Must support cache invalidation") {
		t.Error("Prompt should contain first criterion")
	}
	if !strings.Contains(prompt, "Should handle concurrent access") {
		t.Error("Prompt should contain second criterion")
	}
	if !strings.Contains(prompt, "Must be testable") {
		t.Error("Prompt should contain third criterion")
	}
}

func TestArchitectureReviewBuilder_Build_WithDiff(t *testing.T) {
	request := ReviewRequest{
		TaskID:          "TASK-789",
		TaskDescription: "Refactor database layer",
		CodeContext: CodeContext{
			FilePath: "internal/db/connection.go",
			Diff: `@@ -10,5 +10,7 @@
 func Connect() {
-    db.Open()
+    pool := db.NewPool()
+    pool.Connect()
 }`,
			Language: "go",
		},
	}

	builder := NewArchitectureReviewBuilder()
	prompt, err := builder.Build(request)

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if !strings.Contains(prompt, "### Changes (Git Diff)") {
		t.Error("Prompt should contain diff section header")
	}
	if !strings.Contains(prompt, "```diff") {
		t.Error("Prompt should use diff code block")
	}
	if !strings.Contains(prompt, "pool.Connect()") {
		t.Error("Prompt should contain diff content")
	}
	if !strings.Contains(prompt, "internal/db/connection.go") {
		t.Error("Prompt should contain file path")
	}
}

func TestArchitectureReviewBuilder_Build_WithPackageName(t *testing.T) {
	request := ReviewRequest{
		TaskID:          "TASK-101",
		TaskDescription: "Review service interface",
		CodeContext: CodeContext{
			FilePath:    "internal/service/interface.go",
			PackageName: "service",
			FileContent: "package service\n\ntype Service interface {}",
			Language:    "go",
		},
	}

	builder := NewArchitectureReviewBuilder()
	prompt, err := builder.Build(request)

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if !strings.Contains(prompt, "**Package**: `service`") {
		t.Error("Prompt should contain package name")
	}
}

func TestArchitectureReviewBuilder_Build_WithSurroundingCode(t *testing.T) {
	request := ReviewRequest{
		TaskID:          "TASK-202",
		TaskDescription: "Review handler implementation",
		CodeContext: CodeContext{
			FileContent:     "func HandleRequest() {}",
			SurroundingCode: "package handlers\n\ntype Context struct {}\n\nfunc NewContext() *Context {}",
			Language:        "go",
		},
	}

	builder := NewArchitectureReviewBuilder()
	prompt, err := builder.Build(request)

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if !strings.Contains(prompt, "### Related Code Context") {
		t.Error("Prompt should contain surrounding code section")
	}
	if !strings.Contains(prompt, "func NewContext()") {
		t.Error("Prompt should contain surrounding code content")
	}
}

func TestArchitectureReviewBuilder_Build_WithAdditionalContext(t *testing.T) {
	request := ReviewRequest{
		TaskID:            "TASK-303",
		TaskDescription:   "Review API design",
		AdditionalContext: "This API will be used by mobile clients with limited bandwidth.",
		CodeContext: CodeContext{
			FileContent: "package api\n\ntype Response struct {}",
			Language:    "go",
		},
	}

	builder := NewArchitectureReviewBuilder()
	prompt, err := builder.Build(request)

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if !strings.Contains(prompt, "## Additional Context") {
		t.Error("Prompt should contain additional context section")
	}
	if !strings.Contains(prompt, "limited bandwidth") {
		t.Error("Prompt should contain additional context content")
	}
}

func TestArchitectureReviewBuilder_Build_WithVoteRequired(t *testing.T) {
	request := ReviewRequest{
		TaskID:          "TASK-404",
		TaskDescription: "Review critical security module",
		RequireVote:     true,
		CodeContext: CodeContext{
			FileContent: "package security\n\nfunc Encrypt() {}",
			Language:    "go",
		},
	}

	builder := NewArchitectureReviewBuilder()
	prompt, err := builder.Build(request)

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if !strings.Contains(prompt, "## Vote Required") {
		t.Error("Prompt should contain vote required section")
	}
	if !strings.Contains(prompt, "VOTE: APPROVE") {
		t.Error("Prompt should mention APPROVE vote option")
	}
	if !strings.Contains(prompt, "VOTE: REQUEST_CHANGE") {
		t.Error("Prompt should mention REQUEST_CHANGE vote option")
	}
	if !strings.Contains(prompt, "VOTE: REJECT") {
		t.Error("Prompt should mention REJECT vote option")
	}
}

func TestArchitectureReviewBuilder_Build_WithoutVoteRequired(t *testing.T) {
	request := ReviewRequest{
		TaskID:          "TASK-505",
		TaskDescription: "Review utility functions",
		RequireVote:     false,
		CodeContext: CodeContext{
			FileContent: "package util\n\nfunc Helper() {}",
			Language:    "go",
		},
	}

	builder := NewArchitectureReviewBuilder()
	prompt, err := builder.Build(request)

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if strings.Contains(prompt, "## Vote Required") {
		t.Error("Prompt should NOT contain vote required section when vote not required")
	}
	if !strings.Contains(prompt, "without a formal vote") {
		t.Error("Prompt should indicate no vote required")
	}
}

func TestArchitectureReviewBuilder_Build_ArchitectureReviewCriteria(t *testing.T) {
	request := ReviewRequest{
		TaskID:          "TASK-606",
		TaskDescription: "Review architecture",
		CodeContext: CodeContext{
			FileContent: "package main\n\nfunc main() {}",
			Language:    "go",
		},
	}

	builder := NewArchitectureReviewBuilder()
	prompt, err := builder.Build(request)

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Check for architecture criteria sections
	criteria := []string{
		"## Architecture Review Criteria",
		"### Design Patterns and Structure",
		"### SOLID Principles",
		"Single Responsibility",
		"Open/Closed",
		"Liskov Substitution",
		"Interface Segregation",
		"Dependency Inversion",
		"### Maintainability and Extensibility",
		"### Integration and Consistency",
		"### Scalability and Performance Considerations",
		"### Code Smells and Anti-Patterns",
	}

	for _, criterion := range criteria {
		if !strings.Contains(prompt, criterion) {
			t.Errorf("Prompt should contain '%s'", criterion)
		}
	}
}

func TestArchitectureReviewBuilder_Build_ReviewInstructions(t *testing.T) {
	request := ReviewRequest{
		TaskID:          "TASK-707",
		TaskDescription: "Review code structure",
		CodeContext: CodeContext{
			FileContent: "package test\n\nfunc Test() {}",
			Language:    "go",
		},
	}

	builder := NewArchitectureReviewBuilder()
	prompt, err := builder.Build(request)

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Check for review instruction sections
	instructions := []string{
		"## Review Instructions",
		"**Strengths**",
		"**Issues**",
		"Severity",
		"Category",
		"Location",
		"**Recommendations**",
		"**Trade-offs**",
	}

	for _, instruction := range instructions {
		if !strings.Contains(prompt, instruction) {
			t.Errorf("Prompt should contain instruction '%s'", instruction)
		}
	}
}

func TestArchitectureReviewBuilder_Build_ValidationErrors(t *testing.T) {
	testCases := []struct {
		name        string
		request     ReviewRequest
		expectedErr string
	}{
		{
			name: "missing TaskID",
			request: ReviewRequest{
				TaskDescription: "Some task",
				CodeContext: CodeContext{
					FileContent: "code",
				},
			},
			expectedErr: "TaskID is required",
		},
		{
			name: "missing TaskDescription",
			request: ReviewRequest{
				TaskID: "TASK-999",
				CodeContext: CodeContext{
					FileContent: "code",
				},
			},
			expectedErr: "TaskDescription is required",
		},
		{
			name: "missing both FileContent and Diff",
			request: ReviewRequest{
				TaskID:          "TASK-888",
				TaskDescription: "Some task",
				CodeContext:     CodeContext{},
			},
			expectedErr: "either FileContent or Diff must be provided",
		},
	}

	builder := NewArchitectureReviewBuilder()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := builder.Build(testCase.request)
			if err == nil {
				t.Errorf("Expected error '%s', but got nil", testCase.expectedErr)
			} else if !strings.Contains(err.Error(), testCase.expectedErr) {
				t.Errorf("Expected error containing '%s', got '%s'", testCase.expectedErr, err.Error())
			}
		})
	}
}

func TestArchitectureReviewBuilder_Build_DiffPreferredOverContent(t *testing.T) {
	request := ReviewRequest{
		TaskID:          "TASK-111",
		TaskDescription: "Test diff preference",
		CodeContext: CodeContext{
			FileContent: "package old\n\nfunc Old() {}",
			Diff:        "@@ -1,1 +1,1 @@\n-old\n+new",
			Language:    "go",
		},
	}

	builder := NewArchitectureReviewBuilder()
	prompt, err := builder.Build(request)

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// When both are provided, diff should be preferred
	if !strings.Contains(prompt, "### Changes (Git Diff)") {
		t.Error("Prompt should show diff when both diff and content are provided")
	}
	if strings.Contains(prompt, "### Code\n```go") {
		t.Error("Prompt should NOT show full code section when diff is provided")
	}
}
