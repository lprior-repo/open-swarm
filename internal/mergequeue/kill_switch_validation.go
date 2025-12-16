package mergequeue

import (
	"fmt"
	"strings"
	"time"
)

// BranchValidationError represents detailed validation failure with context
type BranchValidationError struct {
	Code      string // Machine-readable error code
	Message   string // Human-readable error message
	BranchID  string // Branch that failed validation
	Details   string // Additional context
	Timestamp time.Time
}

// Error implements the error interface
func (e *BranchValidationError) Error() string {
	return fmt.Sprintf("[%s] %s: %s (branch: %s)", e.Code, e.Message, e.Details, e.BranchID)
}

// ValidationCode constants define standardized error codes for validation failures
const (
	// ValidationCodeBranchNotFound indicates the branch does not exist in the merge queue
	ValidationCodeBranchNotFound = "BRANCH_NOT_FOUND"

	// ValidationCodeBranchProtected indicates the branch is protected (main/master)
	ValidationCodeBranchProtected = "BRANCH_PROTECTED"

	// ValidationCodePendingWork indicates the branch has pending or in-progress work
	ValidationCodePendingWork = "PENDING_WORK"

	// ValidationCodeInvalidStatus indicates the branch is in an invalid state for killing
	ValidationCodeInvalidStatus = "INVALID_STATUS"

	// ValidationCodeOwnershipMismatch indicates the requester doesn't own the branch
	ValidationCodeOwnershipMismatch = "OWNERSHIP_MISMATCH"

	// ValidationCodeValidationTimeout indicates validation took too long
	ValidationCodeValidationTimeout = "VALIDATION_TIMEOUT"
)

// KillSwitchValidator provides pre-kill validation for branch operations
type KillSwitchValidator struct {
	protectedBranches map[string]bool // Set of protected branch patterns (main, master, release/*)
}

// NewKillSwitchValidator creates a new validator with default protected branch patterns
func NewKillSwitchValidator() *KillSwitchValidator {
	return &KillSwitchValidator{
		protectedBranches: map[string]bool{
			"main":    true,
			"master":  true,
			"develop": false, // Configurable based on project
		},
	}
}

// AddProtectedBranch adds a branch pattern to the protected list
func (v *KillSwitchValidator) AddProtectedBranch(pattern string) {
	if v.protectedBranches == nil {
		v.protectedBranches = make(map[string]bool)
	}
	v.protectedBranches[pattern] = true
}

// ValidateBranchExists checks if the branch exists in the active branches
func (v *KillSwitchValidator) ValidateBranchExists(branch *SpeculativeBranch, branchID string) *BranchValidationError {
	if branch == nil {
		return &BranchValidationError{
			Code:      ValidationCodeBranchNotFound,
			Message:   "Branch does not exist",
			BranchID:  branchID,
			Details:   fmt.Sprintf("No branch with ID '%s' found in merge queue", branchID),
			Timestamp: time.Now(),
		}
	}
	return nil
}

// ValidateBranchNotProtected checks if the branch is a protected branch (main/master)
func (v *KillSwitchValidator) ValidateBranchNotProtected(branchID string) *BranchValidationError {
	// Check exact matches first
	if protected, exists := v.protectedBranches[branchID]; exists && protected {
		return &BranchValidationError{
			Code:      ValidationCodeBranchProtected,
			Message:   "Cannot kill protected branch",
			BranchID:  branchID,
			Details:   fmt.Sprintf("Branch '%s' is protected from kill operations", branchID),
			Timestamp: time.Now(),
		}
	}

	// Check prefix patterns (e.g., release/*, hotfix/*)
	protectedPatterns := []string{"release/", "hotfix/", "production/"}
	for _, pattern := range protectedPatterns {
		if strings.HasPrefix(branchID, pattern) {
			return &BranchValidationError{
				Code:      ValidationCodeBranchProtected,
				Message:   "Cannot kill protected branch",
				BranchID:  branchID,
				Details:   fmt.Sprintf("Branch matches protected pattern '%s*'", pattern),
				Timestamp: time.Now(),
			}
		}
	}

	return nil
}

// ValidateBranchStatus checks if the branch is in a valid state for killing
func (v *KillSwitchValidator) ValidateBranchStatus(branch *SpeculativeBranch) *BranchValidationError {
	if branch == nil {
		return &BranchValidationError{
			Code:      ValidationCodeInvalidStatus,
			Message:   "Cannot validate status of nil branch",
			BranchID:  "unknown",
			Details:   "Branch object is nil",
			Timestamp: time.Now(),
		}
	}

	// Already killed branches are okay (idempotent operation)
	if branch.Status == BranchStatusKilled {
		return nil // Idempotent - already killed is not an error
	}

	// Valid statuses for killing: Pending, Testing, Failed, Passed
	// Invalid: Merged, Archived
	validStatuses := map[BranchStatus]bool{
		BranchStatusPending: true,
		BranchStatusTesting: true,
		BranchStatusFailed:  true,
		BranchStatusPassed:  true,
	}

	if !validStatuses[branch.Status] {
		return &BranchValidationError{
			Code:      ValidationCodeInvalidStatus,
			Message:   "Branch is in an invalid state for killing",
			BranchID:  branch.ID,
			Details:   fmt.Sprintf("Branch status '%s' does not allow kill operations", branch.Status),
			Timestamp: time.Now(),
		}
	}

	return nil
}

// ValidateNoPendingWork checks if the branch has pending or in-progress work
func (v *KillSwitchValidator) ValidateNoPendingWork(branch *SpeculativeBranch) *BranchValidationError {
	if branch == nil {
		return &BranchValidationError{
			Code:      ValidationCodeInvalidStatus,
			Message:   "Cannot validate pending work of nil branch",
			BranchID:  "unknown",
			Details:   "Branch object is nil",
			Timestamp: time.Now(),
		}
	}

	// Check for pending work based on branch status
	if branch.Status == BranchStatusTesting {
		return &BranchValidationError{
			Code:      ValidationCodePendingWork,
			Message:   "Branch has pending work in progress",
			BranchID:  branch.ID,
			Details:   fmt.Sprintf("Branch is currently testing with workflow '%s', please wait for completion or force kill with timeout", branch.WorkflowID),
			Timestamp: time.Now(),
		}
	}

	// Check if branch has active resources
	if branch.ContainerID != "" && branch.Status != BranchStatusFailed && branch.Status != BranchStatusPassed {
		return &BranchValidationError{
			Code:      ValidationCodePendingWork,
			Message:   "Branch has active container resources",
			BranchID:  branch.ID,
			Details:   fmt.Sprintf("Docker container '%s' is still active, resource cleanup may be in progress", branch.ContainerID),
			Timestamp: time.Now(),
		}
	}

	// Check if any unprocessed test result exists (incomplete testing)
	if branch.TestResult == nil && branch.Status == BranchStatusTesting {
		return &BranchValidationError{
			Code:      ValidationCodePendingWork,
			Message:   "Branch test result is pending",
			BranchID:  branch.ID,
			Details:   "Waiting for test result to complete, cannot kill while test is in flight",
			Timestamp: time.Now(),
		}
	}

	return nil
}

// ValidateOwnership checks if the requesting agent owns the branch
// In this context, ownership is determined by the initial change requester (agent ID)
func (v *KillSwitchValidator) ValidateOwnership(branch *SpeculativeBranch, requestingAgent string) *BranchValidationError {
	if branch == nil {
		return &BranchValidationError{
			Code:      ValidationCodeInvalidStatus,
			Message:   "Cannot validate ownership of nil branch",
			BranchID:  "unknown",
			Details:   "Branch object is nil",
			Timestamp: time.Now(),
		}
	}

	if len(branch.Changes) == 0 {
		return &BranchValidationError{
			Code:      ValidationCodeInvalidStatus,
			Message:   "Cannot determine branch ownership",
			BranchID:  branch.ID,
			Details:   "Branch has no associated changes with agent IDs",
			Timestamp: time.Now(),
		}
	}

	// Get the original requester (first change's agent ID)
	originalRequester := branch.Changes[0].ID
	systemAgents := map[string]bool{
		"system":         true,
		"admin":          true,
		"coordinator":    true,
		"merge-queue":    true,
		"automated-test": true,
	}

	// System agents can kill any branch
	if systemAgents[requestingAgent] {
		return nil
	}

	// Regular agents can only kill branches they created
	if requestingAgent != originalRequester {
		return &BranchValidationError{
			Code:      ValidationCodeOwnershipMismatch,
			Message:   "Agent does not own this branch",
			BranchID:  branch.ID,
			Details:   fmt.Sprintf("Branch created by agent '%s', but kill requested by '%s'", originalRequester, requestingAgent),
			Timestamp: time.Now(),
		}
	}

	return nil
}

// ValidateFullKillSwitchPrerequisites performs all pre-kill validations and returns clear errors
//
// This is the primary validation entry point that should be called before any kill operation.
// It validates in order:
//  1. Branch exists
//  2. Branch is not protected
//  3. Branch has valid status
//  4. Branch has no pending work
//  5. Requesting agent owns the branch
//
// Returns nil if all validations pass, or the first validation error encountered
func (v *KillSwitchValidator) ValidateFullKillSwitchPrerequisites(
	branch *SpeculativeBranch,
	branchID string,
	requestingAgent string,
) *BranchValidationError {
	// 1. Check branch exists
	if err := v.ValidateBranchExists(branch, branchID); err != nil {
		return err
	}

	// 2. Check branch is not protected
	if err := v.ValidateBranchNotProtected(branchID); err != nil {
		return err
	}

	// 3. Check branch status is valid
	if err := v.ValidateBranchStatus(branch); err != nil {
		return err
	}

	// 4. Check no pending work
	if err := v.ValidateNoPendingWork(branch); err != nil {
		return err
	}

	// 5. Check ownership
	if err := v.ValidateOwnership(branch, requestingAgent); err != nil {
		return err
	}

	return nil
}

// BranchHealthReport provides a detailed status report for a branch
type BranchHealthReport struct {
	BranchID         string
	Status           BranchStatus
	IsKilled         bool
	IsProtected      bool
	HasPendingWork   bool
	Owner            string
	CreatedAt        time.Time
	KilledAt         *time.Time
	KillReason       string
	ValidationIssues []string
	CanBeKilled      bool
}

// GenerateHealthReport creates a detailed report on branch state and killability
func (v *KillSwitchValidator) GenerateHealthReport(branch *SpeculativeBranch, branchID string) *BranchHealthReport {
	report := &BranchHealthReport{
		BranchID:         branchID,
		IsKilled:         false,
		IsProtected:      false,
		HasPendingWork:   false,
		CanBeKilled:      true,
		ValidationIssues: []string{},
	}

	if branch == nil {
		report.ValidationIssues = append(report.ValidationIssues, "Branch does not exist")
		report.CanBeKilled = false
		return report
	}

	report.Status = branch.Status
	report.IsKilled = branch.Status == BranchStatusKilled
	report.KilledAt = branch.KilledAt
	report.KillReason = branch.KillReason

	if len(branch.Changes) > 0 {
		report.Owner = branch.Changes[0].ID
		report.CreatedAt = branch.Changes[0].CreatedAt
	}

	// Check protected status
	if err := v.ValidateBranchNotProtected(branchID); err != nil {
		report.IsProtected = true
		report.ValidationIssues = append(report.ValidationIssues, err.Details)
		report.CanBeKilled = false
	}

	// Check pending work
	if err := v.ValidateNoPendingWork(branch); err != nil {
		report.HasPendingWork = true
		report.ValidationIssues = append(report.ValidationIssues, err.Details)
		report.CanBeKilled = false
	}

	// Check status validity
	if err := v.ValidateBranchStatus(branch); err != nil {
		report.ValidationIssues = append(report.ValidationIssues, err.Details)
		report.CanBeKilled = false
	}

	return report
}
