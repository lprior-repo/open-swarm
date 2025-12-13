# Prompts Package

The `prompts` package provides structured prompt builders for code review agents. It generates comprehensive, context-rich prompts for three specialized review types: architecture, functional, and testing.

## Overview

This package helps orchestrate multi-agent code reviews by providing:

- **Specialized Review Types**: Architecture, Functional, and Testing reviews
- **Context-Rich Prompts**: Include code, diffs, acceptance criteria, and surrounding context
- **Structured Output**: Enforce consistent review format with optional voting
- **Fluent API**: Easy-to-use builder pattern for constructing review requests
- **Validation**: Ensure all required information is present before generating prompts

## Review Types

### Architecture Review (`ReviewTypeArchitecture`)

Focuses on:
- Design patterns and structure
- SOLID principles
- Maintainability and extensibility
- Integration and consistency
- Scalability and performance
- Code smells and anti-patterns

### Functional Review (`ReviewTypeFunctional`)

Focuses on:
- Requirements compliance
- Logic correctness
- Edge cases and error handling
- Data integrity and validation
- Go best practices
- Resource management
- Potential bugs

### Testing Review (`ReviewTypeTesting`)

Focuses on:
- Test coverage
- Test quality and clarity
- Assertions and verification
- TDD principles
- Go testing best practices
- Test reliability
- Error testing
- Test organization

## Usage

### Basic Usage

```go
import "open-swarm/internal/prompts"

// Create a review request
request := prompts.ReviewRequest{
    Type:            prompts.ReviewTypeArchitecture,
    TaskID:          "TASK-123",
    TaskDescription: "Implement user authentication",
    CodeContext: prompts.CodeContext{
        FilePath:    "auth/service.go",
        FileContent: "package auth\n\nfunc Login() error { return nil }",
        Language:    "go",
        PackageName: "auth",
    },
    AcceptanceCriteria: []string{
        "Must follow SOLID principles",
        "Must be easily testable",
    },
    RequireVote: true,
}

// Build the prompt
prompt, err := prompts.BuildArchitecturePrompt(request)
if err != nil {
    log.Fatal(err)
}

fmt.Println(prompt)
```

### Fluent API

```go
// Create request using fluent builder
request := prompts.NewReviewRequest(
    prompts.ReviewTypeFunctional,
    "TASK-456",
    "Add input validation",
).
    WithDiff("+func Validate() error { ... }").
    WithAcceptanceCriteria(
        "Must validate all inputs",
        "Must return descriptive errors",
    ).
    WithAdditionalContext("This is part of the API layer").
    WithoutVote()

prompt, err := prompts.BuildPrompt(request)
```

### Loading Code from Files

```go
// Load file content
ctx, err := prompts.LoadFileContent("path/to/file.go")
if err != nil {
    log.Fatal(err)
}

request := prompts.ReviewRequest{
    TaskID:          "TASK-789",
    TaskDescription: "Review authentication logic",
    CodeContext:     ctx,
    RequireVote:     true,
}

prompt, err := prompts.BuildFunctionalPrompt(request)
```

### Loading with Git Diff

```go
// Load file with diff
ctx, err := prompts.LoadFileWithDiff(
    "path/to/file.go",
    "+func NewFunc() {}\n-func OldFunc() {}",
)
if err != nil {
    log.Fatal(err)
}

request := prompts.ReviewRequest{
    TaskID:          "TASK-101",
    TaskDescription: "Refactor authentication",
    CodeContext:     ctx,
    RequireVote:     true,
}
```

### Generate All Review Types

```go
// Generate prompts for all three review types
request := prompts.ReviewRequest{
    TaskID:          "TASK-202",
    TaskDescription: "Payment processing implementation",
    CodeContext: prompts.CodeContext{
        FilePath:    "payment/processor.go",
        FileContent: "...",
    },
}

allPrompts, err := prompts.BuildAllReviewPrompts(request)
if err != nil {
    log.Fatal(err)
}

// Use each prompt with different reviewers
archPrompt := allPrompts[prompts.ReviewTypeArchitecture]
funcPrompt := allPrompts[prompts.ReviewTypeFunctional]
testPrompt := allPrompts[prompts.ReviewTypeTesting]
```

### Using Specific Builders

```go
// Create specific builder
builder := prompts.NewArchitectureReviewBuilder()

request := prompts.ReviewRequest{
    TaskID:          "ARCH-001",
    TaskDescription: "Design microservice architecture",
    CodeContext: prompts.CodeContext{
        FileContent: "...",
    },
    RequireVote: true,
}

prompt, err := builder.Build(request)
fmt.Printf("Review type: %s\n", builder.GetReviewType())
```

### Factory Pattern

```go
// Get builder by type
builder, err := prompts.GetBuilder(prompts.ReviewTypeFunctional)
if err != nil {
    log.Fatal(err)
}

prompt, err := builder.Build(request)
```

## Types

### ReviewRequest

```go
type ReviewRequest struct {
    Type               ReviewType      // Type of review
    TaskID             string          // Unique task identifier
    TaskDescription    string          // What the task accomplishes
    AcceptanceCriteria []string        // Requirements to meet
    CodeContext        CodeContext     // Code to review
    AdditionalContext  string          // Extra information
    RequireVote        bool            // Whether vote is required
}
```

### CodeContext

```go
type CodeContext struct {
    FilePath        string  // Path to file
    FileContent     string  // Full file content
    Diff            string  // Git diff (optional)
    SurroundingCode string  // Context around changes (optional)
    Language        string  // Programming language
    PackageName     string  // Go package name
}
```

### ReviewResponse

```go
type ReviewResponse struct {
    ReviewerName string          // Reviewer identifier
    ReviewType   ReviewType      // Type of review
    Vote         string          // APPROVE, REQUEST_CHANGE, REJECT
    Feedback     string          // Detailed feedback
    Issues       []ReviewIssue   // Specific problems
    Suggestions  []string        // Improvements
    Duration     time.Duration   // Review duration
}
```

### ReviewIssue

```go
type ReviewIssue struct {
    Severity    string  // critical, major, minor, suggestion
    Category    string  // Issue category
    Description string  // Problem description
    Location    string  // Where the issue was found
    Suggestion  string  // Recommended fix
}
```

## Prompt Structure

All prompts follow this structure:

1. **Role and Focus**: Establishes the reviewer's expertise and focus area
2. **Task Information**: Task ID and description
3. **Acceptance Criteria**: Requirements to validate (if provided)
4. **Code Under Review**: File path, package, and code content
5. **Changes**: Git diff or full code
6. **Surrounding Context**: Related code context (if provided)
7. **Additional Context**: Extra information (if provided)
8. **Review Criteria**: Specific evaluation criteria for the review type
9. **Review Instructions**: How to structure the review
10. **Vote Instructions**: Voting format (if required)

## Voting

When `RequireVote` is true, the prompt includes instructions for the LLM to end its response with:

- `VOTE: APPROVE` - No significant issues
- `VOTE: REQUEST_CHANGE` - Needs improvements
- `VOTE: REJECT` - Critical issues

The vote should appear on the last line of the response.

## Integration Example

```go
// Integration with OpenCode executor
func ReviewCode(taskID, description, filePath string) error {
    // Load code
    ctx, err := prompts.LoadFileContent(filePath)
    if err != nil {
        return err
    }

    // Create review request
    request := prompts.NewReviewRequest(
        prompts.ReviewTypeFunctional,
        taskID,
        description,
    ).WithAcceptanceCriteria(
        "Must handle all edge cases",
        "Must follow Go best practices",
    )
    request.CodeContext = ctx

    // Generate all review prompts
    allPrompts, err := prompts.BuildAllReviewPrompts(request)
    if err != nil {
        return err
    }

    // Execute reviews with different agents
    var reviews []prompts.ReviewResponse
    for reviewType, prompt := range allPrompts {
        response, err := executeReview(prompt, reviewType)
        if err != nil {
            return err
        }
        reviews = append(reviews, response)
    }

    // Aggregate votes
    return processReviews(reviews)
}
```

## Best Practices

1. **Always provide context**: Include file paths, package names, and surrounding code
2. **Use diffs when possible**: More focused reviews on actual changes
3. **Specify acceptance criteria**: Clear requirements lead to better reviews
4. **Enable voting for gates**: Use `RequireVote: true` for quality gates
5. **Use all three review types**: Architecture, functional, and testing together provide comprehensive coverage
6. **Load files properly**: Use helper functions to extract language and package info
7. **Chain fluent methods**: Build requests step-by-step with the fluent API

## Testing

Run tests with:

```bash
go test ./internal/prompts/...
```

Run with coverage:

```bash
go test -cover ./internal/prompts/...
```

## Examples

See `example_test.go` for runnable examples:

```bash
go test -run Example ./internal/prompts/
```
