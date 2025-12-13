// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package temporal

import (
	"fmt"
	"regexp"
	"strings"
)

// Minimum number of matched groups in regex for error location detection
const MinErrorLocationMatches = 3

// Maximum error message length before truncation
const MaxErrorMessageLength = 500

// TestFailure represents a single test failure
type TestFailure struct {
	// TestName is the name of the failed test (e.g., "TestFoo")
	TestName string

	// Package is the Go package where the test failed
	Package string

	// ErrorMessage contains the failure message/output
	ErrorMessage string

	// IsPanic indicates if this was a panic
	IsPanic bool

	// FileName is the file where the failure occurred (if available)
	FileName string

	// LineNumber is the line number where the failure occurred (if available)
	LineNumber string
}

// TestParseResult represents the outcome of parsing test output
type TestParseResult struct {
	// Failures contains all detected test failures
	Failures []TestFailure

	// HasFailures indicates if any failures were found
	HasFailures bool

	// TotalTests is the total number of tests run (if detectable)
	TotalTests int

	// PassedTests is the number of tests that passed (if detectable)
	PassedTests int

	// FailedTests is the number of tests that failed
	FailedTests int

	// RawFailureOutput contains the raw failure lines (no PASS, no timing)
	RawFailureOutput string
}

// TestParser provides utilities for parsing Go test output
type TestParser struct{}

// NewTestParser creates a new TestParser instance
func NewTestParser() *TestParser {
	return &TestParser{}
}

// ParseTestOutput parses Go test output and extracts only failures
// Handles:
// - FAIL lines
// - Error messages
// - Panics
// Excludes:
// - PASS lines
// - Timing information (ok/FAIL with durations)
func (p *TestParser) ParseTestOutput(rawOutput string) *TestParseResult {
	result := &TestParseResult{
		Failures:         []TestFailure{},
		HasFailures:      false,
		RawFailureOutput: "",
	}

	lines := strings.Split(rawOutput, "\n")
	var failureLines []string
	var currentFailureIdx = -1
	var inErrorBlock bool

	// Regex patterns
	failLineRegex := regexp.MustCompile(`^FAIL\s+([^\s]+)`)                // FAIL package
	testFailRegex := regexp.MustCompile(`^\s*---\s*FAIL:\s*([^\s]+)`)      // --- FAIL: TestName
	panicRegex := regexp.MustCompile(`^panic:`)                            // panic: message (at start of line)
	errorLocationRegex := regexp.MustCompile(`^\s*([^:]+):(\d+):\s*(.*)$`) // file.go:123: message
	okPassRegex := regexp.MustCompile(`^(ok|PASS)\s+`)                     // ok/PASS lines (skip)
	passTestRegex := regexp.MustCompile(`^\s*---\s*PASS:`)                 // --- PASS: lines (skip)
	timingRegex := regexp.MustCompile(`\s+\d+\.\d+s$`)                     // lines ending in timing
	coverageRegex := regexp.MustCompile(`coverage:`)                       // coverage: lines (skip)
	buildFailRegex := regexp.MustCompile(`^#\s+([^\s]+)`)                  // # package [build failed]
	runLineRegex := regexp.MustCompile(`^=== RUN\s+`)                      // === RUN lines (skip in failure output)
	contLineRegex := regexp.MustCompile(`^=== CONT\s+`)                    // === CONT lines (skip in failure output)

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if inErrorBlock {
				inErrorBlock = false
			}
			continue
		}

		// Skip PASS and ok lines
		if okPassRegex.MatchString(line) || passTestRegex.MatchString(line) {
			inErrorBlock = false
			continue
		}

		// Skip coverage lines
		if coverageRegex.MatchString(line) {
			continue
		}

		// Skip RUN and CONT lines from failure output
		if runLineRegex.MatchString(line) || contLineRegex.MatchString(line) {
			continue
		}

		// Detect build failures (e.g., "# package [build failed]")
		if matches := buildFailRegex.FindStringSubmatch(line); len(matches) > 1 {
			result.Failures = append(result.Failures, TestFailure{
				TestName:     "BUILD",
				Package:      matches[1],
				ErrorMessage: "",
				IsPanic:      false,
			})
			currentFailureIdx = len(result.Failures) - 1
			inErrorBlock = true
			failureLines = append(failureLines, line)
			continue
		}

		// Detect "FAIL package" lines
		if matches := failLineRegex.FindStringSubmatch(line); len(matches) > 1 {
			// Don't include timing lines in raw output
			if !timingRegex.MatchString(line) {
				failureLines = append(failureLines, line)
			}
			result.FailedTests++
			continue
		}

		// Detect "--- FAIL: TestName" lines
		if matches := testFailRegex.FindStringSubmatch(line); len(matches) > 1 {
			// If current failure is a panic with Unknown name, update it instead of creating new
			if currentFailureIdx >= 0 && result.Failures[currentFailureIdx].TestName == "Unknown" &&
				result.Failures[currentFailureIdx].IsPanic {
				result.Failures[currentFailureIdx].TestName = matches[1]
			} else {
				// Start new failure
				result.Failures = append(result.Failures, TestFailure{
					TestName:     matches[1],
					ErrorMessage: "",
					IsPanic:      false,
				})
				currentFailureIdx = len(result.Failures) - 1
			}
			inErrorBlock = true

			// Don't include timing in raw output
			if !timingRegex.MatchString(line) {
				failureLines = append(failureLines, line)
			}
			continue
		}

		// Detect panic
		if panicRegex.MatchString(line) {
			// If no failure exists yet, create one for the panic
			if currentFailureIdx < 0 {
				result.Failures = append(result.Failures, TestFailure{
					TestName:     "Unknown",
					ErrorMessage: "",
					IsPanic:      true,
				})
				currentFailureIdx = len(result.Failures) - 1
			} else {
				result.Failures[currentFailureIdx].IsPanic = true
			}
			failureLines = append(failureLines, line)
			inErrorBlock = true
			continue
		}

		// Detect error location (file:line: message)
		if matches := errorLocationRegex.FindStringSubmatch(line); len(matches) > MinErrorLocationMatches {
			if currentFailureIdx >= 0 {
				result.Failures[currentFailureIdx].FileName = matches[1]
				result.Failures[currentFailureIdx].LineNumber = matches[2]
				// Capture the message after the line number
				if matches[3] != "" {
					if result.Failures[currentFailureIdx].ErrorMessage != "" {
						result.Failures[currentFailureIdx].ErrorMessage += "\n"
					}
					result.Failures[currentFailureIdx].ErrorMessage += matches[3]
				}
			}
			failureLines = append(failureLines, line)
			inErrorBlock = true
			continue
		}

		// If we're in an error block, collect the error message
		if inErrorBlock {
			// Skip lines that are just timing info
			if !timingRegex.MatchString(line) {
				failureLines = append(failureLines, line)
				if currentFailureIdx >= 0 {
					if result.Failures[currentFailureIdx].ErrorMessage != "" {
						result.Failures[currentFailureIdx].ErrorMessage += "\n"
					}
					result.Failures[currentFailureIdx].ErrorMessage += line
				}
			}
		} else {
			// Check if this looks like an error message (indented or starts with error indicators)
			if strings.HasPrefix(line, "\t") || strings.HasPrefix(line, "    ") ||
				strings.HasPrefix(trimmed, "Error:") ||
				strings.HasPrefix(trimmed, "expected:") || strings.HasPrefix(trimmed, "got:") {
				failureLines = append(failureLines, line)
				inErrorBlock = true
				if currentFailureIdx >= 0 {
					if result.Failures[currentFailureIdx].ErrorMessage != "" {
						result.Failures[currentFailureIdx].ErrorMessage += "\n"
					}
					result.Failures[currentFailureIdx].ErrorMessage += line
				}
			}
		}

		// Look ahead to detect the end of test output
		if i+1 < len(lines) {
			nextLine := strings.TrimSpace(lines[i+1])
			if okPassRegex.MatchString(nextLine) || passTestRegex.MatchString(nextLine) ||
				failLineRegex.MatchString(nextLine) {
				inErrorBlock = false
			}
		}
	}

	// Trim error messages
	for i := range result.Failures {
		result.Failures[i].ErrorMessage = strings.TrimSpace(result.Failures[i].ErrorMessage)
	}

	result.HasFailures = len(result.Failures) > 0 || result.FailedTests > 0
	result.RawFailureOutput = strings.Join(failureLines, "\n")

	return result
}

// GetFailureSummary returns a concise summary of failures suitable for feedback
func (p *TestParser) GetFailureSummary(result *TestParseResult) string {
	if !result.HasFailures {
		return "All tests passed"
	}

	var summary strings.Builder
	summary.WriteString("Test Failures:\n")

	for _, failure := range result.Failures {
		p.addFailureToSummary(&summary, failure)
	}

	if result.FailedTests > 0 {
		summary.WriteString(fmt.Sprintf("\nTotal failed: %d\n", result.FailedTests))
	}

	return summary.String()
}

// addFailureToSummary formats and adds a single failure to the summary
func (p *TestParser) addFailureToSummary(summary *strings.Builder, failure TestFailure) {
	if failure.TestName == "BUILD" {
		p.addBuildFailure(summary, failure)
	} else {
		p.addTestFailure(summary, failure)
	}
}

// addBuildFailure formats a build failure message
func (p *TestParser) addBuildFailure(summary *strings.Builder, failure TestFailure) {
	summary.WriteString(fmt.Sprintf("\n❌ Build failed in package: %s\n", failure.Package))
}

// addTestFailure formats a test failure message with location and error details
func (p *TestParser) addTestFailure(summary *strings.Builder, failure TestFailure) {
	summary.WriteString(fmt.Sprintf("\n❌ %s", failure.TestName))
	if failure.Package != "" {
		summary.WriteString(fmt.Sprintf(" (package: %s)", failure.Package))
	}
	if failure.IsPanic {
		summary.WriteString(" [PANIC]")
	}
	summary.WriteString("\n")

	p.addFailureLocation(summary, failure)
	p.addFailureError(summary, failure)
}

// addFailureLocation adds the file and line number info to the summary
func (p *TestParser) addFailureLocation(summary *strings.Builder, failure TestFailure) {
	if failure.FileName != "" && failure.LineNumber != "" {
		summary.WriteString(fmt.Sprintf("   Location: %s:%s\n", failure.FileName, failure.LineNumber))
	}
}

// addFailureError adds the error message (truncated if necessary) to the summary
func (p *TestParser) addFailureError(summary *strings.Builder, failure TestFailure) {
	if failure.ErrorMessage != "" {
		errMsg := failure.ErrorMessage
		if len(errMsg) > MaxErrorMessageLength {
			errMsg = errMsg[:MaxErrorMessageLength] + "..."
		}
		summary.WriteString(fmt.Sprintf("   %s\n", errMsg))
	}
}

// GetRawFailures returns only the failure output without PASS lines or timing
// This is useful for feeding back to agents for retry attempts
func (p *TestParser) GetRawFailures(result *TestParseResult) string {
	return result.RawFailureOutput
}
