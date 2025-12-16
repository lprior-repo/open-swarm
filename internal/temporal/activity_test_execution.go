// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package temporal

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/log"
)

// TestExecutionOptions configures test execution parameters
type TestExecutionOptions struct {
	// Pattern specifies which tests to run (e.g., "./...", "./pkg/...", "specific_test")
	Pattern string `json:"pattern"`

	// Timeout is the maximum duration in seconds for test execution
	Timeout int64 `json:"timeout"`

	// RaceCheck enables Go race detector (-race flag)
	RaceCheck bool `json:"race_check"`

	// Coverage enables test coverage reporting (-coverprofile flag)
	Coverage bool `json:"coverage"`

	// Verbose enables verbose test output (-v flag)
	Verbose bool `json:"verbose"`

	// ShortMode enables short mode to skip long-running tests (-short flag)
	ShortMode bool `json:"short_mode"`

	// MaxRetries defines maximum retry attempts on timeout (0 = no retries)
	MaxRetries int `json:"max_retries"`

	// FailFast stops execution after first test failure (-failfast flag)
	FailFast bool `json:"fail_fast"`
}

// TestExecutionActivity wraps test execution with proper activity semantics
type TestExecutionActivity struct{}

// NewTestExecutionActivity creates a new test execution activity
func NewTestExecutionActivity() *TestExecutionActivity {
	return &TestExecutionActivity{}
}

// ExecuteTests runs Go tests with specified options and returns structured results
// This activity:
// - Executes `go test` with configurable options
// - Captures stdout and stderr separately
// - Handles timeouts with retry support
// - Parses test results using TestParser
// - Records activity heartbeats for long-running tests
// - Returns structured TestResult for workflow consumption
//
// Parameters:
//   - ctx: Temporal activity context
//   - opts: Test execution options (pattern, timeout, flags)
//
// Returns:
//   - *TestResult: Structured test results with pass/fail counts
//   - error: If test execution fails (not if tests fail, only if runner fails)
//
// Example usage in workflow:
//
//	result, err := activities.ExecuteTests(ctx, &TestExecutionOptions{
//		Pattern:    "./...",
//		Timeout:    30,
//		RaceCheck:  true,
//		Coverage:   true,
//		Verbose:    true,
//	})
func (tea *TestExecutionActivity) ExecuteTests(ctx context.Context, opts *TestExecutionOptions) (*TestResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Starting test execution",
		"pattern", opts.Pattern,
		"timeout", opts.Timeout,
		"race", opts.RaceCheck,
		"coverage", opts.Coverage,
	)

	startTime := time.Now()

	// Validate options
	if opts == nil {
		return nil, fmt.Errorf("test execution options cannot be nil")
	}
	if opts.Pattern == "" {
		return nil, fmt.Errorf("test pattern cannot be empty")
	}
	if opts.Timeout <= 0 {
		return nil, fmt.Errorf("timeout must be positive")
	}

	// Initialize retry logic
	var result *TestResult
	var err error
	attempt := 0
	maxAttempts := opts.MaxRetries + 1

	for attempt < maxAttempts {
		activity.RecordHeartbeat(ctx, "executing", "attempt", attempt+1)

		if attempt > 0 {
			logger.Info("Retrying test execution", "attempt", attempt+1, "of", maxAttempts)
			time.Sleep(time.Second) // Brief delay before retry
		}

		result, err = tea.runTests(ctx, opts, logger, startTime)

		// If execution succeeded or max retries reached, break
		if err == nil || attempt == maxAttempts-1 {
			break
		}

		// Check if error is timeout-related
		if strings.Contains(err.Error(), "context deadline exceeded") || strings.Contains(err.Error(), "timeout") {
			attempt++
			continue
		}

		// Non-timeout errors shouldn't be retried
		break
	}

	if err != nil {
		logger.Error("Test execution failed", "error", err, "attempt", attempt+1)
		return &TestResult{
			Passed:       false,
			TotalTests:   0,
			PassedTests:  0,
			FailedTests:  0,
			Output:       fmt.Sprintf("Test execution error: %v", err),
			Duration:     time.Since(startTime),
			FailureTests: []string{},
		}, fmt.Errorf("test execution failed: %w", err)
	}

	logger.Info("Test execution completed",
		"passed", result.Passed,
		"totalTests", result.TotalTests,
		"passedTests", result.PassedTests,
		"failedTests", result.FailedTests,
		"duration", result.Duration,
	)

	return result, nil
}

// runTests performs the actual test execution
func (tea *TestExecutionActivity) runTests(ctx context.Context, opts *TestExecutionOptions, logger log.Logger, startTime time.Time) (*TestResult, error) {
	// Build command arguments
	args := []string{"test"}

	// Add flags
	if opts.Verbose {
		args = append(args, "-v")
	}
	if opts.RaceCheck {
		args = append(args, "-race")
	}
	if opts.Coverage {
		args = append(args, "-coverprofile=coverage.out")
	}
	if opts.ShortMode {
		args = append(args, "-short")
	}
	if opts.FailFast {
		args = append(args, "-failfast")
	}

	// Add timeout (convert seconds to Duration)
	timeoutStr := fmt.Sprintf("%ds", opts.Timeout)
	args = append(args, "-timeout", timeoutStr)

	// Add test pattern
	args = append(args, opts.Pattern)

	logger.Info("Executing command", "cmd", "go", "args", args)

	// Create command with timeout context
	ctx, cancel := context.WithTimeout(ctx, time.Duration(opts.Timeout)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", args...)

	// Capture stdout and stderr separately
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute command
	err := cmd.Run()

	// Get output (combined for TestResult)
	output := stdout.String()
	if len(stderr.String()) > 0 {
		if output != "" {
			output += "\n--- STDERR ---\n"
		}
		output += stderr.String()
	}

	duration := time.Since(startTime)

	// If context was cancelled due to timeout, report it
	if ctx.Err() == context.DeadlineExceeded {
		logger.Error("Test execution timeout", "timeout", opts.Timeout)
		return &TestResult{
			Passed:       false,
			TotalTests:   0,
			PassedTests:  0,
			FailedTests:  0,
			Output:       fmt.Sprintf("Test execution timeout after %d seconds\n%s", opts.Timeout, output),
			Duration:     duration,
			FailureTests: []string{},
		}, fmt.Errorf("test execution timeout: deadline exceeded")
	}

	// Parse test output
	parser := NewTestParser()
	parseResult := parser.ParseTestOutput(output)

	// Check exit code (exit code 0 = all tests passed, non-zero = failures)
	testsPassed := err == nil && !parseResult.HasFailures

	// Extract failed test names
	failedNames := make([]string, len(parseResult.Failures))
	for i, failure := range parseResult.Failures {
		failedNames[i] = failure.TestName
	}

	return &TestResult{
		Passed:       testsPassed,
		TotalTests:   parseResult.TotalTests,
		PassedTests:  parseResult.PassedTests,
		FailedTests:  parseResult.FailedTests,
		Output:       output,
		Duration:     duration,
		FailureTests: failedNames,
	}, nil
}

// ExecuteTestsInDir runs tests in a specific directory
// Useful for testing specific packages or ensuring correct working directory
func (tea *TestExecutionActivity) ExecuteTestsInDir(ctx context.Context, dir string, opts *TestExecutionOptions) (*TestResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Starting test execution in directory",
		"dir", dir,
		"pattern", opts.Pattern,
	)

	// Build command with absolute directory handling
	args := []string{"test"}

	if opts.Verbose {
		args = append(args, "-v")
	}
	if opts.RaceCheck {
		args = append(args, "-race")
	}
	if opts.Coverage {
		args = append(args, "-coverprofile=coverage.out")
	}
	if opts.ShortMode {
		args = append(args, "-short")
	}
	if opts.FailFast {
		args = append(args, "-failfast")
	}

	timeoutStr := fmt.Sprintf("%ds", opts.Timeout)
	args = append(args, "-timeout", timeoutStr)
	args = append(args, opts.Pattern)

	ctx, cancel := context.WithTimeout(ctx, time.Duration(opts.Timeout)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	startTime := time.Now()
	err := cmd.Run()
	duration := time.Since(startTime)

	output := stdout.String()
	if len(stderr.String()) > 0 {
		if output != "" {
			output += "\n--- STDERR ---\n"
		}
		output += stderr.String()
	}

	parser := NewTestParser()
	parseResult := parser.ParseTestOutput(output)

	testsPassed := err == nil && !parseResult.HasFailures

	failedNames := make([]string, len(parseResult.Failures))
	for i, failure := range parseResult.Failures {
		failedNames[i] = failure.TestName
	}

	return &TestResult{
		Passed:       testsPassed,
		TotalTests:   parseResult.TotalTests,
		PassedTests:  parseResult.PassedTests,
		FailedTests:  parseResult.FailedTests,
		Output:       output,
		Duration:     duration,
		FailureTests: failedNames,
	}, nil
}

// ExecuteSpecificTest runs a single test by name
// Useful for debugging or re-running specific failing tests
func (tea *TestExecutionActivity) ExecuteSpecificTest(ctx context.Context, testName string, opts *TestExecutionOptions) (*TestResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Starting specific test execution", "testName", testName)

	// Modify pattern to run specific test
	modifiedOpts := *opts
	modifiedOpts.Pattern = fmt.Sprintf("-run %s %s", testName, opts.Pattern)

	return tea.ExecuteTests(ctx, &modifiedOpts)
}

// TestResultSummary returns a human-readable summary of test results
// Useful for workflow logging and reporting
func (tr *TestResult) Summary() string {
	var summary strings.Builder

	if tr.Passed {
		summary.WriteString(fmt.Sprintf("PASSED: All %d tests passed", tr.TotalTests))
	} else {
		summary.WriteString(fmt.Sprintf("FAILED: %d/%d tests failed",
			tr.FailedTests, tr.TotalTests))
	}

	summary.WriteString(fmt.Sprintf(" (Duration: %.2fs)", tr.Duration.Seconds()))

	if len(tr.FailureTests) > 0 && len(tr.FailureTests) <= 5 {
		summary.WriteString(fmt.Sprintf("\nFailed tests: %s",
			strings.Join(tr.FailureTests, ", ")))
	} else if len(tr.FailureTests) > 5 {
		summary.WriteString(fmt.Sprintf("\nFailed tests (%d): %s, ...",
			len(tr.FailureTests),
			strings.Join(tr.FailureTests[:5], ", ")))
	}

	return summary.String()
}
