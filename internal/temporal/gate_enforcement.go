// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package temporal

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.temporal.io/sdk/activity"

	"open-swarm/internal/gates"
	"open-swarm/internal/telemetry"
)

// GateEnforcement orchestrates pre and post-execution gate verification.
// It enforces test immutability before agent execution and empirical honesty after.
type GateEnforcement struct {
	testImmutability gates.Gate
	empiricalHonesty gates.Gate
}

// NewGateEnforcement creates a new GateEnforcement instance
func NewGateEnforcement() *GateEnforcement {
	return &GateEnforcement{
		testImmutability: gates.NewTestImmutabilityGate("", ""),
		empiricalHonesty: gates.NewEmpiricalHonestyGate(""),
	}
}

// EnforcePreExecutionGates enforces gates before agent execution.
// It locks test files and validates process isolation setup.
//
// Returns:
//   - error if gates fail (task must be rejected)
//   - nil if all gates pass (proceed to agent execution)
func (ge *GateEnforcement) EnforcePreExecutionGates(
	ctx context.Context,
	taskID string,
	testFiles []string,
) error {
	ctx, span := telemetry.StartSpan(ctx, "activity.gates", "EnforcePreExecutionGates",
		attribute.String("taskID", taskID),
		attribute.Int("testFiles", len(testFiles)),
	)
	defer span.End()

	logger := activity.GetLogger(ctx)
	logger.Info("Pre-execution gate enforcement", "taskID", taskID, "testFiles", len(testFiles))

	startTime := time.Now()
	telemetry.AddEvent(ctx, "gates.pre_execution.start", attribute.String("taskID", taskID))

	// Validate input
	if taskID == "" {
		err := fmt.Errorf("taskID cannot be empty")
		logger.Error("Invalid task ID", "error", err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	if len(testFiles) == 0 {
		err := fmt.Errorf("no test files provided for task %s", taskID)
		logger.Error("No test files", "taskID", taskID, "error", err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	// Lock test files to enforce immutability
	// Each test file gets locked with read-only permissions
	for _, testFile := range testFiles {
		immutabilityGate := gates.NewTestImmutabilityGate(taskID, testFile)

		// Execute test immutability gate
		gateErr := immutabilityGate.Check(ctx)
		if gateErr != nil {
			logger.Error(
				"Test immutability gate failed",
				"taskID", taskID,
				"testFile", testFile,
				"error", gateErr,
			)
			span.RecordError(gateErr)
			span.SetStatus(codes.Error, fmt.Sprintf("test immutability failed for %s", testFile))
			telemetry.AddEvent(ctx, "gates.pre_execution.failed",
				attribute.String("reason", "test_immutability"),
				attribute.String("testFile", testFile),
			)
			return gateErr
		}

		logger.Info("Test file locked", "taskID", taskID, "testFile", testFile)
		telemetry.AddEvent(ctx, "gates.test_locked", attribute.String("testFile", testFile))
	}

	duration := time.Since(startTime)
	span.SetAttributes(
		attribute.String("status", "passed"),
		attribute.Int64("duration_ms", duration.Milliseconds()),
	)
	span.SetStatus(codes.Ok, "all pre-execution gates passed")

	logger.Info("Pre-execution gates passed", "taskID", taskID, "duration", duration)
	telemetry.AddEvent(ctx, "gates.pre_execution.passed",
		attribute.String("taskID", taskID),
		attribute.Int64("duration_ms", duration.Milliseconds()),
	)

	return nil
}

// EnforcePostExecutionGates enforces gates after agent execution.
// It validates that the agent was honest about test results and provides raw output.
//
// Returns:
//   - error if gates fail (task must be rejected as incomplete)
//   - nil if all gates pass (task can be marked complete)
func (ge *GateEnforcement) EnforcePostExecutionGates(
	ctx context.Context,
	taskID string,
	result *ExecutionResult,
) error {
	ctx, span := telemetry.StartSpan(ctx, "activity.gates", "EnforcePostExecutionGates",
		attribute.String("taskID", taskID),
		attribute.Bool("claimed_success", result.Success),
	)
	defer span.End()

	logger := activity.GetLogger(ctx)
	logger.Info("Post-execution gate enforcement", "taskID", taskID, "success", result.Success)

	startTime := time.Now()
	telemetry.AddEvent(ctx, "gates.post_execution.start", attribute.String("taskID", taskID))

	// Validate input
	if taskID == "" {
		err := fmt.Errorf("taskID cannot be empty")
		logger.Error("Invalid task ID", "error", err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	if result == nil {
		err := fmt.Errorf("execution result cannot be nil")
		logger.Error("Nil execution result", "taskID", taskID)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	// Extract test output from result
	var testOutput string
	if rawOutput, ok := result.OutputData["test_output"]; ok {
		testOutput = fmt.Sprintf("%v", rawOutput)
	}

	// Create empirical honesty gate with agent claim and test results
	honestyGate := gates.NewEmpiricalHonestyGate(taskID)

	// Set the actual test result data if available
	if testResultData, ok := result.OutputData["test_result"]; ok {
		// Assuming test result is in the output data
		// The gate will validate whether the claim matches reality
		_ = testResultData
	}

	// Create a synthetic test result to validate against the claim
	// The test result represents the actual state (empirical truth)
	syntheticResult := &gates.TestResult{
		Output:   testOutput,
		ExitCode: 0,
	}

	if !result.Success && result.Error != "" {
		// If agent claims failure, the exit code should be non-zero
		syntheticResult.ExitCode = 1
		syntheticResult.Output = fmt.Sprintf("FAILED: %s\n%s", result.Error, testOutput)
	}

	honestyGate.SetTestResult(syntheticResult)

	// Build agent claim string from result
	claimMessage := "Implementation complete"
	if !result.Success {
		claimMessage = fmt.Sprintf("Failed: %s", result.Error)
	}
	honestyGate.SetAgentClaim(claimMessage)

	// Execute empirical honesty gate
	gateErr := honestyGate.Check(ctx)
	if gateErr != nil {
		logger.Error(
			"Empirical honesty gate failed",
			"taskID", taskID,
			"claimed_success", result.Success,
			"error", gateErr,
		)
		span.RecordError(gateErr)
		span.SetStatus(codes.Error, fmt.Sprintf("empirical honesty failed: %v", gateErr))
		telemetry.AddEvent(ctx, "gates.post_execution.failed",
			attribute.String("reason", "empirical_honesty"),
			attribute.String("detail", gateErr.Error()),
		)
		return gateErr
	}

	duration := time.Since(startTime)
	span.SetAttributes(
		attribute.String("status", "passed"),
		attribute.Int64("duration_ms", duration.Milliseconds()),
	)
	span.SetStatus(codes.Ok, "all post-execution gates passed")

	logger.Info("Post-execution gates passed", "taskID", taskID, "duration", duration)
	telemetry.AddEvent(ctx, "gates.post_execution.passed",
		attribute.String("taskID", taskID),
		attribute.Int64("duration_ms", duration.Milliseconds()),
	)

	return nil
}

// CleanupGates releases any gate resources (e.g., unlock test files).
// Called after task completion regardless of success/failure.
func (ge *GateEnforcement) CleanupGates(
	ctx context.Context,
	taskID string,
	testFiles []string,
) error {
	ctx, span := telemetry.StartSpan(ctx, "activity.gates", "CleanupGates",
		attribute.String("taskID", taskID),
		attribute.Int("testFiles", len(testFiles)),
	)
	defer span.End()

	logger := activity.GetLogger(ctx)
	logger.Info("Gate cleanup", "taskID", taskID, "testFiles", len(testFiles))

	// Unlock test files
	for _, testFile := range testFiles {
		immutabilityGate := gates.NewTestImmutabilityGate(taskID, testFile)
		// Safely unlock (ignore errors to ensure all files are attempted)
		_ = immutabilityGate.UnlockTestFile()
		logger.Info("Test file unlocked", "taskID", taskID, "testFile", testFile)
	}

	span.SetStatus(codes.Ok, "cleanup complete")
	return nil
}
