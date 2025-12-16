package mergequeue

import (
	"context"
	"fmt"
	"log/slog"
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

	validator *KillSwitchValidator
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
		validator:      NewKillSwitchValidator(),
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
func (c *Coordinator) createSpeculativeBranches(ctx context.Context, batch []*ChangeRequest) {
	// Delegate to implementation in speculative_execution.go
	c.createSpeculativeBranchesImpl(ctx, batch)
}

// processBypass handles independent changes in bypass lane
func (c *Coordinator) processBypass(ctx context.Context, change *ChangeRequest) {
	// Delegate to implementation in speculative_execution.go
	c.processBypassImpl(ctx, change)
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
		cascadeErr := c.KillDependentBranchesWithValidation(ctx, failedBranchID)
		if cascadeErr != nil {
			// Log cascade error but continue to kill the main branch
			// The cascade may have partially succeeded
			if userErr, ok := cascadeErr.(*UserFacingError); ok {
				// Already formatted for user display
				_ = userErr // would be logged in production
			} else if killErr, ok := cascadeErr.(*KillSwitchError); ok {
				// Log kill switch context
				_ = killErr // would be logged in production
			} else {
				// Wrap unknown error
				_ = fmt.Errorf("cascade kill partially failed: %w", cascadeErr)
			}
		}

		// Then kill the failed branch itself
		mainKillErr := c.KillFailedBranchWithValidation(ctx, failedBranchID, fmt.Sprintf("tests failed: %s", result.ErrorMessage))
		if mainKillErr != nil {
			// Log error but branch state is still updated
			_ = mainKillErr // would be logged in production
		}
		// TODO: Promote next change to base
	}

	// Update metrics
	c.updateStats(result)
}

// updateStats updates queue performance metrics
func (c *Coordinator) updateStats(result *TestResult) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Track test failures
	if !result.Passed {
		c.stats.TotalFailures++
	}

	// Check if this was a timeout (duration exceeds or equals test timeout)
	if result.Duration >= c.config.TestTimeout {
		c.stats.TotalTimeouts++
	}
}

// GetStats returns current queue statistics
func (c *Coordinator) GetStats() QueueStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.stats
}

// KillFailedBranchWithValidation kills a single speculative branch and cleans up its resources
func (c *Coordinator) KillFailedBranchWithValidation(ctx context.Context, branchID string, reason string, requestingAgent string) *BranchValidationError {
	logger := slog.Default()
	startTime := time.Now()

	// Log kill initiation
	logger.Info("Kill switch initiated (basic version)",
		"branch_id", branchID,
		"reason", reason,
		"requesting_agent", requestingAgent,
	)

	// Pre-validation (read-only access)
	c.mu.RLock()
	branch, exists := c.activeBranches[branchID]
	c.mu.RUnlock()

	if !exists {
		return c.validator.ValidateBranchExists(nil, branchID)
	}

	if validationErr := c.validator.ValidateFullKillSwitchPrerequisites(branch, branchID, requestingAgent); validationErr != nil {
		return validationErr
	}

	// If validation passes, proceed with modification (write-lock)
	c.mu.Lock()
	defer c.mu.Unlock()

	// Re-check existence under write lock (though unlikely to change after RLock)
	branch, exists = c.activeBranches[branchID]
	if !exists {
		// Should not happen if pre-validation passed, but defensive check
		return c.validator.ValidateBranchExists(nil, branchID)
	}

	// Idempotent: if already killed, return success
	if branch.Status == BranchStatusKilled {
		logger.Debug("Branch already killed (idempotent)",
			"branch_id", branchID,
			"previous_reason", branch.KillReason,
		)
		return nil
	}

	logger.Debug("Beginning branch cleanup",
		"branch_id", branchID,
		"current_status", branch.Status,
		"changes_count", len(branch.Changes),
		"depth", branch.Depth,
	)

	// TODO: Docker container cleanup removed - restore when DockerManager is implemented
	// Container cleanup will be handled externally for now

	// TODO: Cancel Temporal workflow

	// Clean up worktrees for this branch
	if err := c.cleanupBranchWorktrees(context.Background(), branch); err != nil {
		logger.Warn("Worktree cleanup had errors",
			"branch_id", branchID,
			"error", err.Error(),
		)
	} else {
		logger.Debug("Worktree cleanup completed",
			"branch_id", branchID,
		)
	}

	// Update branch status
	now := time.Now()
	branch.Status = BranchStatusKilled
	branch.KilledAt = &now
	branch.KillReason = reason

	// Track kill switch activation
	c.stats.TotalKills++

	duration := time.Since(startTime)
	logger.Info("Kill switch completed",
		"branch_id", branchID,
		"reason", reason,
		"killed_at", now,
		"total_kills", c.stats.TotalKills,
		"duration_ms", duration.Milliseconds(),
	)

	// Send notification to affected agents (unlock before notification to avoid deadlock)
	c.mu.Unlock()
	notifyStart := time.Now()
	notifyErr := c.notifyBranchKilled(ctx, branch, reason)
	notifyDuration := time.Since(notifyStart)

	if notifyErr != nil {
		logger.Warn("Branch kill notification failed",
			"branch_id", branchID,
			"error", notifyErr.Error(),
			"notification_duration_ms", notifyDuration.Milliseconds(),
		)
	} else {
		logger.Debug("Branch kill notification sent successfully",
			"branch_id", branchID,
			"notified_agents", len(branch.Changes),
			"notification_duration_ms", notifyDuration.Milliseconds(),
		)
	}
	c.mu.Lock()

	return nil
}

// KillDependentBranchesWithValidation recursively kills all child branches when a parent fails
func (c *Coordinator) KillDependentBranchesWithValidation(ctx context.Context, branchID string, requestingAgent string) (*BranchValidationError, error) {
	logger := slog.Default()
	startTime := time.Now()

	logger.Info("Cascade kill initiated (basic version)",
		"parent_branch_id", branchID,
		"requesting_agent", requestingAgent,
	)

	// Pre-validation (read-only access)
	c.mu.RLock()
	parentBranch, exists := c.activeBranches[branchID]
	c.mu.RUnlock()

	if !exists {
		return c.validator.ValidateBranchExists(nil, branchID), nil // Validation error, no cascade error
	}

	if validationErr := c.validator.ValidateFullKillSwitchPrerequisites(parentBranch, branchID, requestingAgent); validationErr != nil {
		return validationErr, nil // Validation error, no cascade error
	}

	// Acquire write lock for modifications
	c.mu.Lock()
	// Re-check existence under write lock
	parentBranch, exists = c.activeBranches[branchID]
	if !exists {
		// Should not happen if pre-validation passed, but defensive check
		c.mu.Unlock()
		return c.validator.ValidateBranchExists(nil, branchID), nil
	}
	childrenIDs := make([]string, len(parentBranch.ChildrenIDs))
	copy(childrenIDs, parentBranch.ChildrenIDs)
	c.mu.Unlock() // Release lock briefly to allow recursive calls

	logger.Debug("Processing dependent branches",
		"parent_branch_id", branchID,
		"dependent_branches", len(childrenIDs),
		"children_ids", childrenIDs,
	)

	var cascadeError error // For errors during cascade
	var killedCount int
	var failureCount int

	// Recursively kill all children
	for idx, childID := range childrenIDs {
		logger.Debug("Processing child branch",
			"parent_branch_id", branchID,
			"child_branch_id", childID,
			"position", idx+1,
			"total_children", len(childrenIDs),
		)

		// First kill the child's descendants
		if valErr, err := c.KillDependentBranchesWithValidation(ctx, childID, requestingAgent); valErr != nil || err != nil {
			logger.Error("Failed to cascade kill descendants",
				"child_branch_id", childID,
				"parent_branch_id", branchID,
				"validation_error", valErr,
				"cascade_error", err,
			)
			if cascadeError == nil {
				if valErr != nil {
					cascadeError = valErr
				} else {
					cascadeError = err
				}
			}
			failureCount++
			continue
		}

		// Then kill the child itself
		childKillReason := fmt.Sprintf("parent branch %s failed", branchID)
		if valErr := c.KillFailedBranchWithValidation(ctx, childID, childKillReason, requestingAgent); valErr != nil {
			logger.Error("Failed to kill dependent branch",
				"child_branch_id", childID,
				"parent_branch_id", branchID,
				"validation_error", valErr,
				"kill_reason", childKillReason,
			)
			if cascadeError == nil {
				cascadeError = valErr
			}
			failureCount++
			continue
		}

		killedCount++
		logger.Debug("Successfully killed dependent branch",
			"child_branch_id", childID,
			"parent_branch_id", branchID,
			"killed_count", killedCount,
			"total_killed", len(childrenIDs),
		)
	}

	duration := time.Since(startTime)

	if failureCount > 0 {
		logger.Warn("Cascade kill completed with failures",
			"parent_branch_id", branchID,
			"total_children", len(childrenIDs),
			"successfully_killed", killedCount,
			"failed_kills", failureCount,
			"duration_ms", duration.Milliseconds(),
		)
	} else if len(childrenIDs) > 0 {
		logger.Info("Cascade kill completed successfully",
			"parent_branch_id", branchID,
			"dependent_branches_killed", killedCount,
			"duration_ms", duration.Milliseconds(),
		)
	} else {
		logger.Debug("No dependent branches to kill",
			"parent_branch_id", branchID,
			"duration_ms", duration.Milliseconds(),
		)
	}

	return nil, cascadeError
}

// GetBranchHealthReport provides a detailed status report for a branch
func (c *Coordinator) GetBranchHealthReport(branchID string) *BranchHealthReport {
	c.mu.RLock()
	branch, exists := c.activeBranches[branchID]
	c.mu.RUnlock()

	if !exists {
		return c.validator.GenerateHealthReport(nil, branchID)
	}
	return c.validator.GenerateHealthReport(branch, branchID)
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
