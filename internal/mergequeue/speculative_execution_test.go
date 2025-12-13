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

// TestGetBranchAncestry_CreatesCorrectChain verifies ancestry chain retrieval
func TestGetBranchAncestry_CreatesCorrectChain(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{})

	batch := []*ChangeRequest{
		{ID: "agent-1"},
		{ID: "agent-2"},
		{ID: "agent-3"},
	}

	ctx := context.Background()
	coord.createSpeculativeBranches(ctx, batch)
	time.Sleep(50 * time.Millisecond)

	// Test ancestry for deepest branch
	ancestry := coord.GetBranchAncestry("branch-agent-1-agent-2-agent-3")
	assert.Equal(t, 3, len(ancestry), "Should have 3 branches in ancestry chain")
	assert.Equal(t, "branch-agent-1", ancestry[0], "First should be root")
	assert.Equal(t, "branch-agent-1-agent-2", ancestry[1], "Second should be parent")
	assert.Equal(t, "branch-agent-1-agent-2-agent-3", ancestry[2], "Third should be self")

	// Test ancestry for middle branch
	ancestry = coord.GetBranchAncestry("branch-agent-1-agent-2")
	assert.Equal(t, 2, len(ancestry), "Should have 2 branches in ancestry chain")

	// Test ancestry for root branch
	ancestry = coord.GetBranchAncestry("branch-agent-1")
	assert.Equal(t, 1, len(ancestry), "Root should have ancestry of just itself")
	assert.Equal(t, "branch-agent-1", ancestry[0])
}

// TestGetBranchAncestry_NonExistent returns empty for non-existent branch
func TestGetBranchAncestry_NonExistent(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{})

	ancestry := coord.GetBranchAncestry("branch-nonexistent")
	assert.Equal(t, 0, len(ancestry), "Non-existent branch should return empty ancestry")
}

// TestGetBranchDescendants_ReturnsAllDescendants verifies descendant retrieval
func TestGetBranchDescendants_ReturnsAllDescendants(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{})

	batch := []*ChangeRequest{
		{ID: "agent-1"},
		{ID: "agent-2"},
		{ID: "agent-3"},
	}

	ctx := context.Background()
	coord.createSpeculativeBranches(ctx, batch)
	time.Sleep(50 * time.Millisecond)

	// Test descendants of root
	descendants := coord.GetBranchDescendants("branch-agent-1")
	assert.Equal(t, 2, len(descendants), "Root should have 2 descendants")
	assert.Contains(t, descendants, "branch-agent-1-agent-2")
	assert.Contains(t, descendants, "branch-agent-1-agent-2-agent-3")

	// Test descendants of middle branch
	descendants = coord.GetBranchDescendants("branch-agent-1-agent-2")
	assert.Equal(t, 1, len(descendants), "Middle should have 1 descendant")
	assert.Equal(t, "branch-agent-1-agent-2-agent-3", descendants[0])

	// Test descendants of leaf
	descendants = coord.GetBranchDescendants("branch-agent-1-agent-2-agent-3")
	assert.Equal(t, 0, len(descendants), "Leaf should have no descendants")
}

// TestGetBranchDescendants_NonExistent returns empty for non-existent branch
func TestGetBranchDescendants_NonExistent(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{})

	descendants := coord.GetBranchDescendants("branch-nonexistent")
	assert.Equal(t, 0, len(descendants), "Non-existent branch should return empty descendants")
}

// TestGetBranchSiblings_ReturnsCorrectSiblings verifies sibling retrieval
func TestGetBranchSiblings_ReturnsCorrectSiblings(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{})

	// Create two branches from same parent
	batch := []*ChangeRequest{
		{ID: "agent-1"},
		{ID: "agent-2"},
		{ID: "agent-3"},
	}

	ctx := context.Background()
	coord.createSpeculativeBranches(ctx, batch)
	time.Sleep(50 * time.Millisecond)

	// Root has no siblings
	siblings := coord.GetBranchSiblings("branch-agent-1")
	assert.Equal(t, 0, len(siblings), "Root should have no siblings")

	// Middle branch has no siblings in linear hierarchy
	siblings = coord.GetBranchSiblings("branch-agent-1-agent-2")
	assert.Equal(t, 0, len(siblings), "Linear hierarchy has no siblings")
}

// TestGetBranchSiblings_NonExistent returns empty for non-existent branch
func TestGetBranchSiblings_NonExistent(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{})

	siblings := coord.GetBranchSiblings("branch-nonexistent")
	assert.Equal(t, 0, len(siblings), "Non-existent branch should return empty siblings")
}

// TestIsAncestorOf_CorrectlyIdentifiesAncestry verifies ancestor relationship checking
func TestIsAncestorOf_CorrectlyIdentifiesAncestry(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{})

	batch := []*ChangeRequest{
		{ID: "agent-1"},
		{ID: "agent-2"},
		{ID: "agent-3"},
	}

	ctx := context.Background()
	coord.createSpeculativeBranches(ctx, batch)
	time.Sleep(50 * time.Millisecond)

	// Root is ancestor of middle and leaf
	assert.True(t, coord.IsAncestorOf("branch-agent-1", "branch-agent-1-agent-2"))
	assert.True(t, coord.IsAncestorOf("branch-agent-1", "branch-agent-1-agent-2-agent-3"))

	// Middle is ancestor of leaf but not root
	assert.True(t, coord.IsAncestorOf("branch-agent-1-agent-2", "branch-agent-1-agent-2-agent-3"))
	assert.False(t, coord.IsAncestorOf("branch-agent-1-agent-2-agent-3", "branch-agent-1"))

	// Leaf is not ancestor of anything
	assert.False(t, coord.IsAncestorOf("branch-agent-1-agent-2-agent-3", "branch-agent-1-agent-2"))

	// Non-existent branches
	assert.False(t, coord.IsAncestorOf("branch-nonexistent", "branch-agent-1"))
}

// TestCascadeStatusUpdate_UpdatesAllDescendants verifies cascade status updates
func TestCascadeStatusUpdate_UpdatesAllDescendants(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{})

	batch := []*ChangeRequest{
		{ID: "agent-1"},
		{ID: "agent-2"},
		{ID: "agent-3"},
	}

	ctx := context.Background()
	coord.createSpeculativeBranches(ctx, batch)
	time.Sleep(50 * time.Millisecond)

	// Cascade a status update from root
	err := coord.CascadeStatusUpdate("branch-agent-1", BranchStatusFailed)
	require.NoError(t, err)

	// All branches should be updated
	coord.mu.RLock()
	assert.Equal(t, BranchStatusFailed, coord.activeBranches["branch-agent-1"].Status)
	assert.Equal(t, BranchStatusFailed, coord.activeBranches["branch-agent-1-agent-2"].Status)
	assert.Equal(t, BranchStatusFailed, coord.activeBranches["branch-agent-1-agent-2-agent-3"].Status)
	coord.mu.RUnlock()
}

// TestCascadeStatusUpdate_NonExistentBranch returns error
func TestCascadeStatusUpdate_NonExistentBranch(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{})

	err := coord.CascadeStatusUpdate("branch-nonexistent", BranchStatusFailed)
	assert.Error(t, err, "Should return error for non-existent branch")
}

// TestCollectBranchFamily_ReturnsRootAndDescendants verifies family collection
func TestCollectBranchFamily_ReturnsRootAndDescendants(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{})

	batch := []*ChangeRequest{
		{ID: "agent-1"},
		{ID: "agent-2"},
		{ID: "agent-3"},
	}

	ctx := context.Background()
	coord.createSpeculativeBranches(ctx, batch)
	time.Sleep(50 * time.Millisecond)

	// Collect family starting from middle branch - should get all three
	family := coord.CollectBranchFamily("branch-agent-1-agent-2")
	assert.Equal(t, 3, len(family), "Family should include root and all descendants")
	assert.Contains(t, family, "branch-agent-1")
	assert.Contains(t, family, "branch-agent-1-agent-2")
	assert.Contains(t, family, "branch-agent-1-agent-2-agent-3")

	// Collect family starting from root
	family = coord.CollectBranchFamily("branch-agent-1")
	assert.Equal(t, 3, len(family), "Family from root should include all branches")

	// Collect family starting from leaf
	family = coord.CollectBranchFamily("branch-agent-1-agent-2-agent-3")
	assert.Equal(t, 3, len(family), "Family from leaf should include root and all ancestors + descendants")
}

// TestCollectBranchFamily_NonExistent returns empty for non-existent branch
func TestCollectBranchFamily_NonExistent(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{})

	family := coord.CollectBranchFamily("branch-nonexistent")
	assert.Equal(t, 0, len(family), "Non-existent branch should return empty family")
}

// TestGetBranchHierarchy_BuildsCorrectTree verifies hierarchy tree building
func TestGetBranchHierarchy_BuildsCorrectTree(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{})

	batch := []*ChangeRequest{
		{ID: "agent-1"},
		{ID: "agent-2"},
		{ID: "agent-3"},
	}

	ctx := context.Background()
	coord.createSpeculativeBranches(ctx, batch)
	time.Sleep(50 * time.Millisecond)

	// Get hierarchy from root
	root := coord.GetBranchHierarchy("branch-agent-1")
	require.NotNil(t, root)
	assert.Equal(t, "branch-agent-1", root.ID)
	assert.Equal(t, 1, root.Depth)
	assert.Equal(t, 1, len(root.Children), "Root should have 1 child")

	// Check child
	child := root.Children[0]
	assert.Equal(t, "branch-agent-1-agent-2", child.ID)
	assert.Equal(t, 2, child.Depth)
	assert.Equal(t, 1, len(child.Children), "Child should have 1 child")

	// Check grandchild
	grandchild := child.Children[0]
	assert.Equal(t, "branch-agent-1-agent-2-agent-3", grandchild.ID)
	assert.Equal(t, 3, grandchild.Depth)
	assert.Equal(t, 0, len(grandchild.Children), "Grandchild should have no children")
}

// TestGetBranchHierarchy_NonExistent returns nil for non-existent branch
func TestGetBranchHierarchy_NonExistent(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{})

	root := coord.GetBranchHierarchy("branch-nonexistent")
	assert.Nil(t, root, "Non-existent branch should return nil")
}

// TestBranchHierarchyIntegration verifies all hierarchy operations work together
func TestBranchHierarchyIntegration(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{})

	batch := []*ChangeRequest{
		{ID: "agent-1"},
		{ID: "agent-2"},
		{ID: "agent-3"},
		{ID: "agent-4"},
	}

	ctx := context.Background()
	coord.createSpeculativeBranches(ctx, batch)
	time.Sleep(50 * time.Millisecond)

	branchID := "branch-agent-1-agent-2-agent-3"

	// Verify ancestry
	ancestry := coord.GetBranchAncestry(branchID)
	assert.Equal(t, 3, len(ancestry))

	// Verify descendants from root
	descendants := coord.GetBranchDescendants("branch-agent-1")
	assert.Greater(t, len(descendants), 0)

	// Verify ancestor relationship
	assert.True(t, coord.IsAncestorOf("branch-agent-1", branchID))

	// Verify family
	family := coord.CollectBranchFamily(branchID)
	assert.Greater(t, len(family), 0)

	// Verify hierarchy tree
	tree := coord.GetBranchHierarchy("branch-agent-1")
	assert.NotNil(t, tree)
	assert.Equal(t, "branch-agent-1", tree.ID)
}
