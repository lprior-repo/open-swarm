// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package temporal

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

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

	// GATE 1: GenTest - Generate Tests
	logger.Info("Gate 1: GenTest")
	gateStart := workflow.Now(ctx)

	var genTestResult *GateResult
	err = workflow.ExecuteActivity(ctx, enhancedActivities.ExecuteGenTest,
		bootstrap, input.TaskID, input.AcceptanceCriteria).Get(ctx, &genTestResult)

	if err != nil || !genTestResult.Passed {
		genTestResult.Duration = workflow.Now(ctx).Sub(gateStart)
		result.GateResults = append(result.GateResults, *genTestResult)
		result.Error = fmt.Sprintf("GenTest failed: %v", genTestResult.Error)
		return result, nil
	}
	genTestResult.Duration = workflow.Now(ctx).Sub(gateStart)
	result.GateResults = append(result.GateResults, *genTestResult)
	result.FilesChanged = append(result.FilesChanged, genTestResult.AgentResults[0].FilesChanged...)

	// GATE 2: LintTest - Lint Test Files
	logger.Info("Gate 2: LintTest")
	gateStart = workflow.Now(ctx)

	var lintTestResult *GateResult
	err = workflow.ExecuteActivity(ctx, enhancedActivities.ExecuteLintTest,
		bootstrap).Get(ctx, &lintTestResult)

	if err != nil || !lintTestResult.Passed {
		lintTestResult.Duration = workflow.Now(ctx).Sub(gateStart)
		result.GateResults = append(result.GateResults, *lintTestResult)
		result.Error = fmt.Sprintf("LintTest failed: %v", lintTestResult.Error)
		// Revert on lint failure
		_ = workflow.ExecuteActivity(ctx, cellActivities.RevertChanges, bootstrap).Get(ctx, nil)
		return result, nil
	}
	lintTestResult.Duration = workflow.Now(ctx).Sub(gateStart)
	result.GateResults = append(result.GateResults, *lintTestResult)

	// GATE 3: VerifyRED - Tests Must Fail
	logger.Info("Gate 3: VerifyRED")
	gateStart = workflow.Now(ctx)

	var verifyRedResult *GateResult
	err = workflow.ExecuteActivity(ctx, enhancedActivities.ExecuteVerifyRED,
		bootstrap).Get(ctx, &verifyRedResult)

	if err != nil || !verifyRedResult.Passed {
		verifyRedResult.Duration = workflow.Now(ctx).Sub(gateStart)
		result.GateResults = append(result.GateResults, *verifyRedResult)
		result.Error = fmt.Sprintf("VerifyRED failed: tests should fail but passed: %v", verifyRedResult.Error)
		_ = workflow.ExecuteActivity(ctx, cellActivities.RevertChanges, bootstrap).Get(ctx, nil)
		return result, nil
	}
	verifyRedResult.Duration = workflow.Now(ctx).Sub(gateStart)
	result.GateResults = append(result.GateResults, *verifyRedResult)

	// GATE 4: GenImpl - Generate Implementation
	logger.Info("Gate 4: GenImpl")
	gateStart = workflow.Now(ctx)

	var genImplResult *GateResult
	err = workflow.ExecuteActivity(ctx, enhancedActivities.ExecuteGenImpl,
		bootstrap, input.TaskID, input.Description, input.AcceptanceCriteria, "").Get(ctx, &genImplResult)

	if err != nil || !genImplResult.Passed {
		genImplResult.Duration = workflow.Now(ctx).Sub(gateStart)
		result.GateResults = append(result.GateResults, *genImplResult)
		result.Error = fmt.Sprintf("GenImpl failed: %v", genImplResult.Error)
		_ = workflow.ExecuteActivity(ctx, cellActivities.RevertChanges, bootstrap).Get(ctx, nil)
		return result, nil
	}
	genImplResult.Duration = workflow.Now(ctx).Sub(gateStart)
	result.GateResults = append(result.GateResults, *genImplResult)
	result.FilesChanged = append(result.FilesChanged, genImplResult.AgentResults[0].FilesChanged...)

	// GATE 5: VerifyGREEN - Tests Must Pass
	logger.Info("Gate 5: VerifyGREEN")
	gateStart = workflow.Now(ctx)

	var verifyGreenResult *GateResult
	err = workflow.ExecuteActivity(ctx, enhancedActivities.ExecuteVerifyGREEN,
		bootstrap).Get(ctx, &verifyGreenResult)

	if err != nil || !verifyGreenResult.Passed {
		verifyGreenResult.Duration = workflow.Now(ctx).Sub(gateStart)
		result.GateResults = append(result.GateResults, *verifyGreenResult)
		result.Error = fmt.Sprintf("VerifyGREEN failed: tests did not pass: %v", verifyGreenResult.Error)
		_ = workflow.ExecuteActivity(ctx, cellActivities.RevertChanges, bootstrap).Get(ctx, nil)
		return result, nil
	}
	verifyGreenResult.Duration = workflow.Now(ctx).Sub(gateStart)
	result.GateResults = append(result.GateResults, *verifyGreenResult)

	// GATE 6: MultiReview - 3 Reviewers with Unanimous Approval
	logger.Info("Gate 6: MultiReview")
	gateStart = workflow.Now(ctx)

	reviewersCount := input.ReviewersCount
	if reviewersCount == 0 {
		reviewersCount = 3 // Default: 3 reviewers
	}

	var multiReviewResult *GateResult
	err = workflow.ExecuteActivity(ctx, enhancedActivities.ExecuteMultiReview,
		bootstrap, input.TaskID, input.Description, reviewersCount).Get(ctx, &multiReviewResult)

	if err != nil || !multiReviewResult.Passed {
		multiReviewResult.Duration = workflow.Now(ctx).Sub(gateStart)
		result.GateResults = append(result.GateResults, *multiReviewResult)
		result.Error = fmt.Sprintf("MultiReview failed: not unanimous approval: %v", multiReviewResult.Error)
		_ = workflow.ExecuteActivity(ctx, cellActivities.RevertChanges, bootstrap).Get(ctx, nil)
		return result, nil
	}
	multiReviewResult.Duration = workflow.Now(ctx).Sub(gateStart)
	result.GateResults = append(result.GateResults, *multiReviewResult)

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
