package mergequeue

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKillFailedBranchWithTimeout_Success(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{
		KillSwitchTimeout: 500 * time.Millisecond,
	})
	ctx := context.Background()

	// Create a test branch
	branch := &SpeculativeBranch{
		ID:     "branch-1",
		Status: BranchStatusTesting,
		Changes: []ChangeRequest{
			{
				ID:            "agent-1",
				FilesModified: []string{"src/auth/login.ts"},
			},
		},
	}

	coord.mu.Lock()
	coord.activeBranches = map[string]*SpeculativeBranch{
		"branch-1": branch,
	}
	coord.mu.Unlock()

	// Kill the branch (should complete quickly)
	startTime := time.Now()
	err := coord.killFailedBranchWithTimeout(ctx, "branch-1", "test failure")
	duration := time.Since(startTime)

	require.NoError(t, err)
	assert.Less(t, duration, 500*time.Millisecond, "Should complete faster than timeout")

	// Verify branch was killed
	coord.mu.RLock()
	killedBranch := coord.activeBranches["branch-1"]
	coord.mu.RUnlock()

	assert.Equal(t, BranchStatusKilled, killedBranch.Status)
	assert.NotNil(t, killedBranch.KilledAt)
	assert.Equal(t, "test failure", killedBranch.KillReason)
	assert.NotContains(t, killedBranch.KillReason, "timeout", "Should not indicate timeout when successful")
}

func TestKillFailedBranchWithTimeout_NonExistentBranch(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{
		KillSwitchTimeout: 500 * time.Millisecond,
	})
	ctx := context.Background()

	coord.mu.Lock()
	coord.activeBranches = map[string]*SpeculativeBranch{}
	coord.mu.Unlock()

	// Try to kill a non-existent branch
	err := coord.killFailedBranchWithTimeout(ctx, "non-existent", "test failure")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "branch non-existent not found")
}

func TestKillFailedBranchWithTimeout_Idempotent(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{
		KillSwitchTimeout: 500 * time.Millisecond,
	})
	ctx := context.Background()

	// Create a branch that's already killed
	killedTime := time.Now().Add(-1 * time.Hour)
	branch := &SpeculativeBranch{
		ID:         "branch-1",
		Status:     BranchStatusKilled,
		KilledAt:   &killedTime,
		KillReason: "previous failure",
	}

	coord.mu.Lock()
	coord.activeBranches = map[string]*SpeculativeBranch{
		"branch-1": branch,
	}
	initialKills := coord.stats.TotalKills
	coord.mu.Unlock()

	// Try to kill it again
	err := coord.killFailedBranchWithTimeout(ctx, "branch-1", "new failure")
	require.NoError(t, err)

	// Verify the original kill data is preserved
	coord.mu.RLock()
	killedBranch := coord.activeBranches["branch-1"]
	finalKills := coord.stats.TotalKills
	coord.mu.RUnlock()

	assert.Equal(t, BranchStatusKilled, killedBranch.Status)
	assert.Equal(t, killedTime, *killedBranch.KilledAt, "KilledAt should not change")
	assert.Equal(t, "previous failure", killedBranch.KillReason, "KillReason should not change")
	assert.Equal(t, initialKills, finalKills, "TotalKills should not increment for idempotent kill")
}

func TestKillFailedBranchWithTimeout_GracefulDegradation(t *testing.T) {
	// Use a very short timeout to simulate timeout scenario
	coord := NewCoordinator(CoordinatorConfig{
		KillSwitchTimeout: 1 * time.Nanosecond, // Extremely short timeout to force timeout
	})
	ctx := context.Background()

	// Create a test branch
	branch := &SpeculativeBranch{
		ID:     "branch-1",
		Status: BranchStatusTesting,
	}

	coord.mu.Lock()
	coord.activeBranches = map[string]*SpeculativeBranch{
		"branch-1": branch,
	}
	coord.mu.Unlock()

	// Kill the branch (should timeout)
	err := coord.killFailedBranchWithTimeout(ctx, "branch-1", "test failure")

	// Should return timeout error
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timeout", "Error should mention timeout")

	// But branch should still be marked as killed (graceful degradation)
	coord.mu.RLock()
	killedBranch := coord.activeBranches["branch-1"]
	coord.mu.RUnlock()

	assert.Equal(t, BranchStatusKilled, killedBranch.Status, "Branch should be killed despite timeout")
	assert.NotNil(t, killedBranch.KilledAt, "KilledAt should be set")
	assert.Contains(t, killedBranch.KillReason, "timeout during cleanup", "Reason should indicate timeout")
	assert.Contains(t, killedBranch.KillReason, "test failure", "Reason should include original reason")
}

func TestKillFailedBranchWithTimeout_PreservesOtherBranches(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{
		KillSwitchTimeout: 500 * time.Millisecond,
	})
	ctx := context.Background()

	// Create multiple branches
	branch1 := &SpeculativeBranch{
		ID:     "branch-1",
		Status: BranchStatusTesting,
	}
	branch2 := &SpeculativeBranch{
		ID:     "branch-2",
		Status: BranchStatusTesting,
	}

	coord.mu.Lock()
	coord.activeBranches = map[string]*SpeculativeBranch{
		"branch-1": branch1,
		"branch-2": branch2,
	}
	coord.mu.Unlock()

	// Kill only branch-1
	err := coord.killFailedBranchWithTimeout(ctx, "branch-1", "test failure")
	require.NoError(t, err)

	// Verify branch-1 is killed but branch-2 is untouched
	coord.mu.RLock()
	defer coord.mu.RUnlock()

	assert.Equal(t, BranchStatusKilled, coord.activeBranches["branch-1"].Status)
	assert.Equal(t, BranchStatusTesting, coord.activeBranches["branch-2"].Status)
	assert.Nil(t, coord.activeBranches["branch-2"].KilledAt)
	assert.Empty(t, coord.activeBranches["branch-2"].KillReason)
}

func TestKillFailedBranchWithTimeout_MetricsTracking(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{
		KillSwitchTimeout: 500 * time.Millisecond,
	})
	ctx := context.Background()

	// Create test branches
	branch1 := &SpeculativeBranch{
		ID:     "branch-1",
		Status: BranchStatusTesting,
	}
	branch2 := &SpeculativeBranch{
		ID:     "branch-2",
		Status: BranchStatusTesting,
	}

	coord.mu.Lock()
	coord.activeBranches = map[string]*SpeculativeBranch{
		"branch-1": branch1,
		"branch-2": branch2,
	}
	coord.mu.Unlock()

	// Verify initial stats
	stats := coord.GetStats()
	assert.Equal(t, int64(0), stats.TotalKills)

	// Kill branch-1
	err := coord.killFailedBranchWithTimeout(ctx, "branch-1", "test failure")
	require.NoError(t, err)

	// Verify kill counter incremented
	stats = coord.GetStats()
	assert.Equal(t, int64(1), stats.TotalKills)

	// Kill branch-2
	err = coord.killFailedBranchWithTimeout(ctx, "branch-2", "another failure")
	require.NoError(t, err)

	// Verify kill counter incremented again
	stats = coord.GetStats()
	assert.Equal(t, int64(2), stats.TotalKills)
}

func TestKillDependentBranchesWithTimeout_SimpleHierarchy(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{
		KillSwitchTimeout: 500 * time.Millisecond,
	})
	ctx := context.Background()

	// Create a simple parent-child hierarchy
	parent := &SpeculativeBranch{
		ID:          "parent",
		Status:      BranchStatusFailed,
		ChildrenIDs: []string{"child-1", "child-2"},
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

	// Kill all dependent branches
	err := coord.killDependentBranchesWithTimeout(ctx, "parent")
	require.NoError(t, err)

	// Verify all children are killed
	coord.mu.RLock()
	defer coord.mu.RUnlock()

	assert.Equal(t, BranchStatusKilled, coord.activeBranches["child-1"].Status)
	assert.Equal(t, BranchStatusKilled, coord.activeBranches["child-2"].Status)
	assert.Contains(t, coord.activeBranches["child-1"].KillReason, "parent branch parent failed")
	assert.Contains(t, coord.activeBranches["child-2"].KillReason, "parent branch parent failed")
}

func TestKillDependentBranchesWithTimeout_DeepHierarchy(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{
		KillSwitchTimeout: 500 * time.Millisecond,
	})
	ctx := context.Background()

	// Create a 3-level deep hierarchy:
	// parent -> child -> grandchild
	parent := &SpeculativeBranch{
		ID:          "parent",
		Status:      BranchStatusFailed,
		ChildrenIDs: []string{"child"},
	}
	child := &SpeculativeBranch{
		ID:          "child",
		Status:      BranchStatusTesting,
		ParentID:    "parent",
		ChildrenIDs: []string{"grandchild"},
	}
	grandchild := &SpeculativeBranch{
		ID:       "grandchild",
		Status:   BranchStatusTesting,
		ParentID: "child",
	}

	coord.mu.Lock()
	coord.activeBranches = map[string]*SpeculativeBranch{
		"parent":     parent,
		"child":      child,
		"grandchild": grandchild,
	}
	coord.mu.Unlock()

	// Kill all dependent branches (should cascade down)
	err := coord.killDependentBranchesWithTimeout(ctx, "parent")
	require.NoError(t, err)

	// Verify entire hierarchy is killed
	coord.mu.RLock()
	defer coord.mu.RUnlock()

	assert.Equal(t, BranchStatusKilled, coord.activeBranches["child"].Status)
	assert.Equal(t, BranchStatusKilled, coord.activeBranches["grandchild"].Status)
	assert.Contains(t, coord.activeBranches["child"].KillReason, "parent branch parent failed")
	assert.Contains(t, coord.activeBranches["grandchild"].KillReason, "parent branch child failed")
}

func TestKillDependentBranchesWithTimeout_MultipleChildren(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{
		KillSwitchTimeout: 500 * time.Millisecond,
	})
	ctx := context.Background()

	// Create a hierarchy with multiple children at each level:
	//        parent
	//       /      \
	//    child-1  child-2
	//     /  \       |
	//   gc1  gc2    gc3
	parent := &SpeculativeBranch{
		ID:          "parent",
		Status:      BranchStatusFailed,
		ChildrenIDs: []string{"child-1", "child-2"},
	}
	child1 := &SpeculativeBranch{
		ID:          "child-1",
		Status:      BranchStatusTesting,
		ParentID:    "parent",
		ChildrenIDs: []string{"grandchild-1", "grandchild-2"},
	}
	child2 := &SpeculativeBranch{
		ID:          "child-2",
		Status:      BranchStatusTesting,
		ParentID:    "parent",
		ChildrenIDs: []string{"grandchild-3"},
	}
	grandchild1 := &SpeculativeBranch{
		ID:       "grandchild-1",
		Status:   BranchStatusTesting,
		ParentID: "child-1",
	}
	grandchild2 := &SpeculativeBranch{
		ID:       "grandchild-2",
		Status:   BranchStatusTesting,
		ParentID: "child-1",
	}
	grandchild3 := &SpeculativeBranch{
		ID:       "grandchild-3",
		Status:   BranchStatusTesting,
		ParentID: "child-2",
	}

	coord.mu.Lock()
	coord.activeBranches = map[string]*SpeculativeBranch{
		"parent":       parent,
		"child-1":      child1,
		"child-2":      child2,
		"grandchild-1": grandchild1,
		"grandchild-2": grandchild2,
		"grandchild-3": grandchild3,
	}
	coord.mu.Unlock()

	// Kill all dependent branches
	err := coord.killDependentBranchesWithTimeout(ctx, "parent")
	require.NoError(t, err)

	// Verify entire tree is killed
	coord.mu.RLock()
	defer coord.mu.RUnlock()

	assert.Equal(t, BranchStatusKilled, coord.activeBranches["child-1"].Status)
	assert.Equal(t, BranchStatusKilled, coord.activeBranches["child-2"].Status)
	assert.Equal(t, BranchStatusKilled, coord.activeBranches["grandchild-1"].Status)
	assert.Equal(t, BranchStatusKilled, coord.activeBranches["grandchild-2"].Status)
	assert.Equal(t, BranchStatusKilled, coord.activeBranches["grandchild-3"].Status)
}

func TestKillDependentBranchesWithTimeout_Timeout(t *testing.T) {
	// Create coordinator with very short timeout
	coord := NewCoordinator(CoordinatorConfig{
		KillSwitchTimeout: 1 * time.Nanosecond, // Extremely short to force timeout
	})

	// Use a context that's already cancelled to force immediate timeout
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Create a simple hierarchy
	parent := &SpeculativeBranch{
		ID:          "parent",
		Status:      BranchStatusFailed,
		ChildrenIDs: []string{"child"},
	}
	child := &SpeculativeBranch{
		ID:       "child",
		Status:   BranchStatusTesting,
		ParentID: "parent",
	}

	coord.mu.Lock()
	coord.activeBranches = map[string]*SpeculativeBranch{
		"parent": parent,
		"child":  child,
	}
	coord.mu.Unlock()

	// Kill should timeout
	err := coord.killDependentBranchesWithTimeout(ctx, "parent")
	require.Error(t, err)
	// Check for timeout in error message or error type
	assert.True(t,
		contains(err.Error(), "timeout") || contains(err.Error(), "timed out"),
		"Error should mention timeout. Got: %v", err)
}

func TestKillDependentBranchesWithTimeout_NonExistentBranch(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{
		KillSwitchTimeout: 500 * time.Millisecond,
	})
	ctx := context.Background()

	coord.mu.Lock()
	coord.activeBranches = map[string]*SpeculativeBranch{}
	coord.mu.Unlock()

	// Try to kill dependents of non-existent branch
	err := coord.killDependentBranchesWithTimeout(ctx, "non-existent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "branch non-existent not found")
}

func TestKillDependentBranchesWithTimeout_AlreadyKilledChildren(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{
		KillSwitchTimeout: 500 * time.Millisecond,
	})
	ctx := context.Background()

	// Create hierarchy where some children are already killed
	killedTime := time.Now().Add(-1 * time.Hour)
	parent := &SpeculativeBranch{
		ID:          "parent",
		Status:      BranchStatusFailed,
		ChildrenIDs: []string{"child-1", "child-2"},
	}
	child1 := &SpeculativeBranch{
		ID:         "child-1",
		Status:     BranchStatusKilled,
		ParentID:   "parent",
		KilledAt:   &killedTime,
		KillReason: "already killed",
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

	// Kill dependents (should be idempotent for child-1)
	err := coord.killDependentBranchesWithTimeout(ctx, "parent")
	require.NoError(t, err)

	// Verify child-1 is unchanged (idempotent) and child-2 is killed
	coord.mu.RLock()
	defer coord.mu.RUnlock()

	assert.Equal(t, BranchStatusKilled, coord.activeBranches["child-1"].Status)
	assert.Equal(t, "already killed", coord.activeBranches["child-1"].KillReason, "Should preserve original reason")
	assert.Equal(t, killedTime, *coord.activeBranches["child-1"].KilledAt, "Should preserve original timestamp")

	assert.Equal(t, BranchStatusKilled, coord.activeBranches["child-2"].Status)
	assert.Contains(t, coord.activeBranches["child-2"].KillReason, "parent branch parent failed")
}
