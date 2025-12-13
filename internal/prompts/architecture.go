package prompts

import (
	"fmt"
	"strings"
)

// ArchitectureReviewBuilder builds prompts for architecture-focused code reviews
type ArchitectureReviewBuilder struct{}

// NewArchitectureReviewBuilder creates a new architecture review prompt builder
func NewArchitectureReviewBuilder() *ArchitectureReviewBuilder {
	return &ArchitectureReviewBuilder{}
}

// Build creates an architecture review prompt from the request
func (b *ArchitectureReviewBuilder) Build(request ReviewRequest) (string, error) {
	if err := validateRequest(request); err != nil {
		return "", err
	}

	var sb strings.Builder

	// Role and context
	sb.WriteString("You are a senior software architect specializing in architectural reviews.\n\n")
	sb.WriteString("# Review Focus: Architecture and Design Patterns\n\n")

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
	sb.WriteString("## Architecture Review Criteria\n\n")
	sb.WriteString("Evaluate the code against these architectural principles:\n\n")

	sb.WriteString("### Design Patterns and Structure\n")
	sb.WriteString("- Does the design follow established patterns appropriately?\n")
	sb.WriteString("- Are patterns being used correctly (not over-engineered or misapplied)?\n")
	sb.WriteString("- Is the module/package structure logical and well-organized?\n")
	sb.WriteString("- Are responsibilities clearly separated?\n\n")

	sb.WriteString("### SOLID Principles\n")
	sb.WriteString("- **Single Responsibility**: Does each component have one clear purpose?\n")
	sb.WriteString("- **Open/Closed**: Is the code open for extension but closed for modification?\n")
	sb.WriteString("- **Liskov Substitution**: Are abstractions used correctly?\n")
	sb.WriteString("- **Interface Segregation**: Are interfaces focused and minimal?\n")
	sb.WriteString("- **Dependency Inversion**: Does the code depend on abstractions, not concretions?\n\n")

	sb.WriteString("### Maintainability and Extensibility\n")
	sb.WriteString("- Is the code easy to understand and modify?\n")
	sb.WriteString("- Can new features be added without major refactoring?\n")
	sb.WriteString("- Are there appropriate extension points?\n")
	sb.WriteString("- Is coupling minimized and cohesion maximized?\n\n")

	sb.WriteString("### Integration and Consistency\n")
	sb.WriteString("- Does it integrate well with the existing codebase?\n")
	sb.WriteString("- Are naming conventions and code style consistent?\n")
	sb.WriteString("- Does it follow the project's architectural patterns?\n")
	sb.WriteString("- Are dependencies managed appropriately?\n\n")

	sb.WriteString("### Scalability and Performance Considerations\n")
	sb.WriteString("- Will this design scale as the system grows?\n")
	sb.WriteString("- Are there any obvious performance bottlenecks?\n")
	sb.WriteString("- Is resource management handled appropriately?\n")
	sb.WriteString("- Are concurrency/parallelism concerns addressed?\n\n")

	sb.WriteString("### Code Smells and Anti-Patterns\n")
	sb.WriteString("- Are there any design smells (e.g., feature envy, inappropriate intimacy)?\n")
	sb.WriteString("- Are there circular dependencies or tight coupling issues?\n")
	sb.WriteString("- Is there duplicated or redundant code?\n")
	sb.WriteString("- Are there any obvious anti-patterns?\n\n")

	// Instructions
	sb.WriteString("## Review Instructions\n\n")
	sb.WriteString("Provide a thorough architectural review:\n\n")
	sb.WriteString("1. **Strengths**: What architectural decisions are sound?\n")
	sb.WriteString("2. **Issues**: Identify specific architectural problems with:\n")
	sb.WriteString("   - Severity (critical/major/minor/suggestion)\n")
	sb.WriteString("   - Category (design/coupling/abstraction/scalability/etc.)\n")
	sb.WriteString("   - Location (file:line or component name)\n")
	sb.WriteString("   - Clear description of the problem\n")
	sb.WriteString("   - Suggested improvement\n")
	sb.WriteString("3. **Recommendations**: Suggest architectural improvements\n")
	sb.WriteString("4. **Trade-offs**: Note any architectural trade-offs being made\n\n")

	// Voting instructions
	if request.RequireVote {
		sb.WriteString("## Vote Required\n\n")
		sb.WriteString("Your response MUST end with one of these votes:\n\n")
		sb.WriteString("- **VOTE: APPROVE** - Architecture is sound, no significant issues\n")
		sb.WriteString("- **VOTE: REQUEST_CHANGE** - Good design with improvements needed\n")
		sb.WriteString("- **VOTE: REJECT** - Fundamental architectural issues must be addressed\n\n")
		sb.WriteString("Format your vote on the last line as: `VOTE: [APPROVE|REQUEST_CHANGE|REJECT]`\n")
	} else {
		sb.WriteString("Provide your architectural assessment without a formal vote.\n")
	}

	return sb.String(), nil
}

// GetReviewType returns the review type this builder handles
func (b *ArchitectureReviewBuilder) GetReviewType() ReviewType {
	return ReviewTypeArchitecture
}

// validateRequest checks if the request has required fields
func validateRequest(request ReviewRequest) error {
	if request.TaskID == "" {
		return fmt.Errorf("TaskID is required")
	}
	if request.TaskDescription == "" {
		return fmt.Errorf("TaskDescription is required")
	}
	if request.CodeContext.FileContent == "" && request.CodeContext.Diff == "" {
		return fmt.Errorf("either FileContent or Diff must be provided")
	}
	return nil
}
