package mergequeue

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestKillDependentBranches_ThreeLevelHierarchy tests recursive kill with 3-level deep hierarchy
func TestKillDependentBranches_ThreeLevelHierarchy(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{})
	ctx := context.Background()

	// Create 3-level hierarchy: root -> child -> grandchild
	root := &SpeculativeBranch{
		ID:          "root",
		Status:      BranchStatusTesting,
		ChildrenIDs: []string{"child"},
	}
	child := &SpeculativeBranch{
		ID:          "child",
		ParentID:    "root",
		Status:      BranchStatusTesting,
		ChildrenIDs: []string{"grandchild"},
	}
	grandchild := &SpeculativeBranch{
		ID:          "grandchild",
		ParentID:    "child",
		Status:      BranchStatusTesting,
		ChildrenIDs: []string{},
	}

	coord.mu.Lock()
	coord.activeBranches = map[string]*SpeculativeBranch{
		"root":       root,
		"child":      child,
		"grandchild": grandchild,
	}
	coord.mu.Unlock()

	// Kill dependents of root
	err := coord.killDependentBranches(ctx, "root")
	require.NoError(t, err)

	// Verify entire hierarchy was killed recursively
	coord.mu.RLock()
	defer coord.mu.RUnlock()

	assert.Equal(t, BranchStatusTesting, coord.activeBranches["root"].Status, "Root should not be killed")
	assert.Equal(t, BranchStatusKilled, coord.activeBranches["child"].Status, "Child should be killed")
	assert.Equal(t, BranchStatusKilled, coord.activeBranches["grandchild"].Status, "Grandchild should be killed")

	// Verify kill reasons reference the correct parent
	assert.Contains(t, coord.activeBranches["child"].KillReason, "parent branch root failed")
	assert.Contains(t, coord.activeBranches["grandchild"].KillReason, "parent branch child failed")
}

// TestKillDependentBranches_FourLevelHierarchy tests recursive kill with 4-level deep hierarchy
func TestKillDependentBranches_FourLevelHierarchy(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{})
	ctx := context.Background()

	// Create 4-level hierarchy: level0 -> level1 -> level2 -> level3
	level0 := &SpeculativeBranch{
		ID:          "level0",
		Status:      BranchStatusTesting,
		ChildrenIDs: []string{"level1"},
	}
	level1 := &SpeculativeBranch{
		ID:          "level1",
		ParentID:    "level0",
		Status:      BranchStatusTesting,
		ChildrenIDs: []string{"level2"},
	}
	level2 := &SpeculativeBranch{
		ID:          "level2",
		ParentID:    "level1",
		Status:      BranchStatusTesting,
		ChildrenIDs: []string{"level3"},
	}
	level3 := &SpeculativeBranch{
		ID:          "level3",
		ParentID:    "level2",
		Status:      BranchStatusTesting,
		ChildrenIDs: []string{},
	}

	coord.mu.Lock()
	coord.activeBranches = map[string]*SpeculativeBranch{
		"level0": level0,
		"level1": level1,
		"level2": level2,
		"level3": level3,
	}
	coord.mu.Unlock()

	// Kill dependents of level0
	err := coord.killDependentBranches(ctx, "level0")
	require.NoError(t, err)

	// Verify entire 4-level hierarchy was killed recursively
	coord.mu.RLock()
	defer coord.mu.RUnlock()

	assert.Equal(t, BranchStatusTesting, coord.activeBranches["level0"].Status, "Level0 should not be killed")
	assert.Equal(t, BranchStatusKilled, coord.activeBranches["level1"].Status, "Level1 should be killed")
	assert.Equal(t, BranchStatusKilled, coord.activeBranches["level2"].Status, "Level2 should be killed")
	assert.Equal(t, BranchStatusKilled, coord.activeBranches["level3"].Status, "Level3 should be killed")
}

// TestKillDependentBranches_FiveLevelHierarchy tests recursive kill with 5-level deep hierarchy
func TestKillDependentBranches_FiveLevelHierarchy(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{})
	ctx := context.Background()

	// Create 5-level deep hierarchy
	level0 := &SpeculativeBranch{
		ID:          "level0",
		Status:      BranchStatusTesting,
		ChildrenIDs: []string{"level1"},
	}
	level1 := &SpeculativeBranch{
		ID:          "level1",
		ParentID:    "level0",
		Status:      BranchStatusTesting,
		ChildrenIDs: []string{"level2"},
	}
	level2 := &SpeculativeBranch{
		ID:          "level2",
		ParentID:    "level1",
		Status:      BranchStatusTesting,
		ChildrenIDs: []string{"level3"},
	}
	level3 := &SpeculativeBranch{
		ID:          "level3",
		ParentID:    "level2",
		Status:      BranchStatusTesting,
		ChildrenIDs: []string{"level4"},
	}
	level4 := &SpeculativeBranch{
		ID:          "level4",
		ParentID:    "level3",
		Status:      BranchStatusTesting,
		ChildrenIDs: []string{},
	}

	coord.mu.Lock()
	coord.activeBranches = map[string]*SpeculativeBranch{
		"level0": level0,
		"level1": level1,
		"level2": level2,
		"level3": level3,
		"level4": level4,
	}
	coord.mu.Unlock()

	// Kill dependents of level0
	err := coord.killDependentBranches(ctx, "level0")
	require.NoError(t, err)

	// Verify entire 5-level hierarchy was killed recursively
	coord.mu.RLock()
	defer coord.mu.RUnlock()

	assert.Equal(t, BranchStatusTesting, coord.activeBranches["level0"].Status, "Level0 should not be killed")
	assert.Equal(t, BranchStatusKilled, coord.activeBranches["level1"].Status, "Level1 should be killed")
	assert.Equal(t, BranchStatusKilled, coord.activeBranches["level2"].Status, "Level2 should be killed")
	assert.Equal(t, BranchStatusKilled, coord.activeBranches["level3"].Status, "Level3 should be killed")
	assert.Equal(t, BranchStatusKilled, coord.activeBranches["level4"].Status, "Level4 should be killed")

	// Verify all have kill timestamps
	assert.NotNil(t, coord.activeBranches["level1"].KilledAt)
	assert.NotNil(t, coord.activeBranches["level2"].KilledAt)
	assert.NotNil(t, coord.activeBranches["level3"].KilledAt)
	assert.NotNil(t, coord.activeBranches["level4"].KilledAt)
}

// TestKillDependentBranches_ComplexTreeStructure tests recursive kill with branching hierarchy
func TestKillDependentBranches_ComplexTreeStructure(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{})
	ctx := context.Background()

	// Create complex tree: root has 2 children, each child has 2 children (7 nodes total)
	//        root
	//       /    \
	//    child1  child2
	//     / \      / \
	//   gc1 gc2  gc3 gc4
	root := &SpeculativeBranch{
		ID:          "root",
		Status:      BranchStatusTesting,
		ChildrenIDs: []string{"child1", "child2"},
	}
	child1 := &SpeculativeBranch{
		ID:          "child1",
		ParentID:    "root",
		Status:      BranchStatusTesting,
		ChildrenIDs: []string{"gc1", "gc2"},
	}
	child2 := &SpeculativeBranch{
		ID:          "child2",
		ParentID:    "root",
		Status:      BranchStatusTesting,
		ChildrenIDs: []string{"gc3", "gc4"},
	}
	gc1 := &SpeculativeBranch{
		ID:          "gc1",
		ParentID:    "child1",
		Status:      BranchStatusTesting,
		ChildrenIDs: []string{},
	}
	gc2 := &SpeculativeBranch{
		ID:          "gc2",
		ParentID:    "child1",
		Status:      BranchStatusTesting,
		ChildrenIDs: []string{},
	}
	gc3 := &SpeculativeBranch{
		ID:          "gc3",
		ParentID:    "child2",
		Status:      BranchStatusTesting,
		ChildrenIDs: []string{},
	}
	gc4 := &SpeculativeBranch{
		ID:          "gc4",
		ParentID:    "child2",
		Status:      BranchStatusTesting,
		ChildrenIDs: []string{},
	}

	coord.mu.Lock()
	coord.activeBranches = map[string]*SpeculativeBranch{
		"root":   root,
		"child1": child1,
		"child2": child2,
		"gc1":    gc1,
		"gc2":    gc2,
		"gc3":    gc3,
		"gc4":    gc4,
	}
	coord.mu.Unlock()

	// Kill dependents of root
	err := coord.killDependentBranches(ctx, "root")
	require.NoError(t, err)

	// Verify entire tree was killed
	coord.mu.RLock()
	defer coord.mu.RUnlock()

	assert.Equal(t, BranchStatusTesting, coord.activeBranches["root"].Status, "Root should not be killed")
	assert.Equal(t, BranchStatusKilled, coord.activeBranches["child1"].Status)
	assert.Equal(t, BranchStatusKilled, coord.activeBranches["child2"].Status)
	assert.Equal(t, BranchStatusKilled, coord.activeBranches["gc1"].Status)
	assert.Equal(t, BranchStatusKilled, coord.activeBranches["gc2"].Status)
	assert.Equal(t, BranchStatusKilled, coord.activeBranches["gc3"].Status)
	assert.Equal(t, BranchStatusKilled, coord.activeBranches["gc4"].Status)
}

// TestKillDependentBranches_PartialTreeKill tests killing a middle branch in hierarchy
func TestKillDependentBranches_PartialTreeKill(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{})
	ctx := context.Background()

	// Create tree where we kill a middle branch
	//        root
	//       /    \
	//    child1  child2
	//     / \
	//   gc1 gc2
	root := &SpeculativeBranch{
		ID:          "root",
		Status:      BranchStatusTesting,
		ChildrenIDs: []string{"child1", "child2"},
	}
	child1 := &SpeculativeBranch{
		ID:          "child1",
		ParentID:    "root",
		Status:      BranchStatusTesting,
		ChildrenIDs: []string{"gc1", "gc2"},
	}
	child2 := &SpeculativeBranch{
		ID:          "child2",
		ParentID:    "root",
		Status:      BranchStatusTesting,
		ChildrenIDs: []string{},
	}
	gc1 := &SpeculativeBranch{
		ID:          "gc1",
		ParentID:    "child1",
		Status:      BranchStatusTesting,
		ChildrenIDs: []string{},
	}
	gc2 := &SpeculativeBranch{
		ID:          "gc2",
		ParentID:    "child1",
		Status:      BranchStatusTesting,
		ChildrenIDs: []string{},
	}

	coord.mu.Lock()
	coord.activeBranches = map[string]*SpeculativeBranch{
		"root":   root,
		"child1": child1,
		"child2": child2,
		"gc1":    gc1,
		"gc2":    gc2,
	}
	coord.mu.Unlock()

	// Kill dependents of child1 (not root)
	err := coord.killDependentBranches(ctx, "child1")
	require.NoError(t, err)

	// Verify only child1's descendants were killed
	coord.mu.RLock()
	defer coord.mu.RUnlock()

	assert.Equal(t, BranchStatusTesting, coord.activeBranches["root"].Status, "Root should not be killed")
	assert.Equal(t, BranchStatusTesting, coord.activeBranches["child1"].Status, "Child1 should not be killed")
	assert.Equal(t, BranchStatusTesting, coord.activeBranches["child2"].Status, "Child2 should not be killed")
	assert.Equal(t, BranchStatusKilled, coord.activeBranches["gc1"].Status, "gc1 should be killed")
	assert.Equal(t, BranchStatusKilled, coord.activeBranches["gc2"].Status, "gc2 should be killed")
}

// TestKillDependentBranches_DeepAsymmetricTree tests killing with asymmetric depth
func TestKillDependentBranches_DeepAsymmetricTree(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{})
	ctx := context.Background()

	// Create asymmetric tree: one branch goes deep, other is shallow
	//        root
	//       /    \
	//    deep1  shallow
	//      |
	//    deep2
	//      |
	//    deep3
	//      |
	//    deep4
	root := &SpeculativeBranch{
		ID:          "root",
		Status:      BranchStatusTesting,
		ChildrenIDs: []string{"deep1", "shallow"},
	}
	deep1 := &SpeculativeBranch{
		ID:          "deep1",
		ParentID:    "root",
		Status:      BranchStatusTesting,
		ChildrenIDs: []string{"deep2"},
	}
	shallow := &SpeculativeBranch{
		ID:          "shallow",
		ParentID:    "root",
		Status:      BranchStatusTesting,
		ChildrenIDs: []string{},
	}
	deep2 := &SpeculativeBranch{
		ID:          "deep2",
		ParentID:    "deep1",
		Status:      BranchStatusTesting,
		ChildrenIDs: []string{"deep3"},
	}
	deep3 := &SpeculativeBranch{
		ID:          "deep3",
		ParentID:    "deep2",
		Status:      BranchStatusTesting,
		ChildrenIDs: []string{"deep4"},
	}
	deep4 := &SpeculativeBranch{
		ID:          "deep4",
		ParentID:    "deep3",
		Status:      BranchStatusTesting,
		ChildrenIDs: []string{},
	}

	coord.mu.Lock()
	coord.activeBranches = map[string]*SpeculativeBranch{
		"root":    root,
		"deep1":   deep1,
		"shallow": shallow,
		"deep2":   deep2,
		"deep3":   deep3,
		"deep4":   deep4,
	}
	coord.mu.Unlock()

	// Kill dependents of root
	err := coord.killDependentBranches(ctx, "root")
	require.NoError(t, err)

	// Verify entire asymmetric tree was killed
	coord.mu.RLock()
	defer coord.mu.RUnlock()

	assert.Equal(t, BranchStatusTesting, coord.activeBranches["root"].Status)
	assert.Equal(t, BranchStatusKilled, coord.activeBranches["deep1"].Status)
	assert.Equal(t, BranchStatusKilled, coord.activeBranches["deep2"].Status)
	assert.Equal(t, BranchStatusKilled, coord.activeBranches["deep3"].Status)
	assert.Equal(t, BranchStatusKilled, coord.activeBranches["deep4"].Status)
	assert.Equal(t, BranchStatusKilled, coord.activeBranches["shallow"].Status)
}

// TestKillDependentBranches_MixedStatuses tests killing with different branch statuses
func TestKillDependentBranches_MixedStatuses(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{})
	ctx := context.Background()

	// Create hierarchy with different statuses
	parent := &SpeculativeBranch{
		ID:          "parent",
		Status:      BranchStatusTesting,
		ChildrenIDs: []string{"pending", "testing", "passed", "failed"},
	}
	pending := &SpeculativeBranch{
		ID:       "pending",
		ParentID: "parent",
		Status:   BranchStatusPending,
	}
	testing := &SpeculativeBranch{
		ID:       "testing",
		ParentID: "parent",
		Status:   BranchStatusTesting,
	}
	passed := &SpeculativeBranch{
		ID:       "passed",
		ParentID: "parent",
		Status:   BranchStatusPassed,
	}
	failed := &SpeculativeBranch{
		ID:       "failed",
		ParentID: "parent",
		Status:   BranchStatusFailed,
	}

	coord.mu.Lock()
	coord.activeBranches = map[string]*SpeculativeBranch{
		"parent":  parent,
		"pending": pending,
		"testing": testing,
		"passed":  passed,
		"failed":  failed,
	}
	coord.mu.Unlock()

	// Kill dependents of parent
	err := coord.killDependentBranches(ctx, "parent")
	require.NoError(t, err)

	// Verify all children are killed regardless of initial status
	coord.mu.RLock()
	defer coord.mu.RUnlock()

	assert.Equal(t, BranchStatusTesting, coord.activeBranches["parent"].Status)
	assert.Equal(t, BranchStatusKilled, coord.activeBranches["pending"].Status)
	assert.Equal(t, BranchStatusKilled, coord.activeBranches["testing"].Status)
	assert.Equal(t, BranchStatusKilled, coord.activeBranches["passed"].Status)
	assert.Equal(t, BranchStatusKilled, coord.activeBranches["failed"].Status)
}

// TestKillDependentBranches_SevenLevelHierarchy tests extreme depth recursion
func TestKillDependentBranches_SevenLevelHierarchy(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{})
	ctx := context.Background()

	// Create 7-level deep hierarchy to test extreme recursion
	branches := make(map[string]*SpeculativeBranch)
	for i := 0; i < 7; i++ {
		id := fmt.Sprintf("level%d", i)
		branch := &SpeculativeBranch{
			ID:          id,
			Status:      BranchStatusTesting,
			ChildrenIDs: []string{},
		}
		if i > 0 {
			branch.ParentID = fmt.Sprintf("level%d", i-1)
			branches[fmt.Sprintf("level%d", i-1)].ChildrenIDs = []string{id}
		}
		branches[id] = branch
	}

	coord.mu.Lock()
	coord.activeBranches = branches
	coord.mu.Unlock()

	// Kill dependents of level0
	err := coord.killDependentBranches(ctx, "level0")
	require.NoError(t, err)

	// Verify entire 7-level hierarchy was killed recursively
	coord.mu.RLock()
	defer coord.mu.RUnlock()

	assert.Equal(t, BranchStatusTesting, coord.activeBranches["level0"].Status, "Level0 should not be killed")
	for i := 1; i < 7; i++ {
		id := fmt.Sprintf("level%d", i)
		assert.Equal(t, BranchStatusKilled, coord.activeBranches[id].Status, "%s should be killed", id)
		assert.NotNil(t, coord.activeBranches[id].KilledAt)
	}
}
