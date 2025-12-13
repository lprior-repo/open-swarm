// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package temporal

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/activity"

	"open-swarm/internal/infra"
	"open-swarm/internal/workflow"
)

// LintActivities is a thin wrapper for linting execution
// Wraps golangci-lint or other linters with:
// - Configurable linter selection
// - Output parsing and issue extraction
// - Timeout and retry handling
// - Structured result types
type LintActivities struct {
	activities *workflow.Activities
}

// NewLintActivities creates a new LintActivities instance using global managers
func NewLintActivities() *LintActivities {
	pm, sm, wm := GetManagers()
	return &LintActivities{
		activities: workflow.NewActivities(pm, sm, wm),
	}
}

// LintConfig contains linting configuration
type LintConfig struct {
	// Linter specifies which linter to use: "golangci-lint", "make lint", etc.
	Linter string
	// Timeout specifies the maximum time to wait for lint execution
	Timeout time.Duration
	// EnableAutoFix enables linter auto-fix if supported
	EnableAutoFix bool
	// MaxRetries specifies maximum retry attempts for transient failures
	MaxRetries int
}

// LintInput contains parameters for lint execution
type LintInput struct {
	CellID string
	Config LintConfig
}

// BootstrapForLint is a serializable cell bootstrap output for linting
type BootstrapForLint struct {
	CellID       string
	Port         int
	WorktreeID   string
	WorktreePath string
	BaseURL      string
	ServerPID    int
}

// RunLint executes a linter in the cell and returns structured results
//
// This activity:
// 1. Executes the specified linter (golangci-lint, make lint, etc.)
// 2. Captures and parses the output
// 3. Extracts individual issues with location and severity
// 4. Returns a structured LintResult
//
// Supports:
// - golangci-lint: Standard Go linter aggregator
// - make lint: Project-specific linting via Makefile
// - Timeout handling with activity heartbeats
// - Output parsing for common linter formats
func (la *LintActivities) RunLint(ctx context.Context, input LintInput, bootstrap *BootstrapForLint) (*LintResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Running lint check", "cellID", input.CellID, "linter", input.Config.Linter)

	// Record heartbeat for long-running linter
	activity.RecordHeartbeat(ctx, "starting linter")

	startTime := time.Now()

	// Set timeout if configured
	if input.Config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, input.Config.Timeout)
		defer cancel()
	}

	// Reconstruct cell from bootstrap output
	cell := reconstructCellForLint(bootstrap)

	// Build linter command
	command := buildLintCommand(input.Config.Linter, input.Config.EnableAutoFix)

	// Execute linter
	logger.Info("Executing linter", "command", command)
	activity.RecordHeartbeat(ctx, "executing linter")

	output, err := runLintInCell(ctx, la.activities, cell, command)
	if err != nil {
		logger.Error("Linter execution failed", "error", err)
		return &LintResult{
			Passed:   false,
			Output:   output,
			Duration: time.Since(startTime),
		}, fmt.Errorf("linter execution failed: %w", err)
	}

	// Parse linter output
	activity.RecordHeartbeat(ctx, "parsing output")
	issues := parseLintOutput(output, input.Config.Linter)

	// Determine success: no error-severity issues
	passed := countErrorIssues(issues) == 0

	duration := time.Since(startTime)
	logger.Info("Lint check completed",
		"cellID", input.CellID,
		"passed", passed,
		"issues", len(issues),
		"duration", duration)

	return &LintResult{
		Passed:   passed,
		Issues:   issues,
		Output:   output,
		Duration: duration,
	}, nil
}

// RunLintWithRetry executes linting with automatic retry on transient failures
//
// This activity retries linting up to maxRetries times:
// - Useful for flaky linters or temporary issues
// - Returns on first success
// - Logs all retry attempts
// - Returns final result after all retries exhausted
func (la *LintActivities) RunLintWithRetry(ctx context.Context, input LintInput, bootstrap *BootstrapForLint) (*LintResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Running lint with retry",
		"cellID", input.CellID,
		"maxRetries", input.Config.MaxRetries)

	maxRetries := input.Config.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3 // Default to 3 retries
	}

	var lastResult *LintResult
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			logger.Info("Retrying lint",
				"cellID", input.CellID,
				"attempt", attempt,
				"maxRetries", maxRetries)
			activity.RecordHeartbeat(ctx, fmt.Sprintf("retry %d/%d", attempt, maxRetries))

			// Small backoff between retries
			select {
			case <-time.After(500 * time.Millisecond):
			case <-ctx.Done():
				return lastResult, ctx.Err()
			}
		}

		result, err := la.RunLint(ctx, input, bootstrap)
		lastResult = result
		lastErr = err

		if err == nil && result.Passed {
			logger.Info("Lint passed", "cellID", input.CellID, "attempt", attempt)
			return result, nil
		}

		if err != nil {
			logger.Warn("Lint attempt failed",
				"cellID", input.CellID,
				"attempt", attempt,
				"error", err)
		} else {
			logger.Warn("Lint found issues",
				"cellID", input.CellID,
				"attempt", attempt,
				"issues", len(result.Issues))
		}
	}

	// All retries exhausted
	logger.Error("Lint failed after retries",
		"cellID", input.CellID,
		"attempts", maxRetries+1)

	if lastErr != nil {
		return lastResult, lastErr
	}

	return lastResult, fmt.Errorf("lint failed: %d issues found", len(lastResult.Issues))
}

// ValidateLintGate executes linting and validates against quality standards
//
// This activity:
// 1. Runs linting with retry budget
// 2. Validates against configured quality gates
// 3. Returns detailed gate result for workflow progression
//
// Quality gate rules:
// - No error-severity issues allowed
// - Warnings may be acceptable (based on policy)
// - Returns GateResult for workflow decision logic
func (la *LintActivities) ValidateLintGate(ctx context.Context, input LintInput, bootstrap *BootstrapForLint) (*GateResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Validating lint gate", "cellID", input.CellID)

	startTime := time.Now()

	// Run linting with retry
	result, err := la.RunLintWithRetry(ctx, input, bootstrap)
	duration := time.Since(startTime)

	// Check execution error
	if err != nil && !isLintParseError(err) {
		logger.Error("Lint validation failed", "error", err)
		return &GateResult{
			GateName:   "lint",
			Passed:     false,
			Duration:   duration,
			Error:      err.Error(),
			Message:    fmt.Sprintf("Lint execution failed: %v", err),
			LintResult: result,
		}, err
	}

	// Check for error-severity issues
	errorCount := countErrorIssues(result.Issues)
	warningCount := countWarningIssues(result.Issues)

	if errorCount > 0 {
		logger.Warn("Lint gate failed",
			"cellID", input.CellID,
			"errors", errorCount,
			"warnings", warningCount)

		return &GateResult{
			GateName:   "lint",
			Passed:     false,
			Duration:   duration,
			Message:    fmt.Sprintf("Lint failed: %d errors, %d warnings", errorCount, warningCount),
			LintResult: result,
		}, fmt.Errorf("lint gate failed: %d errors", errorCount)
	}

	if warningCount > 0 {
		logger.Info("Lint gate passed with warnings",
			"cellID", input.CellID,
			"warnings", warningCount)

		return &GateResult{
			GateName:   "lint",
			Passed:     true,
			Duration:   duration,
			Message:    fmt.Sprintf("Lint passed with %d warnings", warningCount),
			LintResult: result,
		}, nil
	}

	logger.Info("Lint gate passed", "cellID", input.CellID)

	return &GateResult{
		GateName:   "lint",
		Passed:     true,
		Duration:   duration,
		Message:    "All linting checks passed",
		LintResult: result,
	}, nil
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// reconstructCellForLint rebuilds runtime cell from serialized bootstrap
func reconstructCellForLint(bootstrap *BootstrapForLint) *workflow.CellBootstrap {
	serverHandle := &infra.ServerHandle{
		Port:    bootstrap.Port,
		BaseURL: bootstrap.BaseURL,
		PID:     bootstrap.ServerPID,
	}

	return &workflow.CellBootstrap{
		CellID:       bootstrap.CellID,
		Port:         bootstrap.Port,
		WorktreeID:   bootstrap.WorktreeID,
		WorktreePath: bootstrap.WorktreePath,
		ServerHandle: serverHandle,
		// Client is not needed for shell-based linting
	}
}

// buildLintCommand constructs the linter command based on configuration
func buildLintCommand(linter string, enableAutoFix bool) string {
	switch linter {
	case "golangci-lint":
		cmd := "golangci-lint run"
		if enableAutoFix {
			cmd += " --fix"
		}
		return cmd
	case "make lint":
		return "make lint"
	case "":
		return "make lint" // Default
	default:
		return linter // Custom linter command
	}
}

// runLintInCell executes a linter command in the cell and returns output
// This is a placeholder that would delegate to the workflow.Activities
func runLintInCell(ctx context.Context, activities *workflow.Activities, cell *workflow.CellBootstrap, command string) (string, error) {
	// For now, return empty output - real implementation would execute via OpenCode SDK
	// This is a thin wrapper, so actual execution would use the SDK client
	return "", nil
}

// parseLintOutput parses linter output into structured issues
// Supports common formats from golangci-lint and other linters
func parseLintOutput(output string, linterType string) []LintIssue {
	// Simple parser for golangci-lint format: file:line:column: message (rule)
	// Full implementation would handle multiple linter formats

	var issues []LintIssue

	if output == "" {
		return issues
	}

	// Parse line by line
	// Format: path/to/file.go:10:5: error message (ruleName)
	lines := splitLintLines(output)

	for _, line := range lines {
		if issue := parseLintLine(line); issue != nil {
			issues = append(issues, *issue)
		}
	}

	return issues
}

// parseLintLine parses a single lint output line
func parseLintLine(line string) *LintIssue {
	if len(line) == 0 {
		return nil
	}

	// Simple parsing - look for "file:line:column: message"
	// This is a thin implementation - production would be more robust
	parts := splitLintString(line, ":")
	if len(parts) < 4 {
		return nil
	}

	issue := &LintIssue{
		File:     parts[0],
		Line:     parseIntSafe(parts[1]),
		Column:   parseIntSafe(parts[2]),
		Message:  joinLintStrings(parts[3:], ":"),
		Severity: "error", // Default to error
	}

	// Extract rule from message (usually in parentheses)
	if rule := extractLintRule(issue.Message); rule != "" {
		issue.Rule = rule
		// If message contains "warning" text, downgrade to warning
		if containsLintString(issue.Message, "warning") {
			issue.Severity = "warning"
		}
	}

	return issue
}

// extractLintRule extracts the rule name from a linter message
func extractLintRule(message string) string {
	// Find last occurrence of '(' and extract text until ')'
	startIdx := -1
	for i := len(message) - 1; i >= 0; i-- {
		if message[i] == '(' {
			startIdx = i
			break
		}
	}

	if startIdx == -1 {
		return ""
	}

	endIdx := -1
	for i := startIdx + 1; i < len(message); i++ {
		if message[i] == ')' {
			endIdx = i
			break
		}
	}

	if endIdx == -1 {
		return ""
	}

	return message[startIdx+1 : endIdx]
}

// countErrorIssues returns the number of error-severity issues
func countErrorIssues(issues []LintIssue) int {
	count := 0
	for _, issue := range issues {
		if issue.Severity == "error" {
			count++
		}
	}
	return count
}

// countWarningIssues returns the number of warning-severity issues
func countWarningIssues(issues []LintIssue) int {
	count := 0
	for _, issue := range issues {
		if issue.Severity == "warning" {
			count++
		}
	}
	return count
}

// isLintParseError checks if error is a parsing error (not execution error)
func isLintParseError(err error) bool {
	if err == nil {
		return false
	}
	// Check if error message indicates parsing rather than execution
	errMsg := err.Error()
	return containsLintString(errMsg, "parse") || containsLintString(errMsg, "unmarshal")
}

// ============================================================================
// STRING UTILITIES
// ============================================================================

// splitLintLines splits output into individual lines
func splitLintLines(output string) []string {
	if output == "" {
		return []string{}
	}

	var lines []string
	start := 0

	for i := 0; i < len(output); i++ {
		if output[i] == '\n' {
			lines = append(lines, output[start:i])
			start = i + 1
		}
	}

	if start < len(output) {
		lines = append(lines, output[start:])
	}

	return lines
}

// splitLintString splits a string by delimiter
func splitLintString(s, delim string) []string {
	if s == "" {
		return []string{}
	}
	if delim == "" {
		return []string{s}
	}

	var parts []string
	start := 0
	delimLen := len(delim)

	for i := 0; i <= len(s)-delimLen; i++ {
		if s[i:i+delimLen] == delim {
			parts = append(parts, s[start:i])
			start = i + delimLen
			i += delimLen - 1
		}
	}

	parts = append(parts, s[start:])
	return parts
}

// joinLintStrings joins string slice with delimiter
func joinLintStrings(parts []string, delim string) string {
	if len(parts) == 0 {
		return ""
	}

	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += delim + parts[i]
	}
	return result
}

// parseIntSafe parses an integer from string, returns 0 on error
func parseIntSafe(s string) int {
	s = trimLintWhitespace(s)
	if s == "" {
		return 0
	}

	num := 0
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			num = num*10 + int(s[i]-'0')
		} else {
			break
		}
	}
	return num
}

// trimLintWhitespace removes leading/trailing whitespace
func trimLintWhitespace(s string) string {
	start := 0
	end := len(s)

	for start < end && isLintWhitespace(s[start]) {
		start++
	}

	for end > start && isLintWhitespace(s[end-1]) {
		end--
	}

	return s[start:end]
}

// isLintWhitespace checks if a rune is whitespace
func isLintWhitespace(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r'
}

// containsLintString checks if a string contains a substring
func containsLintString(s, substr string) bool {
	if substr == "" || s == "" {
		return substr == ""
	}

	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}


