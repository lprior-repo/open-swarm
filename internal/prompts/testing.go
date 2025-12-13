package prompts

import (
	"fmt"
	"strings"
)

// TestingReviewBuilder builds prompts for test coverage and quality reviews
type TestingReviewBuilder struct{}

// NewTestingReviewBuilder creates a new testing review prompt builder
func NewTestingReviewBuilder() *TestingReviewBuilder {
	return &TestingReviewBuilder{}
}

// Build creates a testing review prompt from the request
func (b *TestingReviewBuilder) Build(request ReviewRequest) (string, error) {
	if err := validateRequest(request); err != nil {
		return "", err
	}

	var sb strings.Builder

	// Role and context
	sb.WriteString("You are a senior QA engineer and testing specialist.\n\n")
	sb.WriteString("# Review Focus: Test Coverage and Quality\n\n")

	// Task information
	sb.WriteString(fmt.Sprintf("## Task Information\n"))
	sb.WriteString(fmt.Sprintf("- **Task ID**: %s\n", request.TaskID))
	sb.WriteString(fmt.Sprintf("- **Description**: %s\n\n", request.TaskDescription))

	// Acceptance criteria
	if len(request.AcceptanceCriteria) > 0 {
		sb.WriteString("## Acceptance Criteria\n")
		for _, criterion := range request.AcceptanceCriteria {
			sb.WriteString(fmt.Sprintf("- %s\n", criterion))
		}
		sb.WriteString("\n")
	}

	// Code context
	sb.WriteString("## Code Under Review\n\n")
	if request.CodeContext.FilePath != "" {
		sb.WriteString(fmt.Sprintf("**File**: `%s`\n", request.CodeContext.FilePath))
		if request.CodeContext.PackageName != "" {
			sb.WriteString(fmt.Sprintf("**Package**: `%s`\n", request.CodeContext.PackageName))
		}
		sb.WriteString("\n")
	}

	// Show diff if available, otherwise full content
	if request.CodeContext.Diff != "" {
		sb.WriteString("### Changes (Git Diff)\n")
		sb.WriteString("```diff\n")
		sb.WriteString(request.CodeContext.Diff)
		sb.WriteString("\n```\n\n")
	} else if request.CodeContext.FileContent != "" {
		sb.WriteString("### Code\n")
		sb.WriteString(fmt.Sprintf("```%s\n", request.CodeContext.Language))
		sb.WriteString(request.CodeContext.FileContent)
		sb.WriteString("\n```\n\n")
	}

	// Surrounding context if provided
	if request.CodeContext.SurroundingCode != "" {
		sb.WriteString("### Related Code Context\n")
		sb.WriteString("```go\n")
		sb.WriteString(request.CodeContext.SurroundingCode)
		sb.WriteString("\n```\n\n")
	}

	// Additional context
	if request.AdditionalContext != "" {
		sb.WriteString("## Additional Context\n")
		sb.WriteString(request.AdditionalContext)
		sb.WriteString("\n\n")
	}

	// Review criteria
	sb.WriteString("## Testing Review Criteria\n\n")
	sb.WriteString("Evaluate the tests against these quality criteria:\n\n")

	sb.WriteString("### Test Coverage\n")
	sb.WriteString("- Are all critical code paths tested?\n")
	sb.WriteString("- Are edge cases covered by tests?\n")
	sb.WriteString("- Are error conditions tested?\n")
	sb.WriteString("- Are boundary conditions tested (min/max values, empty/nil inputs)?\n")
	sb.WriteString("- Are all public APIs/functions tested?\n")
	sb.WriteString("- What critical scenarios are missing tests?\n\n")

	sb.WriteString("### Test Quality and Clarity\n")
	sb.WriteString("- Are tests clear and easy to understand?\n")
	sb.WriteString("- Do test names clearly describe what is being tested?\n")
	sb.WriteString("- Is the test structure clear (Arrange-Act-Assert or Given-When-Then)?\n")
	sb.WriteString("- Are tests focused on one thing each?\n")
	sb.WriteString("- Is test code clean and maintainable?\n\n")

	sb.WriteString("### Assertions and Verification\n")
	sb.WriteString("- Are assertions specific and meaningful?\n")
	sb.WriteString("- Do assertions check the right things?\n")
	sb.WriteString("- Are error messages helpful for debugging?\n")
	sb.WriteString("- Are assertions complete (checking all relevant outputs)?\n")
	sb.WriteString("- Are there any missing assertions?\n\n")

	sb.WriteString("### TDD Principles\n")
	sb.WriteString("- Do tests follow the Red-Green-Refactor cycle?\n")
	sb.WriteString("- Are tests written at the right level of abstraction?\n")
	sb.WriteString("- Do tests drive the design (not just verify it)?\n")
	sb.WriteString("- Are tests isolated and independent?\n")
	sb.WriteString("- Can tests run in any order?\n\n")

	sb.WriteString("### Go Testing Best Practices\n")
	sb.WriteString("- Are table-driven tests used where appropriate?\n")
	sb.WriteString("- Are subtests used for better organization (t.Run)?\n")
	sb.WriteString("- Are test helpers used to reduce duplication?\n")
	sb.WriteString("- Is t.Parallel() used where tests can run concurrently?\n")
	sb.WriteString("- Are benchmarks provided for performance-critical code?\n")
	sb.WriteString("- Are examples provided for documentation?\n\n")

	sb.WriteString("### Test Reliability\n")
	sb.WriteString("- Are tests deterministic (no flaky tests)?\n")
	sb.WriteString("- Do tests clean up after themselves?\n")
	sb.WriteString("- Are there race conditions in tests?\n")
	sb.WriteString("- Do tests have appropriate timeouts?\n")
	sb.WriteString("- Are external dependencies mocked/stubbed?\n\n")

	sb.WriteString("### Error Testing\n")
	sb.WriteString("- Are error cases thoroughly tested?\n")
	sb.WriteString("- Are error types and messages validated?\n")
	sb.WriteString("- Are panic scenarios tested?\n")
	sb.WriteString("- Is context cancellation tested?\n")
	sb.WriteString("- Are timeout scenarios tested?\n\n")

	sb.WriteString("### Test Organization\n")
	sb.WriteString("- Are tests in the right package (_test.go files)?\n")
	sb.WriteString("- Are test files organized logically?\n")
	sb.WriteString("- Are test fixtures and helpers well-organized?\n")
	sb.WriteString("- Is test data managed appropriately?\n\n")

	// Instructions
	sb.WriteString("## Review Instructions\n\n")
	sb.WriteString("Provide a thorough testing review:\n\n")
	sb.WriteString("1. **Coverage Assessment**: What is covered and what is missing?\n")
	sb.WriteString("2. **Quality Issues**: Identify specific problems with:\n")
	sb.WriteString("   - Severity (critical/major/minor/suggestion)\n")
	sb.WriteString("   - Category (coverage/assertions/clarity/reliability/etc.)\n")
	sb.WriteString("   - Location (test name or file:line)\n")
	sb.WriteString("   - Clear description of the problem\n")
	sb.WriteString("   - Impact (what could go wrong)\n")
	sb.WriteString("   - Suggested improvement\n")
	sb.WriteString("3. **Missing Tests**: What critical scenarios lack tests?\n")
	sb.WriteString("4. **Test Examples**: Provide specific test cases that should be added\n")
	sb.WriteString("5. **Recommendations**: How to improve test quality and coverage\n\n")

	// Voting instructions
	if request.RequireVote {
		sb.WriteString("## Vote Required\n\n")
		sb.WriteString("Your response MUST end with one of these votes:\n\n")
		sb.WriteString("- **VOTE: APPROVE** - Comprehensive tests, good coverage, high quality\n")
		sb.WriteString("- **VOTE: REQUEST_CHANGE** - Tests exist but need improvements or additions\n")
		sb.WriteString("- **VOTE: REJECT** - Insufficient coverage, poor quality, or critical gaps\n\n")
		sb.WriteString("Format your vote on the last line as: `VOTE: [APPROVE|REQUEST_CHANGE|REJECT]`\n")
	} else {
		sb.WriteString("Provide your testing assessment without a formal vote.\n")
	}

	return sb.String(), nil
}

// GetReviewType returns the review type this builder handles
func (b *TestingReviewBuilder) GetReviewType() ReviewType {
	return ReviewTypeTesting
}
