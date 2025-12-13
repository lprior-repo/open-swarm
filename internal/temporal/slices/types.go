// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

// Package slices provides vertical slice architecture for Open Swarm workflows.
//
// Each file in this package represents a complete vertical slice containing:
// - Domain types
// - Activities (data access)
// - Business logic
// - Workflow integration
//
// This follows CUPID principles:
// - Composable: Each slice is self-contained and can be composed
// - Unix philosophy: Each slice does one thing well
// - Predictable: Clear inputs/outputs, no hidden state
// - Idiomatic: Go idioms and Temporal patterns
// - Domain-centric: Organized by capability, not technical layer
package slices

import "time"

// ============================================================================
// CELL LIFECYCLE TYPES
// ============================================================================

// BootstrapInput defines input for cell bootstrap
type BootstrapInput struct {
	CellID string
	Branch string
}

// BootstrapOutput contains bootstrap results
type BootstrapOutput struct {
	CellID       string
	Port         int
	WorktreeID   string
	WorktreePath string
	BaseURL      string
	ServerPID    int
}

// ============================================================================
// TASK EXECUTION TYPES
// ============================================================================

// TaskInput defines input for task execution
type TaskInput struct {
	TaskID      string
	Prompt      string
	Description string
}

// TaskOutput contains task execution results
type TaskOutput struct {
	FilesModified []string
	Output        string
	Success       bool
	Error         string
}

// ============================================================================
// WORKFLOW STATE TYPES
// ============================================================================

// WorkflowState tracks the current state of a workflow
type WorkflowState string

const (
	// StateBootstrap represents the initial bootstrap state
	StateBootstrap WorkflowState = "bootstrap"
	// StateGenTest represents the test generation state
	StateGenTest WorkflowState = "gen_test"
	// StateLintTest represents the lint testing state
	StateLintTest WorkflowState = "lint_test"
	// StateVerifyRED represents the RED verification state
	StateVerifyRED WorkflowState = "verify_red"
	// StateGenImpl represents the implementation generation state
	StateGenImpl WorkflowState = "gen_impl"
	// StateVerifyGREEN represents the GREEN verification state
	StateVerifyGREEN WorkflowState = "verify_green"
	// StateMultiReview represents the multi-reviewer approval state
	StateMultiReview WorkflowState = "multi_review"
	// StateCommit represents the commit state
	StateCommit WorkflowState = "commit"
	// StateComplete represents the completion state
	StateComplete WorkflowState = "complete"
	// StateFailed represents the failed state
	StateFailed WorkflowState = "failed"
)

// ============================================================================
// GATE RESULT TYPES
// ============================================================================

// GateResult represents the result of a single gate in a workflow
type GateResult struct {
	GateName      string
	Passed        bool
	AgentResults  []AgentResult
	Duration      time.Duration
	Error         string
	Message       string // Optional informational message
	RetryAttempts int
	TestResult    *TestResult  // For test gates
	LintResult    *LintResult  // For lint gates
	ReviewVotes   []ReviewVote // For review gate
}

// AgentResult contains the result from a single agent execution
type AgentResult struct {
	AgentName    string
	Model        string
	Prompt       string
	Response     string
	Success      bool
	Duration     time.Duration
	Error        string
	FilesChanged []string
}

// ============================================================================
// TEST EXECUTION TYPES
// ============================================================================

// TestResult contains test execution results
type TestResult struct {
	Passed       bool
	TotalTests   int
	PassedTests  int
	FailedTests  int
	Output       string
	Duration     time.Duration
	FailureTests []string // Names of failed tests
}

// TestFailure represents a single test failure
type TestFailure struct {
	TestName string
	Package  string
	Output   string
	File     string
	Line     int
}

// TestParseResult contains parsed test output
type TestParseResult struct {
	Passed       bool
	TotalTests   int
	PassedTests  int
	FailedTests  int
	SkippedTests int
	Failures     []TestFailure
	Duration     time.Duration
	RawOutput    string
}

// ============================================================================
// LINTING TYPES
// ============================================================================

// LintResult contains linting results
type LintResult struct {
	Passed   bool
	Issues   []LintIssue
	Output   string
	Duration time.Duration
}

// LintIssue represents a single linting issue
type LintIssue struct {
	File     string
	Line     int
	Column   int
	Severity string // "error", "warning", "info"
	Message  string
	Rule     string
}

// ParsedLintResult contains parsed lint output
type ParsedLintResult struct {
	Passed bool
	Issues []LintIssue
}

// ============================================================================
// CODE REVIEW TYPES
// ============================================================================

// ReviewVote represents a single reviewer's vote
type ReviewVote struct {
	ReviewerName string
	ReviewType   ReviewType
	Vote         VoteResult
	Feedback     string
	Duration     time.Duration
}

// ReviewType categorizes the review focus
type ReviewType string

const (
	// ReviewTypeTesting represents testing focus review
	ReviewTypeTesting ReviewType = "testing"
	// ReviewTypeFunctional represents functional correctness review
	ReviewTypeFunctional ReviewType = "functional"
	// ReviewTypeArchitecture represents architecture and design review
	ReviewTypeArchitecture ReviewType = "architecture"
)

// VoteResult represents a reviewer's decision
type VoteResult string

const (
	// VoteApprove represents approval decision
	VoteApprove VoteResult = "APPROVE"
	// VoteRequestChange represents request changes decision
	VoteRequestChange VoteResult = "REQUEST_CHANGE"
	// VoteReject represents reject decision
	VoteReject VoteResult = "REJECT"
)

// ParsedVote contains parsed review vote
type ParsedVote struct {
	Vote     VoteResult
	Feedback string
}

// ============================================================================
// FILE LOCKING TYPES
// ============================================================================

// LockError represents a file locking error
type LockError struct {
	Path    string
	Holder  string
	Message string
}

func (e *LockError) Error() string {
	return e.Message
}

// ============================================================================
// DAG WORKFLOW TYPES
// ============================================================================

// Task represents a single task in a DAG
type Task struct {
	Name    string
	Command string
	Deps    []string // Dependencies (task names)
}

// DAGWorkflowInput defines input for DAG workflow
type DAGWorkflowInput struct {
	Tasks []Task
}

// ============================================================================
// TCR WORKFLOW TYPES
// ============================================================================

// TCRWorkflowInput defines input for basic TCR workflow
type TCRWorkflowInput struct {
	CellID  string
	Branch  string
	TaskID  string
	Prompt  string
	Message string
}

// TCRWorkflowResult contains TCR workflow results
type TCRWorkflowResult struct {
	Success      bool
	Committed    bool
	FilesChanged []string
	TestOutput   string
	Error        string
}

// EnhancedTCRInput defines input for the Enhanced 6-Gate TCR workflow
type EnhancedTCRInput struct {
	CellID             string
	Branch             string
	TaskID             string
	Description        string
	AcceptanceCriteria string
	ReviewersCount     int // Default: 3 (unanimous vote required)
}

// EnhancedTCRResult contains the complete result of the Enhanced TCR workflow
type EnhancedTCRResult struct {
	Success      bool
	GateResults  []GateResult
	FilesChanged []string
	Error        string
}

// ============================================================================
// RETRY BUDGET TYPES
// ============================================================================

// GateType represents different gate types for retry budgeting
type GateType string

const (
	// GateTypeTest represents test execution gates
	GateTypeTest GateType = "test"
	// GateTypeLint represents linting gates
	GateTypeLint GateType = "lint"
	// GateTypeCodeGen represents code generation gates
	GateTypeCodeGen GateType = "codegen"
	// GateTypeReview represents code review gates
	GateTypeReview GateType = "review"
)
