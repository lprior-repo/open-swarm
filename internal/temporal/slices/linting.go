// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

// Package slices provides vertical slice architecture for Open Swarm workflows.
//
// linting.go: Complete vertical slice for linting and code quality validation
// - Linting execution via SDK
// - Output parsing and issue extraction
// - Gate validation for workflow progression
//
// This slice follows CUPID principles:
// - Composable: Self-contained linting operations
// - Unix philosophy: Does linting and validation, nothing else
// - Predictable: Clear pass/fail with structured issue list
// - Idiomatic: Go linting conventions, Temporal patterns
// - Domain-centric: Organized around code quality capability
package slices

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/activity"

	"open-swarm/internal/agent"
)

// ============================================================================
// ACTIVITIES
// ============================================================================

// LintingActivities handles all linting and code quality operations
type LintingActivities struct {
	// No external dependencies - uses SDK client from bootstrap output
}

// NewLintingActivities creates a new linting activities instance
func NewLintingActivities() *LintingActivities {
	return &LintingActivities{}
}

// RunLint executes linting checks in a cell and returns parsed results
//
// This activity:
// 1. Reconstructs SDK client from bootstrap output
// 2. Executes linting via SDK (golangci-lint or make lint)
// 3. Parses output to extract issues
// 4. Returns structured LintResult
//
// Used in both test and implementation phases to ensure code quality.
func (l *LintingActivities) RunLint(ctx context.Context, output BootstrapOutput) (*LintResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Running linting checks", "cellID", output.CellID)

	activity.RecordHeartbeat(ctx, "executing linting")

	startTime := time.Now()

	// Reconstruct SDK client
	client := ReconstructClient(output)

	// Execute linting via SDK
	result, err := client.ExecutePrompt(ctx, "Run linting checks: make lint", &agent.PromptOptions{
		Agent: "build",
	})
	if err != nil {
		return &LintResult{
			Passed:   false,
			Output:   err.Error(),
			Duration: time.Since(startTime),
		}, fmt.Errorf("failed to execute linting in cell %q: %w", output.CellID, err)
	}

	duration := time.Since(startTime)

	// Parse linting output
	lintResult := parseLintOutput(result.GetText())
	lintResult.Duration = duration

	logger.Info("Linting completed",
		"cellID", output.CellID,
		"passed", lintResult.Passed,
		"issues", len(lintResult.Issues),
		"duration", duration)

	return lintResult, nil
}

// ValidateLintGate checks if linting passes the quality gate
//
// This activity executes linting and validates against quality standards:
// - No errors allowed
// - Warnings may be acceptable (based on policy)
//
// Returns GateResult for workflow progression decision.
func (l *LintingActivities) ValidateLintGate(ctx context.Context, output BootstrapOutput) (*GateResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Validating lint gate", "cellID", output.CellID)

	startTime := time.Now()

	lintResult, err := l.RunLint(ctx, output)
	if err != nil {
		return &GateResult{
			GateName:      "lint_validation",
			Passed:        false,
			Duration:      time.Since(startTime),
			Error:         err.Error(),
			LintResult:    lintResult,
			RetryAttempts: 0,
		}, err
	}

	// Quality gate: linting must pass (no errors)
	if !lintResult.Passed {
		errorCount := countLintErrors(lintResult.Issues)
		return &GateResult{
			GateName:   "lint_validation",
			Passed:     false,
			Duration:   time.Since(startTime),
			Message:    fmt.Sprintf("Linting failed: %d errors found", errorCount),
			LintResult: lintResult,
		}, fmt.Errorf("linting gate failed: %d errors", errorCount)
	}

	logger.Info("Lint gate passed", "cellID", output.CellID)

	return &GateResult{
		GateName:   "lint_validation",
		Passed:     true,
		Duration:   time.Since(startTime),
		Message:    "All linting checks passed",
		LintResult: lintResult,
	}, nil
}

// RunLintWithRetry executes linting with retry budget for transient failures
//
// Some linting failures may be fixed by the agent (auto-fix).
// This activity retries linting up to maxRetries times.
func (l *LintingActivities) RunLintWithRetry(ctx context.Context, output BootstrapOutput, maxRetries int) (*GateResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Running linting with retry", "cellID", output.CellID, "maxRetries", maxRetries)

	var lastLintResult *LintResult
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			logger.Info("Retrying linting", "cellID", output.CellID, "attempt", attempt)
			activity.RecordHeartbeat(ctx, fmt.Sprintf("retry attempt %d/%d", attempt, maxRetries))
		}

		startTime := time.Now()
		lintResult, err := l.RunLint(ctx, output)
		duration := time.Since(startTime)

		if err != nil {
			lastErr = err
			lastLintResult = lintResult
			continue
		}

		if lintResult.Passed {
			// Success - linting passed
			return &GateResult{
				GateName:      "lint_validation",
				Passed:        true,
				Duration:      duration,
				LintResult:    lintResult,
				RetryAttempts: attempt,
			}, nil
		}

		// Linting failed - save result for potential retry
		lastLintResult = lintResult
		errorCount := countLintErrors(lintResult.Issues)
		lastErr = fmt.Errorf("%d linting errors", errorCount)
	}

	// All retries exhausted
	logger.Warn("Linting failed after retries", "cellID", output.CellID, "attempts", maxRetries+1)

	return &GateResult{
		GateName:      "lint_validation",
		Passed:        false,
		Duration:      0, // Total duration not tracked across retries
		Error:         lastErr.Error(),
		LintResult:    lastLintResult,
		RetryAttempts: maxRetries,
	}, lastErr
}

// ============================================================================
// BUSINESS LOGIC
// ============================================================================

// parseLintOutput parses golangci-lint output into structured LintResult
//
// Parses output from golangci-lint to extract:
// - Pass/fail status
// - Individual issues with location and severity
// - Error/warning counts
//
// This is a simplified parser. Production code should handle various formats.
func parseLintOutput(output string) *LintResult {
	// Simple heuristic parsing
	// golangci-lint format: file:line:column: message (linter)

	passed := true
	var issues []LintIssue

	lines := splitLines(output)

	for _, line := range lines {
		// Skip empty lines
		if len(line) == 0 {
			continue
		}

		// Check for issue lines (contains ": ")
		if containsString(line, ": ") {
			issue := parseLintLine(line)
			if issue != nil {
				issues = append(issues, *issue)
				if issue.Severity == "error" {
					passed = false
				}
			}
		}

		// Check for overall failure indicators
		if containsString(line, "FAIL") || containsString(line, "error") {
			passed = false
		}
	}

	// If no issues found and output contains "OK", assume passing
	if len(issues) == 0 && containsString(output, "OK") {
		passed = true
	}

	return &LintResult{
		Passed: passed,
		Issues: issues,
		Output: output,
	}
}

// parseLintLine parses a single lint issue line
// Format: file:line:column: message (linter)
func parseLintLine(line string) *LintIssue {
	// Simple parsing - look for "file:line:column: message"
	parts := splitString(line, ":")
	if len(parts) < 4 {
		return nil
	}

	// Extract file
	file := parts[0]

	// Extract line number (parse as int)
	lineNum := parseIntSimple(parts[1])

	// Extract column number
	colNum := parseIntSimple(parts[2])

	// Rest is message
	message := joinStrings(parts[3:], ":")

	// Determine severity (simple heuristic)
	severity := "warning"
	if containsString(message, "error") || containsString(message, "fatal") {
		severity = "error"
	}

	// Extract rule (text in parentheses)
	rule := extractRule(message)

	return &LintIssue{
		File:     file,
		Line:     lineNum,
		Column:   colNum,
		Severity: severity,
		Message:  message,
		Rule:     rule,
	}
}

// countLintErrors counts errors in lint issues
func countLintErrors(issues []LintIssue) int {
	count := 0
	for _, issue := range issues {
		if issue.Severity == "error" {
			count++
		}
	}
	return count
}

// ============================================================================
// HELPERS
// ============================================================================

// splitString splits a string by delimiter
func splitString(s, delim string) []string {
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
			i = i + delimLen - 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}

// joinStrings joins string slice with delimiter
func joinStrings(parts []string, delim string) string {
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += delim + parts[i]
	}
	return result
}

// parseIntSimple parses an integer from string (simple version)
func parseIntSimple(s string) int {
	// Trim whitespace
	s = trimWhitespace(s)
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

// trimWhitespace removes leading/trailing whitespace
//
//nolint:cyclop // complexity 11 is acceptable for character-level processing
func trimWhitespace(s string) string {
	start := 0
	end := len(s)

	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}

	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}

	return s[start:end]
}

// extractRule extracts linter rule from message (text in parentheses)
func extractRule(message string) string {
	// Find last occurrence of '('
	start := -1
	for i := len(message) - 1; i >= 0; i-- {
		if message[i] == '(' {
			start = i
			break
		}
	}
	if start == -1 {
		return ""
	}

	// Find closing ')'
	end := -1
	for i := start + 1; i < len(message); i++ {
		if message[i] == ')' {
			end = i
			break
		}
	}
	if end == -1 {
		return ""
	}

	return message[start+1 : end]
}
