package prompts

import (
	"strings"
	"testing"
)

func TestNewFunctionalReviewBuilder(t *testing.T) {
	builder := NewFunctionalReviewBuilder()
	if builder == nil {
		t.Fatal("NewFunctionalReviewBuilder should return a non-nil builder")
	}
}

func TestFunctionalReviewBuilder_GetReviewType(t *testing.T) {
	builder := NewFunctionalReviewBuilder()
	reviewType := builder.GetReviewType()
	if reviewType != ReviewTypeFunctional {
		t.Errorf("Expected ReviewTypeFunctional, got %v", reviewType)
	}
}

func TestFunctionalReviewBuilder_Build_MinimalRequest(t *testing.T) {
	request := ReviewRequest{
		TaskID:          "FUNC-001",
		TaskDescription: "Implement user validation logic",
		CodeContext: CodeContext{
			FileContent: "package user\n\nfunc Validate(u User) error {\n\treturn nil\n}",
			Language:    "go",
		},
	}

	builder := NewFunctionalReviewBuilder()
	prompt, err := builder.Build(request)

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Check essential sections
	if !strings.Contains(prompt, "senior software engineer") {
		t.Error("Prompt should contain role description")
	}
	if !strings.Contains(prompt, "functional correctness reviews") {
		t.Error("Prompt should mention functional correctness")
	}
	if !strings.Contains(prompt, "Review Focus: Business Logic") {
		t.Error("Prompt should contain review focus")
	}
	if !strings.Contains(prompt, "FUNC-001") {
		t.Error("Prompt should contain task ID")
	}
	if !strings.Contains(prompt, "Implement user validation logic") {
		t.Error("Prompt should contain task description")
	}
	if !strings.Contains(prompt, "func Validate") {
		t.Error("Prompt should contain code content")
	}
}

func TestFunctionalReviewBuilder_Build_WithAcceptanceCriteria(t *testing.T) {
	request := ReviewRequest{
		TaskID:          "FUNC-002",
		TaskDescription: "Add email validation",
		AcceptanceCriteria: []string{
			"Must validate email format",
			"Should reject empty emails",
			"Must handle international domains",
		},
		CodeContext: CodeContext{
			FileContent: "package validator\n\nfunc ValidateEmail(email string) bool {}",
			Language:    "go",
		},
	}

	builder := NewFunctionalReviewBuilder()
	prompt, err := builder.Build(request)

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if !strings.Contains(prompt, "## Acceptance Criteria") {
		t.Error("Prompt should contain acceptance criteria section")
	}
	if !strings.Contains(prompt, "Must validate email format") {
		t.Error("Prompt should contain first criterion")
	}
	if !strings.Contains(prompt, "Should reject empty emails") {
		t.Error("Prompt should contain second criterion")
	}
	if !strings.Contains(prompt, "Must handle international domains") {
		t.Error("Prompt should contain third criterion")
	}
}

func TestFunctionalReviewBuilder_Build_WithDiff(t *testing.T) {
	request := ReviewRequest{
		TaskID:          "FUNC-003",
		TaskDescription: "Fix password validation bug",
		CodeContext: CodeContext{
			FilePath: "internal/auth/password.go",
			Diff: `@@ -15,3 +15,5 @@
 func ValidatePassword(pwd string) bool {
-    return len(pwd) > 6
+    minLen := 8
+    return len(pwd) >= minLen && hasSpecialChar(pwd)
 }`,
			Language: "go",
		},
	}

	builder := NewFunctionalReviewBuilder()
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
	if !strings.Contains(prompt, "hasSpecialChar(pwd)") {
		t.Error("Prompt should contain diff content")
	}
	if !strings.Contains(prompt, "internal/auth/password.go") {
		t.Error("Prompt should contain file path")
	}
}

func TestFunctionalReviewBuilder_Build_WithPackageName(t *testing.T) {
	request := ReviewRequest{
		TaskID:          "FUNC-004",
		TaskDescription: "Review calculation logic",
		CodeContext: CodeContext{
			FilePath:    "internal/calc/math.go",
			PackageName: "calc",
			FileContent: "package calc\n\nfunc Add(a, b int) int { return a + b }",
			Language:    "go",
		},
	}

	builder := NewFunctionalReviewBuilder()
	prompt, err := builder.Build(request)

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if !strings.Contains(prompt, "**Package**: `calc`") {
		t.Error("Prompt should contain package name")
	}
}

func TestFunctionalReviewBuilder_Build_WithSurroundingCode(t *testing.T) {
	request := ReviewRequest{
		TaskID:          "FUNC-005",
		TaskDescription: "Review data processing function",
		CodeContext: CodeContext{
			FileContent:     "func Process(data []byte) (Result, error) {}",
			SurroundingCode: "type Result struct {\n\tStatus string\n\tData []byte\n}",
			Language:        "go",
		},
	}

	builder := NewFunctionalReviewBuilder()
	prompt, err := builder.Build(request)

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if !strings.Contains(prompt, "### Related Code Context") {
		t.Error("Prompt should contain surrounding code section")
	}
	if !strings.Contains(prompt, "type Result struct") {
		t.Error("Prompt should contain surrounding code content")
	}
}

func TestFunctionalReviewBuilder_Build_WithAdditionalContext(t *testing.T) {
	request := ReviewRequest{
		TaskID:            "FUNC-006",
		TaskDescription:   "Review rate limiting logic",
		AdditionalContext: "This function must handle high concurrency (10k+ req/s).",
		CodeContext: CodeContext{
			FileContent: "func RateLimit() bool { return true }",
			Language:    "go",
		},
	}

	builder := NewFunctionalReviewBuilder()
	prompt, err := builder.Build(request)

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if !strings.Contains(prompt, "## Additional Context") {
		t.Error("Prompt should contain additional context section")
	}
	if !strings.Contains(prompt, "high concurrency") {
		t.Error("Prompt should contain additional context content")
	}
}

func TestFunctionalReviewBuilder_Build_WithVoteRequired(t *testing.T) {
	request := ReviewRequest{
		TaskID:          "FUNC-007",
		TaskDescription: "Review critical payment processing logic",
		RequireVote:     true,
		CodeContext: CodeContext{
			FileContent: "func ProcessPayment(amount float64) error {}",
			Language:    "go",
		},
	}

	builder := NewFunctionalReviewBuilder()
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

func TestFunctionalReviewBuilder_Build_WithoutVoteRequired(t *testing.T) {
	request := ReviewRequest{
		TaskID:          "FUNC-008",
		TaskDescription: "Review helper utility",
		RequireVote:     false,
		CodeContext: CodeContext{
			FileContent: "func FormatString(s string) string {}",
			Language:    "go",
		},
	}

	builder := NewFunctionalReviewBuilder()
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

func TestFunctionalReviewBuilder_Build_FunctionalReviewCriteria(t *testing.T) {
	request := ReviewRequest{
		TaskID:          "FUNC-009",
		TaskDescription: "Review business logic",
		CodeContext: CodeContext{
			FileContent: "func Calculate() int { return 42 }",
			Language:    "go",
		},
	}

	builder := NewFunctionalReviewBuilder()
	prompt, err := builder.Build(request)

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Check for functional criteria sections
	criteria := []string{
		"## Functional Review Criteria",
		"### Requirements Compliance",
		"acceptance criteria satisfied",
		"### Logic Correctness",
		"algorithm/logic correct",
		"### Edge Cases and Error Handling",
		"nil/null/empty inputs",
		"### Data Integrity and Validation",
		"input validation",
		"### Go Best Practices",
		"Go idioms",
		"error handling done the Go way",
		"### Resource Management",
		"defer statements",
		"### Potential Bugs",
		"obvious bugs",
	}

	for _, criterion := range criteria {
		if !strings.Contains(prompt, criterion) {
			t.Errorf("Prompt should contain '%s'", criterion)
		}
	}
}

func TestFunctionalReviewBuilder_Build_ReviewInstructions(t *testing.T) {
	request := ReviewRequest{
		TaskID:          "FUNC-010",
		TaskDescription: "Review core function",
		CodeContext: CodeContext{
			FileContent: "func Core() {}",
			Language:    "go",
		},
	}

	builder := NewFunctionalReviewBuilder()
	prompt, err := builder.Build(request)

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Check for review instruction sections
	instructions := []string{
		"## Review Instructions",
		"**Correctness Assessment**",
		"**Issues Found**",
		"Severity",
		"Category",
		"Location",
		"**Missing Functionality**",
		"**Edge Cases**",
		"**Recommendations**",
	}

	for _, instruction := range instructions {
		if !strings.Contains(prompt, instruction) {
			t.Errorf("Prompt should contain instruction '%s'", instruction)
		}
	}
}

func TestFunctionalReviewBuilder_Build_ValidationErrors(t *testing.T) {
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
				TaskID: "FUNC-999",
				CodeContext: CodeContext{
					FileContent: "code",
				},
			},
			expectedErr: "TaskDescription is required",
		},
		{
			name: "missing both FileContent and Diff",
			request: ReviewRequest{
				TaskID:          "FUNC-888",
				TaskDescription: "Some task",
				CodeContext:     CodeContext{},
			},
			expectedErr: "either FileContent or Diff must be provided",
		},
	}

	builder := NewFunctionalReviewBuilder()

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

func TestFunctionalReviewBuilder_Build_DiffPreferredOverContent(t *testing.T) {
	request := ReviewRequest{
		TaskID:          "FUNC-111",
		TaskDescription: "Test diff preference",
		CodeContext: CodeContext{
			FileContent: "func Old() {}",
			Diff:        "@@ -1,1 +1,1 @@\n-old\n+new",
			Language:    "go",
		},
	}

	builder := NewFunctionalReviewBuilder()
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

func TestFunctionalReviewBuilder_Build_GoBestPractices(t *testing.T) {
	request := ReviewRequest{
		TaskID:          "FUNC-200",
		TaskDescription: "Review Go code",
		CodeContext: CodeContext{
			FileContent: "func DoWork(ctx context.Context) error { return nil }",
			Language:    "go",
		},
	}

	builder := NewFunctionalReviewBuilder()
	prompt, err := builder.Build(request)

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Verify Go-specific criteria are present
	goCriteria := []string{
		"goroutines and channels",
		"context properly propagated",
		"defer, panic, and recover",
		"nil checks",
		"multiple return values",
	}

	for _, criterion := range goCriteria {
		if !strings.Contains(prompt, criterion) {
			t.Errorf("Prompt should contain Go-specific criterion: '%s'", criterion)
		}
	}
}
