// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package temporal

import "time"

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

// WorkflowState tracks the current state of the workflow
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

// GateResult represents the result of a single gate in the workflow
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
