package mergequeue

import (
	"time"
)

// ChangeRequest represents a completed agent's code ready for merging
type ChangeRequest struct {
	ID            string   // Agent ID
	WorktreePath  string   // Git worktree location
	FilesModified []string // List of files touched
	CommitSHA     string   // Git commit hash
	CreatedAt     time.Time
	Metadata      map[string]string // Additional metadata
}

// TestResult represents the outcome of testing a change or combination
type TestResult struct {
	ChangeIDs    []string // Which changes were tested together
	Passed       bool     // Did tests pass?
	Duration     time.Duration
	ErrorMessage string // If failed, why?
	TestOutput   string // Full test output
}

// SpeculativeBranch represents a parallel test of combined changes
type SpeculativeBranch struct {
	ID          string          // Unique branch ID
	Changes     []ChangeRequest // Changes being tested together
	Depth       int             // How many levels deep (1 = base, 2 = base+1, etc)
	Status      BranchStatus
	TestResult  *TestResult
	ContainerID string // Docker container running tests
	WorkflowID  string // Temporal workflow ID

	// Kill switch hierarchy tracking
	ParentID    string   // ID of parent branch (empty for base branch)
	ChildrenIDs []string // IDs of child branches spawned from this one

	// Kill switch metadata
	KilledAt   *time.Time // Timestamp when branch was killed (nil if not killed)
	KillReason string     // Explanation of why branch was killed
}

// BranchStatus represents the current state of a speculative branch
type BranchStatus string

const (
	BranchStatusPending BranchStatus = "pending" // Waiting to start
	BranchStatusTesting BranchStatus = "testing" // Tests running
	BranchStatusPassed  BranchStatus = "passed"  // Tests passed
	BranchStatusFailed  BranchStatus = "failed"  // Tests failed
	BranchStatusKilled  BranchStatus = "killed"  // Killed due to parent failure
)

// QueueStats tracks merge queue performance metrics
type QueueStats struct {
	MergedPerHour   float64 // Average merge rate
	SuccessRate     float64 // Percentage of successful merges
	AvgQueueTime    time.Duration
	BypassedPercent float64 // Percentage that used bypass lane
	KilledPercent   float64 // Percentage killed due to parent failures
	AvgDepth        float64 // Average speculation depth
}

// ConflictAnalysis represents overlap between changes
type ConflictAnalysis struct {
	Change1         string   // First change ID
	Change2         string   // Second change ID
	HasConflict     bool     // Do they conflict?
	ConflictType    string   // "file", "directory", "dependency"
	ConflictDetails []string // Specific files/paths that conflict
}
