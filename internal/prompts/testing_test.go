package prompts

import (
	"strings"
	"testing"
)

func TestNewTestingReviewBuilder(t *testing.T) {
	builder := NewTestingReviewBuilder()
	if builder == nil {
		t.Fatal("NewTestingReviewBuilder should return a non-nil builder")
	}
}

func TestTestingReviewBuilder_GetReviewType(t *testing.T) {
	builder := NewTestingReviewBuilder()
	reviewType := builder.GetReviewType()
	if reviewType != ReviewTypeTesting {
		t.Errorf("Expected ReviewTypeTesting, got %v", reviewType)
	}
}

func TestTestingReviewBuilder_Build_MinimalRequest(t *testing.T) {
	request := ReviewRequest{
		TaskID:          "TEST-001",
		TaskDescription: "Review unit tests for user service",
		CodeContext: CodeContext{
			FileContent: "package user_test\n\nfunc TestValidate(t *testing.T) {\n\t// test code\n}",
			Language:    "go",
		},
	}

	builder := NewTestingReviewBuilder()
	prompt, err := builder.Build(request)

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Check essential sections
	if !strings.Contains(prompt, "senior QA engineer") {
		t.Error("Prompt should contain role description")
	}
	if !strings.Contains(prompt, "testing specialist") {
		t.Error("Prompt should mention testing specialist")
	}
	if !strings.Contains(prompt, "Review Focus: Test Coverage") {
		t.Error("Prompt should contain review focus")
	}
	if !strings.Contains(prompt, "TEST-001") {
		t.Error("Prompt should contain task ID")
	}
	if !strings.Contains(prompt, "Review unit tests") {
		t.Error("Prompt should contain task description")
	}
	if !strings.Contains(prompt, "func TestValidate") {
		t.Error("Prompt should contain test code")
	}
}

func TestTestingReviewBuilder_Build_WithAcceptanceCriteria(t *testing.T) {
	request := ReviewRequest{
		TaskID:          "TEST-002",
		TaskDescription: "Review test coverage",
		AcceptanceCriteria: []string{
			"Must test all edge cases",
			"Should achieve 80% coverage",
			"Must test error paths",
		},
		CodeContext: CodeContext{
			FileContent: "func TestEdgeCases(t *testing.T) {}",
			Language:    "go",
		},
	}

	builder := NewTestingReviewBuilder()
	prompt, err := builder.Build(request)

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if !strings.Contains(prompt, "## Acceptance Criteria") {
		t.Error("Prompt should contain acceptance criteria section")
	}
	if !strings.Contains(prompt, "Must test all edge cases") {
		t.Error("Prompt should contain first criterion")
	}
	if !strings.Contains(prompt, "Should achieve 80% coverage") {
		t.Error("Prompt should contain second criterion")
	}
	if !strings.Contains(prompt, "Must test error paths") {
		t.Error("Prompt should contain third criterion")
	}
}

func TestTestingReviewBuilder_Build_WithDiff(t *testing.T) {
	request := ReviewRequest{
		TaskID:          "TEST-003",
		TaskDescription: "Review new test cases",
		CodeContext: CodeContext{
			FilePath: "internal/auth/auth_test.go",
			Diff: `@@ -20,3 +20,8 @@
 func TestLogin(t *testing.T) {
+    t.Run("empty credentials", func(t *testing.T) {
+        err := Login("", "")
+        assert.Error(t, err)
+    })
 }`,
			Language: "go",
		},
	}

	builder := NewTestingReviewBuilder()
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
	if !strings.Contains(prompt, "empty credentials") {
		t.Error("Prompt should contain diff content")
	}
	if !strings.Contains(prompt, "internal/auth/auth_test.go") {
		t.Error("Prompt should contain file path")
	}
}

func TestTestingReviewBuilder_Build_WithPackageName(t *testing.T) {
	request := ReviewRequest{
		TaskID:          "TEST-004",
		TaskDescription: "Review test suite",
		CodeContext: CodeContext{
			FilePath:    "internal/calc/calc_test.go",
			PackageName: "calc_test",
			FileContent: "package calc_test\n\nfunc TestAdd(t *testing.T) {}",
			Language:    "go",
		},
	}

	builder := NewTestingReviewBuilder()
	prompt, err := builder.Build(request)

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if !strings.Contains(prompt, "**Package**: `calc_test`") {
		t.Error("Prompt should contain package name")
	}
}

func TestTestingReviewBuilder_Build_WithSurroundingCode(t *testing.T) {
	request := ReviewRequest{
		TaskID:          "TEST-005",
		TaskDescription: "Review integration tests",
		CodeContext: CodeContext{
			FileContent:     "func TestIntegration(t *testing.T) {}",
			SurroundingCode: "type TestFixture struct {\n\tDB *sql.DB\n}\n\nfunc setupTest() *TestFixture {}",
			Language:        "go",
		},
	}

	builder := NewTestingReviewBuilder()
	prompt, err := builder.Build(request)

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if !strings.Contains(prompt, "### Related Code Context") {
		t.Error("Prompt should contain surrounding code section")
	}
	if !strings.Contains(prompt, "type TestFixture") {
		t.Error("Prompt should contain surrounding code content")
	}
}

func TestTestingReviewBuilder_Build_WithAdditionalContext(t *testing.T) {
	request := ReviewRequest{
		TaskID:            "TEST-006",
		TaskDescription:   "Review benchmark tests",
		AdditionalContext: "These benchmarks need to complete in under 1ms each.",
		CodeContext: CodeContext{
			FileContent: "func BenchmarkProcess(b *testing.B) {}",
			Language:    "go",
		},
	}

	builder := NewTestingReviewBuilder()
	prompt, err := builder.Build(request)

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if !strings.Contains(prompt, "## Additional Context") {
		t.Error("Prompt should contain additional context section")
	}
	if !strings.Contains(prompt, "under 1ms") {
		t.Error("Prompt should contain additional context content")
	}
}

func TestTestingReviewBuilder_Build_WithVoteRequired(t *testing.T) {
	request := ReviewRequest{
		TaskID:          "TEST-007",
		TaskDescription: "Review critical payment tests",
		RequireVote:     true,
		CodeContext: CodeContext{
			FileContent: "func TestPayment(t *testing.T) {}",
			Language:    "go",
		},
	}

	builder := NewTestingReviewBuilder()
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

func TestTestingReviewBuilder_Build_WithoutVoteRequired(t *testing.T) {
	request := ReviewRequest{
		TaskID:          "TEST-008",
		TaskDescription: "Review helper tests",
		RequireVote:     false,
		CodeContext: CodeContext{
			FileContent: "func TestHelper(t *testing.T) {}",
			Language:    "go",
		},
	}

	builder := NewTestingReviewBuilder()
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

func TestTestingReviewBuilder_Build_TestingReviewCriteria(t *testing.T) {
	request := ReviewRequest{
		TaskID:          "TEST-009",
		TaskDescription: "Review test quality",
		CodeContext: CodeContext{
			FileContent: "func TestSomething(t *testing.T) {}",
			Language:    "go",
		},
	}

	builder := NewTestingReviewBuilder()
	prompt, err := builder.Build(request)

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Check for testing criteria sections
	criteria := []string{
		"## Testing Review Criteria",
		"### Test Coverage",
		"critical code paths tested",
		"edge cases covered",
		"### Test Quality and Clarity",
		"tests clear and easy to understand",
		"### Assertions and Verification",
		"assertions specific and meaningful",
		"### TDD Principles",
		"Red-Green-Refactor",
		"### Go Testing Best Practices",
		"table-driven tests",
		"t.Run",
		"t.Parallel()",
		"### Test Reliability",
		"deterministic",
		"no flaky tests",
		"### Error Testing",
		"error cases thoroughly tested",
		"### Test Organization",
		"_test.go files",
	}

	for _, criterion := range criteria {
		if !strings.Contains(prompt, criterion) {
			t.Errorf("Prompt should contain '%s'", criterion)
		}
	}
}

func TestTestingReviewBuilder_Build_ReviewInstructions(t *testing.T) {
	request := ReviewRequest{
		TaskID:          "TEST-010",
		TaskDescription: "Review test suite",
		CodeContext: CodeContext{
			FileContent: "func TestAll(t *testing.T) {}",
			Language:    "go",
		},
	}

	builder := NewTestingReviewBuilder()
	prompt, err := builder.Build(request)

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Check for review instruction sections
	instructions := []string{
		"## Review Instructions",
		"**Coverage Assessment**",
		"**Quality Issues**",
		"Severity",
		"Category",
		"Location",
		"**Missing Tests**",
		"**Test Examples**",
		"**Recommendations**",
	}

	for _, instruction := range instructions {
		if !strings.Contains(prompt, instruction) {
			t.Errorf("Prompt should contain instruction '%s'", instruction)
		}
	}
}

func TestTestingReviewBuilder_Build_ValidationErrors(t *testing.T) {
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
					FileContent: "test code",
				},
			},
			expectedErr: "TaskID is required",
		},
		{
			name: "missing TaskDescription",
			request: ReviewRequest{
				TaskID: "TEST-999",
				CodeContext: CodeContext{
					FileContent: "test code",
				},
			},
			expectedErr: "TaskDescription is required",
		},
		{
			name: "missing both FileContent and Diff",
			request: ReviewRequest{
				TaskID:          "TEST-888",
				TaskDescription: "Some task",
				CodeContext:     CodeContext{},
			},
			expectedErr: "either FileContent or Diff must be provided",
		},
	}

	builder := NewTestingReviewBuilder()

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

func TestTestingReviewBuilder_Build_DiffPreferredOverContent(t *testing.T) {
	request := ReviewRequest{
		TaskID:          "TEST-111",
		TaskDescription: "Test diff preference",
		CodeContext: CodeContext{
			FileContent: "func TestOld(t *testing.T) {}",
			Diff:        "@@ -1,1 +1,1 @@\n-old\n+new",
			Language:    "go",
		},
	}

	builder := NewTestingReviewBuilder()
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

func TestTestingReviewBuilder_Build_GoTestingBestPractices(t *testing.T) {
	request := ReviewRequest{
		TaskID:          "TEST-200",
		TaskDescription: "Review Go test practices",
		CodeContext: CodeContext{
			FileContent: "func TestWithSubtests(t *testing.T) {\n\tt.Run(\"subtest\", func(t *testing.T) {})\n}",
			Language:    "go",
		},
	}

	builder := NewTestingReviewBuilder()
	prompt, err := builder.Build(request)

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Verify Go testing-specific criteria are present
	goCriteria := []string{
		"table-driven tests",
		"t.Run",
		"t.Parallel()",
		"benchmarks",
		"examples",
	}

	for _, criterion := range goCriteria {
		if !strings.Contains(prompt, criterion) {
			t.Errorf("Prompt should contain Go testing criterion: '%s'", criterion)
		}
	}
}

func TestTestingReviewBuilder_Build_TDDPrinciples(t *testing.T) {
	request := ReviewRequest{
		TaskID:          "TEST-300",
		TaskDescription: "Review TDD approach",
		CodeContext: CodeContext{
			FileContent: "func TestFeature(t *testing.T) {}",
			Language:    "go",
		},
	}

	builder := NewTestingReviewBuilder()
	prompt, err := builder.Build(request)

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Verify TDD principles are mentioned
	tddPrinciples := []string{
		"Red-Green-Refactor",
		"tests isolated and independent",
		"run in any order",
	}

	for _, principle := range tddPrinciples {
		if !strings.Contains(prompt, principle) {
			t.Errorf("Prompt should contain TDD principle: '%s'", principle)
		}
	}
}
