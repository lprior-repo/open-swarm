package gates

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const defaultCoverageThreshold = 0.90

// RequirementsVerificationGate ensures the agent proves understanding of the task
// by generating test cases that cover all stated requirements.
type RequirementsVerificationGate struct {
	taskID            string
	requirement       *Requirement
	generatedTests    []string
	coverageThreshold float64 // Minimum coverage % required (default 90%).
	timestamp         int64
}

// NewRequirementsVerificationGate creates a new requirements verification gate.
func NewRequirementsVerificationGate(taskID string, req *Requirement) *RequirementsVerificationGate {
	return &RequirementsVerificationGate{
		taskID:            taskID,
		requirement:       req,
		coverageThreshold: defaultCoverageThreshold,
		timestamp:         time.Now().Unix(),
	}
}

// SetCoverageThreshold sets the minimum acceptable test coverage percentage.
func (rvg *RequirementsVerificationGate) SetCoverageThreshold(threshold float64) {
	if threshold > 0 && threshold <= 1.0 {
		rvg.coverageThreshold = threshold
	}
}

// SetGeneratedTests sets the test cases the agent generated.
func (rvg *RequirementsVerificationGate) SetGeneratedTests(tests []string) {
	rvg.generatedTests = tests
}

// Type returns the gate type.
func (rvg *RequirementsVerificationGate) Type() GateType {
	return GateRequirements
}

// Name returns the human-readable name.
func (rvg *RequirementsVerificationGate) Name() string {
	return "Requirements Verification"
}

// Check verifies that the agent's test cases adequately cover all requirements.
func (rvg *RequirementsVerificationGate) Check(_ context.Context) error {
	// Validate inputs
	if rvg.requirement == nil {
		return &GateError{
			Gate:      rvg.Type(),
			TaskID:    rvg.taskID,
			Message:   "requirement not set",
			Details:   "Cannot verify tests without a requirement object",
			Timestamp: time.Now().Unix(),
		}
	}

	if len(rvg.generatedTests) == 0 {
		return &GateError{
			Gate:      rvg.Type(),
			TaskID:    rvg.taskID,
			Message:   "no tests generated",
			Details:   "Agent did not generate any test cases. Tests must prove understanding of the requirement.",
			Timestamp: time.Now().Unix(),
		}
	}

	// Check for minimum coverage of key scenarios
	coverage := rvg.calculateCoverage()
	if coverage < rvg.coverageThreshold {
		return &GateError{
			Gate:      rvg.Type(),
			TaskID:    rvg.taskID,
			Message:   fmt.Sprintf("insufficient test coverage: %.1f%% (need %.1f%%)", coverage*percentageMultiplier, rvg.coverageThreshold*percentageMultiplier),
			Details:   rvg.generateCoverageFeedback(),
			Timestamp: time.Now().Unix(),
		}
	}

	// Verify test clarity (no vague test names)
	if issues := rvg.checkTestClarity(); len(issues) > 0 {
		return &GateError{
			Gate:      rvg.Type(),
			TaskID:    rvg.taskID,
			Message:   "tests contain ambiguous language",
			Details:   strings.Join(issues, "; "),
			Timestamp: time.Now().Unix(),
		}
	}

	// All checks passed
	return nil
}

// calculateCoverage determines what percentage of requirement scenarios are covered by tests.
func (rvg *RequirementsVerificationGate) calculateCoverage() float64 {
	if len(rvg.requirement.Scenarios) == 0 {
		// If no specific scenarios listed, use all tests as coverage
		return 1.0
	}

	coveredScenarios := 0
	for _, scenario := range rvg.requirement.Scenarios {
		if rvg.testCoversScenario(scenario) {
			coveredScenarios++
		}
	}

	return float64(coveredScenarios) / float64(len(rvg.requirement.Scenarios))
}

// testCoversScenario checks if any generated test covers a specific requirement scenario.
func (rvg *RequirementsVerificationGate) testCoversScenario(scenario string) bool {
	scenarioLower := strings.ToLower(scenario)

	for _, test := range rvg.generatedTests {
		testLower := strings.ToLower(test)
		// Check if test name contains key words from scenario
		if strings.Contains(testLower, scenarioLower) {
			return true
		}

		// Check for semantic similarity (simple substring matching)
		words := strings.Fields(scenarioLower)
		matchedWords := 0
		for _, word := range words {
			if len(word) > 3 && strings.Contains(testLower, word) { // Ignore short words
				matchedWords++
			}
		}
		if len(words) > 0 && float64(matchedWords)/float64(len(words)) >= 0.6 {
			return true
		}
	}

	return false
}

// checkTestClarity verifies tests use clear, unambiguous language.
func (rvg *RequirementsVerificationGate) checkTestClarity() []string {
	var issues []string

	vagueWords := []string{"easily", "quickly", "properly", "nicely", "well", "maybe", "possibly", "sometimes", "probably"}

	for _, test := range rvg.generatedTests {
		testLower := strings.ToLower(test)

		// Check for vague language
		for _, vague := range vagueWords {
			if strings.Contains(testLower, vague) {
				issues = append(issues, fmt.Sprintf("Test '%s' uses vague word '%s' - tests must be precise", test, vague))
			}
		}

		// Check for missing assertions (test names without expected outcomes)
		if !rvg.hasExpectedOutcome(test) {
			issues = append(issues, fmt.Sprintf("Test '%s' doesn't indicate expected outcome - must clarify what should happen", test))
		}
	}

	return issues
}

// hasExpectedOutcome checks if a test indicates what the expected result is.
func (rvg *RequirementsVerificationGate) hasExpectedOutcome(testName string) bool {
	// Look for outcome keywords
	outcomeKeywords := []string{
		"success", "fail", "error", "pass", "reject", "accept", "accepted", "rejected",
		"valid", "invalid", "true", "false", "returns", "should",
		"must", "can", "empty", "nil", "panic", "timeout", "works", "handled",
	}

	testLower := strings.ToLower(testName)
	for _, keyword := range outcomeKeywords {
		if strings.Contains(testLower, keyword) {
			return true
		}
	}

	return false
}

// generateCoverageFeedback provides detailed feedback about what's missing.
func (rvg *RequirementsVerificationGate) generateCoverageFeedback() string {
	var feedback strings.Builder

	feedback.WriteString("Required test coverage:\n")
	feedback.WriteString("Generated tests:\n")
	for _, test := range rvg.generatedTests {
		feedback.WriteString(fmt.Sprintf("  ✓ %s\n", test))
	}

	feedback.WriteString("\nRequired scenarios to cover:\n")
	for _, scenario := range rvg.requirement.Scenarios {
		if rvg.testCoversScenario(scenario) {
			feedback.WriteString(fmt.Sprintf("  ✓ %s\n", scenario))
		} else {
			feedback.WriteString(fmt.Sprintf("  ✗ %s (NOT COVERED)\n", scenario))
		}
	}

	if len(rvg.requirement.EdgeCases) > 0 {
		feedback.WriteString("\nEdge cases to cover:\n")
		for _, edgeCase := range rvg.requirement.EdgeCases {
			if rvg.testCoversScenario(edgeCase) {
				feedback.WriteString(fmt.Sprintf("  ✓ %s\n", edgeCase))
			} else {
				feedback.WriteString(fmt.Sprintf("  ✗ %s (NOT COVERED)\n", edgeCase))
			}
		}
	}

	return feedback.String()
}
