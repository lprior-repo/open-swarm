package mergequeue

import (
	"context"
	"fmt"
	"sync"
	"time"
)

const (
	// Default kill switch timeout duration (milliseconds)
	defaultKillSwitchTimeout = 500 * time.Millisecond
	// Default speculation depth for merge queue
	defaultDepth = 5
	// Default depth offset for adaptive depth calculation
	depthOffset = 2
	// Minimum adaptive depth
	minAdaptiveDepth = 2
	// Process queue tick interval
	processQueueTickInterval = 100 * time.Millisecond
	// Default channel buffer capacity
	defaultChannelCapacity = 100
)

// Coordinator implements Uber-style speculative merge queue
type Coordinator struct {
	mu sync.RWMutex

	// Main queue (conflicting changes)
	mainQueue []*ChangeRequest

	// Bypass lane (independent changes)
	bypassLane []*ChangeRequest

	// Active speculative branches
	activeBranches map[string]*SpeculativeBranch

	// Configuration
	config CoordinatorConfig

	// Metrics
	stats QueueStats

	// Channels for coordination
	changesChan  chan *ChangeRequest
	resultsChan  chan *TestResult
	shutdownChan chan struct{}
	shutdownOnce sync.Once
}

// CoordinatorConfig holds merge queue configuration
type CoordinatorConfig struct {
	// Sliding window size (default 5)
	WindowSize int

	// Max bypass lane concurrency (default 3)
	MaxBypassSlots int

	// Default speculation depth (default 5)
	DefaultDepth int

	// Minimum pass rate to increase depth (default 0.90)
	HighPassRateThreshold float64

	// Maximum pass rate to decrease depth (default 0.70)
	LowPassRateThreshold float64

	// Kill switch timeout (default 500ms)
	KillSwitchTimeout time.Duration

	// Test timeout per change (default 5 minutes)
	TestTimeout time.Duration
}

// NewCoordinator creates a new merge queue coordinator
func NewCoordinator(config CoordinatorConfig) *Coordinator {
	// Set defaults
	if config.WindowSize == 0 {
		config.WindowSize = 5
	}
	if config.MaxBypassSlots == 0 {
		config.MaxBypassSlots = 3
	}
	if config.DefaultDepth == 0 {
		config.DefaultDepth = defaultDepth
	}
	if config.HighPassRateThreshold == 0 {
		config.HighPassRateThreshold = 0.90
	}
	if config.LowPassRateThreshold == 0 {
		config.LowPassRateThreshold = 0.70
	}
	if config.KillSwitchTimeout == 0 {
		config.KillSwitchTimeout = defaultKillSwitchTimeout
	}
	if config.TestTimeout == 0 {
		config.TestTimeout = 5 * time.Minute
	}

	return &Coordinator{
		mainQueue:      make([]*ChangeRequest, 0),
		bypassLane:     make([]*ChangeRequest, 0),
		activeBranches: make(map[string]*SpeculativeBranch),
		config:         config,
		changesChan:    make(chan *ChangeRequest, defaultChannelCapacity),
		resultsChan:    make(chan *TestResult, defaultChannelCapacity),
		shutdownChan:   make(chan struct{}),
	}
}

// Start begins processing the merge queue
func (c *Coordinator) Start(ctx context.Context) error {
	go c.processQueue(ctx)
	go c.handleResults(ctx)
	return nil
}

// Stop gracefully shuts down the coordinator
func (c *Coordinator) Stop() error {
	c.shutdownOnce.Do(func() {
		close(c.shutdownChan)
	})
	return nil
}

// Submit adds a change request to the queue
func (c *Coordinator) Submit(ctx context.Context, change *ChangeRequest) error {
	select {
	case c.changesChan <- change:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("failed to submit change: %w", ctx.Err())
	case <-c.shutdownChan:
		return fmt.Errorf("coordinator is shutting down")
	}
}

// processQueue is the main coordinator loop
func (c *Coordinator) processQueue(ctx context.Context) {
	ticker := time.NewTicker(processQueueTickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.shutdownChan:
			return
		case change := <-c.changesChan:
			c.handleNewChange(ctx, change)
		case <-ticker.C:
			c.processMainQueue(ctx)
		}
	}
}

// handleNewChange routes change to bypass lane or main queue
func (c *Coordinator) handleNewChange(ctx context.Context, change *ChangeRequest) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if independent (can bypass)
	if c.isIndependent(change) && len(c.bypassLane) < c.config.MaxBypassSlots {
		c.bypassLane = append(c.bypassLane, change)
		go c.processBypass(ctx, change)
	} else {
		c.mainQueue = append(c.mainQueue, change)
	}
}

// isIndependent checks if change conflicts with any queued changes
func (c *Coordinator) isIndependent(change *ChangeRequest) bool {
	// Check against main queue
	for _, queued := range c.mainQueue {
		if c.hasConflict(change, queued) {
			return false
		}
	}

	// Check against bypass lane
	for _, queued := range c.bypassLane {
		if c.hasConflict(change, queued) {
			return false
		}
	}

	return true
}

// hasConflict determines if two changes conflict (directory-based)
func (c *Coordinator) hasConflict(a, b *ChangeRequest) bool {
	// Simple directory-based conflict detection
	// TODO: Integrate with Agent Mail reservations
	for _, fileA := range a.FilesModified {
		for _, fileB := range b.FilesModified {
			// If files share same directory, consider it a conflict
			if sharesDirectory(fileA, fileB) {
				return true
			}
		}
	}
	return false
}

// processMainQueue implements Uber's speculative execution
func (c *Coordinator) processMainQueue(ctx context.Context) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.mainQueue) == 0 {
		return
	}

	// Calculate adaptive depth based on success rate
	depth := c.calculateDepth()

	// Take up to 'depth' changes from queue
	batchSize := minInt(depth, len(c.mainQueue))
	batch := c.mainQueue[:batchSize]

	// Create speculative branches
	c.createSpeculativeBranches(ctx, batch)
}

// calculateDepth adapts speculation depth based on historical success rate
func (c *Coordinator) calculateDepth() int {
	if c.stats.SuccessRate >= c.config.HighPassRateThreshold {
		return c.config.DefaultDepth + depthOffset // Increase depth
	} else if c.stats.SuccessRate <= c.config.LowPassRateThreshold {
		return maxInt(minAdaptiveDepth, c.config.DefaultDepth-depthOffset) // Decrease depth
	}
	return c.config.DefaultDepth
}

// createSpeculativeBranches spawns parallel tests for batch
func (c *Coordinator) createSpeculativeBranches(_ context.Context, batch []*ChangeRequest) {
	// TODO: Implement speculative branch creation
	// 1. Create base test (change[0])
	// 2. Create speculative tests (change[0]+change[1], change[0]+change[1]+change[2], etc)
	// 3. Run all tests in parallel
}

// processBypass handles independent changes in bypass lane
func (c *Coordinator) processBypass(_ context.Context, change *ChangeRequest) {
	// TODO: Implement bypass lane processing
	// 1. Test change in isolation
	// 2. If pass -> merge directly to main
	// 3. If fail -> revert and move to manual review
}

// handleResults processes test results and makes merge decisions
func (c *Coordinator) handleResults(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-c.shutdownChan:
			return
		case result := <-c.resultsChan:
			c.processTestResult(ctx, result)
		}
	}
}

// processTestResult handles a completed test
func (c *Coordinator) processTestResult(ctx context.Context, result *TestResult) {
	c.mu.Lock()

	// Find which branch this result belongs to
	var failedBranchID string
	for id, branch := range c.activeBranches {
		if branch.TestResult == result || len(result.ChangeIDs) > 0 {
			// Match by change IDs
			match := true
			if len(branch.Changes) == len(result.ChangeIDs) {
				for i, change := range branch.Changes {
					if i >= len(result.ChangeIDs) || change.ID != result.ChangeIDs[i] {
						match = false
						break
					}
				}
			} else {
				match = false
			}
			if match {
				failedBranchID = id
				break
			}
		}
	}
	c.mu.Unlock()

	if result.Passed {
		// TODO: Merge successful changes
		// TODO: Promote queue (remove merged, advance next)
	} else if failedBranchID != "" {
		// Kill switch: Kill failed branch and all its dependent children
		// First kill all dependent branches
		if err := c.killDependentBranches(ctx, failedBranchID); err != nil {
			// Log error but continue
			_ = err
		}

		// Then kill the failed branch itself
		if err := c.killFailedBranch(ctx, failedBranchID, fmt.Sprintf("tests failed: %s", result.ErrorMessage)); err != nil {
			// Log error
			_ = err
		}
		// TODO: Promote next change to base
	}

	// Update metrics
	c.updateStats(result)
}

// updateStats updates queue performance metrics
func (c *Coordinator) updateStats(_ *TestResult) {
	// TODO: Implement metrics tracking
}

// GetStats returns current queue statistics
func (c *Coordinator) GetStats() QueueStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.stats
}

// killFailedBranch kills a single speculative branch and cleans up its resources
func (c *Coordinator) killFailedBranch(_ context.Context, branchID string, reason string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	branch, exists := c.activeBranches[branchID]
	if !exists {
		return fmt.Errorf("branch %s not found", branchID)
	}

	// Idempotent: if already killed, return success
	if branch.Status == BranchStatusKilled {
		return nil
	}

	// TODO: Cancel Temporal workflow
	// TODO: Stop Docker container
	// TODO: Clean up worktree

	// Update branch status
	now := time.Now()
	branch.Status = BranchStatusKilled
	branch.KilledAt = &now
	branch.KillReason = reason

	return nil
}

// killDependentBranches recursively kills all child branches when a parent fails
func (c *Coordinator) killDependentBranches(ctx context.Context, branchID string) error {
	c.mu.Lock()
	branch, exists := c.activeBranches[branchID]
	if !exists {
		c.mu.Unlock()
		return fmt.Errorf("branch %s not found", branchID)
	}

	childrenIDs := make([]string, len(branch.ChildrenIDs))
	copy(childrenIDs, branch.ChildrenIDs)
	c.mu.Unlock()

	// Recursively kill all children
	for _, childID := range childrenIDs {
		// First kill the child's descendants
		if err := c.killDependentBranches(ctx, childID); err != nil {
			// Log error but continue killing other branches
			continue
		}

		// Then kill the child itself
		if err := c.killFailedBranch(ctx, childID, fmt.Sprintf("parent branch %s failed", branchID)); err != nil {
			// Log error but continue
			continue
		}
	}

	return nil
}

// Helper functions
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func sharesDirectory(pathA, pathB string) bool {
	// TODO: Implement proper directory sharing detection
	// For now, simple check
	return false
}
