package mergequeue

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCoordinator(t *testing.T) {
	config := CoordinatorConfig{
		WindowSize:     5,
		MaxBypassSlots: 3,
		DefaultDepth:   5,
	}

	coord := NewCoordinator(config)
	require.NotNil(t, coord)
	assert.Equal(t, 5, coord.config.WindowSize)
	assert.Equal(t, 3, coord.config.MaxBypassSlots)
	assert.Equal(t, 5, coord.config.DefaultDepth)
}

func TestCoordinatorDefaults(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{})
	require.NotNil(t, coord)

	// Check defaults
	assert.Equal(t, 5, coord.config.WindowSize)
	assert.Equal(t, 3, coord.config.MaxBypassSlots)
	assert.Equal(t, 5, coord.config.DefaultDepth)
	assert.Equal(t, 0.90, coord.config.HighPassRateThreshold)
	assert.Equal(t, 0.70, coord.config.LowPassRateThreshold)
	assert.Equal(t, 500*time.Millisecond, coord.config.KillSwitchTimeout)
	assert.Equal(t, 5*time.Minute, coord.config.TestTimeout)
}

func TestSubmitChange(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{})
	ctx := context.Background()

	err := coord.Start(ctx)
	require.NoError(t, err)
	defer func() {
		_ = coord.Stop()
	}()

	change := &ChangeRequest{
		ID:            "agent-1",
		WorktreePath:  "/tmp/agent-1",
		FilesModified: []string{"src/auth/login.ts"},
		CommitSHA:     "abc123",
		CreatedAt:     time.Now(),
	}

	err = coord.Submit(ctx, change)
	assert.NoError(t, err)

	// Give it time to process
	time.Sleep(200 * time.Millisecond)

	coord.mu.RLock()
	// First change with no conflicts goes to bypass lane
	assert.Len(t, coord.bypassLane, 1, "First independent change should go to bypass lane")
	assert.Len(t, coord.mainQueue, 0, "Main queue should be empty for first independent change")
	coord.mu.RUnlock()
}

func TestIndependentChanges(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{})

	change1 := &ChangeRequest{
		ID:            "agent-1",
		FilesModified: []string{"src/auth/login.ts"},
	}

	change2 := &ChangeRequest{
		ID:            "agent-2",
		FilesModified: []string{"src/payments/stripe.ts"},
	}

	// Currently hasConflict always returns false (TODO)
	// So these should be considered independent
	assert.False(t, coord.hasConflict(change1, change2))
}

func TestCalculateDepth(t *testing.T) {
	tests := []struct {
		name         string
		successRate  float64
		defaultDepth int
		expected     int
	}{
		{
			name:         "high success rate increases depth",
			successRate:  0.95,
			defaultDepth: 5,
			expected:     7,
		},
		{
			name:         "low success rate decreases depth",
			successRate:  0.65,
			defaultDepth: 5,
			expected:     3,
		},
		{
			name:         "medium success rate keeps default",
			successRate:  0.80,
			defaultDepth: 5,
			expected:     5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coord := NewCoordinator(CoordinatorConfig{
				DefaultDepth: tt.defaultDepth,
			})
			coord.stats.SuccessRate = tt.successRate

			depth := coord.calculateDepth()
			assert.Equal(t, tt.expected, depth)
		})
	}
}

func TestGetStats(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{})

	// Set some stats
	coord.mu.Lock()
	coord.stats.MergedPerHour = 42.0
	coord.stats.SuccessRate = 0.87
	coord.mu.Unlock()

	stats := coord.GetStats()
	assert.Equal(t, 42.0, stats.MergedPerHour)
	assert.Equal(t, 0.87, stats.SuccessRate)
}

func TestKillFailedBranch_Success(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{})
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

	// Kill the branch
	err := coord.killFailedBranch(ctx, "branch-1", "test failure")
	require.NoError(t, err)

	// Verify branch was killed
	coord.mu.RLock()
	killedBranch := coord.activeBranches["branch-1"]
	coord.mu.RUnlock()

	assert.Equal(t, BranchStatusKilled, killedBranch.Status)
	assert.NotNil(t, killedBranch.KilledAt)
	assert.Equal(t, "test failure", killedBranch.KillReason)
}

func TestKillFailedBranch_NonExistentBranch(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{})
	ctx := context.Background()

	coord.mu.Lock()
	coord.activeBranches = map[string]*SpeculativeBranch{}
	coord.mu.Unlock()

	// Try to kill a non-existent branch
	err := coord.killFailedBranch(ctx, "non-existent", "test failure")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "branch non-existent not found")
}

func TestKillFailedBranch_Idempotent(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{})
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
	coord.mu.Unlock()

	// Try to kill it again
	err := coord.killFailedBranch(ctx, "branch-1", "new failure")
	require.NoError(t, err)

	// Verify the original kill data is preserved
	coord.mu.RLock()
	killedBranch := coord.activeBranches["branch-1"]
	coord.mu.RUnlock()

	assert.Equal(t, BranchStatusKilled, killedBranch.Status)
	assert.Equal(t, killedTime, *killedBranch.KilledAt, "KilledAt should not change")
	assert.Equal(t, "previous failure", killedBranch.KillReason, "KillReason should not change")
}

func TestKillFailedBranch_DifferentStatuses(t *testing.T) {
	tests := []struct {
		name          string
		initialStatus BranchStatus
	}{
		{"pending branch", BranchStatusPending},
		{"testing branch", BranchStatusTesting},
		{"passed branch", BranchStatusPassed},
		{"failed branch", BranchStatusFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coord := NewCoordinator(CoordinatorConfig{})
			ctx := context.Background()

			branch := &SpeculativeBranch{
				ID:     "branch-1",
				Status: tt.initialStatus,
			}

			coord.mu.Lock()
			coord.activeBranches = map[string]*SpeculativeBranch{
				"branch-1": branch,
			}
			coord.mu.Unlock()

			beforeKill := time.Now()
			err := coord.killFailedBranch(ctx, "branch-1", "cascade kill")
			afterKill := time.Now()

			require.NoError(t, err)

			coord.mu.RLock()
			killedBranch := coord.activeBranches["branch-1"]
			coord.mu.RUnlock()

			assert.Equal(t, BranchStatusKilled, killedBranch.Status)
			assert.NotNil(t, killedBranch.KilledAt)
			assert.True(t, killedBranch.KilledAt.After(beforeKill) || killedBranch.KilledAt.Equal(beforeKill))
			assert.True(t, killedBranch.KilledAt.Before(afterKill) || killedBranch.KilledAt.Equal(afterKill))
			assert.Equal(t, "cascade kill", killedBranch.KillReason)
		})
	}
}

func TestKillFailedBranch_PreservesOtherBranches(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{})
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
	err := coord.killFailedBranch(ctx, "branch-1", "test failure")
	require.NoError(t, err)

	// Verify branch-1 is killed but branch-2 is untouched
	coord.mu.RLock()
	defer coord.mu.RUnlock()

	assert.Equal(t, BranchStatusKilled, coord.activeBranches["branch-1"].Status)
	assert.Equal(t, BranchStatusTesting, coord.activeBranches["branch-2"].Status)
	assert.Nil(t, coord.activeBranches["branch-2"].KilledAt)
	assert.Empty(t, coord.activeBranches["branch-2"].KillReason)
}
