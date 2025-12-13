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

// SpeculativeBranch represents a parallel test of combined changes in the merge queue.
//
// # Kill Switch Architecture
//
// The kill switch is a hierarchical failure propagation mechanism that immediately terminates
// all dependent speculative branches when a parent branch fails its tests. This prevents
// wasted computational resources on branches that are guaranteed to fail.
//
// # Hierarchy Structure
//
// Speculative branches form a parent-child tree:
//   - Base branch (Depth=1): Tests only the first change [C1]
//   - Level 2 (Depth=2): Tests base + next change [C1, C2]
//   - Level 3 (Depth=3): Tests level 2 + next change [C1, C2, C3]
//   - And so on...
//
// Each branch tracks its ParentID and ChildrenIDs to maintain this hierarchy.
//
// # Kill Switch Behavior
//
// When a branch fails:
//  1. All descendant branches are immediately killed recursively (depth-first)
//  2. Each killed branch has its Status set to BranchStatusKilled
//  3. KilledAt timestamp and KillReason are recorded for observability
//  4. Resources (Temporal workflows, Docker containers, worktrees) are cleaned up
//  5. The TotalKills metric is incremented for each killed branch
//
// # Example
//
// Given this hierarchy:
//
//	Branch A [C1]
//	  ├─ Branch B [C1, C2]
//	  │    └─ Branch D [C1, C2, C3]
//	  └─ Branch C [C1, C2, C3]
//
// If Branch B fails:
//   - Branch D is killed (child of B)
//   - Branch A continues (parent of B)
//   - Branch C continues (sibling of B)
//
// # Idempotency
//
// The kill switch is idempotent - killing an already-killed branch is a no-op.
// This prevents race conditions when multiple failures occur simultaneously.
//
// # Performance Benefits
//
// The kill switch provides significant performance improvements:
//   - Prevents wasted test execution on guaranteed failures
//   - Frees up Docker containers and CPU resources immediately
//   - Reduces queue latency by focusing on viable merge candidates
//   - Tracked via KilledPercent metric in QueueStats
//
// See KILLSWITCH.md for detailed architecture documentation.
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
	// BranchStatusKilled indicates the branch was terminated by the kill switch.
	// This happens when a parent branch in the speculation hierarchy fails its tests,
	// causing all dependent child branches to be killed to save resources.
	// See SpeculativeBranch documentation for kill switch architecture details.
	BranchStatusKilled BranchStatus = "killed"
)

// QueueStats tracks merge queue performance metrics.
//
// # Kill Switch Metrics
//
// The kill switch metrics (KilledPercent, TotalKills) measure the effectiveness
// of the hierarchical failure propagation system:
//
//  - KilledPercent: Percentage of branches terminated by the kill switch
//  - TotalKills: Total count of kill switch activations
//
// A higher KilledPercent indicates more resource savings from early termination
// of failing speculative branches. Typical values:
//  - >30%: High kill switch activity (saving significant resources)
//  - 10-30%: Moderate kill switch activity
//  - <10%: Low kill switch activity (high pass rates or shallow speculation)
//
// The ratio TotalKills/TotalFailures shows cascading effectiveness:
//  - Ratio > 2: Effective cascading (each failure kills multiple branches)
//  - Ratio ≈ 1: Mostly shallow hierarchies or isolated failures
//
// See KILLSWITCH.md for detailed architecture documentation.
type QueueStats struct {
	MergedPerHour   float64   // Average merge rate
	SuccessRate     float64   // Percentage of successful merges
	AvgQueueTime    time.Duration
	BypassedPercent float64   // Percentage that used bypass lane
	// KilledPercent is the percentage of branches terminated by the kill switch.
	// High values indicate effective resource savings from hierarchical failure propagation.
	KilledPercent float64
	AvgDepth      float64   // Average speculation depth
	TotalTests    int64     // Total number of tests executed
	TotalPasses   int64     // Total number of passed tests
	TotalFailures int64     // Total number of test failures
	// TotalKills counts how many branches were terminated via the kill switch.
	// This includes both direct failures and cascading kills from parent failures.
	TotalKills    int64
	TotalTimeouts int64     // Total number of test timeouts
	TotalMerges   int       // Total number of successful merges
	LastMergeTime time.Time // Time of last successful merge
}

// ConflictAnalysis represents overlap between changes
type ConflictAnalysis struct {
	Change1         string   // First change ID
	Change2         string   // Second change ID
	HasConflict     bool     // Do they conflict?
	ConflictType    string   // "file", "directory", "dependency"
	ConflictDetails []string // Specific files/paths that conflict
}
