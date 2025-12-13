package prompts

import (
	"fmt"
	"strings"
	"time"
)

// GetBuilder returns the appropriate prompt builder for the given review type
func GetBuilder(reviewType ReviewType) (PromptBuilder, error) {
	switch reviewType {
	case ReviewTypeArchitecture:
		return NewArchitectureReviewBuilder(), nil
	case ReviewTypeFunctional:
		return NewFunctionalReviewBuilder(), nil
	case ReviewTypeTesting:
		return NewTestingReviewBuilder(), nil
	default:
		return nil, fmt.Errorf("unknown review type: %s", reviewType)
	}
}

// BuildPrompt is a convenience function that creates a prompt for the given request
func BuildPrompt(request ReviewRequest) (string, error) {
	builder, err := GetBuilder(request.Type)
	if err != nil {
		return "", err
	}
	return builder.Build(request)
}

// BuildArchitecturePrompt creates an architecture review prompt
func BuildArchitecturePrompt(request ReviewRequest) (string, error) {
	request.Type = ReviewTypeArchitecture
	return NewArchitectureReviewBuilder().Build(request)
}

// BuildFunctionalPrompt creates a functional review prompt
func BuildFunctionalPrompt(request ReviewRequest) (string, error) {
	request.Type = ReviewTypeFunctional
	return NewFunctionalReviewBuilder().Build(request)
}

// BuildTestingPrompt creates a testing review prompt
func BuildTestingPrompt(request ReviewRequest) (string, error) {
	request.Type = ReviewTypeTesting
	return NewTestingReviewBuilder().Build(request)
}

// BuildAllReviewPrompts creates prompts for all three review types
func BuildAllReviewPrompts(baseRequest ReviewRequest) (map[ReviewType]string, error) {
	results := make(map[ReviewType]string)

	// Architecture review
	archRequest := baseRequest
	archRequest.Type = ReviewTypeArchitecture
	archPrompt, err := BuildArchitecturePrompt(archRequest)
	if err != nil {
		return nil, fmt.Errorf("architecture prompt: %w", err)
	}
	results[ReviewTypeArchitecture] = archPrompt

	// Functional review
	funcRequest := baseRequest
	funcRequest.Type = ReviewTypeFunctional
	funcPrompt, err := BuildFunctionalPrompt(funcRequest)
	if err != nil {
		return nil, fmt.Errorf("functional prompt: %w", err)
	}
	results[ReviewTypeFunctional] = funcPrompt

	// Testing review
	testRequest := baseRequest
	testRequest.Type = ReviewTypeTesting
	testPrompt, err := BuildTestingPrompt(testRequest)
	if err != nil {
		return nil, fmt.Errorf("testing prompt: %w", err)
	}
	results[ReviewTypeTesting] = testPrompt

	return results, nil
}

// ImplementationBuilder constructs implementation request prompts for agents
type ImplementationBuilder struct {
	taskDescription string
	requestType     RequestType
	outputPath      string
	testContents    string
	reviewFeedback  string
	testFailures    string
	context         map[string]string
	metadata        map[string]interface{}
}

// NewImplementationBuilder creates a new implementation prompt builder
func NewImplementationBuilder(taskDescription string) *ImplementationBuilder {
	return &ImplementationBuilder{
		taskDescription: taskDescription,
		requestType:     RequestTypeInitial,
		context:         make(map[string]string),
		metadata:        make(map[string]interface{}),
	}
}

// NewBuilder is an alias for NewImplementationBuilder for backward compatibility
func NewBuilder(taskDescription string) *ImplementationBuilder {
	return NewImplementationBuilder(taskDescription)
}

// WithRequestType sets the type of request
func (b *ImplementationBuilder) WithRequestType(rt RequestType) *ImplementationBuilder {
	b.requestType = rt
	return b
}

// WithOutputPath sets the expected output file path
func (b *ImplementationBuilder) WithOutputPath(path string) *ImplementationBuilder {
	b.outputPath = path
	return b
}

// WithTestContents includes test file contents (for TDD)
func (b *ImplementationBuilder) WithTestContents(tests string) *ImplementationBuilder {
	b.testContents = tests
	return b
}

// WithReviewFeedback includes code review feedback for refinement
func (b *ImplementationBuilder) WithReviewFeedback(feedback string) *ImplementationBuilder {
	b.reviewFeedback = feedback
	return b
}

// WithTestFailures includes test failure output for debugging
func (b *ImplementationBuilder) WithTestFailures(failures string) *ImplementationBuilder {
	b.testFailures = failures
	return b
}

// WithContext adds relevant context information (e.g., related files, architecture notes)
func (b *ImplementationBuilder) WithContext(key, value string) *ImplementationBuilder {
	b.context[key] = value
	return b
}

// WithMetadata adds metadata about the request (e.g., agent info, timestamps)
func (b *ImplementationBuilder) WithMetadata(key string, value interface{}) *ImplementationBuilder {
	b.metadata[key] = value
	return b
}

// Build generates the complete prompt string
func (b *ImplementationBuilder) Build() string {
	var sb strings.Builder

	// Write header with timestamp
	sb.WriteString("# Implementation Request\n\n")
	sb.WriteString(fmt.Sprintf("**Generated:** %s\n", time.Now().Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("**Request Type:** %s\n\n", b.requestType))

	// Write task description
	sb.WriteString("## Task Description\n\n")
	sb.WriteString(b.taskDescription)
	sb.WriteString("\n\n")

	// Write output expectations
	if b.outputPath != "" {
		sb.WriteString("## Output Expectations\n\n")
		sb.WriteString(fmt.Sprintf("**Implementation File:** `%s`\n\n", b.outputPath))
	}

	// Write test contents for TDD
	if b.testContents != "" {
		sb.WriteString("## Test-Driven Development\n\n")
		sb.WriteString("Your implementation should pass the following tests:\n\n")
		sb.WriteString("```go\n")
		sb.WriteString(b.testContents)
		sb.WriteString("\n```\n\n")
	}

	// Write review feedback for refinement
	if b.reviewFeedback != "" {
		sb.WriteString("## Code Review Feedback\n\n")
		sb.WriteString("The following feedback from code review should be addressed:\n\n")
		sb.WriteString(b.reviewFeedback)
		sb.WriteString("\n\n")
	}

	// Write test failure details for bug fixes
	if b.testFailures != "" {
		sb.WriteString("## Test Failure Details\n\n")
		sb.WriteString("The following tests are currently failing:\n\n")
		sb.WriteString("```\n")
		sb.WriteString(b.testFailures)
		sb.WriteString("\n```\n\n")
		sb.WriteString("Please fix the implementation to make these tests pass.\n\n")
	}

	// Write additional context
	if len(b.context) > 0 {
		sb.WriteString("## Context & Architecture\n\n")
		for key, value := range b.context {
			sb.WriteString(fmt.Sprintf("### %s\n\n%s\n\n", key, value))
		}
	}

	// Write metadata section if present
	if len(b.metadata) > 0 {
		sb.WriteString("## Metadata\n\n")
		for key, value := range b.metadata {
			sb.WriteString(fmt.Sprintf("- **%s:** %v\n", key, value))
		}
		sb.WriteString("\n")
	}

	// Write closing instructions
	sb.WriteString(b.buildClosingInstructions())

	return sb.String()
}

// buildClosingInstructions generates the appropriate closing instructions based on request type
func (b *ImplementationBuilder) buildClosingInstructions() string {
	switch b.requestType {
	case RequestTypeInitial:
		return `## Implementation Instructions

1. **Implement the feature** based on the test-driven development tests provided above
2. **Ensure all tests pass** before completing the implementation
3. **Follow Go best practices** and project conventions
4. **Update any related documentation** if needed
5. **Commit your changes** with a clear, descriptive message

For any ambiguities, infer intent from the tests and context provided.
`

	case RequestTypeRefinement:
		return `## Refinement Instructions

1. **Review the feedback** provided above carefully
2. **Refactor or improve the implementation** according to the feedback
3. **Ensure all tests still pass** after refinement
4. **Maintain backward compatibility** unless explicitly instructed otherwise
5. **Commit your changes** with a message describing the improvements

Focus on code quality, readability, and maintainability.
`

	case RequestTypeDebug:
		return `## Bug Fix Instructions

1. **Analyze the failing tests** provided above
2. **Identify the root cause** of the failures
3. **Fix the implementation** to address the root cause
4. **Verify that all tests pass** after the fix
5. **Consider if similar issues** exist elsewhere in the codebase
6. **Commit your changes** with a message describing the fix

Ensure the fix is minimal and does not introduce regressions.
`

	default:
		return `## Implementation Instructions

1. Implement the requested changes
2. Ensure all tests pass
3. Commit your changes with a descriptive message
`
	}
}

// String returns the built prompt
func (b *ImplementationBuilder) String() string {
	return b.Build()
}

// BuildImplementationPrompt generates an implementation prompt from a PromptRequest
func BuildImplementationPrompt(request *PromptRequest) string {
	builder := NewImplementationBuilder(request.TaskDescription)
	builder.WithRequestType(request.RequestType)
	builder.WithOutputPath(request.OutputPath)
	builder.WithTestContents(request.TestContents)
	builder.WithReviewFeedback(request.ReviewFeedback)
	builder.WithTestFailures(request.TestFailures)

	for key, value := range request.Context {
		builder.WithContext(key, value)
	}

	return builder.Build()
}
