package mergequeue

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateBranchExists_Success(t *testing.T) {
	validator := NewKillSwitchValidator()
	branch := &SpeculativeBranch{
		ID:     "branch-1",
		Status: BranchStatusTesting,
	}

	err := validator.ValidateBranchExists(branch, "branch-1")
	require.Nil(t, err, "Should not error for existing branch")
}

func TestValidateBranchExists_Failure(t *testing.T) {
	validator := NewKillSwitchValidator()

	err := validator.ValidateBranchExists(nil, "non-existent")
	require.NotNil(t, err, "Should error for non-existent branch")
	assert.Equal(t, ValidationCodeBranchNotFound, err.Code)
	assert.Contains(t, err.Details, "non-existent")
}

func TestValidateBranchNotProtected_MainBranch(t *testing.T) {
	validator := NewKillSwitchValidator()

	err := validator.ValidateBranchNotProtected("main")
	require.NotNil(t, err, "Should error for main branch")
	assert.Equal(t, ValidationCodeBranchProtected, err.Code)
	assert.Contains(t, err.Message, "protected")
}

func TestValidateBranchNotProtected_MasterBranch(t *testing.T) {
	validator := NewKillSwitchValidator()

	err := validator.ValidateBranchNotProtected("master")
	require.NotNil(t, err, "Should error for master branch")
	assert.Equal(t, ValidationCodeBranchProtected, err.Code)
}

func TestValidateBranchNotProtected_ReleaseBranch(t *testing.T) {
	validator := NewKillSwitchValidator()

	err := validator.ValidateBranchNotProtected("release/v1.0.0")
	require.NotNil(t, err, "Should error for release branch")
	assert.Equal(t, ValidationCodeBranchProtected, err.Code)
	assert.Contains(t, err.Details, "release/")
}

func TestValidateBranchNotProtected_HotfixBranch(t *testing.T) {
	validator := NewKillSwitchValidator()

	err := validator.ValidateBranchNotProtected("hotfix/critical-bug")
	require.NotNil(t, err, "Should error for hotfix branch")
	assert.Equal(t, ValidationCodeBranchProtected, err.Code)
	assert.Contains(t, err.Details, "hotfix/")
}

func TestValidateBranchNotProtected_RegularBranch(t *testing.T) {
	validator := NewKillSwitchValidator()

	err := validator.ValidateBranchNotProtected("feature/new-feature")
	require.Nil(t, err, "Should not error for regular feature branch")
}

func TestValidateBranchNotProtected_CustomProtected(t *testing.T) {
	validator := NewKillSwitchValidator()
	validator.AddProtectedBranch("staging")

	err := validator.ValidateBranchNotProtected("staging")
	require.NotNil(t, err, "Should error for custom protected branch")
	assert.Equal(t, ValidationCodeBranchProtected, err.Code)
}

func TestValidateBranchStatus_ValidStatuses(t *testing.T) {
	validator := NewKillSwitchValidator()

	validStatuses := []BranchStatus{
		BranchStatusPending,
		BranchStatusTesting,
		BranchStatusFailed,
		BranchStatusPassed,
	}

	for _, status := range validStatuses {
		branch := &SpeculativeBranch{
			ID:     "branch-1",
			Status: status,
		}
		err := validator.ValidateBranchStatus(branch)
		require.Nil(t, err, "Should allow status %s", status)
	}
}

func TestValidateBranchStatus_AlreadyKilled(t *testing.T) {
	validator := NewKillSwitchValidator()
	killedTime := time.Now()
	branch := &SpeculativeBranch{
		ID:       "branch-1",
		Status:   BranchStatusKilled,
		KilledAt: &killedTime,
	}

	err := validator.ValidateBranchStatus(branch)
	require.Nil(t, err, "Already killed status should be idempotent")
}

func TestValidateBranchStatus_NilBranch(t *testing.T) {
	validator := NewKillSwitchValidator()

	err := validator.ValidateBranchStatus(nil)
	require.NotNil(t, err)
	assert.Equal(t, ValidationCodeInvalidStatus, err.Code)
}

func TestValidateNoPendingWork_Success(t *testing.T) {
	validator := NewKillSwitchValidator()
	branch := &SpeculativeBranch{
		ID:     "branch-1",
		Status: BranchStatusFailed,
	}

	err := validator.ValidateNoPendingWork(branch)
	require.Nil(t, err, "Failed status should have no pending work")
}

func TestValidateNoPendingWork_ActiveTesting(t *testing.T) {
	validator := NewKillSwitchValidator()
	branch := &SpeculativeBranch{
		ID:         "branch-1",
		Status:     BranchStatusTesting,
		WorkflowID: "workflow-123",
	}

	err := validator.ValidateNoPendingWork(branch)
	require.NotNil(t, err)
	assert.Equal(t, ValidationCodePendingWork, err.Code)
	assert.Contains(t, err.Details, "currently testing")
}

func TestValidateNoPendingWork_ActiveContainer(t *testing.T) {
	validator := NewKillSwitchValidator()
	branch := &SpeculativeBranch{
		ID:          "branch-1",
		Status:      BranchStatusPending,
		ContainerID: "container-123",
	}

	err := validator.ValidateNoPendingWork(branch)
	require.NotNil(t, err)
	assert.Equal(t, ValidationCodePendingWork, err.Code)
	assert.Contains(t, err.Details, "Docker container")
}

func TestValidateOwnership_SystemAgent(t *testing.T) {
	validator := NewKillSwitchValidator()
	branch := &SpeculativeBranch{
		ID: "branch-1",
		Changes: []ChangeRequest{
			{ID: "user-agent-1"},
		},
	}

	err := validator.ValidateOwnership(branch, "system")
	require.Nil(t, err, "System agent should be able to kill any branch")
}

func TestValidateOwnership_AdminAgent(t *testing.T) {
	validator := NewKillSwitchValidator()
	branch := &SpeculativeBranch{
		ID: "branch-1",
		Changes: []ChangeRequest{
			{ID: "user-agent-1"},
		},
	}

	err := validator.ValidateOwnership(branch, "admin")
	require.Nil(t, err, "Admin agent should be able to kill any branch")
}

func TestValidateOwnership_CoordinatorAgent(t *testing.T) {
	validator := NewKillSwitchValidator()
	branch := &SpeculativeBranch{
		ID: "branch-1",
		Changes: []ChangeRequest{
			{ID: "user-agent-1"},
		},
	}

	err := validator.ValidateOwnership(branch, "coordinator")
	require.Nil(t, err, "Coordinator should be able to kill any branch")
}

func TestValidateOwnership_OwnerAgent(t *testing.T) {
	validator := NewKillSwitchValidator()
	branch := &SpeculativeBranch{
		ID: "branch-1",
		Changes: []ChangeRequest{
			{ID: "user-agent-1"},
		},
	}

	err := validator.ValidateOwnership(branch, "user-agent-1")
	require.Nil(t, err, "Owner agent should be able to kill own branch")
}

func TestValidateOwnership_NonOwnerAgent(t *testing.T) {
	validator := NewKillSwitchValidator()
	branch := &SpeculativeBranch{
		ID: "branch-1",
		Changes: []ChangeRequest{
			{ID: "user-agent-1"},
		},
	}

	err := validator.ValidateOwnership(branch, "user-agent-2")
	require.NotNil(t, err)
	assert.Equal(t, ValidationCodeOwnershipMismatch, err.Code)
	assert.Contains(t, err.Details, "user-agent-1")
	assert.Contains(t, err.Details, "user-agent-2")
}

func TestValidateFullKillSwitchPrerequisites_AllValid(t *testing.T) {
	validator := NewKillSwitchValidator()
	branch := &SpeculativeBranch{
		ID:     "feature-branch",
		Status: BranchStatusFailed,
		Changes: []ChangeRequest{
			{ID: "agent-1"},
		},
	}

	err := validator.ValidateFullKillSwitchPrerequisites(branch, "feature-branch", "agent-1")
	require.Nil(t, err, "All validations should pass")
}

func TestValidateFullKillSwitchPrerequisites_ProtectedBranch(t *testing.T) {
	validator := NewKillSwitchValidator()
	branch := &SpeculativeBranch{
		ID:     "main",
		Status: BranchStatusFailed,
		Changes: []ChangeRequest{
			{ID: "agent-1"},
		},
	}

	err := validator.ValidateFullKillSwitchPrerequisites(branch, "main", "agent-1")
	require.NotNil(t, err)
	assert.Equal(t, ValidationCodeBranchProtected, err.Code)
}

func TestValidateFullKillSwitchPrerequisites_OwnershipMismatch(t *testing.T) {
	validator := NewKillSwitchValidator()
	branch := &SpeculativeBranch{
		ID:     "feature-branch",
		Status: BranchStatusFailed,
		Changes: []ChangeRequest{
			{ID: "agent-1"},
		},
	}

	err := validator.ValidateFullKillSwitchPrerequisites(branch, "feature-branch", "agent-2")
	require.NotNil(t, err)
	assert.Equal(t, ValidationCodeOwnershipMismatch, err.Code)
}

func TestValidateFullKillSwitchPrerequisites_BranchNotFound(t *testing.T) {
	validator := NewKillSwitchValidator()

	err := validator.ValidateFullKillSwitchPrerequisites(nil, "non-existent", "agent-1")
	require.NotNil(t, err)
	assert.Equal(t, ValidationCodeBranchNotFound, err.Code)
}

func TestValidateFullKillSwitchPrerequisites_PendingWork(t *testing.T) {
	validator := NewKillSwitchValidator()
	branch := &SpeculativeBranch{
		ID:         "feature-branch",
		Status:     BranchStatusTesting,
		WorkflowID: "workflow-123",
		Changes: []ChangeRequest{
			{ID: "agent-1"},
		},
	}

	err := validator.ValidateFullKillSwitchPrerequisites(branch, "feature-branch", "agent-1")
	require.NotNil(t, err)
	assert.Equal(t, ValidationCodePendingWork, err.Code)
}

func TestGenerateHealthReport_HealthyBranch(t *testing.T) {
	validator := NewKillSwitchValidator()
	createdAt := time.Now()
	branch := &SpeculativeBranch{
		ID:     "feature-branch",
		Status: BranchStatusFailed,
		Changes: []ChangeRequest{
			{ID: "agent-1", CreatedAt: createdAt},
		},
	}

	report := validator.GenerateHealthReport(branch, "feature-branch")

	assert.Equal(t, "feature-branch", report.BranchID)
	assert.Equal(t, BranchStatusFailed, report.Status)
	assert.False(t, report.IsKilled)
	assert.False(t, report.IsProtected)
	assert.False(t, report.HasPendingWork)
	assert.True(t, report.CanBeKilled)
	assert.Equal(t, "agent-1", report.Owner)
	assert.Equal(t, createdAt, report.CreatedAt)
	assert.Empty(t, report.ValidationIssues)
}

func TestGenerateHealthReport_ProtectedBranch(t *testing.T) {
	validator := NewKillSwitchValidator()
	branch := &SpeculativeBranch{
		ID:     "main",
		Status: BranchStatusFailed,
	}

	report := validator.GenerateHealthReport(branch, "main")

	assert.True(t, report.IsProtected)
	assert.False(t, report.CanBeKilled)
	assert.NotEmpty(t, report.ValidationIssues)
	// Check that at least one validation issue contains "protected"
	foundProtected := false
	for _, issue := range report.ValidationIssues {
		if strings.Contains(issue, "protected") {
			foundProtected = true
			break
		}
	}
	assert.True(t, foundProtected, "Expected validation issues to contain mention of 'protected'")
}

func TestGenerateHealthReport_KilledBranch(t *testing.T) {
	validator := NewKillSwitchValidator()
	killedAt := time.Now()
	branch := &SpeculativeBranch{
		ID:         "feature-branch",
		Status:     BranchStatusKilled,
		KilledAt:   &killedAt,
		KillReason: "test timeout",
	}

	report := validator.GenerateHealthReport(branch, "feature-branch")

	assert.True(t, report.IsKilled)
	assert.Equal(t, killedAt, *report.KilledAt)
	assert.Equal(t, "test timeout", report.KillReason)
}

func TestGenerateHealthReport_NonExistentBranch(t *testing.T) {
	validator := NewKillSwitchValidator()

	report := validator.GenerateHealthReport(nil, "non-existent")

	assert.Equal(t, "non-existent", report.BranchID)
	assert.False(t, report.CanBeKilled)
	assert.NotEmpty(t, report.ValidationIssues)
	// Check that at least one validation issue contains "does not exist"
	foundIssue := false
	for _, issue := range report.ValidationIssues {
		if strings.Contains(issue, "does not exist") {
			foundIssue = true
			break
		}
	}
	assert.True(t, foundIssue, "Expected validation issues to contain mention of 'does not exist'")
}

func TestBranchValidationError_Error(t *testing.T) {
	err := &BranchValidationError{
		Code:      ValidationCodeBranchProtected,
		Message:   "Cannot kill protected branch",
		BranchID:  "main",
		Details:   "Branch 'main' is a protected branch",
		Timestamp: time.Now(),
	}

	errorStr := err.Error()
	assert.Contains(t, errorStr, ValidationCodeBranchProtected)
	assert.Contains(t, errorStr, "Cannot kill protected branch")
	assert.Contains(t, errorStr, "main")
}
