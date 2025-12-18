// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package temporal

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.temporal.io/sdk/activity"

	"open-swarm/internal/agent"
	"open-swarm/internal/filelock"
	"open-swarm/internal/telemetry"
	"open-swarm/internal/workflow"
)

// EnhancedActivities contains activities for Enhanced TCR workflow
type EnhancedActivities struct {
	lockRegistry *filelock.MemoryRegistry
}

// NewEnhancedActivities creates a new EnhancedActivities instance
func NewEnhancedActivities() *EnhancedActivities {
	return &EnhancedActivities{
		lockRegistry: GetFileLockRegistry(),
	}
}

// newFailedGateResult creates a GateResult for failed gate execution.
// Consolidates the common error result pattern across all gate functions.
func newFailedGateResult(gateName string, err error, startTime time.Time) *GateResult {
	return &GateResult{
		GateName: gateName,
		Passed:   false,
		Error:    err.Error(),
		Duration: time.Since(startTime),
	}
}

// getChangedFiles extracts file paths from agent file status.
// Returns a slice of non-empty file paths that were modified.
func getChangedFiles(ctx context.Context, cell *workflow.CellBootstrap) []string {
	fileStatus, _ := cell.Client.GetFileStatus(ctx)
	filesChanged := make([]string, 0, len(fileStatus))
	for _, file := range fileStatus {
		if file.Path != "" {
			filesChanged = append(filesChanged, file.Path)
		}
	}
	return filesChanged
}

// AcquireFileLocks acquires locks on task-related files
// Returns list of locked file patterns
func (ea *EnhancedActivities) AcquireFileLocks(ctx context.Context, cellID string, taskID string) ([]string, error) {
	ctx, span := telemetry.StartSpan(ctx, "activity.enhanced", "AcquireFileLocks",
		trace.WithAttributes(telemetry.TCRAttrs("", taskID)...),
	)
	defer span.End()

	logger := activity.GetLogger(ctx)
	logger.Info("Acquiring file locks", "cellID", cellID, "taskID", taskID)

	// Define file patterns to lock (task-specific patterns)
	// In a real implementation, these would be derived from task analysis
	patterns := []string{
		fmt.Sprintf("**/%s_test.go", taskID),
		fmt.Sprintf("**/%s.go", taskID),
		"**/test/**",
	}

	lockedPatterns := []string{}
	ttl := 15 * time.Minute // Lock TTL

	for _, pattern := range patterns {
		req := filelock.LockRequest{
			Path:      pattern,
			Holder:    cellID,
			Exclusive: true,
			TTL:       ttl,
		}
		result, err := ea.lockRegistry.Acquire(req)
		if err != nil || !result.Granted {
			// Rollback: release already acquired locks
			releaseLocks(ea.lockRegistry, cellID, lockedPatterns)
			return nil, fmt.Errorf("failed to acquire lock on %s: %w", pattern, err)
		}
		lockedPatterns = append(lockedPatterns, pattern)
		logger.Info("Lock acquired", "pattern", pattern, "holder", cellID)
		telemetry.AddEvent(ctx, "lock.acquired", attribute.String("pattern", pattern))
	}

	span.SetAttributes(attribute.Int("locks.acquired", len(lockedPatterns)))
	span.SetStatus(codes.Ok, "all locks acquired")
	return lockedPatterns, nil
}

// ReleaseFileLocks releases all locks held by a cell
func (ea *EnhancedActivities) ReleaseFileLocks(ctx context.Context, cellID string, patterns []string) error {
	ctx, span := telemetry.StartSpan(ctx, "activity.enhanced", "ReleaseFileLocks",
		trace.WithAttributes(attribute.Int("locks.count", len(patterns))),
	)
	defer span.End()

	logger := activity.GetLogger(ctx)
	logger.Info("Releasing file locks", "cellID", cellID, "count", len(patterns))

	var errs []error
	for _, pattern := range patterns {
		err := ea.lockRegistry.Release(pattern, cellID)
		if err != nil {
			logger.Warn("Failed to release lock", "pattern", pattern, "error", err)
			errs = append(errs, err)
		} else {
			logger.Info("Lock released", "pattern", pattern)
		}
	}

	if len(errs) > 0 {
		span.SetStatus(codes.Error, "failed to release some locks")
		return fmt.Errorf("failed to release some locks: %v", errs)
	}
	span.SetStatus(codes.Ok, "all locks released")
	return nil
}

// ExecuteGenTest - Gate 1: Generate test files based on acceptance criteria
func (ea *EnhancedActivities) ExecuteGenTest(ctx context.Context, bootstrap *BootstrapOutput, taskID string, acceptanceCriteria string) (*GateResult, error) {
	ctx, span := telemetry.StartSpan(ctx, "activity.enhanced", "ExecuteGenTest",
		trace.WithAttributes(telemetry.TCRAttrs("", taskID)...),
	)
	defer span.End()

	logger := activity.GetLogger(ctx)
	logger.Info("Gate: GenTest", "taskID", taskID)

	startTime := time.Now()
	telemetry.AddEvent(ctx, "gate.start", telemetry.AttrGateName.String("gen_test"))
	cellActivities := NewCellActivities()
	cell := cellActivities.reconstructCell(bootstrap)

	// Execute agent to generate tests
	prompt := fmt.Sprintf(`Generate comprehensive tests for task: %s

Acceptance Criteria:
%s

Requirements:
- Write tests in Go using testing package
- Cover all edge cases and error conditions
- Follow TDD principles - tests should fail initially
- Use table-driven tests where appropriate`, taskID, acceptanceCriteria)

	result, err := cell.Client.ExecutePrompt(ctx, prompt, &agent.PromptOptions{
		Title: fmt.Sprintf("GenTest: %s", taskID),
		Agent: "test-generator",
		Model: "github-copilot/claude-haiku-4.5",
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "test generation failed")
		telemetry.AddEvent(ctx, "gate.failed", telemetry.AttrGateName.String("gen_test"))
		return newFailedGateResult("gen_test", err, startTime), err
	}

	filesChanged := getChangedFiles(ctx, cell)

	span.SetAttributes(
		telemetry.AttrGateName.String("gen_test"),
		telemetry.AttrGatePassed.Bool(true),
		attribute.Int("files.changed", len(filesChanged)),
	)
	telemetry.AddEvent(ctx, "gate.passed",
		telemetry.AttrGateName.String("gen_test"),
		attribute.Int("files.changed", len(filesChanged)),
	)
	span.SetStatus(codes.Ok, "test generation completed")

	return &GateResult{
		GateName: "gen_test",
		Passed:   true,
		AgentResults: []AgentResult{
			{
				AgentName:    "test-generator",
				Model:        "github-copilot/claude-haiku-4.5",
				Prompt:       prompt,
				Response:     result.GetText(),
				Success:      true,
				Duration:     time.Since(startTime),
				FilesChanged: filesChanged,
			},
		},
		Duration: time.Since(startTime),
	}, nil
}

// ExecuteLintTest - Gate 2: Lint test files
func (ea *EnhancedActivities) ExecuteLintTest(ctx context.Context, bootstrap *BootstrapOutput) (*GateResult, error) {
	ctx, span := telemetry.StartSpan(ctx, "activity.enhanced", "ExecuteLintTest")
	defer span.End()

	logger := activity.GetLogger(ctx)
	logger.Info("Gate: LintTest")

	startTime := time.Now()
	telemetry.AddEvent(ctx, "gate.start", telemetry.AttrGateName.String("lint_test"))
	cellActivities := NewCellActivities()
	cell := cellActivities.reconstructCell(bootstrap)

	// Run golangci-lint on test files
	result, _ := cell.Client.ExecuteCommand(ctx, "", "shell", []string{"golangci-lint", "run", "--disable-all", "--enable=errcheck,staticcheck,unused", "*_test.go"})

	output := ""
	if result != nil {
		output = result.GetText()
	}

	// Parse lint output using LintParser
	parser := NewLintParser()
	parseResult := parser.ParseGolangciLint(output)

	span.SetAttributes(
		telemetry.AttrGateName.String("lint_test"),
		telemetry.AttrGatePassed.Bool(!parseResult.HasErrors),
		attribute.Int("lint.issues", len(parseResult.Issues)),
	)

	if parseResult.HasErrors {
		span.SetStatus(codes.Error, "lint check failed")
		telemetry.AddEvent(ctx, "gate.failed", telemetry.AttrGateName.String("lint_test"))
	} else {
		span.SetStatus(codes.Ok, "lint check passed")
		telemetry.AddEvent(ctx, "gate.passed", telemetry.AttrGateName.String("lint_test"))
	}

	return &GateResult{
		GateName: "lint_test",
		Passed:   !parseResult.HasErrors,
		LintResult: &LintResult{
			Passed:   !parseResult.HasErrors,
			Output:   output,
			Duration: time.Since(startTime),
			Issues:   parseResult.Issues,
		},
		Duration: time.Since(startTime),
		Message:  parseResult.Summary,
	}, nil
}

// ExecuteVerifyRED - Gate 3: Verify tests fail (RED phase)
func (ea *EnhancedActivities) ExecuteVerifyRED(ctx context.Context, bootstrap *BootstrapOutput) (*GateResult, error) {
	ctx, span := telemetry.StartSpan(ctx, "activity.enhanced", "ExecuteVerifyRED")
	defer span.End()

	logger := activity.GetLogger(ctx)
	logger.Info("Gate: VerifyRED")

	startTime := time.Now()
	telemetry.AddEvent(ctx, "gate.start", telemetry.AttrGateName.String("verify_red"))
	cellActivities := NewCellActivities()
	cell := cellActivities.reconstructCell(bootstrap)

	// Run tests - they SHOULD fail
	result, _ := cell.Client.ExecuteCommand(ctx, "", "shell", []string{"go", "test", "-v", "./..."})
	output := ""
	if result != nil {
		output = result.GetText()
	}

	// Parse test output using TestParser
	parser := NewTestParser()
	parseResult := parser.ParseTestOutput(output)

	// Tests should FAIL - if they pass, that's an error
	testsFailed := parseResult.HasFailures
	redPassed := testsFailed // RED means tests failed as expected

	// Build debug info with parsed failure information
	debugInfo := ""
	if parseResult.HasFailures {
		debugInfo = parser.GetFailureSummary(parseResult)
	}

	span.SetAttributes(
		telemetry.AttrGateName.String("verify_red"),
		telemetry.AttrGatePassed.Bool(redPassed),
		telemetry.AttrTestsPassed.Int(parseResult.PassedTests),
		telemetry.AttrTestsFailed.Int(parseResult.FailedTests),
	)

	if redPassed {
		span.SetStatus(codes.Ok, "RED phase verified - tests failed as expected")
		telemetry.AddEvent(ctx, "gate.passed", telemetry.AttrGateName.String("verify_red"))
	} else {
		span.SetStatus(codes.Error, "RED phase failed - tests passed but should fail")
		telemetry.AddEvent(ctx, "gate.failed", telemetry.AttrGateName.String("verify_red"))
	}

	return &GateResult{
		GateName: "verify_red",
		Passed:   redPassed,
		TestResult: &TestResult{
			Passed:      !testsFailed, // Inverted: we want false here
			Output:      output,
			Duration:    time.Since(startTime),
			TotalTests:  parseResult.TotalTests,
			FailedTests: parseResult.FailedTests,
		},
		Duration: time.Since(startTime),
		Error: func() string {
			if !redPassed {
				return "tests passed but should fail (not RED)"
			} else {
				return ""
			}
		}(),
		Message: debugInfo,
	}, nil
}

// ExecuteGenImpl - Gate 4: Generate implementation
// testFailureOutput: Optional. If provided (non-empty), includes test failure feedback for retry attempts.
//
// When retrying after VerifyGREEN failures, the workflow should:
//  1. Extract test output from verifyGreenResult.TestResult.Output
//  2. Pass it as testFailureOutput parameter
//  3. The agent will receive parsed, structured failure information
//
// Example retry workflow:
//
//	if !verifyGreenResult.Passed {
//	  testOutput := verifyGreenResult.TestResult.Output
//	  genImplResult, _ := ExecuteGenImpl(ctx, bootstrap, taskID, desc, criteria, testOutput)
//	}
func (ea *EnhancedActivities) ExecuteGenImpl(ctx context.Context, bootstrap *BootstrapOutput, taskID string, description string, acceptanceCriteria string, testFailureOutput string) (*GateResult, error) {
	ctx, span := telemetry.StartSpan(ctx, "activity.enhanced", "ExecuteGenImpl",
		trace.WithAttributes(telemetry.TCRAttrs("", taskID)...),
	)
	defer span.End()

	logger := activity.GetLogger(ctx)
	logger.Info("Gate: GenImpl", "taskID", taskID)

	startTime := time.Now()
	telemetry.AddEvent(ctx, "gate.start", telemetry.AttrGateName.String("gen_impl"))

	isRetry := testFailureOutput != ""
	if isRetry {
		span.SetAttributes(attribute.Bool("retry", true))
		telemetry.AddEvent(ctx, "impl.retry", attribute.String("reason", "test_failures"))
	}
	cellActivities := NewCellActivities()
	cell := cellActivities.reconstructCell(bootstrap)

	// Build base prompt
	var promptBuilder strings.Builder

	// If retry feedback is provided, include parsed test failures
	if testFailureOutput != "" {
		parser := NewTestParser()
		parseResult := parser.ParseTestOutput(testFailureOutput)

		if parseResult.HasFailures {
			failureSummary := parser.GetFailureSummary(parseResult)
			promptBuilder.WriteString("Previous implementation attempt failed with test failures:\n\n")
			promptBuilder.WriteString(failureSummary)
			promptBuilder.WriteString("\n\nPlease fix the implementation to address these failures.\n\n")
			logger.Info("GenImpl retry with test failure feedback", "failures", len(parseResult.Failures))
		}
	}

	promptBuilder.WriteString(fmt.Sprintf(`Implement the solution for task: %s

Description: %s

Acceptance Criteria:
%s

Requirements:
- Implement code to make all tests pass
- Follow Go best practices and idioms
- Include proper error handling
- Add documentation comments`, taskID, description, acceptanceCriteria))

	prompt := promptBuilder.String()

	result, err := cell.Client.ExecutePrompt(ctx, prompt, &agent.PromptOptions{
		Title: fmt.Sprintf("GenImpl: %s", taskID),
		Agent: "implementation",
		Model: "github-copilot/claude-haiku-4.5",
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "implementation generation failed")
		telemetry.AddEvent(ctx, "gate.failed", telemetry.AttrGateName.String("gen_impl"))
		return newFailedGateResult("gen_impl", err, startTime), err
	}

	filesChanged := getChangedFiles(ctx, cell)

	span.SetAttributes(
		telemetry.AttrGateName.String("gen_impl"),
		telemetry.AttrGatePassed.Bool(true),
		attribute.Int("files.changed", len(filesChanged)),
	)
	telemetry.AddEvent(ctx, "gate.passed",
		telemetry.AttrGateName.String("gen_impl"),
		attribute.Int("files.changed", len(filesChanged)),
	)
	span.SetStatus(codes.Ok, "implementation generation completed")

	return &GateResult{
		GateName: "gen_impl",
		Passed:   true,
		AgentResults: []AgentResult{
			{
				AgentName:    "implementation",
				Model:        "github-copilot/claude-haiku-4.5",
				Prompt:       prompt,
				Response:     result.GetText(),
				Success:      true,
				Duration:     time.Since(startTime),
				FilesChanged: filesChanged,
			},
		},
		Duration: time.Since(startTime),
	}, nil
}

// ExecuteVerifyGREEN - Gate 5: Verify tests pass (GREEN phase)
func (ea *EnhancedActivities) ExecuteVerifyGREEN(ctx context.Context, bootstrap *BootstrapOutput) (*GateResult, error) {
	ctx, span := telemetry.StartSpan(ctx, "activity.enhanced", "ExecuteVerifyGREEN")
	defer span.End()

	logger := activity.GetLogger(ctx)
	logger.Info("Gate: VerifyGREEN")

	startTime := time.Now()
	telemetry.AddEvent(ctx, "gate.start", telemetry.AttrGateName.String("verify_green"))
	cellActivities := NewCellActivities()
	cell := cellActivities.reconstructCell(bootstrap)

	// Run tests - they SHOULD pass
	result, err := cell.Client.ExecuteCommand(ctx, "", "shell", []string{"go", "test", "-v", "./..."})
	output := ""
	if result != nil {
		output = result.GetText()
	}

	// Parse test output using TestParser
	parser := NewTestParser()
	parseResult := parser.ParseTestOutput(output)

	// GREEN means all tests pass - no failures allowed
	testsPassed := err == nil && !parseResult.HasFailures

	// Extract failed test names for detailed reporting
	failedTestNames := make([]string, 0, len(parseResult.Failures))
	for _, failure := range parseResult.Failures {
		failedTestNames = append(failedTestNames, failure.TestName)
	}

	span.SetAttributes(
		telemetry.AttrGateName.String("verify_green"),
		telemetry.AttrGatePassed.Bool(testsPassed),
		telemetry.AttrTestsPassed.Int(parseResult.PassedTests),
		telemetry.AttrTestsFailed.Int(parseResult.FailedTests),
	)

	if testsPassed {
		span.SetStatus(codes.Ok, "GREEN phase verified - all tests passed")
		telemetry.AddEvent(ctx, "gate.passed", telemetry.AttrGateName.String("verify_green"))
	} else {
		span.SetStatus(codes.Error, "GREEN phase failed - tests failed")
		telemetry.AddEvent(ctx, "gate.failed",
			telemetry.AttrGateName.String("verify_green"),
			attribute.Int("failed_tests", len(failedTestNames)),
		)
	}

	return &GateResult{
		GateName: "verify_green",
		Passed:   testsPassed,
		TestResult: &TestResult{
			Passed:       testsPassed,
			TotalTests:   parseResult.TotalTests,
			PassedTests:  parseResult.PassedTests,
			FailedTests:  parseResult.FailedTests,
			Output:       output,
			Duration:     time.Since(startTime),
			FailureTests: failedTestNames,
		},
		Duration: time.Since(startTime),
		Error: func() string {
			if !testsPassed {
				if len(parseResult.Failures) > 0 {
					return fmt.Sprintf("tests failed (not GREEN): %d failure(s) detected", len(parseResult.Failures))
				}
				return "tests failed (not GREEN)"
			}
			return ""
		}(),
	}, nil
}

// ExecuteMultiReview - Gate 6: Multi-reviewer approval (3 reviewers, unanimous)
func (ea *EnhancedActivities) ExecuteMultiReview(ctx context.Context, bootstrap *BootstrapOutput, taskID string, description string, reviewersCount int) (*GateResult, error) {
	ctx, span := telemetry.StartSpan(ctx, "activity.enhanced", "ExecuteMultiReview",
		trace.WithAttributes(telemetry.TCRAttrs("", taskID)...),
	)
	defer span.End()

	logger := activity.GetLogger(ctx)
	logger.Info("Gate: MultiReview", "reviewers", reviewersCount)

	startTime := time.Now()
	telemetry.AddEvent(ctx, "gate.start",
		telemetry.AttrGateName.String("multi_review"),
		attribute.Int("reviewers.count", reviewersCount),
	)
	cellActivities := NewCellActivities()
	cell := cellActivities.reconstructCell(bootstrap)

	reviewTypes := []ReviewType{ReviewTypeTesting, ReviewTypeFunctional, ReviewTypeArchitecture}
	votes := []ReviewVote{}
	voteParser := NewVoteParser()

	for i := 0; i < reviewersCount; i++ {
		reviewType := reviewTypes[i%len(reviewTypes)]
		reviewStart := time.Now()

		prompt := fmt.Sprintf(`Review the code changes for task: %s

Description: %s

Review Focus: %s

Provide:
1. Vote: APPROVE, REQUEST_CHANGE, or REJECT
2. Detailed feedback on the implementation
3. Specific issues or improvements needed

Your review should focus on: %s`, taskID, description, reviewType, getReviewFocus(reviewType))

		result, err := cell.Client.ExecutePrompt(ctx, prompt, &agent.PromptOptions{
			Title: fmt.Sprintf("Review %d (%s): %s", i+1, reviewType, taskID),
			Agent: getReviewerAgent(reviewType),
			Model: "github-copilot/claude-haiku-4.5",
		})

		var vote VoteResult
		var feedback string

		if err != nil {
			vote = VoteReject
			feedback = fmt.Sprintf("Review failed: %v", err)
		} else {
			feedback = result.GetText()
			// Use VoteParser to extract vote
			parsed := voteParser.ParseVote(feedback)
			vote = parsed.Vote
		}

		votes = append(votes, ReviewVote{
			ReviewerName: fmt.Sprintf("reviewer-%d", i+1),
			ReviewType:   reviewType,
			Vote:         vote,
			Feedback:     feedback,
			Duration:     time.Since(reviewStart),
		})

		telemetry.AddEvent(ctx, "review.completed",
			attribute.Int("reviewer.number", i+1),
			attribute.String("review.type", string(reviewType)),
			attribute.String("vote", string(vote)),
		)
	}

	// Check for unanimous approval
	allApproved := voteParser.CheckUnanimousApproval(votes)

	// Generate aggregated feedback if not approved
	aggregator := NewReviewAggregator()
	errorMsg := ""
	if !allApproved {
		errorMsg = aggregator.GetRejectionSummary(votes)
	}

	// Count votes by type
	approvals := 0
	rejections := 0
	requestChanges := 0
	for _, vote := range votes {
		switch vote.Vote {
		case VoteApprove:
			approvals++
		case VoteReject:
			rejections++
		case VoteRequestChange:
			requestChanges++
		}
	}

	span.SetAttributes(
		telemetry.AttrGateName.String("multi_review"),
		telemetry.AttrGatePassed.Bool(allApproved),
		attribute.Int("reviews.total", len(votes)),
		attribute.Int("reviews.approved", approvals),
		attribute.Int("reviews.rejected", rejections),
		attribute.Int("reviews.request_change", requestChanges),
	)

	if allApproved {
		span.SetStatus(codes.Ok, "all reviewers approved")
		telemetry.AddEvent(ctx, "gate.passed", telemetry.AttrGateName.String("multi_review"))
	} else {
		span.SetStatus(codes.Error, "review not unanimously approved")
		telemetry.AddEvent(ctx, "gate.failed",
			telemetry.AttrGateName.String("multi_review"),
			attribute.Int("rejections", rejections),
		)
	}

	return &GateResult{
		GateName:    "multi_review",
		Passed:      allApproved,
		ReviewVotes: votes,
		Duration:    time.Since(startTime),
		Error:       errorMsg,
		Message:     aggregator.AggregateReviewFeedback(votes),
	}, nil
}

// GetCommitSHA retrieves the current commit SHA
func (ea *EnhancedActivities) GetCommitSHA(ctx context.Context, bootstrap *BootstrapOutput) (string, error) {
	ctx, span := telemetry.StartSpan(ctx, "activity.enhanced", "GetCommitSHA")
	defer span.End()

	cellActivities := NewCellActivities()
	cell := cellActivities.reconstructCell(bootstrap)

	result, err := cell.Client.ExecuteCommand(ctx, "", "shell", []string{"git", "rev-parse", "HEAD"})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get commit SHA")
		return "", fmt.Errorf("failed to get commit SHA: %w", err)
	}

	sha := strings.TrimSpace(result.GetText())
	span.SetAttributes(attribute.String("git.commit_sha", sha))
	span.SetStatus(codes.Ok, "commit SHA retrieved")
	telemetry.AddEvent(ctx, "commit.sha.retrieved", attribute.String("sha", sha))
	return sha, nil
}

// Helper function to get review focus description
func getReviewFocus(reviewType ReviewType) string {
	switch reviewType {
	case ReviewTypeTesting:
		return "test coverage, quality, edge cases, and test maintainability"
	case ReviewTypeFunctional:
		return "correctness, requirements satisfaction, and behavior"
	case ReviewTypeArchitecture:
		return "design patterns, code structure, and long-term maintainability"
	default:
		return "overall code quality"
	}
}

// Helper function to get the appropriate reviewer agent based on review type
func getReviewerAgent(reviewType ReviewType) string {
	switch reviewType {
	case ReviewTypeTesting:
		return "reviewer-testing"
	case ReviewTypeFunctional:
		return "reviewer-functional"
	case ReviewTypeArchitecture:
		return "reviewer-architecture"
	default:
		return "reviewer"
	}
}
