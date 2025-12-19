package gates

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// HardWorkEnforcementGate ensures that stubbed implementations fail tests.
// No escape hatches, no mocking exemptions - forced to do real work.
type HardWorkEnforcementGate struct {
	taskID               string
	implementationPath   string      // Path to the implementation file
	implementationCode   string      // The implementation source code
	testResult           *TestResult // Test results after running implementation
	timestamp            int64
	detectedStubPatterns []string // Stub patterns found
}

// NewHardWorkEnforcementGate creates a new hard work enforcement gate.
func NewHardWorkEnforcementGate(taskID string, implPath string) *HardWorkEnforcementGate {
	return &HardWorkEnforcementGate{
		taskID:             taskID,
		implementationPath: implPath,
		timestamp:          time.Now().Unix(),
	}
}

// SetImplementationCode sets the source code of the implementation.
func (hwg *HardWorkEnforcementGate) SetImplementationCode(code string) {
	hwg.implementationCode = code
}

// SetTestResult sets the results of running tests against the implementation.
func (hwg *HardWorkEnforcementGate) SetTestResult(result *TestResult) {
	hwg.testResult = result
}

// Type returns the gate type.
func (hwg *HardWorkEnforcementGate) Type() GateType {
	return GateHardWork
}

// Name returns the human-readable name.
func (hwg *HardWorkEnforcementGate) Name() string {
	return "Hard Work Enforcement"
}

// Check verifies that stubbed code fails tests and forces real implementation.
func (hwg *HardWorkEnforcementGate) Check(_ context.Context) error {
	// Validate inputs
	if hwg.implementationCode == "" {
		return &GateError{
			Gate:      hwg.Type(),
			TaskID:    hwg.taskID,
			Message:   "implementation code not provided",
			Details:   "Cannot verify implementation without source code",
			Timestamp: time.Now().Unix(),
		}
	}

	if hwg.testResult == nil {
		return &GateError{
			Gate:      hwg.Type(),
			TaskID:    hwg.taskID,
			Message:   "test result not set",
			Details:   "Cannot verify work enforced without test results",
			Timestamp: time.Now().Unix(),
		}
	}

	// Check 1: Detect stub patterns (panic, return nil, return empty, TODO comments)
	stubs := hwg.detectStubPatterns()
	if len(stubs) > 0 {
		hwg.detectedStubPatterns = stubs

		// If tests passed with stubs, that's dishonest
		if hwg.testResult.IsPassing() {
			return &GateError{
				Gate:      hwg.Type(),
				TaskID:    hwg.taskID,
				Message:   "stubbed implementation passed tests",
				Details:   hwg.generateStubReport(),
				Timestamp: time.Now().Unix(),
			}
		}

		// Tests correctly failed with stubs - this is what we want
		// (Stubs are fine if tests fail and agent is trying to fix them)
	}

	// Check 2: If tests are failing, verify they're not mocked/skipped
	if !hwg.testResult.IsPassing() {
		if hwg.hasDisabledTests() {
			return &GateError{
				Gate:      hwg.Type(),
				TaskID:    hwg.taskID,
				Message:   "tests have been disabled or skipped",
				Details:   "Tests must be enabled. Skipping tests is not real work. Cannot disable tests to hide failures.",
				Timestamp: time.Now().Unix(),
			}
		}
	}

	// Check 3: Verify no cheating patterns (mocking framework bypass, test framework disabling, etc.)
	cheatPatterns := hwg.detectCheatingPatterns()
	if len(cheatPatterns) > 0 {
		return &GateError{
			Gate:      hwg.Type(),
			TaskID:    hwg.taskID,
			Message:   "cheating patterns detected in implementation",
			Details:   hwg.generateCheatingReport(cheatPatterns),
			Timestamp: time.Now().Unix(),
		}
	}

	// Check 4: Verify implementation is not trivial (at least some real logic)
	if hwg.isImplementationTrivial() {
		return &GateError{
			Gate:      hwg.Type(),
			TaskID:    hwg.taskID,
			Message:   "implementation is too trivial",
			Details:   "Implementation must contain real logic. Cannot pass tests with minimal/boilerplate code.",
			Timestamp: time.Now().Unix(),
		}
	}

	return nil
}

// detectStubPatterns finds common stubbing patterns in code.
func (hwg *HardWorkEnforcementGate) detectStubPatterns() []string {
	var patterns []string

	stubPatternChecks := map[string]*regexp.Regexp{
		"panic calls":         regexp.MustCompile(`panic\s*\(`),
		"unimplemented":       regexp.MustCompile(`(?i)(unimplemented|not\s+implemented|todo)`),
		"return nil":          regexp.MustCompile(`return\s+nil\s*[,;]`),
		"return empty":        regexp.MustCompile(`return\s+(?:""|''|\[\]|\{\})\s*[,;]`),
		"return false":        regexp.MustCompile(`return\s+false\s*[,;]`),
		"return 0":            regexp.MustCompile(`return\s+0\s*[,;]`),
		"empty function body": regexp.MustCompile(`func\s+\w+\s*\([^)]*\)\s*(?:\w+\s+)*[{]\s*[}]`),
	}

	for name, pattern := range stubPatternChecks {
		if pattern.MatchString(hwg.implementationCode) {
			patterns = append(patterns, name)
		}
	}

	return patterns
}

// hasDisabledTests checks if tests have been skipped/disabled.
func (hwg *HardWorkEnforcementGate) hasDisabledTests() bool {
	disabledPatterns := []string{
		`t\.Skip`,
		`SkipNow`,
		`skip\s*\(`,
		`@skip`,
		`x\.test`,
		`xit\(`,
	}

	for _, pattern := range disabledPatterns {
		regex := regexp.MustCompile(pattern)
		if regex.MatchString(hwg.implementationCode) {
			return true
		}
	}

	return false
}

// detectCheatingPatterns finds patterns that try to bypass test framework integrity.
func (hwg *HardWorkEnforcementGate) detectCheatingPatterns() []string {
	var patterns []string

	cheatChecks := map[string]*regexp.Regexp{
		"test mocking injection":   regexp.MustCompile(`(?i)(mock\s+.*test|patch\s+test|override\s+test)`),
		"assertion disabling":      regexp.MustCompile(`(?i)(disable.*assert|skip.*assert|mock.*assert)`),
		"exit code suppression":    regexp.MustCompile(`suppressExitCode|ignore.*exit|suppress.*error`),
		"test framework bypass":    regexp.MustCompile(`(?i)(bypass.*test|circumvent.*test|override.*fail)`),
		"environment manipulation": regexp.MustCompile(`(?i)(setenv.*TEST|override.*TEST|mock.*env.*test)`),
	}

	for name, pattern := range cheatChecks {
		if pattern.MatchString(hwg.implementationCode) {
			patterns = append(patterns, name)
		}
	}

	return patterns
}

// isImplementationTrivial checks if implementation is too short or has no real logic.
func (hwg *HardWorkEnforcementGate) isImplementationTrivial() bool {
	// Count non-comment, non-whitespace lines
	lines := strings.Split(hwg.implementationCode, "\n")
	logicLines := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "//") && !strings.HasPrefix(trimmed, "/*") {
			logicLines++
		}
	}

	// Implementation must have at least 10 lines of actual logic
	return logicLines < 10
}

// generateStubReport provides details about stub patterns found.
func (hwg *HardWorkEnforcementGate) generateStubReport() string {
	var report strings.Builder

	report.WriteString("=== Stub Pattern Detection Report ===\n\n")
	report.WriteString("ISSUE: Stubbed code passed tests (dishonest)\n\n")
	report.WriteString("DETECTED STUB PATTERNS:\n")

	for i, pattern := range hwg.detectedStubPatterns {
		report.WriteString(fmt.Sprintf("  %d. %s\n", i+1, pattern))
	}

	report.WriteString("\nREQUIREMENT:\n")
	report.WriteString("  • Tests must FAIL when implementation is stubbed\n")
	report.WriteString("  • Agent must implement real logic\n")
	report.WriteString("  • Running tests with stubs is the first check\n")
	report.WriteString("  • If stubs pass tests, the tests are insufficient\n\n")

	report.WriteString("ACTION:\n")
	report.WriteString("  1. Improve tests to catch stub implementations\n")
	report.WriteString("  2. Agent must implement real functionality\n")
	report.WriteString("  3. Tests must fail on stubs, pass on real implementation\n")

	return report.String()
}

// generateCheatingReport details cheating patterns discovered.
func (hwg *HardWorkEnforcementGate) generateCheatingReport(patterns []string) string {
	var report strings.Builder

	report.WriteString("=== Cheating Pattern Detection Report ===\n\n")
	report.WriteString("VIOLATION: Code contains cheating patterns\n\n")
	report.WriteString("DETECTED PATTERNS:\n")

	for i, pattern := range patterns {
		report.WriteString(fmt.Sprintf("  %d. %s\n", i+1, pattern))
	}

	report.WriteString("\nVIOLATION DETAILS:\n")
	report.WriteString("  • Attempting to bypass test framework integrity\n")
	report.WriteString("  • Trying to disable or suppress test failures\n")
	report.WriteString("  • Mocking test assertions or validation\n")
	report.WriteString("  • Manipulating environment to hide failures\n\n")

	report.WriteString("CONSEQUENCE:\n")
	report.WriteString("  This is UNACCEPTABLE. Task is REJECTED.\n")
	report.WriteString("  • Submit honest work\n")
	report.WriteString("  • Tests are immutable (cannot be modified)\n")
	report.WriteString("  • Implementation must pass all tests as written\n")

	return report.String()
}
