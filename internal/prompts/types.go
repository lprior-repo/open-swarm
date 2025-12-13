package prompts

import "time"

// RequestType indicates the type of prompt request
type RequestType string

const (
	// RequestTypeInitial generates code from test specifications
	RequestTypeInitial RequestType = "initial"
	// RequestTypeRefinement refines code based on feedback
	RequestTypeRefinement RequestType = "refinement"
	// RequestTypeDebug fixes code based on test failures
	RequestTypeDebug RequestType = "debug"
	// RequestTypeBugFix is an alias for RequestTypeDebug
	RequestTypeBugFix RequestType = "bugfix"
)

// PromptRequest represents a request for code generation or test generation
type PromptRequest struct {
	// ID is a unique identifier for the request
	ID string
	// RequestType indicates the type of request
	RequestType RequestType
	// TaskDescription describes what should be implemented
	TaskDescription string
	// OutputPath specifies where the generated code should be written
	OutputPath string
	// TestContents contains test code (for TDD requests)
	TestContents string
	// ReviewFeedback contains feedback from code reviews
	ReviewFeedback string
	// TestFailures contains test failure output
	TestFailures string
	// Context contains additional contextual information
	Context map[string]string
	// CreatedAt is when the request was created
	CreatedAt time.Time
}

// ReviewType represents the type of code review
type ReviewType string

const (
	// ReviewTypeArchitecture focuses on architectural decisions and patterns
	ReviewTypeArchitecture ReviewType = "architecture"
	// ReviewTypeFunctional focuses on business logic and correctness
	ReviewTypeFunctional ReviewType = "functional"
	// ReviewTypeTesting focuses on test coverage and quality
	ReviewTypeTesting ReviewType = "testing"
)

// CodeContext contains contextual information about the code being reviewed
type CodeContext struct {
	// FilePath is the path to the file being reviewed
	FilePath string
	// FileContent is the full content of the file
	FileContent string
	// Diff is the git diff of changes (optional)
	Diff string
	// SurroundingCode is context around the changed code (optional)
	SurroundingCode string
	// Language is the programming language
	Language string
	// PackageName is the Go package name
	PackageName string
}

// ReviewRequest contains all information needed to build a review prompt
type ReviewRequest struct {
	// Type is the type of review to perform
	Type ReviewType
	// TaskID is the unique identifier for the task
	TaskID string
	// TaskDescription describes what the task is meant to accomplish
	TaskDescription string
	// AcceptanceCriteria are the requirements that must be met
	AcceptanceCriteria []string
	// CodeContext contains the code to review
	CodeContext CodeContext
	// AdditionalContext is any extra information for the reviewer
	AdditionalContext string
	// RequireVote indicates if the reviewer must provide a vote
	RequireVote bool
}

// ReviewResponse contains the structured review from the LLM
type ReviewResponse struct {
	// ReviewerName is the name/type of reviewer
	ReviewerName string
	// ReviewType is the type of review performed
	ReviewType ReviewType
	// Vote is the reviewer's decision (APPROVE, REQUEST_CHANGE, REJECT)
	Vote string
	// Feedback is the detailed review feedback
	Feedback string
	// Issues are specific problems found
	Issues []ReviewIssue
	// Suggestions are improvement recommendations
	Suggestions []string
	// Duration is how long the review took
	Duration time.Duration
}

// ReviewIssue represents a specific problem found during review
type ReviewIssue struct {
	// Severity is the importance level (critical, major, minor, suggestion)
	Severity string
	// Category is the issue category (logic, design, testing, etc.)
	Category string
	// Description explains the issue
	Description string
	// Location is where the issue was found (file:line or function name)
	Location string
	// Suggestion is a recommended fix
	Suggestion string
}

// PromptBuilder is the interface for building review prompts
type PromptBuilder interface {
	// Build creates a review prompt from the request
	Build(request ReviewRequest) (string, error)
	// GetReviewType returns the type of review this builder creates
	GetReviewType() ReviewType
}
