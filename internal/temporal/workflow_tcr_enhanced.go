// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package temporal

import (
	"fmt"
	"strings"

	"go.temporal.io/sdk/log"
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
func (ge *gateExecutor) executeGate(gateName string, activityFn interface{}, args ...interface{}) error {
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
		return fmt.Errorf("gate failed")
	}

	// Track files changed if available
	if len(gateResult.AgentResults) > 0 {
		for _, agentResult := range gateResult.AgentResults {
			ge.result.FilesChanged = append(ge.result.FilesChanged, agentResult.FilesChanged...)
		}
	}

	return nil
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

	// Set defaults
	maxRetries := input.MaxRetries
	if maxRetries == 0 {
		maxRetries = 2 // Default: 2 full regeneration attempts
	}

	maxFixAttempts := input.MaxFixAttempts
	if maxFixAttempts == 0 {
		maxFixAttempts = 5 // Default: 5 targeted fix attempts per regeneration
	}

	reviewersCount := input.ReviewersCount
	if reviewersCount == 0 {
		reviewersCount = 2 // Default: 2 reviewers (reduced from 3 for faster iteration)
	}

	// Activity options - use shared non-idempotent options
	ctx = WithNonIdempotentOptions(ctx)

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
		sagaCtx, _ := NewSagaContext(ctx)

		// Release all acquired locks
		if len(locksAcquired) > 0 {
			logger.Info("Saga: Releasing file locks", "count", len(locksAcquired))
			_ = workflow.ExecuteActivity(sagaCtx, enhancedActivities.ReleaseFileLocks,
				input.CellID, locksAcquired).Get(sagaCtx, nil)
		}

		// Teardown cell
		logger.Info("Saga: Tearing down cell", "cellID", bootstrap.CellID)
		_ = workflow.ExecuteActivity(sagaCtx, cellActivities.TeardownCell, bootstrap).Get(sagaCtx, nil)
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

	// GATES 1-3: Test Generation Phase (no retry on these - they're foundational)
	// GATE 1: GenTest - Generate Tests
	if err := executor.executeGate("GenTest", enhancedActivities.ExecuteGenTest,
		bootstrap, input.TaskID, input.AcceptanceCriteria); err != nil {
		return result, nil
	}

	// GATE 2: LintTest - Lint Test Files
	if err := executor.executeGate("LintTest", enhancedActivities.ExecuteLintTest, bootstrap); err != nil {
		return result, nil
	}

	// GATE 3: VerifyRED - Tests Must Fail
	if err := executor.executeGate("VerifyRED", enhancedActivities.ExecuteVerifyRED, bootstrap, input.TaskID); err != nil {
		return result, nil
	}

	// GATES 4-6: Implementation & Review Loop (two-tier: regeneration + targeted fixes)
	// Outer loop: Full regeneration attempts
	// Inner loop: Targeted fix attempts (preserves working code)
	var feedback string
	success := false

OuterLoop:
	for regenAttempt := 1; regenAttempt <= maxRetries; regenAttempt++ {
		logger.Info("Regeneration attempt", "attempt", regenAttempt, "maxRetries", maxRetries)

		// GATE 4: GenImpl - Generate Implementation (full generation)
		if err := executor.executeGate("GenImpl", enhancedActivities.ExecuteGenImpl,
			bootstrap, input.TaskID, input.Description, input.AcceptanceCriteria, feedback); err != nil {
			// GenImpl itself failed - don't retry, it's a fundamental issue
			return result, nil
		}

		// Inner loop: Targeted fixes after initial generation
		for fixAttempt := 1; fixAttempt <= maxFixAttempts; fixAttempt++ {
			logger.Info("Fix attempt", "regenAttempt", regenAttempt, "fixAttempt", fixAttempt, "maxFixAttempts", maxFixAttempts)

			// GATE 5: VerifyGREEN - Tests Must Pass
			var verifyGreenResult *GateResult
			logger.Info("Gate: VerifyGREEN")
			gateStart := workflow.Now(ctx)
			err := workflow.ExecuteActivity(ctx, enhancedActivities.ExecuteVerifyGREEN, bootstrap, input.TaskID).Get(ctx, &verifyGreenResult)

			if err != nil {
				verifyGreenResult = &GateResult{
					GateName: "VerifyGREEN",
					Passed:   false,
					Error:    err.Error(),
					Duration: workflow.Now(ctx).Sub(gateStart),
				}
			} else {
				verifyGreenResult.Duration = workflow.Now(ctx).Sub(gateStart)
			}
			result.GateResults = append(result.GateResults, *verifyGreenResult)

			if !verifyGreenResult.Passed { //nolint:dupl // Similar but contextually different from MultiReview handling
				// Tests failed - try targeted fix (don't revert!)
				if fixAttempt < maxFixAttempts {
					logger.Info("VerifyGREEN failed, applying targeted fix", "fixAttempt", fixAttempt)
					testFeedback := extractTestFeedback(verifyGreenResult)

					// Apply targeted fix instead of full regeneration
					var fixResult *GateResult
					err := workflow.ExecuteActivity(ctx, enhancedActivities.ExecuteFixFromFeedback,
						bootstrap, input.TaskID, testFeedback).Get(ctx, &fixResult)
					if err != nil || (fixResult != nil && !fixResult.Passed) {
						logger.Warn("Targeted fix failed", "error", err)
					}
					if fixResult != nil {
						result.GateResults = append(result.GateResults, *fixResult)
					}
					continue
				}
				// Max fix attempts reached - try full regeneration
				if regenAttempt < maxRetries {
					logger.Info("Max fix attempts reached, reverting for full regeneration")
					feedback = extractTestFeedback(verifyGreenResult)
					_ = workflow.ExecuteActivity(ctx, cellActivities.RevertChanges, bootstrap).Get(ctx, nil)
					continue OuterLoop
				}
				result.Error = fmt.Sprintf("VerifyGREEN failed after %d regen + %d fix attempts: %v",
					regenAttempt, fixAttempt, verifyGreenResult.Error)
				_ = workflow.ExecuteActivity(ctx, cellActivities.RevertChanges, bootstrap).Get(ctx, nil)
				return result, nil
			}

			// GATE 6: MultiReview - Reviewers with Unanimous Approval
			var reviewResult *GateResult
			logger.Info("Gate: MultiReview")
			gateStart = workflow.Now(ctx)
			err = workflow.ExecuteActivity(ctx, enhancedActivities.ExecuteMultiReview,
				bootstrap, input.TaskID, input.Description, reviewersCount).Get(ctx, &reviewResult)

			if err != nil {
				reviewResult = &GateResult{
					GateName: "MultiReview",
					Passed:   false,
					Error:    err.Error(),
					Duration: workflow.Now(ctx).Sub(gateStart),
				}
			} else {
				reviewResult.Duration = workflow.Now(ctx).Sub(gateStart)
			}
			result.GateResults = append(result.GateResults, *reviewResult)

			if !reviewResult.Passed { //nolint:dupl // Similar but contextually different from VerifyGREEN handling
				// Reviewers requested changes - try targeted fix (don't revert!)
				if fixAttempt < maxFixAttempts {
					logger.Info("MultiReview failed, applying targeted fix", "fixAttempt", fixAttempt)
					reviewFeedback := extractReviewerFeedback(reviewResult)

					// Apply targeted fix instead of full regeneration
					var fixResult *GateResult
					err := workflow.ExecuteActivity(ctx, enhancedActivities.ExecuteFixFromFeedback,
						bootstrap, input.TaskID, reviewFeedback).Get(ctx, &fixResult)
					if err != nil || (fixResult != nil && !fixResult.Passed) {
						logger.Warn("Targeted fix failed", "error", err)
					}
					if fixResult != nil {
						result.GateResults = append(result.GateResults, *fixResult)
					}
					continue
				}
				// Max fix attempts reached - try full regeneration
				if regenAttempt < maxRetries {
					logger.Info("Max fix attempts reached, reverting for full regeneration")
					feedback = extractReviewerFeedback(reviewResult)
					_ = workflow.ExecuteActivity(ctx, cellActivities.RevertChanges, bootstrap).Get(ctx, nil)
					continue OuterLoop
				}
				result.Error = fmt.Sprintf("MultiReview failed after %d regen + %d fix attempts: %v",
					regenAttempt, fixAttempt, reviewResult.Error)
				_ = workflow.ExecuteActivity(ctx, cellActivities.RevertChanges, bootstrap).Get(ctx, nil)
				return result, nil
			}

			// All gates passed!
			success = true
			break OuterLoop
		}
	}

	if !success {
		result.Error = "Workflow exhausted all retry attempts"
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
	logger.Info("Enhanced TCR Workflow completed successfully", "taskID", input.TaskID)

	return result, nil
}

// executeEnhancedTCRGates executes all 6 gates of the Enhanced TCR workflow in sequence
// extractTestFeedback extracts structured test failure feedback for targeted fixes
func extractTestFeedback(testResult *GateResult) string {
	if testResult == nil {
		return ""
	}

	if testResult.TestResult != nil && testResult.TestResult.Output != "" {
		// Use the TestParser to create structured feedback
		parser := NewTestParser()
		parseResult := parser.ParseTestOutput(testResult.TestResult.Output)
		return parser.GetFailureSummary(parseResult)
	}

	// Fallback to error message
	if testResult.Error != "" {
		return fmt.Sprintf("Test Error: %s", testResult.Error)
	}

	return "Tests failed (no detailed output available)"
}

func extractReviewerFeedback(reviewResult *GateResult) string {
	if reviewResult == nil {
		return ""
	}

	var feedback strings.Builder
	feedback.WriteString("Reviewer feedback from previous attempt:\n\n")

	// Extract feedback from review votes
	if len(reviewResult.ReviewVotes) > 0 {
		for _, vote := range reviewResult.ReviewVotes {
			if vote.Vote != VoteApprove {
				feedback.WriteString(fmt.Sprintf("- %s (%s): %s\n", vote.ReviewerName, vote.Vote, vote.Feedback))
			}
		}
	}

	// Include error message if present
	if reviewResult.Error != "" {
		feedback.WriteString(fmt.Sprintf("\nError: %s\n", reviewResult.Error))
	}

	// Include message if present
	if reviewResult.Message != "" {
		feedback.WriteString(fmt.Sprintf("\nDetails: %s\n", reviewResult.Message))
	}

	return feedback.String()
}
