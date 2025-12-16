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

// parseContext holds mutable state during line processing
type parseContext struct {
	result            *TestParseResult
	failureLines      []string
	currentFailureIdx int
	inErrorBlock      bool
	patterns          *regexPatterns
}

// regexPatterns caches compiled regex patterns for reuse
type regexPatterns struct {
	failLineRegex      *regexp.Regexp
	testFailRegex      *regexp.Regexp
	panicRegex         *regexp.Regexp
	errorLocationRegex *regexp.Regexp
	okPassRegex        *regexp.Regexp
	passTestRegex      *regexp.Regexp
	timingRegex        *regexp.Regexp
	coverageRegex      *regexp.Regexp
	buildFailRegex     *regexp.Regexp
	runLineRegex       *regexp.Regexp
	contLineRegex      *regexp.Regexp
}

// newPatterns creates compiled regex patterns
func newPatterns() *regexPatterns {
	return &regexPatterns{
		failLineRegex:      regexp.MustCompile(`^FAIL\s+([^\s]+)`),
		testFailRegex:      regexp.MustCompile(`^\s*---\s*FAIL:\s*([^\s]+)`),
		panicRegex:         regexp.MustCompile(`^panic:`),
		errorLocationRegex: regexp.MustCompile(`^\s*([^:]+):(\d+):\s*(.*)$`),
		okPassRegex:        regexp.MustCompile(`^(ok|PASS)\s+`),
		passTestRegex:      regexp.MustCompile(`^\s*---\s*PASS:`),
		timingRegex:        regexp.MustCompile(`\s+\d+\.\d+s$`),
		coverageRegex:      regexp.MustCompile(`coverage:`),
		buildFailRegex:     regexp.MustCompile(`^#\s+([^\s]+)`),
		runLineRegex:       regexp.MustCompile(`^=== RUN\s+`),
		contLineRegex:      regexp.MustCompile(`^=== CONT\s+`),
	}
}

// ParseTestOutput parses Go test output and extracts only failures
func (p *TestParser) ParseTestOutput(rawOutput string) *TestParseResult {
	ctx := &parseContext{
		result:            &TestParseResult{Failures: []TestFailure{}, HasFailures: false, RawFailureOutput: ""},
		failureLines:      []string{},
		currentFailureIdx: -1,
		inErrorBlock:      false,
		patterns:          newPatterns(),
	}

	lines := strings.Split(rawOutput, "\n")
	for i, line := range lines {
		p.processTestLine(ctx, line, i, lines)
	}

	p.finalizeResults(ctx)
	return ctx.result
}

// processTestLine processes a single line of test output
func (p *TestParser) processTestLine(ctx *parseContext, line string, idx int, lines []string) {
	trimmed := strings.TrimSpace(line)

	if trimmed == "" {
		ctx.inErrorBlock = false
		return
	}

	if p.shouldSkipLine(ctx.patterns, line) {
		return
	}

	if p.tryProcessBuildFailure(ctx, line) ||
		p.tryProcessFailLine(ctx, line) ||
		p.tryProcessTestFailure(ctx, line) ||
		p.tryProcessPanic(ctx, line) ||
		p.tryProcessErrorLocation(ctx, line) {
		return
	}

	p.processErrorMessage(ctx, line, trimmed, idx, lines)
}

// shouldSkipLine checks if a line should be skipped
func (p *TestParser) shouldSkipLine(patterns *regexPatterns, line string) bool {
	if patterns.okPassRegex.MatchString(line) || patterns.passTestRegex.MatchString(line) {
		return true
	}
	if patterns.coverageRegex.MatchString(line) || patterns.runLineRegex.MatchString(line) ||
		patterns.contLineRegex.MatchString(line) {
		return true
	}
	return false
}

// tryProcessBuildFailure handles build failure lines
func (p *TestParser) tryProcessBuildFailure(ctx *parseContext, line string) bool {
	matches := ctx.patterns.buildFailRegex.FindStringSubmatch(line)
	if len(matches) <= 1 {
		return false
	}
	ctx.result.Failures = append(ctx.result.Failures, TestFailure{
		TestName: "BUILD", Package: matches[1], ErrorMessage: "", IsPanic: false,
	})
	ctx.currentFailureIdx = len(ctx.result.Failures) - 1
	ctx.inErrorBlock = true
	ctx.failureLines = append(ctx.failureLines, line)
	return true
}

// tryProcessFailLine handles FAIL package lines
func (p *TestParser) tryProcessFailLine(ctx *parseContext, line string) bool {
	matches := ctx.patterns.failLineRegex.FindStringSubmatch(line)
	if len(matches) <= 1 {
		return false
	}
	if !ctx.patterns.timingRegex.MatchString(line) {
		ctx.failureLines = append(ctx.failureLines, line)
	}
	ctx.result.FailedTests++
	return true
}

// tryProcessTestFailure handles test failure lines
func (p *TestParser) tryProcessTestFailure(ctx *parseContext, line string) bool {
	matches := ctx.patterns.testFailRegex.FindStringSubmatch(line)
	if len(matches) <= 1 {
		return false
	}
	if ctx.currentFailureIdx >= 0 && ctx.result.Failures[ctx.currentFailureIdx].TestName == "Unknown" &&
		ctx.result.Failures[ctx.currentFailureIdx].IsPanic {
		ctx.result.Failures[ctx.currentFailureIdx].TestName = matches[1]
	} else {
		ctx.result.Failures = append(ctx.result.Failures, TestFailure{
			TestName: matches[1], ErrorMessage: "", IsPanic: false,
		})
		ctx.currentFailureIdx = len(ctx.result.Failures) - 1
	}
	ctx.inErrorBlock = true
	if !ctx.patterns.timingRegex.MatchString(line) {
		ctx.failureLines = append(ctx.failureLines, line)
	}
	return true
}

// tryProcessPanic handles panic lines
func (p *TestParser) tryProcessPanic(ctx *parseContext, line string) bool {
	if !ctx.patterns.panicRegex.MatchString(line) {
		return false
	}
	if ctx.currentFailureIdx < 0 {
		ctx.result.Failures = append(ctx.result.Failures, TestFailure{
			TestName: "Unknown", ErrorMessage: "", IsPanic: true,
		})
		ctx.currentFailureIdx = len(ctx.result.Failures) - 1
	} else {
		ctx.result.Failures[ctx.currentFailureIdx].IsPanic = true
	}
	ctx.failureLines = append(ctx.failureLines, line)
	ctx.inErrorBlock = true
	return true
}

// tryProcessErrorLocation handles error location lines
func (p *TestParser) tryProcessErrorLocation(ctx *parseContext, line string) bool {
	matches := ctx.patterns.errorLocationRegex.FindStringSubmatch(line)
	if len(matches) <= MinErrorLocationMatches {
		return false
	}
	if ctx.currentFailureIdx >= 0 {
		ctx.result.Failures[ctx.currentFailureIdx].FileName = matches[1]
		ctx.result.Failures[ctx.currentFailureIdx].LineNumber = matches[2]
		if matches[3] != "" {
			p.appendToErrorMessage(ctx, matches[3])
		}
	}
	ctx.failureLines = append(ctx.failureLines, line)
	ctx.inErrorBlock = true
	return true
}

// processErrorMessage processes general error message lines
func (p *TestParser) processErrorMessage(ctx *parseContext, line, trimmed string, idx int, lines []string) {
	if ctx.inErrorBlock {
		if !ctx.patterns.timingRegex.MatchString(line) {
			ctx.failureLines = append(ctx.failureLines, line)
			if ctx.currentFailureIdx >= 0 {
				p.appendToErrorMessage(ctx, line)
			}
		}
	} else if isErrorIndicator(line, trimmed) {
		ctx.failureLines = append(ctx.failureLines, line)
		ctx.inErrorBlock = true
		if ctx.currentFailureIdx >= 0 {
			p.appendToErrorMessage(ctx, line)
		}
	}

	p.checkLineAhead(ctx, idx, lines)
}

// appendToErrorMessage appends a message to current failure
func (p *TestParser) appendToErrorMessage(ctx *parseContext, msg string) {
	if ctx.result.Failures[ctx.currentFailureIdx].ErrorMessage != "" {
		ctx.result.Failures[ctx.currentFailureIdx].ErrorMessage += "\n"
	}
	ctx.result.Failures[ctx.currentFailureIdx].ErrorMessage += msg
}

// checkLineAhead checks if end of test output is approaching
func (p *TestParser) checkLineAhead(ctx *parseContext, idx int, lines []string) {
	if idx+1 < len(lines) {
		nextLine := strings.TrimSpace(lines[idx+1])
		if ctx.patterns.okPassRegex.MatchString(nextLine) || ctx.patterns.passTestRegex.MatchString(nextLine) ||
			ctx.patterns.failLineRegex.MatchString(nextLine) {
			ctx.inErrorBlock = false
		}
	}
}

// finalizeResults finalizes the parse result
func (p *TestParser) finalizeResults(ctx *parseContext) {
	for i := range ctx.result.Failures {
		ctx.result.Failures[i].ErrorMessage = strings.TrimSpace(ctx.result.Failures[i].ErrorMessage)
	}
	ctx.result.HasFailures = len(ctx.result.Failures) > 0 || ctx.result.FailedTests > 0
	ctx.result.RawFailureOutput = strings.Join(ctx.failureLines, "\n")
}

// isErrorIndicator checks if a line looks like an error
func isErrorIndicator(line, trimmed string) bool {
	return strings.HasPrefix(line, "\t") || strings.HasPrefix(line, "    ") ||
		strings.HasPrefix(trimmed, "Error:") ||
		strings.HasPrefix(trimmed, "expected:") || strings.HasPrefix(trimmed, "got:")
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
