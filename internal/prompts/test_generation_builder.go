// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package prompts

import (
	"fmt"
	"strings"
	"time"
)

// TestGenerationMode indicates the mode of test generation
type TestGenerationMode string

const (
	// TestModeInitial generates tests from a fresh task description
	TestModeInitial TestGenerationMode = "initial"
	// TestModeRefinement refines existing tests based on lint feedback
	TestModeRefinement TestGenerationMode = "refinement"
	// TestModeEnhancement enhances tests based on coverage reports
	TestModeEnhancement TestGenerationMode = "enhancement"
)

// TestGenerationCriteria defines acceptance criteria for test generation
type TestGenerationCriteria struct {
	// Required defines mandatory acceptance criteria
	Required []string
	// Nice defines optional acceptance criteria
	Nice []string
	// Constraints defines any constraints or limitations
	Constraints []string
}

// LintFeedback represents lint errors and warnings
type LintFeedback struct {
	// Errors are critical lint violations
	Errors []string
	// Warnings are non-critical lint violations
	Warnings []string
	// Summary is a brief summary of lint issues
	Summary string
}

// CoverageReport represents code coverage information
type CoverageReport struct {
	// TotalCoverage is the overall coverage percentage
	TotalCoverage float64
	// UncoveredLines lists line numbers not covered by tests
	UncoveredLines []int
	// UncoveredFunctions lists functions without test coverage
	UncoveredFunctions []string
	// Report is the raw coverage report content
	Report string
}

// TestCodeContext provides relevant source code context
type TestCodeContext struct {
	// FilePath is the path to the source file
	FilePath string
	// Content is the source code content
	Content string
	// Excerpt is a relevant excerpt (e.g., function being tested)
	Excerpt string
	// Language is the programming language (e.g., "go", "python")
	Language string
}

// TestGenerationRequest encapsulates all parameters for test generation
type TestGenerationRequest struct {
	// Mode indicates the type of test generation
	Mode TestGenerationMode
	// TaskDescription describes what needs to be tested
	TaskDescription string
	// AcceptanceCriteria defines requirements
	AcceptanceCriteria *TestGenerationCriteria
	// OutputPath specifies where tests should be written
	OutputPath string
	// LintFeedback provides feedback from linting (optional)
	LintFeedback *LintFeedback
	// CoverageReport provides coverage information (optional)
	CoverageReport *CoverageReport
	// CodeContext provides source code context (optional)
	CodeContext *TestCodeContext
	// AdditionalNotes are any extra instructions
	AdditionalNotes string
	// Language is the programming language (default: "Go")
	Language string
	// TestFramework is the testing framework to use (default: "testing")
	TestFramework string
	// GeneratedAt is when the request was created
	GeneratedAt time.Time
}

// TestGenerationPromptBuilder builds test generation prompts
type TestGenerationPromptBuilder struct {
	request *TestGenerationRequest
}

// NewTestGenerationPromptBuilder creates a new TestGenerationPromptBuilder
func NewTestGenerationPromptBuilder(request *TestGenerationRequest) *TestGenerationPromptBuilder {
	if request.GeneratedAt.IsZero() {
		request.GeneratedAt = time.Now()
	}
	if request.Language == "" {
		request.Language = "Go"
	}
	if request.TestFramework == "" {
		request.TestFramework = "testing"
	}
	if request.AcceptanceCriteria == nil {
		request.AcceptanceCriteria = &TestGenerationCriteria{}
	}
	return &TestGenerationPromptBuilder{request: request}
}

// Build generates the complete prompt string for test generation
func (b *TestGenerationPromptBuilder) Build() string {
	var sb strings.Builder

	// Add header with context
	sb.WriteString("# Test Generation Request\n\n")
	sb.WriteString(fmt.Sprintf("**Mode:** %s\n", string(b.request.Mode)))
	sb.WriteString(fmt.Sprintf("**Language:** %s\n", b.request.Language))
	sb.WriteString(fmt.Sprintf("**Framework:** %s\n\n", b.request.TestFramework))

	// Add task description
	sb.WriteString("## Task Description\n\n")
	sb.WriteString(b.request.TaskDescription)
	sb.WriteString("\n\n")

	// Add acceptance criteria if present
	if b.request.AcceptanceCriteria != nil && len(b.request.AcceptanceCriteria.Required) > 0 {
		b.buildAcceptanceCriteria(&sb)
	}

	// Add code context if present
	if b.request.CodeContext != nil {
		b.buildCodeContext(&sb)
	}

	// Add lint feedback if present
	if b.request.LintFeedback != nil && (len(b.request.LintFeedback.Errors) > 0 || len(b.request.LintFeedback.Warnings) > 0) {
		b.buildLintFeedback(&sb)
	}

	// Add coverage report if present
	if b.request.CoverageReport != nil {
		b.buildCoverageReport(&sb)
	}

	// Add output path instructions
	b.buildOutputInstructions(&sb)

	// Add mode-specific guidelines
	b.buildModeGuidelines(&sb)

	// Add additional notes if present
	if b.request.AdditionalNotes != "" {
		sb.WriteString("## Additional Notes\n\n")
		sb.WriteString(b.request.AdditionalNotes)
		sb.WriteString("\n\n")
	}

	// Add footer with generation details
	sb.WriteString(fmt.Sprintf("<!-- Request generated on %s -->\n", b.request.GeneratedAt.Format(time.RFC3339)))

	return sb.String()
}

func (b *TestGenerationPromptBuilder) buildAcceptanceCriteria(sb *strings.Builder) {
	sb.WriteString("## Acceptance Criteria\n\n")

	if len(b.request.AcceptanceCriteria.Required) > 0 {
		sb.WriteString("### Required\n\n")
		for _, criteria := range b.request.AcceptanceCriteria.Required {
			sb.WriteString(fmt.Sprintf("- %s\n", criteria))
		}
		sb.WriteString("\n")
	}

	if len(b.request.AcceptanceCriteria.Nice) > 0 {
		sb.WriteString("### Nice to Have\n\n")
		for _, criteria := range b.request.AcceptanceCriteria.Nice {
			sb.WriteString(fmt.Sprintf("- %s\n", criteria))
		}
		sb.WriteString("\n")
	}

	if len(b.request.AcceptanceCriteria.Constraints) > 0 {
		sb.WriteString("### Constraints\n\n")
		for _, constraint := range b.request.AcceptanceCriteria.Constraints {
			sb.WriteString(fmt.Sprintf("- %s\n", constraint))
		}
		sb.WriteString("\n")
	}
}

func (b *TestGenerationPromptBuilder) buildCodeContext(sb *strings.Builder) {
	ctx := b.request.CodeContext
	sb.WriteString("## Code Context\n\n")

	if ctx.FilePath != "" {
		sb.WriteString("### File\n\n")
		sb.WriteString(fmt.Sprintf("`%s`\n\n", ctx.FilePath))
	}

	if ctx.Excerpt != "" {
		sb.WriteString("### Relevant Code\n\n")
		lang := "go"
		if ctx.Language != "" {
			lang = ctx.Language
		}
		sb.WriteString(fmt.Sprintf("```%s\n%s\n```\n\n", lang, ctx.Excerpt))
	}

	if ctx.Content != "" && ctx.Excerpt == "" {
		sb.WriteString("### Full Source\n\n")
		lang := "go"
		if ctx.Language != "" {
			lang = ctx.Language
		}
		sb.WriteString(fmt.Sprintf("```%s\n%s\n```\n\n", lang, ctx.Content))
	}
}

func (b *TestGenerationPromptBuilder) buildLintFeedback(sb *strings.Builder) {
	feedback := b.request.LintFeedback
	sb.WriteString("## Lint Feedback\n\n")

	if feedback.Summary != "" {
		sb.WriteString("### Summary\n\n")
		sb.WriteString(feedback.Summary)
		sb.WriteString("\n\n")
	}

	if len(feedback.Errors) > 0 {
		sb.WriteString("### Errors\n\n")
		for _, err := range feedback.Errors {
			sb.WriteString(fmt.Sprintf("- %s\n", err))
		}
		sb.WriteString("\n")
	}

	if len(feedback.Warnings) > 0 {
		sb.WriteString("### Warnings\n\n")
		for _, warn := range feedback.Warnings {
			sb.WriteString(fmt.Sprintf("- %s\n", warn))
		}
		sb.WriteString("\n")
	}
}

func (b *TestGenerationPromptBuilder) buildCoverageReport(sb *strings.Builder) {
	report := b.request.CoverageReport
	sb.WriteString("## Coverage Report\n\n")

	sb.WriteString("### Current Coverage\n\n")
	sb.WriteString(fmt.Sprintf("**Total Coverage:** %.2f%%\n\n", report.TotalCoverage))

	if len(report.UncoveredFunctions) > 0 {
		sb.WriteString("### Uncovered Functions\n\n")
		for _, fn := range report.UncoveredFunctions {
			sb.WriteString(fmt.Sprintf("- %s\n", fn))
		}
		sb.WriteString("\n")
	}

	if report.Report != "" {
		sb.WriteString("### Full Report\n\n")
		sb.WriteString("```\n")
		sb.WriteString(report.Report)
		sb.WriteString("\n```\n\n")
	}
}

func (b *TestGenerationPromptBuilder) buildOutputInstructions(sb *strings.Builder) {
	sb.WriteString("## Output Instructions\n\n")
	sb.WriteString(fmt.Sprintf("- Write tests in %s\n", b.request.Language))
	sb.WriteString(fmt.Sprintf("- Use %s framework\n", b.request.TestFramework))
	sb.WriteString(fmt.Sprintf("- Output path: `%s`\n", b.request.OutputPath))
	sb.WriteString(fmt.Sprintf("- Follow idiomatic %s conventions\n", b.request.Language))
	sb.WriteString("- Include proper error handling and edge cases\n")
	sb.WriteString("- Add clear test descriptions and assertions\n\n")
}

func (b *TestGenerationPromptBuilder) buildModeGuidelines(sb *strings.Builder) {
	sb.WriteString("## Generation Guidelines\n\n")

	switch b.request.Mode {
	case TestModeInitial:
		sb.WriteString("Generate comprehensive test suite from scratch covering:\n\n")
		sb.WriteString("- Basic functionality (happy path)\n")
		sb.WriteString("- Error cases and edge cases\n")
		sb.WriteString("- Boundary conditions\n")
		sb.WriteString("- Integration points\n")
		sb.WriteString("- Performance considerations\n\n")

	case TestModeRefinement:
		sb.WriteString("Refine tests based on lint feedback:\n\n")
		sb.WriteString("- Address all flagged lint issues\n")
		sb.WriteString("- Improve code quality and maintainability\n")
		sb.WriteString("- Ensure tests follow project standards\n")
		sb.WriteString("- Maintain or improve coverage\n")
		sb.WriteString("- Optimize test execution time\n\n")

	case TestModeEnhancement:
		sb.WriteString("Enhance tests to improve coverage:\n\n")
		sb.WriteString("- Add tests for currently uncovered functions\n")
		sb.WriteString("- Cover remaining edge cases\n")
		sb.WriteString("- Improve existing test quality\n")
		sb.WriteString("- Add integration tests where appropriate\n")
		sb.WriteString("- Test error recovery paths\n\n")
	}
}

// TestGenerationBuilder provides a fluent API for building test generation requests
type TestGenerationBuilder struct {
	request *TestGenerationRequest
}

// NewTestGenerationBuilder creates a new fluent builder for test generation
func NewTestGenerationBuilder() *TestGenerationBuilder {
	return &TestGenerationBuilder{
		request: &TestGenerationRequest{
			Mode:           TestModeInitial,
			Language:       "Go",
			TestFramework:  "testing",
			AcceptanceCriteria: &TestGenerationCriteria{},
			GeneratedAt:    time.Now(),
		},
	}
}

// WithMode sets the test generation mode
func (b *TestGenerationBuilder) WithMode(mode TestGenerationMode) *TestGenerationBuilder {
	b.request.Mode = mode
	return b
}

// WithTaskDescription sets the task description
func (b *TestGenerationBuilder) WithTaskDescription(desc string) *TestGenerationBuilder {
	b.request.TaskDescription = desc
	return b
}

// WithOutputPath sets the output path for tests
func (b *TestGenerationBuilder) WithOutputPath(path string) *TestGenerationBuilder {
	b.request.OutputPath = path
	return b
}

// WithLanguage sets the programming language
func (b *TestGenerationBuilder) WithLanguage(lang string) *TestGenerationBuilder {
	b.request.Language = lang
	return b
}

// WithTestFramework sets the testing framework
func (b *TestGenerationBuilder) WithTestFramework(framework string) *TestGenerationBuilder {
	b.request.TestFramework = framework
	return b
}

// AddRequiredCriteria adds a required acceptance criterion
func (b *TestGenerationBuilder) AddRequiredCriteria(criteria string) *TestGenerationBuilder {
	b.request.AcceptanceCriteria.Required = append(
		b.request.AcceptanceCriteria.Required,
		criteria,
	)
	return b
}

// AddNiceCriteria adds an optional acceptance criterion
func (b *TestGenerationBuilder) AddNiceCriteria(criteria string) *TestGenerationBuilder {
	b.request.AcceptanceCriteria.Nice = append(
		b.request.AcceptanceCriteria.Nice,
		criteria,
	)
	return b
}

// AddConstraint adds a constraint
func (b *TestGenerationBuilder) AddConstraint(constraint string) *TestGenerationBuilder {
	b.request.AcceptanceCriteria.Constraints = append(
		b.request.AcceptanceCriteria.Constraints,
		constraint,
	)
	return b
}

// WithLintFeedback sets the lint feedback
func (b *TestGenerationBuilder) WithLintFeedback(feedback *LintFeedback) *TestGenerationBuilder {
	b.request.LintFeedback = feedback
	return b
}

// AddLintError adds a lint error
func (b *TestGenerationBuilder) AddLintError(err string) *TestGenerationBuilder {
	if b.request.LintFeedback == nil {
		b.request.LintFeedback = &LintFeedback{}
	}
	b.request.LintFeedback.Errors = append(b.request.LintFeedback.Errors, err)
	return b
}

// AddLintWarning adds a lint warning
func (b *TestGenerationBuilder) AddLintWarning(warn string) *TestGenerationBuilder {
	if b.request.LintFeedback == nil {
		b.request.LintFeedback = &LintFeedback{}
	}
	b.request.LintFeedback.Warnings = append(b.request.LintFeedback.Warnings, warn)
	return b
}

// WithLintSummary sets the lint summary
func (b *TestGenerationBuilder) WithLintSummary(summary string) *TestGenerationBuilder {
	if b.request.LintFeedback == nil {
		b.request.LintFeedback = &LintFeedback{}
	}
	b.request.LintFeedback.Summary = summary
	return b
}

// WithCoverageReport sets the coverage report
func (b *TestGenerationBuilder) WithCoverageReport(report *CoverageReport) *TestGenerationBuilder {
	b.request.CoverageReport = report
	return b
}

// WithCodeContext sets the code context
func (b *TestGenerationBuilder) WithCodeContext(ctx *TestCodeContext) *TestGenerationBuilder {
	b.request.CodeContext = ctx
	return b
}

// WithCodeContextFromFile sets code context from file path and content
func (b *TestGenerationBuilder) WithCodeContextFromFile(filePath string, content string) *TestGenerationBuilder {
	b.request.CodeContext = &TestCodeContext{
		FilePath: filePath,
		Content:  content,
		Language: "go",
	}
	return b
}

// WithCodeExcerpt sets a specific code excerpt
func (b *TestGenerationBuilder) WithCodeExcerpt(excerpt string) *TestGenerationBuilder {
	if b.request.CodeContext == nil {
		b.request.CodeContext = &TestCodeContext{}
	}
	b.request.CodeContext.Excerpt = excerpt
	return b
}

// WithAdditionalNotes sets additional notes
func (b *TestGenerationBuilder) WithAdditionalNotes(notes string) *TestGenerationBuilder {
	b.request.AdditionalNotes = notes
	return b
}

// Build returns the constructed TestGenerationRequest
func (b *TestGenerationBuilder) Build() *TestGenerationRequest {
	return b.request
}

// BuildPrompt constructs and returns the prompt string directly
func (b *TestGenerationBuilder) BuildPrompt() string {
	builder := NewTestGenerationPromptBuilder(b.request)
	return builder.Build()
}
