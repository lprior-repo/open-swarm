// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package prompts

import (
	"strings"
	"testing"
)

func TestNewTestGenerationPromptBuilder(t *testing.T) {
	request := &TestGenerationRequest{
		TaskDescription: "Write tests for user authentication",
		OutputPath:      "auth_test.go",
	}

	builder := NewTestGenerationPromptBuilder(request)

	if builder == nil {
		t.Fatal("NewTestGenerationPromptBuilder returned nil")
	}

	if builder.request != request {
		t.Error("Builder should contain the request")
	}

	if request.Language == "" {
		t.Error("Language should be set to default")
	}

	if request.TestFramework == "" {
		t.Error("TestFramework should be set to default")
	}
}

func TestTestGenerationPromptBuilder_Build_Initial(t *testing.T) {
	request := &TestGenerationRequest{
		Mode:            TestModeInitial,
		TaskDescription: "Write unit tests for user validation",
		OutputPath:      "validation_test.go",
		Language:        "Go",
		TestFramework:   "testing",
		AcceptanceCriteria: &TestGenerationCriteria{
			Required: []string{
				"Test valid user input",
				"Test invalid email format",
				"Minimum password length validation",
			},
			Nice: []string{
				"Test password strength requirements",
			},
			Constraints: []string{
				"Must complete in < 100ms",
				"No external dependencies",
			},
		},
	}

	builder := NewTestGenerationPromptBuilder(request)
	prompt := builder.Build()

	// Check header
	if !strings.Contains(prompt, "# Test Generation Request") {
		t.Error("Prompt should contain main header")
	}

	// Check metadata
	if !strings.Contains(prompt, "Mode") {
		t.Error("Prompt should contain mode metadata")
	}
	if !strings.Contains(prompt, "initial") {
		t.Error("Prompt should contain initial mode")
	}

	// Check task description
	if !strings.Contains(prompt, "## Task Description") {
		t.Error("Prompt should contain task description section")
	}
	if !strings.Contains(prompt, "Write unit tests for user validation") {
		t.Error("Prompt should contain task description text")
	}

	// Check acceptance criteria
	if !strings.Contains(prompt, "## Acceptance Criteria") {
		t.Error("Prompt should contain acceptance criteria section")
	}
	if !strings.Contains(prompt, "### Required") {
		t.Error("Prompt should contain required criteria subsection")
	}
	if !strings.Contains(prompt, "Test valid user input") {
		t.Error("Prompt should contain required criteria")
	}
	if !strings.Contains(prompt, "### Constraints") {
		t.Error("Prompt should contain constraints subsection")
	}

	// Check output instructions
	if !strings.Contains(prompt, "## Output Instructions") {
		t.Error("Prompt should contain output instructions section")
	}
	if !strings.Contains(prompt, "validation_test.go") {
		t.Error("Prompt should contain output path")
	}

	// Check guidelines
	if !strings.Contains(prompt, "## Generation Guidelines") {
		t.Error("Prompt should contain generation guidelines section")
	}
	if !strings.Contains(prompt, "Basic functionality (happy path)") {
		t.Error("Prompt should contain initial mode guidelines")
	}
}

func TestTestGenerationPromptBuilder_Build_Refinement(t *testing.T) {
	request := &TestGenerationRequest{
		Mode:            TestModeRefinement,
		TaskDescription: "Fix lint issues in test file",
		OutputPath:      "user_test.go",
		Language:        "Go",
		TestFramework:   "testing",
		LintFeedback: &LintFeedback{
			Summary: "Multiple test functions missing test prefix",
			Errors: []string{
				"TestUserCreate: missing error handling test",
				"TestUserDelete: insufficient assertion coverage",
			},
			Warnings: []string{
				"Mock setup could be simplified",
			},
		},
	}

	builder := NewTestGenerationPromptBuilder(request)
	prompt := builder.Build()

	// Check refinement mode
	if !strings.Contains(prompt, "refinement") {
		t.Error("Prompt should contain refinement mode")
	}

	// Check lint feedback section
	if !strings.Contains(prompt, "## Lint Feedback") {
		t.Error("Prompt should contain lint feedback section")
	}
	if !strings.Contains(prompt, "### Summary") {
		t.Error("Prompt should contain lint summary")
	}
	if !strings.Contains(prompt, "Multiple test functions missing test prefix") {
		t.Error("Prompt should contain lint summary text")
	}
	if !strings.Contains(prompt, "### Errors") {
		t.Error("Prompt should contain errors subsection")
	}
	if !strings.Contains(prompt, "TestUserCreate: missing error handling test") {
		t.Error("Prompt should contain lint error")
	}
	if !strings.Contains(prompt, "### Warnings") {
		t.Error("Prompt should contain warnings subsection")
	}

	// Check refinement guidelines
	if !strings.Contains(prompt, "Refine tests based on lint feedback") {
		t.Error("Prompt should contain refinement guidelines")
	}
}

func TestTestGenerationPromptBuilder_Build_Enhancement(t *testing.T) {
	request := &TestGenerationRequest{
		Mode:            TestModeEnhancement,
		TaskDescription: "Improve code coverage for database package",
		OutputPath:      "db_test.go",
		Language:        "Go",
		TestFramework:   "testing",
		CoverageReport: &CoverageReport{
			TotalCoverage: 65.5,
			UncoveredFunctions: []string{
				"QueryBuilder.WithIndex",
				"Transaction.Rollback",
				"Connection.Close",
			},
			Report: "Total coverage: 65.50%\ndb/query.go:45:12\ndb/transaction.go:120:8",
		},
	}

	builder := NewTestGenerationPromptBuilder(request)
	prompt := builder.Build()

	// Check enhancement mode
	if !strings.Contains(prompt, "enhancement") {
		t.Error("Prompt should contain enhancement mode")
	}

	// Check coverage report section
	if !strings.Contains(prompt, "## Coverage Report") {
		t.Error("Prompt should contain coverage report section")
	}
	if !strings.Contains(prompt, "### Current Coverage") {
		t.Error("Prompt should contain current coverage subsection")
	}
	if !strings.Contains(prompt, "65.50%") {
		t.Error("Prompt should contain coverage percentage")
	}
	if !strings.Contains(prompt, "### Uncovered Functions") {
		t.Error("Prompt should contain uncovered functions subsection")
	}
	if !strings.Contains(prompt, "QueryBuilder.WithIndex") {
		t.Error("Prompt should contain uncovered function names")
	}

	// Check enhancement guidelines
	if !strings.Contains(prompt, "Enhance tests to improve coverage") {
		t.Error("Prompt should contain enhancement guidelines")
	}
}

func TestTestGenerationPromptBuilder_Build_WithCodeContext(t *testing.T) {
	request := &TestGenerationRequest{
		Mode:            TestModeInitial,
		TaskDescription: "Write tests for validation",
		OutputPath:      "validation_test.go",
		Language:        "Go",
		TestFramework:   "testing",
		CodeContext: &TestCodeContext{
			FilePath: "pkg/user/validation.go",
			Excerpt: `func ValidateEmail(email string) error {
	if !strings.Contains(email, "@") {
		return ErrInvalidEmail
	}
	return nil
}`,
			Language: "go",
		},
	}

	builder := NewTestGenerationPromptBuilder(request)
	prompt := builder.Build()

	// Check code context section
	if !strings.Contains(prompt, "## Code Context") {
		t.Error("Prompt should contain code context section")
	}
	if !strings.Contains(prompt, "### Relevant Code") {
		t.Error("Prompt should contain relevant code subsection")
	}
	if !strings.Contains(prompt, "ValidateEmail") {
		t.Error("Prompt should contain code excerpt")
	}
}

func TestNewTestGenerationBuilder(t *testing.T) {
	builder := NewTestGenerationBuilder()

	if builder == nil {
		t.Fatal("NewTestGenerationBuilder returned nil")
	}

	if builder.request.Mode != TestModeInitial {
		t.Error("Default mode should be initial")
	}

	if builder.request.Language != "Go" {
		t.Error("Default language should be Go")
	}

	if builder.request.TestFramework != "testing" {
		t.Error("Default framework should be testing")
	}
}

func TestTestGenerationBuilder_FluentAPI(t *testing.T) {
	prompt := NewTestGenerationBuilder().
		WithMode(TestModeInitial).
		WithTaskDescription("Write parser tests").
		WithOutputPath("parser_test.go").
		WithLanguage("Go").
		WithTestFramework("testing").
		AddRequiredCriteria("Parse valid JSON").
		AddRequiredCriteria("Reject invalid JSON").
		AddNiceCriteria("Handle large files efficiently").
		AddConstraint("Must be deterministic").
		WithAdditionalNotes("Focus on error recovery").
		BuildPrompt()

	// Verify prompt contains all elements
	if !strings.Contains(prompt, "Write parser tests") {
		t.Error("Prompt should contain task description")
	}
	if !strings.Contains(prompt, "parser_test.go") {
		t.Error("Prompt should contain output path")
	}
	if !strings.Contains(prompt, "Parse valid JSON") {
		t.Error("Prompt should contain required criteria")
	}
	if !strings.Contains(prompt, "Handle large files efficiently") {
		t.Error("Prompt should contain nice criteria")
	}
	if !strings.Contains(prompt, "Must be deterministic") {
		t.Error("Prompt should contain constraint")
	}
	if !strings.Contains(prompt, "Focus on error recovery") {
		t.Error("Prompt should contain additional notes")
	}
}

func TestTestGenerationBuilder_Build(t *testing.T) {
	builder := NewTestGenerationBuilder().
		WithMode(TestModeRefinement).
		WithTaskDescription("Refine storage tests").
		WithOutputPath("storage_test.go")

	request := builder.Build()

	if request == nil {
		t.Fatal("Build returned nil")
	}
	if request.Mode != TestModeRefinement {
		t.Error("Mode should be set to refinement")
	}
	if request.TaskDescription != "Refine storage tests" {
		t.Error("Task description should be set")
	}
	if request.OutputPath != "storage_test.go" {
		t.Error("Output path should be set")
	}
}

func TestTestGenerationBuilder_WithLintFeedback(t *testing.T) {
	builder := NewTestGenerationBuilder().
		WithMode(TestModeRefinement).
		WithTaskDescription("Fix tests").
		WithOutputPath("test.go").
		AddLintError("Missing test cases").
		AddLintError("Inconsistent naming").
		AddLintWarning("Slow test execution").
		WithLintSummary("2 errors, 1 warning found")

	request := builder.Build()

	if request.LintFeedback == nil {
		t.Fatal("LintFeedback should not be nil")
	}
	if len(request.LintFeedback.Errors) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(request.LintFeedback.Errors))
	}
	if len(request.LintFeedback.Warnings) != 1 {
		t.Errorf("Expected 1 warning, got %d", len(request.LintFeedback.Warnings))
	}
	if request.LintFeedback.Summary != "2 errors, 1 warning found" {
		t.Error("Summary should be set correctly")
	}
}

func TestTestGenerationBuilder_WithCodeContext(t *testing.T) {
	builder := NewTestGenerationBuilder().
		WithTaskDescription("Test storage").
		WithOutputPath("storage_test.go").
		WithCodeContextFromFile("storage/storage.go", "package storage\n\nfunc Store(key string, value interface{}) error {\n\treturn nil\n}").
		WithCodeExcerpt("func Store(key string, value interface{}) error {")

	request := builder.Build()

	if request.CodeContext == nil {
		t.Fatal("CodeContext should not be nil")
	}
	if request.CodeContext.FilePath != "storage/storage.go" {
		t.Error("FilePath should be set")
	}
	if !strings.Contains(request.CodeContext.Content, "func Store") {
		t.Error("Content should be set")
	}
	if request.CodeContext.Excerpt != "func Store(key string, value interface{}) error {" {
		t.Error("Excerpt should be set")
	}
}

func TestTestGenerationBuilder_WithCoverageReport(t *testing.T) {
	report := &CoverageReport{
		TotalCoverage: 75.5,
		UncoveredFunctions: []string{"Helper", "Cleanup"},
	}

	builder := NewTestGenerationBuilder().
		WithMode(TestModeEnhancement).
		WithTaskDescription("Improve coverage").
		WithOutputPath("api_test.go").
		WithCoverageReport(report)

	request := builder.Build()

	if request.CoverageReport == nil {
		t.Fatal("CoverageReport should not be nil")
	}
	if request.CoverageReport.TotalCoverage != 75.5 {
		t.Error("Coverage should be set correctly")
	}
	if len(request.CoverageReport.UncoveredFunctions) != 2 {
		t.Error("Should have 2 uncovered functions")
	}
}
