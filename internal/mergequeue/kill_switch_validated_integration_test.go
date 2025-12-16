//go:build integration
// +build integration

package mergequeue

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKillFailedBranchWithValidation_Success(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{
		KillSwitchTimeout: 500 * time.Millisecond,
	})
	ctx := context.Background()

	// Create a test branch
	branch := &SpeculativeBranch{
		ID:     "feature-branch",
		Status: BranchStatusFailed,
		Changes: []ChangeRequest{
			{
				ID:            "agent-1",
				FilesModified: []string{"src/main.go"},
			},
		},
	}

	coord.mu.Lock()
	coord.activeBranches = map[string]*SpeculativeBranch{
		"feature-branch": branch,
	}
	coord.mu.Unlock()

	// Kill with validation should succeed
	err := coord.KillFailedBranchWithValidation(ctx, "feature-branch", "test failure")
	require.Nil(t, err, "Kill should succeed with valid prerequisites")

	// Verify branch was killed
	coord.mu.RLock()
	killedBranch := coord.activeBranches["feature-branch"]
	coord.mu.RUnlock()

	assert.Equal(t, BranchStatusKilled, killedBranch.Status)
	assert.NotNil(t, killedBranch.KilledAt)
	assert.Equal(t, "test failure", killedBranch.KillReason)
}

func TestKillFailedBranchWithValidation_ProtectedBranch(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{
		KillSwitchTimeout: 500 * time.Millisecond,
	})
	ctx := context.Background()

	// Create a protected main branch
	branch := &SpeculativeBranch{
		ID:     "main",
		Status: BranchStatusFailed,
		Changes: []ChangeRequest{
			{ID: "agent-1"},
		},
	}

	coord.mu.Lock()
	coord.activeBranches = map[string]*SpeculativeBranch{
		"main": branch,
	}
	coord.mu.Unlock()

	// Kill should fail due to protection
	err := coord.KillFailedBranchWithValidation(ctx, "main", "test failure")
	require.NotNil(t, err)
	assert.Equal(t, ValidationCodeBranchProtected, err.Code)
	assert.Contains(t, err.Message, "protected")

	// Verify branch was NOT killed
	coord.mu.RLock()
	mainBranch := coord.activeBranches["main"]
	coord.mu.RUnlock()

	assert.NotEqual(t, BranchStatusKilled, mainBranch.Status)
}

func TestKillFailedBranchWithValidation_BranchNotFound(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{
		KillSwitchTimeout: 500 * time.Millisecond,
	})
	ctx := context.Background()

	coord.mu.Lock()
	coord.activeBranches = map[string]*SpeculativeBranch{}
	coord.mu.Unlock()

	// Kill non-existent branch should fail
	err := coord.KillFailedBranchWithValidation(ctx, "non-existent", "test failure")
	require.NotNil(t, err)
	assert.Equal(t, ValidationCodeBranchNotFound, err.Code)
}

func TestKillFailedBranchWithValidation_OwnershipMismatch(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{
		KillSwitchTimeout: 500 * time.Millisecond,
	})
	ctx := context.Background()

	// Create a branch owned by agent-1
	branch := &SpeculativeBranch{
		ID:     "feature-branch",
		Status: BranchStatusFailed,
		Changes: []ChangeRequest{
			{ID: "agent-1"},
		},
	}

	coord.mu.Lock()
	coord.activeBranches = map[string]*SpeculativeBranch{
		"feature-branch": branch,
	}
	coord.mu.Unlock()

	// Try to kill as agent-2 (different owner)
	err := coord.KillFailedBranchWithValidation(ctx, "feature-branch", "test failure")
	require.NotNil(t, err)
	assert.Equal(t, ValidationCodeOwnershipMismatch, err.Code)

	// Verify branch was NOT killed
	coord.mu.RLock()
	unchangedBranch := coord.activeBranches["feature-branch"]
	coord.mu.RUnlock()

	assert.NotEqual(t, BranchStatusKilled, unchangedBranch.Status)
}

func TestKillFailedBranchWithValidation_SystemAgentCanKillAny(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{
		KillSwitchTimeout: 500 * time.Millisecond,
	})
	ctx := context.Background()

	// Create a branch owned by agent-1
	branch := &SpeculativeBranch{
		ID:     "feature-branch",
		Status: BranchStatusFailed,
		Changes: []ChangeRequest{
			{ID: "agent-1"},
		},
	}

	coord.mu.Lock()
	coord.activeBranches = map[string]*SpeculativeBranch{
		"feature-branch": branch,
	}
	coord.mu.Unlock()

	// System agent can kill any branch
	err := coord.KillFailedBranchWithValidation(ctx, "feature-branch", "test failure")
	require.Nil(t, err, "System agent should be able to kill any branch")

	// Verify branch was killed
	coord.mu.RLock()
	killedBranch := coord.activeBranches["feature-branch"]
	coord.mu.RUnlock()

	assert.Equal(t, BranchStatusKilled, killedBranch.Status)
}

func TestKillFailedBranchWithValidation_PendingWork(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{
		KillSwitchTimeout: 500 * time.Millisecond,
	})
	ctx := context.Background()

	// Create a branch with active testing
	branch := &SpeculativeBranch{
		ID:         "feature-branch",
		Status:     BranchStatusTesting,
		WorkflowID: "workflow-123",
		Changes: []ChangeRequest{
			{ID: "agent-1"},
		},
	}

	coord.mu.Lock()
	coord.activeBranches = map[string]*SpeculativeBranch{
		"feature-branch": branch,
	}
	coord.mu.Unlock()

	// Kill should fail due to pending work
	err := coord.KillFailedBranchWithValidation(ctx, "feature-branch", "test failure")
	require.NotNil(t, err)
	assert.Equal(t, ValidationCodePendingWork, err.Code)

	// Verify branch was NOT killed
	coord.mu.RLock()
	unchangedBranch := coord.activeBranches["feature-branch"]
	coord.mu.RUnlock()

	assert.NotEqual(t, BranchStatusKilled, unchangedBranch.Status)
}

func TestKillDependentBranchesWithValidation_Success(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{
		KillSwitchTimeout: 500 * time.Millisecond,
	})
	ctx := context.Background()

	// Create a parent-child hierarchy
	parent := &SpeculativeBranch{
		ID:          "parent",
		Status:      BranchStatusFailed,
		ChildrenIDs: []string{"child-1", "child-2"},
		Changes: []ChangeRequest{
			{ID: "agent-1"},
		},
	}
	child1 := &SpeculativeBranch{
		ID:       "child-1",
		Status:   BranchStatusTesting,
		ParentID: "parent",
	}
	child2 := &SpeculativeBranch{
		ID:       "child-2",
		Status:   BranchStatusTesting,
		ParentID: "parent",
	}

	coord.mu.Lock()
	coord.activeBranches = map[string]*SpeculativeBranch{
		"parent":  parent,
		"child-1": child1,
		"child-2": child2,
	}
	coord.mu.Unlock()

	// Kill dependents with validation should succeed
	validationErr, cascadeErr := coord.KillDependentBranchesWithValidation(ctx, "parent")
	require.Nil(t, validationErr, "Validation should pass")
	require.Nil(t, cascadeErr, "Cascade should succeed")

	// Verify all children are killed
	coord.mu.RLock()
	defer coord.mu.RUnlock()

	assert.Equal(t, BranchStatusKilled, coord.activeBranches["child-1"].Status)
	assert.Equal(t, BranchStatusKilled, coord.activeBranches["child-2"].Status)
	assert.Contains(t, coord.activeBranches["child-1"].KillReason, "parent branch parent failed")
	assert.Contains(t, coord.activeBranches["child-2"].KillReason, "parent branch parent failed")
}

func TestKillDependentBranchesWithValidation_ParentProtected(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{
		KillSwitchTimeout: 500 * time.Millisecond,
	})
	ctx := context.Background()

	// Create a protected parent with children
	parent := &SpeculativeBranch{
		ID:          "main",
		Status:      BranchStatusFailed,
		ChildrenIDs: []string{"child-1"},
		Changes: []ChangeRequest{
			{ID: "agent-1"},
		},
	}
	child := &SpeculativeBranch{
		ID:       "child-1",
		Status:   BranchStatusTesting,
		ParentID: "main",
	}

	coord.mu.Lock()
	coord.activeBranches = map[string]*SpeculativeBranch{
		"main":    parent,
		"child-1": child,
	}
	coord.mu.Unlock()

	// Kill dependents should fail validation
	validationErr, cascadeErr := coord.KillDependentBranchesWithValidation(ctx, "main")
	require.NotNil(t, validationErr, "Validation should fail")
	require.Nil(t, cascadeErr, "Cascade should not run")
	assert.Equal(t, ValidationCodeBranchProtected, validationErr.Code)

	// Verify children were NOT killed
	coord.mu.RLock()
	defer coord.mu.RUnlock()

	assert.NotEqual(t, BranchStatusKilled, coord.activeBranches["child-1"].Status)
}

func TestKillDependentBranchesWithValidation_ParentNotFound(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{
		KillSwitchTimeout: 500 * time.Millisecond,
	})
	ctx := context.Background()

	coord.mu.Lock()
	coord.activeBranches = map[string]*SpeculativeBranch{}
	coord.mu.Unlock()

	// Kill dependents of non-existent parent should fail
	validationErr, cascadeErr := coord.KillDependentBranchesWithValidation(ctx, "non-existent")
	require.NotNil(t, validationErr, "Validation should fail")
	require.Nil(t, cascadeErr, "Cascade should not run")
	assert.Equal(t, ValidationCodeBranchNotFound, validationErr.Code)
}

func TestKillDependentBranchesWithValidation_OwnershipMismatch(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{
		KillSwitchTimeout: 500 * time.Millisecond,
	})
	ctx := context.Background()

	// Create a parent owned by agent-1
	parent := &SpeculativeBranch{
		ID:          "parent",
		Status:      BranchStatusFailed,
		ChildrenIDs: []string{"child-1"},
		Changes: []ChangeRequest{
			{ID: "agent-1"},
		},
	}
	child := &SpeculativeBranch{
		ID:       "child-1",
		Status:   BranchStatusTesting,
		ParentID: "parent",
	}

	coord.mu.Lock()
	coord.activeBranches = map[string]*SpeculativeBranch{
		"parent":  parent,
		"child-1": child,
	}
	coord.mu.Unlock()

	// Try to kill as agent-2 (different owner)
	validationErr, cascadeErr := coord.KillDependentBranchesWithValidation(ctx, "parent", "agent-2")
	require.NotNil(t, validationErr, "Validation should fail")
	require.Nil(t, cascadeErr, "Cascade should not run")
	assert.Equal(t, ValidationCodeOwnershipMismatch, validationErr.Code)

	// Verify children were NOT killed
	coord.mu.RLock()
	defer coord.mu.RUnlock()

	assert.NotEqual(t, BranchStatusKilled, coord.activeBranches["child-1"].Status)
}

func TestGetBranchHealthReport_HealthyBranch(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{
		KillSwitchTimeout: 500 * time.Millisecond,
	})

	createdAt := time.Now()
	branch := &SpeculativeBranch{
		ID:     "feature-branch",
		Status: BranchStatusFailed,
		Changes: []ChangeRequest{
			{ID: "agent-1", CreatedAt: createdAt},
		},
	}

	coord.mu.Lock()
	coord.activeBranches = map[string]*SpeculativeBranch{
		"feature-branch": branch,
	}
	coord.mu.Unlock()

	report := coord.GetBranchHealthReport("feature-branch")

	require.NotNil(t, report)
	assert.Equal(t, "feature-branch", report.BranchID)
	assert.Equal(t, BranchStatusFailed, report.Status)
	assert.False(t, report.IsKilled)
	assert.False(t, report.IsProtected)
	assert.False(t, report.HasPendingWork)
	assert.True(t, report.CanBeKilled)
	assert.Equal(t, "agent-1", report.Owner)
	assert.Empty(t, report.ValidationIssues)
}

func TestGetBranchHealthReport_ProtectedBranch(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{
		KillSwitchTimeout: 500 * time.Millisecond,
	})

	branch := &SpeculativeBranch{
		ID:     "main",
		Status: BranchStatusFailed,
	}

	coord.mu.Lock()
	coord.activeBranches = map[string]*SpeculativeBranch{
		"main": branch,
	}
	coord.mu.Unlock()

	report := coord.GetBranchHealthReport("main")

	require.NotNil(t, report)
	assert.True(t, report.IsProtected)
	assert.False(t, report.CanBeKilled)
	assert.NotEmpty(t, report.ValidationIssues)
}

func TestGetBranchHealthReport_NonExistent(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{
		KillSwitchTimeout: 500 * time.Millisecond,
	})

	coord.mu.Lock()
	coord.activeBranches = map[string]*SpeculativeBranch{}
	coord.mu.Unlock()

	report := coord.GetBranchHealthReport("non-existent")

	require.NotNil(t, report)
	assert.False(t, report.CanBeKilled)
	assert.NotEmpty(t, report.ValidationIssues)
	assert.Contains(t, report.ValidationIssues, "does not exist")
}

func TestGetBranchHealthReport_KilledBranch(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{
		KillSwitchTimeout: 500 * time.Millisecond,
	})

	killedAt := time.Now()
	branch := &SpeculativeBranch{
		ID:         "feature-branch",
		Status:     BranchStatusKilled,
		KilledAt:   &killedAt,
		KillReason: "test timeout",
	}

	coord.mu.Lock()
	coord.activeBranches = map[string]*SpeculativeBranch{
		"feature-branch": branch,
	}
	coord.mu.Unlock()

	report := coord.GetBranchHealthReport("feature-branch")

	require.NotNil(t, report)
	assert.True(t, report.IsKilled)
	assert.Equal(t, killedAt, *report.KilledAt)
	assert.Equal(t, "test timeout", report.KillReason)
}
