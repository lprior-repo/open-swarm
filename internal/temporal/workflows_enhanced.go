// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package temporal

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// gateExecutor executes a gate activity and handles common error/duration tracking
type gateExecutor struct {
	ctx            workflow.Context
	logger         log.Logger
	result         *EnhancedTCRResult
	bootstrap      *BootstrapOutput
	cellActivities *CellActivities
}

// executeGate runs a gate activity and handles result tracking
func (ge *gateExecutor) executeGate(gateName string, activityFn interface{}, args ...interface{}) (*GateResult, error) {
	ge.logger.Info(fmt.Sprintf("Gate: %s", gateName))
	gateStart := workflow.Now(ge.ctx)

	var gateResult *GateResult
	err := workflow.ExecuteActivity(ge.ctx, activityFn, args...).Get(ge.ctx, &gateResult)

	if err != nil {
		gateResult = &GateResult{
			GateName: gateName,
			Passed:   false,
			Error:    err.Error(),
			Duration: workflow.Now(ge.ctx).Sub(gateStart),
		}
	} else {
		gateResult.Duration = workflow.Now(ge.ctx).Sub(gateStart)
	}

	ge.result.GateResults = append(ge.result.GateResults, *gateResult)

	if !gateResult.Passed {
		ge.result.Error = fmt.Sprintf("%s failed: %v", gateName, gateResult.Error)
		// Revert changes on gate failure
		_ = workflow.ExecuteActivity(ge.ctx, ge.cellActivities.RevertChanges, ge.bootstrap).Get(ge.ctx, nil)
		return gateResult, fmt.Errorf("gate failed")
	}

	// Track files changed if available
	if len(gateResult.AgentResults) > 0 {
		for _, agentResult := range gateResult.AgentResults {
			ge.result.FilesChanged = append(ge.result.FilesChanged, agentResult.FilesChanged...)
		}
	}

	return gateResult, nil
}

// EnhancedTCRWorkflow implements the 6-Gate Enhanced TCR pattern with file locks
// Flow: Bootstrap → AcquireLocks → [GenTest → LintTest → VerifyRED → GenImpl → VerifyGREEN → MultiReview] → Commit/Revert → ReleaseLocks → Teardown
//
// Uses saga pattern to guarantee lock release even on failure.
// All gates must pass sequentially; failure triggers revert and retry signal.
func EnhancedTCRWorkflow(ctx workflow.Context, input EnhancedTCRInput) (*EnhancedTCRResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting Enhanced TCR Workflow", "taskID", input.TaskID)

	// Initialize result
	result := &EnhancedTCRResult{
		Success:      false,
		GateResults:  []GateResult{},
		FilesChanged: []string{},
		Error:        "",
	}

	// Activity options
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		HeartbeatTimeout:    30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 1, // Don't retry non-idempotent operations
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	cellActivities := NewCellActivities()
	enhancedActivities := NewEnhancedActivities()

	// STEP 1: Bootstrap Cell
	logger.Info("Gate: Bootstrap")
	var bootstrap *BootstrapOutput
	err := workflow.ExecuteActivity(ctx, cellActivities.BootstrapCell, BootstrapInput{
		CellID: input.CellID,
		Branch: input.Branch,
	}).Get(ctx, &bootstrap)

	if err != nil {
		result.Error = fmt.Sprintf("bootstrap failed: %v", err)
		return result, nil
	}

	// SAGA PATTERN: Ensure cleanup happens (teardown + lock release)
	var locksAcquired []string
	defer func() {
		// Use disconnected context for cleanup
		disconnCtx, _ := workflow.NewDisconnectedContext(ctx)
		cleanupAo := workflow.ActivityOptions{
			StartToCloseTimeout: 2 * time.Minute,
			RetryPolicy: &temporal.RetryPolicy{
				MaximumAttempts: 3,
			},
		}
		disconnCtx = workflow.WithActivityOptions(disconnCtx, cleanupAo)

		// Release all acquired locks
		if len(locksAcquired) > 0 {
			logger.Info("Saga: Releasing file locks", "count", len(locksAcquired))
			_ = workflow.ExecuteActivity(disconnCtx, enhancedActivities.ReleaseFileLocks,
				input.CellID, locksAcquired).Get(disconnCtx, nil)
		}

		// Teardown cell
		logger.Info("Saga: Tearing down cell", "cellID", bootstrap.CellID)
		_ = workflow.ExecuteActivity(disconnCtx, cellActivities.TeardownCell, bootstrap).Get(disconnCtx, nil)
	}()

	// STEP 2: Acquire File Locks
	logger.Info("Acquiring file locks for task files")
	var filesLocked []string
	err = workflow.ExecuteActivity(ctx, enhancedActivities.AcquireFileLocks,
		input.CellID, input.TaskID).Get(ctx, &filesLocked)

	if err != nil {
		result.Error = fmt.Sprintf("failed to acquire locks: %v", err)
		return result, nil
	}
	locksAcquired = filesLocked
	logger.Info("File locks acquired", "files", filesLocked)

	// Create gate executor for simplified gate execution
	executor := &gateExecutor{
		ctx:            ctx,
		logger:         logger,
		result:         result,
		bootstrap:      bootstrap,
		cellActivities: cellActivities,
	}

	// GATE 1: GenTest - Generate Tests
	_, err = executor.executeGate("GenTest", enhancedActivities.ExecuteGenTest,
		bootstrap, input.TaskID, input.AcceptanceCriteria)
	if err != nil {
		return result, nil
	}

	// GATE 2: LintTest - Lint Test Files
	_, err = executor.executeGate("LintTest", enhancedActivities.ExecuteLintTest, bootstrap)
	if err != nil {
		return result, nil
	}

	// GATE 3: VerifyRED - Tests Must Fail
	_, err = executor.executeGate("VerifyRED", enhancedActivities.ExecuteVerifyRED, bootstrap)
	if err != nil {
		return result, nil
	}

	// GATE 4: GenImpl - Generate Implementation
	_, err = executor.executeGate("GenImpl", enhancedActivities.ExecuteGenImpl,
		bootstrap, input.TaskID, input.Description, input.AcceptanceCriteria, "")
	if err != nil {
		return result, nil
	}

	// GATE 5: VerifyGREEN - Tests Must Pass
	_, err = executor.executeGate("VerifyGREEN", enhancedActivities.ExecuteVerifyGREEN, bootstrap)
	if err != nil {
		return result, nil
	}

	// GATE 6: MultiReview - 3 Reviewers with Unanimous Approval
	reviewersCount := input.ReviewersCount
	if reviewersCount == 0 {
		reviewersCount = 3 // Default: 3 reviewers
	}

	_, err = executor.executeGate("MultiReview", enhancedActivities.ExecuteMultiReview,
		bootstrap, input.TaskID, input.Description, reviewersCount)
	if err != nil {
		return result, nil
	}

	// ALL GATES PASSED: Commit Changes
	logger.Info("All gates passed - committing changes")

	commitMsg := fmt.Sprintf("Task %s: %s\n\nEnhanced TCR - All 6 gates passed\n\nGenerated by Open Swarm", input.TaskID, input.Description)
	err = workflow.ExecuteActivity(ctx, cellActivities.CommitChanges, bootstrap, commitMsg).Get(ctx, nil)
	if err != nil {
		logger.Warn("Commit failed", "error", err)
		result.Error = fmt.Sprintf("commit failed: %v", err)
		return result, nil
	}

	// Success!
	result.Success = true

	logger.Info("Enhanced TCR Workflow completed successfully",
		"taskID", input.TaskID)

	return result, nil
}
