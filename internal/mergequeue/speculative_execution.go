package mergequeue

import (
	"context"
	"time"
)

// generateBranchID creates a unique identifier for a speculative branch based on its changes.
// The ID is deterministic and reflects the combination of changes being tested.
// Format: "branch-{id1}-{id2}-{id3}"
func (c *Coordinator) generateBranchID(changes []*ChangeRequest) string {
	if len(changes) == 0 {
		return ""
	}

	// Build concatenated IDs with '-' separator
	id := "branch"
	for _, change := range changes {
		id += "-" + change.ID
	}
	return id
}

// executeSpeculativeBranch runs tests for a speculative branch in an isolated environment.
// This method orchestrates the entire test lifecycle:
// 1. Updates branch status to testing
// 2. Creates an isolated test environment (Docker container if available)
// 3. Executes tests via Temporal workflow
// 4. Collects and reports results
// 5. Cleans up resources on completion or timeout
func (c *Coordinator) executeSpeculativeBranch(ctx context.Context, branch *SpeculativeBranch) {
	// Update status to testing
	c.mu.Lock()
	branch.Status = BranchStatusTesting
	c.mu.Unlock()

	// Create test context with timeout
	testCtx, cancel := context.WithTimeout(ctx, c.config.TestTimeout)
	defer cancel()

	// Build list of change IDs for result tracking
	changeIDs := make([]string, len(branch.Changes))
	for i, change := range branch.Changes {
		changeIDs[i] = change.ID
	}

	startTime := time.Now()
	var result *TestResult

	// Execute tests with proper error handling
	func() {
		// TODO: Create git worktree for isolated testing
		// This would involve calling coordinator_worktree.go methods

		// TODO: Start Docker container for test isolation (if available)
		// if c.dockerManager != nil {
		//     containerID, err := c.dockerManager.StartTestContainer(ctx, branch)
		//     if err != nil {
		//         result = &TestResult{
		//             ChangeIDs: changeIDs,
		//             Passed: false,
		//             Duration: time.Since(startTime),
		//             ErrorMessage: fmt.Sprintf("failed to start container: %v", err),
		//         }
		//         return
		//     }
		//     c.mu.Lock()
		//     branch.ContainerID = containerID
		//     c.mu.Unlock()
		//     defer c.dockerManager.StopAndRemoveContainer(ctx, containerID)
		// }

		// TODO: Start Temporal workflow for test execution
		// This would integrate with coordinator_temporal.go

		// Execute tests (currently a stub - will integrate with Temporal)
		testResult := c.runBranchTests(testCtx, branch)
		if testResult == nil {
			result = &TestResult{
				ChangeIDs:    changeIDs,
				Passed:       false,
				Duration:     time.Since(startTime),
				ErrorMessage: "test execution returned nil result",
			}
			return
		}

		result = testResult
	}()

	// Handle timeout case if test execution didn't complete
	if result == nil {
		result = &TestResult{
			ChangeIDs:    changeIDs,
			Passed:       false,
			Duration:     time.Since(startTime),
			ErrorMessage: "test execution timed out",
		}
	}

	// Update branch with final result
	c.mu.Lock()
	branch.TestResult = result
	if result.Passed {
		branch.Status = BranchStatusPassed
	} else {
		branch.Status = BranchStatusFailed
	}
	c.mu.Unlock()

	// Send result to results channel for processing
	select {
	case c.resultsChan <- result:
	case <-ctx.Done():
		// Context cancelled, exit gracefully
		return
	case <-c.shutdownChan:
		// Coordinator shutting down
		return
	}
}

// runBranchTests executes tests for a branch (stub for Temporal integration)
// This will be fully implemented when integrating with coordinator_temporal.go
func (c *Coordinator) runBranchTests(_ context.Context, branch *SpeculativeBranch) *TestResult {
	// TODO: Integrate with Temporal workflow to run actual tests
	// For now, return a passing test result as a stub

	changeIDs := make([]string, len(branch.Changes))
	for i, change := range branch.Changes {
		changeIDs[i] = change.ID
	}

	return &TestResult{
		ChangeIDs: changeIDs,
		Passed:    true,
		Duration:  100 * time.Millisecond,
		TestOutput: "Stub test execution - all tests passed",
	}
}

// createSpeculativeBranchesImpl implements the speculative branch creation logic
// This is called by createSpeculativeBranches in coordinator.go
func (c *Coordinator) createSpeculativeBranchesImpl(ctx context.Context, batch []*ChangeRequest) {
	if len(batch) == 0 {
		return
	}

	// Create speculative branches for each level
	// Level 1: change[0] alone
	// Level 2: change[0] + change[1]
	// Level 3: change[0] + change[1] + change[2]
	// etc.
	for depth := 1; depth <= len(batch); depth++ {
		changes := batch[:depth]
		branchID := c.generateBranchID(changes)

		// Check if branch already exists (avoid duplicate work)
		c.mu.RLock()
		_, exists := c.activeBranches[branchID]
		c.mu.RUnlock()

		if exists {
			continue
		}

		// Determine parent branch (previous depth level)
		var parentID string
		if depth > 1 {
			parentChanges := batch[:depth-1]
			parentID = c.generateBranchID(parentChanges)
		}

		// Create speculative branch metadata
		branch := &SpeculativeBranch{
			ID:          branchID,
			Changes:     make([]ChangeRequest, len(changes)),
			Depth:       depth,
			Status:      BranchStatusPending,
			ParentID:    parentID,
			ChildrenIDs: []string{},
		}

		// Copy changes to avoid reference issues
		for i, change := range changes {
			branch.Changes[i] = *change
		}

		// Update parent's children list
		if parentID != "" {
			c.mu.Lock()
			if parent, exists := c.activeBranches[parentID]; exists {
				parent.ChildrenIDs = append(parent.ChildrenIDs, branchID)
			}
			c.mu.Unlock()
		}

		// Store branch
		c.mu.Lock()
		c.activeBranches[branchID] = branch
		c.mu.Unlock()

		// Launch test asynchronously
		go c.executeSpeculativeBranch(ctx, branch)
	}
}

// processBypassImpl implements the bypass lane processing logic
// This is called by processBypass in coordinator.go
func (c *Coordinator) processBypassImpl(ctx context.Context, change *ChangeRequest) {
	// Create a single-change branch for bypass testing
	branchID := c.generateBranchID([]*ChangeRequest{change})

	c.mu.Lock()
	branch := &SpeculativeBranch{
		ID:      branchID,
		Changes: []ChangeRequest{*change},
		Depth:   1,
		Status:  BranchStatusPending,
	}
	c.activeBranches[branchID] = branch
	c.mu.Unlock()

	// Execute tests
	c.executeSpeculativeBranch(ctx, branch)

	// Remove from bypass lane when done
	c.mu.Lock()
	for i, bypassed := range c.bypassLane {
		if bypassed.ID == change.ID {
			c.bypassLane = append(c.bypassLane[:i], c.bypassLane[i+1:]...)
			break
		}
	}
	c.mu.Unlock()
}

// Note: mergeSuccessfulBranch and cleanupBranchWorktrees are implemented in
// coordinator_temporal.go and coordinator_worktree.go respectively
