package mergequeue

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// BenchmarkKillFailedBranchWithTimeout_SingleBranch measures time to kill a single branch
func BenchmarkKillFailedBranchWithTimeout_SingleBranch(b *testing.B) {
	coord := NewCoordinator(CoordinatorConfig{
		KillSwitchTimeout: 500 * time.Millisecond,
	})
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Setup: Create a test branch
		branchID := fmt.Sprintf("bench-branch-%d", i)
		branch := &SpeculativeBranch{
			ID:     branchID,
			Status: BranchStatusTesting,
			Changes: []ChangeRequest{
				{
					ID:            "agent-1",
					FilesModified: []string{"src/test.go"},
				},
			},
		}

		coord.mu.Lock()
		coord.activeBranches[branchID] = branch
		coord.mu.Unlock()
		b.StartTimer()

		// Benchmark: Kill the branch
		_ = coord.killFailedBranchWithTimeout(ctx, branchID, "test failure")
	}
}

// BenchmarkKillFailedBranchWithTimeout_IdempotentKill measures overhead of idempotent kills
func BenchmarkKillFailedBranchWithTimeout_IdempotentKill(b *testing.B) {
	coord := NewCoordinator(CoordinatorConfig{
		KillSwitchTimeout: 500 * time.Millisecond,
	})
	ctx := context.Background()

	// Setup: Create and kill a branch once
	killedTime := time.Now()
	branch := &SpeculativeBranch{
		ID:         "bench-branch",
		Status:     BranchStatusKilled,
		KilledAt:   &killedTime,
		KillReason: "already killed",
	}

	coord.mu.Lock()
	coord.activeBranches["bench-branch"] = branch
	coord.mu.Unlock()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Benchmark: Kill already-killed branch (idempotent operation)
		_ = coord.killFailedBranchWithTimeout(ctx, "bench-branch", "another failure")
	}
}

// BenchmarkKillFailedBranchWithTimeout_KillWithTimeout measures timeout overhead
func BenchmarkKillFailedBranchWithTimeout_KillWithTimeout(b *testing.B) {
	// Create coordinator with short timeout to measure overhead
	coord := NewCoordinator(CoordinatorConfig{
		KillSwitchTimeout: 1 * time.Nanosecond, // Very short to consistently timeout
	})
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Setup: Create a test branch
		branchID := fmt.Sprintf("bench-branch-%d", i)
		branch := &SpeculativeBranch{
			ID:     branchID,
			Status: BranchStatusTesting,
		}

		coord.mu.Lock()
		coord.activeBranches[branchID] = branch
		coord.mu.Unlock()
		b.StartTimer()

		// Benchmark: Kill branch with guaranteed timeout
		_ = coord.killFailedBranchWithTimeout(ctx, branchID, "test failure")
	}
}

// BenchmarkKillFailedBranchWithTimeout_MutexContention measures lock overhead
func BenchmarkKillFailedBranchWithTimeout_MutexContention(b *testing.B) {
	coord := NewCoordinator(CoordinatorConfig{
		KillSwitchTimeout: 500 * time.Millisecond,
	})
	ctx := context.Background()

	// Pre-populate with many branches to increase contention
	b.StopTimer()
	for i := 0; i < 1000; i++ {
		branchID := fmt.Sprintf("existing-branch-%d", i)
		branch := &SpeculativeBranch{
			ID:     branchID,
			Status: BranchStatusTesting,
		}
		coord.mu.Lock()
		coord.activeBranches[branchID] = branch
		coord.mu.Unlock()
	}
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		branchID := fmt.Sprintf("bench-branch-%d", i)
		branch := &SpeculativeBranch{
			ID:     branchID,
			Status: BranchStatusTesting,
		}
		coord.mu.Lock()
		coord.activeBranches[branchID] = branch
		coord.mu.Unlock()
		b.StartTimer()

		_ = coord.killFailedBranchWithTimeout(ctx, branchID, "test failure")
	}
}

// BenchmarkKillMultipleBranchesSequential measures time to kill multiple branches sequentially
func BenchmarkKillMultipleBranchesSequential(b *testing.B) {
	coord := NewCoordinator(CoordinatorConfig{
		KillSwitchTimeout: 500 * time.Millisecond,
	})
	ctx := context.Background()
	branchCount := 10

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Setup: Create multiple branches
		for j := 0; j < branchCount; j++ {
			branchID := fmt.Sprintf("bench-branch-%d-%d", i, j)
			branch := &SpeculativeBranch{
				ID:     branchID,
				Status: BranchStatusTesting,
			}
			coord.mu.Lock()
			coord.activeBranches[branchID] = branch
			coord.mu.Unlock()
		}
		b.StartTimer()

		// Benchmark: Kill all branches sequentially
		for j := 0; j < branchCount; j++ {
			branchID := fmt.Sprintf("bench-branch-%d-%d", i, j)
			_ = coord.killFailedBranchWithTimeout(ctx, branchID, "test failure")
		}
	}
}

// BenchmarkKillMultipleBranchesParallel measures time to kill multiple branches in parallel
func BenchmarkKillMultipleBranchesParallel(b *testing.B) {
	ctx := context.Background()
	branchCount := 10

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Setup: Create new coordinator and branches for each iteration
		coord := NewCoordinator(CoordinatorConfig{
			KillSwitchTimeout: 500 * time.Millisecond,
		})

		for j := 0; j < branchCount; j++ {
			branchID := fmt.Sprintf("bench-branch-%d-%d", i, j)
			branch := &SpeculativeBranch{
				ID:     branchID,
				Status: BranchStatusTesting,
			}
			coord.mu.Lock()
			coord.activeBranches[branchID] = branch
			coord.mu.Unlock()
		}
		b.StartTimer()

		// Benchmark: Kill all branches in parallel
		var wg sync.WaitGroup
		for j := 0; j < branchCount; j++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				branchID := fmt.Sprintf("bench-branch-%d-%d", i, idx)
				_ = coord.killFailedBranchWithTimeout(ctx, branchID, "test failure")
			}(j)
		}
		wg.Wait()
		b.StopTimer()
	}
}

// BenchmarkKillDependentBranchesWithTimeout_ShallowHierarchy measures cascade kill with shallow tree
func BenchmarkKillDependentBranchesWithTimeout_ShallowHierarchy(b *testing.B) {
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Setup: Create coordinator and shallow hierarchy
		coord := NewCoordinator(CoordinatorConfig{
			KillSwitchTimeout: 500 * time.Millisecond,
		})

		// Create hierarchy: parent -> 3 children
		parent := &SpeculativeBranch{
			ID:          "parent",
			Status:      BranchStatusFailed,
			ChildrenIDs: []string{"child-0", "child-1", "child-2"},
		}
		coord.mu.Lock()
		coord.activeBranches["parent"] = parent
		for j := 0; j < 3; j++ {
			branchID := fmt.Sprintf("child-%d", j)
			child := &SpeculativeBranch{
				ID:       branchID,
				Status:   BranchStatusTesting,
				ParentID: "parent",
			}
			coord.activeBranches[branchID] = child
		}
		coord.mu.Unlock()
		b.StartTimer()

		// Benchmark: Kill the entire hierarchy
		_ = coord.killDependentBranchesWithTimeout(ctx, "parent")
	}
}

// BenchmarkKillDependentBranchesWithTimeout_DeepHierarchy measures cascade kill with deep tree
func BenchmarkKillDependentBranchesWithTimeout_DeepHierarchy(b *testing.B) {
	ctx := context.Background()
	depth := 10

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Setup: Create coordinator and deep hierarchy
		coord := NewCoordinator(CoordinatorConfig{
			KillSwitchTimeout: 500 * time.Millisecond,
		})

		// Create linear deep hierarchy: parent -> child -> grandchild -> ... (10 levels)
		parentID := "root"
		parent := &SpeculativeBranch{
			ID:          parentID,
			Status:      BranchStatusFailed,
			ChildrenIDs: []string{"level-1"},
		}
		coord.mu.Lock()
		coord.activeBranches[parentID] = parent

		for level := 1; level < depth; level++ {
			nodeID := fmt.Sprintf("level-%d", level)
			childNodeID := fmt.Sprintf("level-%d", level+1)
			node := &SpeculativeBranch{
				ID:          nodeID,
				Status:      BranchStatusTesting,
				ParentID:    fmt.Sprintf("level-%d", level-1),
				ChildrenIDs: []string{childNodeID},
			}
			coord.activeBranches[nodeID] = node
		}

		// Add leaf node
		leafID := fmt.Sprintf("level-%d", depth)
		leaf := &SpeculativeBranch{
			ID:       leafID,
			Status:   BranchStatusTesting,
			ParentID: fmt.Sprintf("level-%d", depth-1),
		}
		coord.activeBranches[leafID] = leaf
		coord.mu.Unlock()
		b.StartTimer()

		// Benchmark: Kill entire deep hierarchy
		_ = coord.killDependentBranchesWithTimeout(ctx, "root")
	}
}

// BenchmarkKillDependentBranchesWithTimeout_WideHierarchy measures cascade kill with wide tree
func BenchmarkKillDependentBranchesWithTimeout_WideHierarchy(b *testing.B) {
	ctx := context.Background()
	childCount := 50

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Setup: Create coordinator and wide hierarchy
		coord := NewCoordinator(CoordinatorConfig{
			KillSwitchTimeout: 500 * time.Millisecond,
		})

		// Create hierarchy with one parent and many children
		childIDs := make([]string, childCount)
		for j := 0; j < childCount; j++ {
			childIDs[j] = fmt.Sprintf("child-%d", j)
		}

		parent := &SpeculativeBranch{
			ID:          "parent",
			Status:      BranchStatusFailed,
			ChildrenIDs: childIDs,
		}

		coord.mu.Lock()
		coord.activeBranches["parent"] = parent
		for j := 0; j < childCount; j++ {
			child := &SpeculativeBranch{
				ID:       fmt.Sprintf("child-%d", j),
				Status:   BranchStatusTesting,
				ParentID: "parent",
			}
			coord.activeBranches[fmt.Sprintf("child-%d", j)] = child
		}
		coord.mu.Unlock()
		b.StartTimer()

		// Benchmark: Kill all children
		_ = coord.killDependentBranchesWithTimeout(ctx, "parent")
	}
}

// BenchmarkKillDependentBranchesWithTimeout_ComplexHierarchy measures cascade kill with complex tree
func BenchmarkKillDependentBranchesWithTimeout_ComplexHierarchy(b *testing.B) {
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Setup: Create coordinator and complex hierarchy
		//        root
		//       /    \
		//     l1-0   l1-1
		//     / \     / \
		//  l2-0 l2-1 l2-2 l2-3
		//   |
		//  l3-0
		coord := NewCoordinator(CoordinatorConfig{
			KillSwitchTimeout: 500 * time.Millisecond,
		})

		root := &SpeculativeBranch{
			ID:          "root",
			Status:      BranchStatusFailed,
			ChildrenIDs: []string{"l1-0", "l1-1"},
		}

		l1_0 := &SpeculativeBranch{
			ID:          "l1-0",
			Status:      BranchStatusTesting,
			ParentID:    "root",
			ChildrenIDs: []string{"l2-0", "l2-1"},
		}
		l1_1 := &SpeculativeBranch{
			ID:          "l1-1",
			Status:      BranchStatusTesting,
			ParentID:    "root",
			ChildrenIDs: []string{"l2-2", "l2-3"},
		}

		l2_0 := &SpeculativeBranch{
			ID:          "l2-0",
			Status:      BranchStatusTesting,
			ParentID:    "l1-0",
			ChildrenIDs: []string{"l3-0"},
		}
		l2_1 := &SpeculativeBranch{
			ID:       "l2-1",
			Status:   BranchStatusTesting,
			ParentID: "l1-0",
		}
		l2_2 := &SpeculativeBranch{
			ID:       "l2-2",
			Status:   BranchStatusTesting,
			ParentID: "l1-1",
		}
		l2_3 := &SpeculativeBranch{
			ID:       "l2-3",
			Status:   BranchStatusTesting,
			ParentID: "l1-1",
		}
		l3_0 := &SpeculativeBranch{
			ID:       "l3-0",
			Status:   BranchStatusTesting,
			ParentID: "l2-0",
		}

		coord.mu.Lock()
		coord.activeBranches["root"] = root
		coord.activeBranches["l1-0"] = l1_0
		coord.activeBranches["l1-1"] = l1_1
		coord.activeBranches["l2-0"] = l2_0
		coord.activeBranches["l2-1"] = l2_1
		coord.activeBranches["l2-2"] = l2_2
		coord.activeBranches["l2-3"] = l2_3
		coord.activeBranches["l3-0"] = l3_0
		coord.mu.Unlock()
		b.StartTimer()

		// Benchmark: Kill entire complex hierarchy
		_ = coord.killDependentBranchesWithTimeout(ctx, "root")
	}
}

// BenchmarkKillProtectionMechanism_MutexAcquisition measures lock acquisition time
func BenchmarkKillProtectionMechanism_MutexAcquisition(b *testing.B) {
	coord := NewCoordinator(CoordinatorConfig{
		KillSwitchTimeout: 500 * time.Millisecond,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Benchmark: Only measure lock/unlock time
		coord.mu.Lock()
		// Do minimal work
		_ = len(coord.activeBranches)
		coord.mu.Unlock()
	}
}

// BenchmarkKillProtectionMechanism_TimeoutContextCreation measures timeout context overhead
func BenchmarkKillProtectionMechanism_TimeoutContextCreation(b *testing.B) {
	ctx := context.Background()
	timeout := 500 * time.Millisecond

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Benchmark: Context creation overhead
		killCtx, cancel := context.WithTimeout(ctx, timeout)
		cancel()
		_ = killCtx
	}
}

// BenchmarkKillProtectionMechanism_ChannelSelect measures timeout channel select overhead
func BenchmarkKillProtectionMechanism_ChannelSelect(b *testing.B) {
	ctx := context.Background()
	killCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Benchmark: Channel select on context timeout
		select {
		case <-killCtx.Done():
			// Timeout occurred
		default:
			// Not timed out
		}
	}
}

// BenchmarkKillProtectionMechanism_IdempotencyCheck measures idempotency check overhead
func BenchmarkKillProtectionMechanism_IdempotencyCheck(b *testing.B) {
	coord := NewCoordinator(CoordinatorConfig{
		KillSwitchTimeout: 500 * time.Millisecond,
	})

	// Pre-populate with killed branch
	killedTime := time.Now()
	branch := &SpeculativeBranch{
		ID:         "killed-branch",
		Status:     BranchStatusKilled,
		KilledAt:   &killedTime,
		KillReason: "already killed",
	}
	coord.mu.Lock()
	coord.activeBranches["killed-branch"] = branch
	coord.mu.Unlock()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Benchmark: Just idempotency check (branch already killed)
		coord.mu.Lock()
		b, exists := coord.activeBranches["killed-branch"]
		isKilled := exists && b.Status == BranchStatusKilled
		coord.mu.Unlock()
		_ = isKilled
	}
}

// BenchmarkMemoryAllocation_BranchCreation measures memory overhead of branch creation
func BenchmarkMemoryAllocation_BranchCreation(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = &SpeculativeBranch{
			ID:     fmt.Sprintf("branch-%d", i),
			Status: BranchStatusTesting,
			Changes: []ChangeRequest{
				{
					ID:            "agent-1",
					FilesModified: []string{"src/test.go"},
				},
			},
		}
	}
}

// BenchmarkMemoryAllocation_KilledBranchMetadata measures memory for kill metadata
func BenchmarkMemoryAllocation_KilledBranchMetadata(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		killedTime := time.Now()
		_ = &SpeculativeBranch{
			ID:         fmt.Sprintf("branch-%d", i),
			Status:     BranchStatusKilled,
			KilledAt:   &killedTime,
			KillReason: "test failure with detailed reason for memory measurement",
		}
	}
}

// BenchmarkMemoryAllocation_HierarchyCreation measures memory for hierarchy structures
func BenchmarkMemoryAllocation_HierarchyCreation(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		childrenIDs := make([]string, 10)
		for j := 0; j < 10; j++ {
			childrenIDs[j] = fmt.Sprintf("child-%d-%d", i, j)
		}
		_ = &SpeculativeBranch{
			ID:          fmt.Sprintf("parent-%d", i),
			Status:      BranchStatusTesting,
			ChildrenIDs: childrenIDs,
		}
	}
}

// BenchmarkConcurrentKillOperations measures behavior under concurrent kills
func BenchmarkConcurrentKillOperations(b *testing.B) {
	ctx := context.Background()
	branchCount := 100
	concurrency := 10

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Setup: Create coordinator with many branches
		coord := NewCoordinator(CoordinatorConfig{
			KillSwitchTimeout: 500 * time.Millisecond,
		})

		for j := 0; j < branchCount; j++ {
			branchID := fmt.Sprintf("bench-branch-%d-%d", i, j)
			branch := &SpeculativeBranch{
				ID:     branchID,
				Status: BranchStatusTesting,
			}
			coord.mu.Lock()
			coord.activeBranches[branchID] = branch
			coord.mu.Unlock()
		}
		b.StartTimer()

		// Benchmark: Kill branches with limited concurrency
		var wg sync.WaitGroup
		semaphore := make(chan struct{}, concurrency)

		for j := 0; j < branchCount; j++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				branchID := fmt.Sprintf("bench-branch-%d-%d", i, idx)
				_ = coord.killFailedBranchWithTimeout(ctx, branchID, "test failure")
			}(j)
		}
		wg.Wait()
		b.StopTimer()
	}
}

// buildBinaryTreeHelper recursively builds a binary tree structure
func buildBinaryTreeHelper(coord *Coordinator, parentID string, level, maxLevel int, counter *int) {
	if level > maxLevel {
		return
	}

	*counter++
	leftID := fmt.Sprintf("node-%d-L", *counter)
	*counter++
	rightID := fmt.Sprintf("node-%d-R", *counter)

	parent := coord.activeBranches[parentID]
	parent.ChildrenIDs = []string{leftID, rightID}

	left := &SpeculativeBranch{
		ID:       leftID,
		Status:   BranchStatusTesting,
		ParentID: parentID,
	}
	right := &SpeculativeBranch{
		ID:       rightID,
		Status:   BranchStatusTesting,
		ParentID: parentID,
	}

	coord.activeBranches[leftID] = left
	coord.activeBranches[rightID] = right

	buildBinaryTreeHelper(coord, leftID, level+1, maxLevel, counter)
	buildBinaryTreeHelper(coord, rightID, level+1, maxLevel, counter)
}

// BenchmarkCascadeKillTiming measures cascade kill performance with contention
func BenchmarkCascadeKillTiming(b *testing.B) {
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Setup: Create coordinator with branching hierarchy
		coord := NewCoordinator(CoordinatorConfig{
			KillSwitchTimeout: 500 * time.Millisecond,
		})

		// Create binary tree structure: 63 nodes (7 levels)
		//               root
		//              /    \
		//           n-1      n-2
		//          /  \      /  \
		//        n-3 n-4  n-5 n-6
		//        ...

		root := &SpeculativeBranch{
			ID:     "root",
			Status: BranchStatusFailed,
		}
		coord.mu.Lock()
		coord.activeBranches["root"] = root
		counter := 0
		buildBinaryTreeHelper(coord, "root", 1, 5, &counter)
		coord.mu.Unlock()
		b.StartTimer()

		// Benchmark: Kill entire binary tree
		_ = coord.killDependentBranchesWithTimeout(ctx, "root")
	}
}
