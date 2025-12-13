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
	defer coord.Stop()

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
	// Should be in main queue (first change, no bypass)
	assert.Len(t, coord.mainQueue, 1)
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
