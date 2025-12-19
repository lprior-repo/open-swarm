package gates

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// RequirementDriftDetectionGate ensures agents don't drift from original requirements.
// Periodically verifies code still solves the original problem.
type RequirementDriftDetectionGate struct {
	taskID                string
	originalRequirement   *Requirement
	currentImplementation string
	checkpoints           []DriftCheckpoint
	timestamp             int64
	tokenBudget           int // Re-check every N tokens
	tokensSinceCheck      int
	driftDetected         bool
}

// DriftCheckpoint records a verification of requirement alignment.
type DriftCheckpoint struct {
	Timestamp      int64
	TokensUsed     int
	AlignmentScore float64 // 0.0-1.0 (1.0 = perfectly aligned)
	Issues         []string
	Passed         bool
}

// NewRequirementDriftDetectionGate creates a new drift detection gate.
func NewRequirementDriftDetectionGate(taskID string, req *Requirement) *RequirementDriftDetectionGate {
	return &RequirementDriftDetectionGate{
		taskID:              taskID,
		originalRequirement: req,
		checkpoints:         make([]DriftCheckpoint, 0),
		timestamp:           time.Now().Unix(),
		tokenBudget:         500, //nolint:mnd // Check every ~500 tokens by default
	}
}

// SetTokenBudget sets how often to check alignment (every N tokens).
func (rdd *RequirementDriftDetectionGate) SetTokenBudget(tokens int) {
	if tokens > 0 {
		rdd.tokenBudget = tokens
	}
}

// SetCurrentImplementation updates the implementation being checked.
func (rdd *RequirementDriftDetectionGate) SetCurrentImplementation(code string) {
	rdd.currentImplementation = code
}

// AddTokens tracks token consumption for re-checking periodically.
func (rdd *RequirementDriftDetectionGate) AddTokens(count int) {
	rdd.tokensSinceCheck += count

	// Auto-check if token budget exceeded
	if rdd.tokensSinceCheck >= rdd.tokenBudget {
		rdd.checkAlignment()
		rdd.tokensSinceCheck = 0
	}
}

// Type returns the gate type.
func (rdd *RequirementDriftDetectionGate) Type() GateType {
	return GateDriftDetection
}

// Name returns the human-readable name.
func (rdd *RequirementDriftDetectionGate) Name() string {
	return "Requirement Drift Detection"
}

// Check verifies the implementation is still aligned with original requirements.
func (rdd *RequirementDriftDetectionGate) Check(_ context.Context) error {
	// Validate inputs
	if rdd.originalRequirement == nil {
		return &GateError{
			Gate:      rdd.Type(),
			TaskID:    rdd.taskID,
			Message:   "original requirement not set",
			Details:   "Cannot detect drift without original requirement",
			Timestamp: time.Now().Unix(),
		}
	}

	if rdd.currentImplementation == "" {
		return &GateError{
			Gate:      rdd.Type(),
			TaskID:    rdd.taskID,
			Message:   "current implementation not provided",
			Details:   "Cannot verify alignment without implementation code",
			Timestamp: time.Now().Unix(),
		}
	}

	// Perform alignment check
	checkpoint := rdd.checkAlignment()

	// If alignment too low, return error
	if checkpoint.AlignmentScore < 0.70 { // 70% threshold
		return &GateError{
			Gate:      rdd.Type(),
			TaskID:    rdd.taskID,
			Message:   fmt.Sprintf("requirement drift detected: %.1f%% aligned (need 70%%)", checkpoint.AlignmentScore*100),
			Details:   rdd.generateDriftReport(),
			Timestamp: time.Now().Unix(),
		}
	}

	if !checkpoint.Passed {
		return &GateError{
			Gate:      rdd.Type(),
			TaskID:    rdd.taskID,
			Message:   "implementation diverged from original requirement",
			Details:   rdd.generateDriftReport(),
			Timestamp: time.Now().Unix(),
		}
	}

	return nil
}

// checkAlignment performs the actual alignment verification.
func (rdd *RequirementDriftDetectionGate) checkAlignment() DriftCheckpoint {
	checkpoint := DriftCheckpoint{
		Timestamp:  time.Now().Unix(),
		TokensUsed: rdd.tokensSinceCheck,
		Issues:     make([]string, 0),
		Passed:     true,
	}

	// Check 1: Implementation still contains key requirement terms
	coverage := rdd.checkRequirementCoverage()
	checkpoint.AlignmentScore = coverage

	// Check 2: Verify acceptance criteria are met in code
	if !rdd.verifyAcceptanceCriteria() {
		checkpoint.Issues = append(checkpoint.Issues, "Acceptance criteria may not be satisfied")
		checkpoint.Passed = false
	}

	// Check 3: Verify expected scenarios are implemented
	missingScenarios := rdd.findMissingScenarios()
	if len(missingScenarios) > 0 {
		checkpoint.Issues = append(checkpoint.Issues, fmt.Sprintf("Missing implementations for scenarios: %v", missingScenarios))
		checkpoint.Passed = false
	}

	// Check 4: Detect scope creep (code doing things not in requirement)
	extraFeatures := rdd.detectExtraScope()
	if len(extraFeatures) > 0 {
		checkpoint.Issues = append(checkpoint.Issues, fmt.Sprintf("Scope creep detected: %v", extraFeatures))
	}

	rdd.checkpoints = append(rdd.checkpoints, checkpoint)

	// Mark if drift detected
	if !checkpoint.Passed {
		rdd.driftDetected = true
	}

	return checkpoint
}

// checkRequirementCoverage determines if implementation covers requirement concepts.
func (rdd *RequirementDriftDetectionGate) checkRequirementCoverage() float64 {
	// Extract key terms from requirement
	reqTerms := rdd.extractKeyTerms(rdd.originalRequirement.Description)
	if len(reqTerms) == 0 {
		return 1.0 // No terms to match
	}

	// Count how many are present in implementation
	matchedTerms := 0
	implLower := strings.ToLower(rdd.currentImplementation)

	for _, term := range reqTerms {
		if strings.Contains(implLower, strings.ToLower(term)) {
			matchedTerms++
		}
	}

	return float64(matchedTerms) / float64(len(reqTerms))
}

// extractKeyTerms pulls important concepts from requirement text.
func (rdd *RequirementDriftDetectionGate) extractKeyTerms(text string) []string {
	// Simplified keyword extraction (in production, use NLP)
	keywords := []string{}
	words := strings.Fields(text)

	// Keep words > 5 chars that aren't common stop words
	stopWords := map[string]bool{
		"the": true, "and": true, "or": true, "for": true,
		"with": true, "that": true, "this": true, "from": true,
		"are": true, "not": true, "but": true, "can": true,
		"must": true, "should": true, "will": true, "their": true,
	}

	for _, word := range words {
		if len(word) > 4 && !stopWords[strings.ToLower(word)] {
			keywords = append(keywords, word)
		}
	}

	// Limit to 10 most relevant
	if len(keywords) > 10 {
		keywords = keywords[:10]
	}

	return keywords
}

// verifyAcceptanceCriteria checks if acceptance criteria appear in implementation.
func (rdd *RequirementDriftDetectionGate) verifyAcceptanceCriteria() bool {
	if rdd.originalRequirement.Acceptance == "" {
		return true // No specific criteria
	}

	implLower := strings.ToLower(rdd.currentImplementation)
	acceptLower := strings.ToLower(rdd.originalRequirement.Acceptance)

	// Simple check: acceptance criteria concepts appear in code
	keyPhrases := strings.Split(acceptLower, ";")
	matchedPhrases := 0

	for _, phrase := range keyPhrases {
		phrase = strings.TrimSpace(phrase)
		if len(phrase) > 0 && strings.Contains(implLower, phrase) {
			matchedPhrases++
		}
	}

	// At least 70% of acceptance criteria should be detectable
	if len(keyPhrases) > 0 {
		coverage := float64(matchedPhrases) / float64(len(keyPhrases))
		return coverage >= 0.70
	}

	return true
}

// findMissingScenarios identifies required test scenarios not in implementation.
func (rdd *RequirementDriftDetectionGate) findMissingScenarios() []string {
	missing := []string{}
	implLower := strings.ToLower(rdd.currentImplementation)

	for _, scenario := range rdd.originalRequirement.Scenarios {
		scenarioLower := strings.ToLower(scenario)
		if !strings.Contains(implLower, scenarioLower) {
			// Check for semantic similarity
			words := strings.Fields(scenarioLower)
			matchedWords := 0
			for _, word := range words {
				if len(word) > 3 && strings.Contains(implLower, word) {
					matchedWords++
				}
			}
			if len(words) > 0 && float64(matchedWords)/float64(len(words)) < 0.5 {
				missing = append(missing, scenario)
			}
		}
	}

	return missing
}

// detectExtraScope identifies features not in the original requirement.
func (rdd *RequirementDriftDetectionGate) detectExtraScope() []string {
	extra := []string{}

	extraIndicators := []string{
		"bonus", "extra", "additional", "optimization", "performance improvement",
		"refactor", "cleanup", "restructure", "rename", "comment cleanup",
	}

	implLower := strings.ToLower(rdd.currentImplementation)
	for _, indicator := range extraIndicators {
		if strings.Contains(implLower, indicator) {
			extra = append(extra, indicator)
		}
	}

	return extra
}

// generateDriftReport provides detailed drift analysis.
func (rdd *RequirementDriftDetectionGate) generateDriftReport() string {
	var report strings.Builder

	report.WriteString("=== Requirement Drift Detection Report ===\n\n")
	report.WriteString("ORIGINAL REQUIREMENT:\n")
	report.WriteString(fmt.Sprintf("  Task: %s\n", rdd.originalRequirement.Title))
	report.WriteString(fmt.Sprintf("  Description: %s\n\n", rdd.originalRequirement.Description))

	if len(rdd.checkpoints) > 0 {
		latest := rdd.checkpoints[len(rdd.checkpoints)-1]
		report.WriteString("LATEST ALIGNMENT CHECK:\n")
		report.WriteString(fmt.Sprintf("  Alignment Score: %.1f%%\n", latest.AlignmentScore*100))
		report.WriteString(fmt.Sprintf("  Passed: %v\n", latest.Passed))

		if len(latest.Issues) > 0 {
			report.WriteString("  Issues Detected:\n")
			for _, issue := range latest.Issues {
				report.WriteString(fmt.Sprintf("    • %s\n", issue))
			}
		}
		report.WriteString("\n")
	}

	report.WriteString("REQUIRED ACTIONS:\n")
	report.WriteString("  1. Re-read original requirement\n")
	report.WriteString("  2. Verify implementation still matches specification\n")
	report.WriteString("  3. Remove any scope creep or extra features\n")
	report.WriteString("  4. Focus only on requirements, nothing more\n")
	report.WriteString("  5. Re-run tests to confirm alignment\n")

	return report.String()
}

// GetCheckpoints returns all recorded alignment checkpoints.
func (rdd *RequirementDriftDetectionGate) GetCheckpoints() []DriftCheckpoint {
	return rdd.checkpoints
}

// IsDriftDetected returns whether drift has been detected.
func (rdd *RequirementDriftDetectionGate) IsDriftDetected() bool {
	return rdd.driftDetected
}
