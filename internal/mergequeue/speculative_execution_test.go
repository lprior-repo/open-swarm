package mergequeue

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGenerateBranchID_SingleChange verifies branch ID generation for a single change
func TestGenerateBranchID_SingleChange(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{})

	change := &ChangeRequest{
		ID: "agent-1",
	}

	branchID := coord.generateBranchID([]*ChangeRequest{change})
	assert.Equal(t, "branch-agent-1", branchID)
}

// TestGenerateBranchID_MultipleChanges verifies branch ID generation for multiple changes
func TestGenerateBranchID_MultipleChanges(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{})

	changes := []*ChangeRequest{
		{ID: "agent-1"},
		{ID: "agent-2"},
		{ID: "agent-3"},
	}

	branchID := coord.generateBranchID(changes)
	assert.Equal(t, "branch-agent-1-agent-2-agent-3", branchID)
}

// TestGenerateBranchID_EmptyChanges verifies that empty change list returns empty ID
func TestGenerateBranchID_EmptyChanges(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{})

	branchID := coord.generateBranchID([]*ChangeRequest{})
	assert.Equal(t, "", branchID)
}

// TestGenerateBranchID_Deterministic verifies that branch IDs are deterministic
func TestGenerateBranchID_Deterministic(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{})

	changes := []*ChangeRequest{
		{ID: "agent-1"},
		{ID: "agent-2"},
	}

	// Generate ID twice - should be identical
	branchID1 := coord.generateBranchID(changes)
	branchID2 := coord.generateBranchID(changes)
	assert.Equal(t, branchID1, branchID2, "Branch IDs should be deterministic")
}

// TestExecuteSpeculativeBranch_UpdatesStatus verifies branch status updates during execution
func TestExecuteSpeculativeBranch_UpdatesStatus(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{
		TestTimeout: 100 * time.Millisecond,
	})
	ctx := context.Background()

	// Start coordinator to handle results
	err := coord.Start(ctx)
	require.NoError(t, err)
	defer func() {
		_ = coord.Stop()
	}()

	branch := &SpeculativeBranch{
		ID:     "branch-test",
		Status: BranchStatusPending,
		Changes: []ChangeRequest{
			{ID: "agent-1"},
		},
	}

	coord.mu.Lock()
	coord.activeBranches["branch-test"] = branch
	coord.mu.Unlock()

	// Execute branch
	go coord.executeSpeculativeBranch(ctx, branch)

	// Wait for status to update
	time.Sleep(50 * time.Millisecond)

	coord.mu.RLock()
	status := branch.Status
	coord.mu.RUnlock()

	// Status should have changed from pending
	assert.NotEqual(t, BranchStatusPending, status, "Branch status should be updated during execution")
}

// TestExecuteSpeculativeBranch_SendsResult verifies result is sent to results channel
func TestExecuteSpeculativeBranch_SendsResult(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{
		TestTimeout: 200 * time.Millisecond,
	})
	ctx := context.Background()

	branch := &SpeculativeBranch{
		ID:     "branch-test",
		Status: BranchStatusPending,
		Changes: []ChangeRequest{
			{ID: "agent-1"},
		},
	}

	coord.mu.Lock()
	coord.activeBranches["branch-test"] = branch
	coord.mu.Unlock()

	// Execute branch
	go coord.executeSpeculativeBranch(ctx, branch)

	// Wait for result
	select {
	case result := <-coord.resultsChan:
		require.NotNil(t, result)
		assert.Equal(t, 1, len(result.ChangeIDs))
		assert.Equal(t, "agent-1", result.ChangeIDs[0])
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Expected result to be sent to results channel")
	}
}

// TestExecuteSpeculativeBranch_SetsTestResult verifies test result is set on branch
func TestExecuteSpeculativeBranch_SetsTestResult(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{
		TestTimeout: 200 * time.Millisecond,
	})
	ctx := context.Background()

	branch := &SpeculativeBranch{
		ID:     "branch-test",
		Status: BranchStatusPending,
		Changes: []ChangeRequest{
			{ID: "agent-1"},
		},
	}

	coord.mu.Lock()
	coord.activeBranches["branch-test"] = branch
	coord.mu.Unlock()

	// Execute branch
	go coord.executeSpeculativeBranch(ctx, branch)

	// Wait for execution to complete
	time.Sleep(150 * time.Millisecond)

	coord.mu.RLock()
	testResult := branch.TestResult
	coord.mu.RUnlock()

	assert.NotNil(t, testResult, "Branch should have test result after execution")
	assert.Equal(t, 1, len(testResult.ChangeIDs))
}

// TestCreateSpeculativeBranches_CreatesCorrectHierarchy verifies branch hierarchy is created correctly
func TestCreateSpeculativeBranches_CreatesCorrectHierarchy(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{})

	batch := []*ChangeRequest{
		{ID: "agent-1"},
		{ID: "agent-2"},
		{ID: "agent-3"},
	}

	ctx := context.Background()
	coord.createSpeculativeBranches(ctx, batch)

	// Give goroutines time to start
	time.Sleep(50 * time.Millisecond)

	coord.mu.RLock()
	defer coord.mu.RUnlock()

	// Should create 3 branches: depth 1, 2, and 3
	assert.Equal(t, 3, len(coord.activeBranches), "Should create one branch per depth level")

	// Verify depth 1 branch
	branch1 := coord.activeBranches["branch-agent-1"]
	require.NotNil(t, branch1)
	assert.Equal(t, 1, branch1.Depth)
	assert.Equal(t, 1, len(branch1.Changes))
	assert.Equal(t, "", branch1.ParentID, "Depth 1 should have no parent")

	// Verify depth 2 branch
	branch2 := coord.activeBranches["branch-agent-1-agent-2"]
	require.NotNil(t, branch2)
	assert.Equal(t, 2, branch2.Depth)
	assert.Equal(t, 2, len(branch2.Changes))
	assert.Equal(t, "branch-agent-1", branch2.ParentID)

	// Verify depth 3 branch
	branch3 := coord.activeBranches["branch-agent-1-agent-2-agent-3"]
	require.NotNil(t, branch3)
	assert.Equal(t, 3, branch3.Depth)
	assert.Equal(t, 3, len(branch3.Changes))
	assert.Equal(t, "branch-agent-1-agent-2", branch3.ParentID)

	// Verify parent-child relationships
	assert.Contains(t, branch1.ChildrenIDs, "branch-agent-1-agent-2")
	assert.Contains(t, branch2.ChildrenIDs, "branch-agent-1-agent-2-agent-3")
}

// TestCreateSpeculativeBranches_AvoidsDuplicates verifies no duplicate branches are created
func TestCreateSpeculativeBranches_AvoidsDuplicates(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{})

	batch := []*ChangeRequest{
		{ID: "agent-1"},
		{ID: "agent-2"},
	}

	ctx := context.Background()

	// Create branches twice
	coord.createSpeculativeBranches(ctx, batch)
	time.Sleep(50 * time.Millisecond)

	coord.mu.RLock()
	initialCount := len(coord.activeBranches)
	coord.mu.RUnlock()

	coord.createSpeculativeBranches(ctx, batch)
	time.Sleep(50 * time.Millisecond)

	coord.mu.RLock()
	finalCount := len(coord.activeBranches)
	coord.mu.RUnlock()

	// Count should not increase on second call
	assert.Equal(t, initialCount, finalCount, "Should not create duplicate branches")
}

// TestProcessBypass_CreatesAndExecutesBranch verifies bypass lane processing
func TestProcessBypass_CreatesAndExecutesBranch(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{
		TestTimeout: 200 * time.Millisecond,
	})
	ctx := context.Background()

	change := &ChangeRequest{
		ID:            "agent-bypass",
		FilesModified: []string{"src/independent.ts"},
	}

	coord.mu.Lock()
	coord.bypassLane = []*ChangeRequest{change}
	coord.mu.Unlock()

	// Process bypass
	go coord.processBypass(ctx, change)

	// Wait for execution
	time.Sleep(150 * time.Millisecond)

	coord.mu.RLock()
	defer coord.mu.RUnlock()

	// Bypass lane should be cleaned up
	assert.Equal(t, 0, len(coord.bypassLane), "Bypass lane should be cleaned after processing")

	// Branch should exist
	branchID := "branch-agent-bypass"
	branch, exists := coord.activeBranches[branchID]
	assert.True(t, exists, "Branch should be created for bypass processing")
	if exists {
		assert.Equal(t, 1, branch.Depth)
		assert.Equal(t, 1, len(branch.Changes))
	}
}

// TestRunBranchTests_Stub verifies the stub implementation returns true
func TestRunBranchTests_Stub(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{})
	ctx := context.Background()

	branch := &SpeculativeBranch{
		ID: "branch-test",
		Changes: []ChangeRequest{
			{ID: "agent-1"},
		},
	}

	// Stub should return a passing test result
	result := coord.runBranchTests(ctx, branch)
	require.NotNil(t, result, "runBranchTests should return a test result")
	assert.True(t, result.Passed, "Stub implementation should return passing result")
}

// TestExecuteSpeculativeBranch_ContextCancellation verifies graceful handling of context cancellation
func TestExecuteSpeculativeBranch_ContextCancellation(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{
		TestTimeout: 5 * time.Second,
	})
	ctx, cancel := context.WithCancel(context.Background())

	branch := &SpeculativeBranch{
		ID:     "branch-test",
		Status: BranchStatusPending,
		Changes: []ChangeRequest{
			{ID: "agent-1"},
		},
	}

	coord.mu.Lock()
	coord.activeBranches["branch-test"] = branch
	coord.mu.Unlock()

	// Execute branch
	go coord.executeSpeculativeBranch(ctx, branch)

	// Cancel context immediately
	time.Sleep(10 * time.Millisecond)
	cancel()

	// Wait to ensure goroutine exits gracefully
	time.Sleep(50 * time.Millisecond)

	// Branch should still have its status updated
	coord.mu.RLock()
	status := branch.Status
	coord.mu.RUnlock()

	assert.NotEqual(t, BranchStatusPending, status, "Branch should be marked as testing even if cancelled")
}

// TestExecuteSpeculativeBranch_CoordinatorShutdown verifies graceful handling of coordinator shutdown
func TestExecuteSpeculativeBranch_CoordinatorShutdown(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{
		TestTimeout: 5 * time.Second,
	})
	ctx := context.Background()

	branch := &SpeculativeBranch{
		ID:     "branch-test",
		Status: BranchStatusPending,
		Changes: []ChangeRequest{
			{ID: "agent-1"},
		},
	}

	coord.mu.Lock()
	coord.activeBranches["branch-test"] = branch
	coord.mu.Unlock()

	// Execute branch
	go coord.executeSpeculativeBranch(ctx, branch)

	// Shutdown coordinator immediately
	time.Sleep(10 * time.Millisecond)
	err := coord.Stop()
	require.NoError(t, err)

	// Wait to ensure goroutine exits gracefully
	time.Sleep(50 * time.Millisecond)

	// Test should not hang or panic
}
