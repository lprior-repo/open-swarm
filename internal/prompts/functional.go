package prompts

import (
	"fmt"
	"strings"
)

// FunctionalReviewBuilder builds prompts for functional correctness reviews
type FunctionalReviewBuilder struct{}

// NewFunctionalReviewBuilder creates a new functional review prompt builder
func NewFunctionalReviewBuilder() *FunctionalReviewBuilder {
	return &FunctionalReviewBuilder{}
}

// Build creates a functional review prompt from the request
func (b *FunctionalReviewBuilder) Build(request ReviewRequest) (string, error) {
	if err := validateRequest(request); err != nil {
		return "", err
	}

	var sb strings.Builder

	// Role and context
	sb.WriteString("You are a senior software engineer specializing in functional correctness reviews.\n\n")
	sb.WriteString("# Review Focus: Business Logic and Functional Correctness\n\n")

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
	sb.WriteString("## Functional Review Criteria\n\n")
	sb.WriteString("Evaluate the code against these functional correctness criteria:\n\n")

	sb.WriteString("### Requirements Compliance\n")
	sb.WriteString("- Does the implementation fully meet the stated requirements?\n")
	sb.WriteString("- Are all acceptance criteria satisfied?\n")
	sb.WriteString("- Are there any missing features or incomplete implementations?\n")
	sb.WriteString("- Does the behavior match the task description?\n\n")

	sb.WriteString("### Logic Correctness\n")
	sb.WriteString("- Is the core algorithm/logic correct?\n")
	sb.WriteString("- Are calculations and computations accurate?\n")
	sb.WriteString("- Are conditional statements and branches correct?\n")
	sb.WriteString("- Are loops and iterations implemented correctly?\n")
	sb.WriteString("- Are there any off-by-one errors or boundary issues?\n\n")

	sb.WriteString("### Edge Cases and Error Handling\n")
	sb.WriteString("- Are edge cases identified and handled?\n")
	sb.WriteString("- What happens with nil/null/empty inputs?\n")
	sb.WriteString("- Are boundary conditions checked (min/max values, array bounds)?\n")
	sb.WriteString("- Are errors detected and handled appropriately?\n")
	sb.WriteString("- Are error messages clear and actionable?\n")
	sb.WriteString("- Is error propagation done correctly?\n\n")

	sb.WriteString("### Data Integrity and Validation\n")
	sb.WriteString("- Is input validation thorough and correct?\n")
	sb.WriteString("- Are data transformations accurate?\n")
	sb.WriteString("- Is state managed correctly?\n")
	sb.WriteString("- Are race conditions or concurrency issues possible?\n")
	sb.WriteString("- Is data consistency maintained?\n\n")

	sb.WriteString("### Go Best Practices\n")
	sb.WriteString("- Are Go idioms and conventions followed?\n")
	sb.WriteString("- Is error handling done the Go way (multiple return values)?\n")
	sb.WriteString("- Are goroutines and channels used correctly?\n")
	sb.WriteString("- Is context properly propagated for cancellation/timeouts?\n")
	sb.WriteString("- Are defer, panic, and recover used appropriately?\n")
	sb.WriteString("- Are nil checks done where needed?\n\n")

	sb.WriteString("### Resource Management\n")
	sb.WriteString("- Are resources (files, connections, locks) properly released?\n")
	sb.WriteString("- Are defer statements used correctly for cleanup?\n")
	sb.WriteString("- Are there potential resource leaks?\n")
	sb.WriteString("- Is memory usage reasonable?\n\n")

	sb.WriteString("### Potential Bugs\n")
	sb.WriteString("- Are there any obvious bugs or logic errors?\n")
	sb.WriteString("- Could the code panic in any scenario?\n")
	sb.WriteString("- Are there type conversion issues?\n")
	sb.WriteString("- Are there potential deadlocks or race conditions?\n")
	sb.WriteString("- Are all code paths reachable and tested?\n\n")

	// Instructions
	sb.WriteString("## Review Instructions\n\n")
	sb.WriteString("Provide a thorough functional review:\n\n")
	sb.WriteString("1. **Correctness Assessment**: Does the code do what it's supposed to do?\n")
	sb.WriteString("2. **Issues Found**: List specific problems with:\n")
	sb.WriteString("   - Severity (critical/major/minor/suggestion)\n")
	sb.WriteString("   - Category (logic/error-handling/validation/concurrency/etc.)\n")
	sb.WriteString("   - Location (file:line or function name)\n")
	sb.WriteString("   - Clear description of the problem\n")
	sb.WriteString("   - Example scenario where it could fail\n")
	sb.WriteString("   - Suggested fix\n")
	sb.WriteString("3. **Missing Functionality**: Anything from requirements not implemented?\n")
	sb.WriteString("4. **Edge Cases**: What edge cases should be considered?\n")
	sb.WriteString("5. **Recommendations**: Specific improvements to logic or error handling\n\n")

	// Voting instructions
	if request.RequireVote {
		sb.WriteString("## Vote Required\n\n")
		sb.WriteString("Your response MUST end with one of these votes:\n\n")
		sb.WriteString("- **VOTE: APPROVE** - Logic is correct, requirements met, no significant issues\n")
		sb.WriteString("- **VOTE: REQUEST_CHANGE** - Functional but needs improvements or fixes\n")
		sb.WriteString("- **VOTE: REJECT** - Critical bugs, incorrect logic, or missing requirements\n\n")
		sb.WriteString("Format your vote on the last line as: `VOTE: [APPROVE|REQUEST_CHANGE|REJECT]`\n")
	} else {
		sb.WriteString("Provide your functional assessment without a formal vote.\n")
	}

	return sb.String(), nil
}

// GetReviewType returns the review type this builder handles
func (b *FunctionalReviewBuilder) GetReviewType() ReviewType {
	return ReviewTypeFunctional
}
