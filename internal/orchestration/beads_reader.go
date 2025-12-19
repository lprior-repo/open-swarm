package orchestration

import (
	"fmt"
	"strings"

	"open-swarm/internal/gates"
)

// BeadsTaskReader reads Beads issues and converts them to agent configurations.
type BeadsTaskReader struct {
	logger Logger
}

// NewBeadsTaskReader creates a new Beads task reader.
func NewBeadsTaskReader(logger Logger) *BeadsTaskReader {
	return &BeadsTaskReader{
		logger: logger,
	}
}

// BeadsIssue represents a Beads task/issue in simplified form.
type BeadsIssue struct {
	ID              string   // Issue ID (e.g., "open-swarm-xxxx")
	Title           string   // Issue title
	Description     string   // Issue description
	Acceptance      string   // Acceptance criteria
	Type            string   // Issue type (bug, feature, task, epic, chore)
	Status          string   // Status (open, in_progress, blocked, closed)
	Priority        int      // Priority level (1-5, 1=highest)
	Labels          []string // Issue labels
	Dependencies    []string // Dependent issue IDs
	AssignedTo      string   // Assignee
	EstimatedTokens int      // Estimated token budget
}

// ReadFromIssue converts a Beads issue to an AgentConfig.
func (r *BeadsTaskReader) ReadFromIssue(issue BeadsIssue) (*AgentConfig, error) {
	// Parse scenarios and edge cases from description
	scenarios, edgeCases := r.parseScenarios(issue.Description)

	// Parse dependencies
	dependencies := r.parseDependencies(issue.Dependencies)

	// Extract requirement information
	req := &gates.Requirement{
		TaskID:      issue.ID,
		Title:       issue.Title,
		Description: issue.Description,
		Acceptance:  issue.Acceptance,
		Scenarios:   scenarios,
		EdgeCases:   edgeCases,
	}

	// Create agent config
	config := &AgentConfig{
		TaskID:             issue.ID,
		Title:              issue.Title,
		Description:        issue.Description,
		AcceptanceCriteria: issue.Acceptance,
		Scenarios:          scenarios,
		EdgeCases:          edgeCases,
		DependsOn:          dependencies,
		RequirementsForGate: req,
		MaxRetries:         3, // Default retries
		TimeoutSeconds:     300, // 5 min default
		ReviewersCount:     1, // Single reviewer default
	}

	// Adjust based on issue priority
	config.ReviewersCount = r.getReviewerCountFromPriority(issue.Priority)
	config.MaxRetries = r.getRetriesFromPriority(issue.Priority)

	// Check for parallelism hints in labels
	if r.hasLabel(issue.Labels, "needs-parallel-review") {
		config.ReviewersCount = 3 // Run with 3 parallel reviewers
	}
	if r.hasLabel(issue.Labels, "high-complexity") {
		config.MaxRetries = 5
		config.ReviewersCount = 5
	}

	r.logger.Infof("Parsed Beads issue %s: %s (priority=%d, reviewers=%d, retries=%d)",
		issue.ID, issue.Title, issue.Priority, config.ReviewersCount, config.MaxRetries)

	return config, nil
}

// parseScenarios extracts test scenarios and edge cases from description.
func (r *BeadsTaskReader) parseScenarios(description string) ([]string, []string) {
	var scenarios []string
	var edgeCases []string

	lines := strings.Split(description, "\n")
	inScenarios := false
	inEdgeCases := false

	for _, line := range lines {
		lineLower := strings.ToLower(strings.TrimSpace(line))

		// Detect sections
		if strings.Contains(lineLower, "scenario") || strings.Contains(lineLower, "test case") {
			inScenarios = true
			inEdgeCases = false
			continue
		}
		if strings.Contains(lineLower, "edge case") {
			inEdgeCases = true
			inScenarios = false
			continue
		}

		// Parse bullet points
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "-") || strings.HasPrefix(trimmed, "*") {
			content := strings.TrimPrefix(strings.TrimPrefix(trimmed, "-"), "*")
			content = strings.TrimSpace(content)

			if inScenarios && content != "" {
				scenarios = append(scenarios, content)
			} else if inEdgeCases && content != "" {
				edgeCases = append(edgeCases, content)
			}
		}
	}

	// If no explicit scenarios found, create one from description
	if len(scenarios) == 0 {
		scenarios = append(scenarios, "default: "+description)
	}

	return scenarios, edgeCases
}

// parseDependencies converts dependency IDs from Beads format.
func (r *BeadsTaskReader) parseDependencies(deps []string) []string {
	var result []string
	for _, dep := range deps {
		// Beads stores full dependency info, we just need the ID
		// In real implementation, this might parse a more complex structure
		if strings.HasPrefix(dep, "open-swarm-") {
			result = append(result, dep)
		}
	}
	return result
}

// getReviewerCountFromPriority determines reviewer count based on priority.
func (r *BeadsTaskReader) getReviewerCountFromPriority(priority int) int {
	switch {
	case priority == 1: // Critical
		return 5 // 5 parallel reviewers
	case priority <= 2: // High
		return 3
	case priority <= 3: // Medium
		return 1
	default: // Low
		return 1
	}
}

// getRetriesFromPriority determines max retries based on priority.
func (r *BeadsTaskReader) getRetriesFromPriority(priority int) int {
	switch {
	case priority == 1: // Critical - don't give up
		return 10
	case priority <= 2: // High
		return 5
	case priority <= 3: // Medium
		return 3
	default: // Low
		return 2
	}
}

// hasLabel checks if issue has a specific label.
func (r *BeadsTaskReader) hasLabel(labels []string, target string) bool {
	targetLower := strings.ToLower(target)
	for _, label := range labels {
		if strings.ToLower(label) == targetLower {
			return true
		}
	}
	return false
}

// ValidateTask checks if a task is valid for agent execution.
func (r *BeadsTaskReader) ValidateTask(issue BeadsIssue) error {
	// Check required fields
	if issue.ID == "" {
		return fmt.Errorf("issue ID is required")
	}
	if issue.Title == "" {
		return fmt.Errorf("issue title is required")
	}
	if issue.Description == "" {
		return fmt.Errorf("issue description is required")
	}

	// Check status
	if issue.Status == "closed" {
		return fmt.Errorf("issue is already closed")
	}

	// Check for circular dependencies
	if r.hasCircularDep(issue.ID, issue.Dependencies) {
		return fmt.Errorf("circular dependency detected")
	}

	return nil
}

// hasCircularDep checks for circular dependencies (simplified).
func (r *BeadsTaskReader) hasCircularDep(taskID string, deps []string) bool {
	// Simple check: does any dependency list contain the task itself?
	for _, dep := range deps {
		if dep == taskID {
			return true
		}
	}
	return false
}

// CreateBatch creates multiple agent configs from Beads issues.
func (r *BeadsTaskReader) CreateBatch(issues []BeadsIssue) ([]*AgentConfig, error) {
	var configs []*AgentConfig
	var errs []error

	for _, issue := range issues {
		// Validate first
		if err := r.ValidateTask(issue); err != nil {
			r.logger.Warnf("Invalid task %s: %v", issue.ID, err)
			errs = append(errs, fmt.Errorf("task %s: %w", issue.ID, err))
			continue
		}

		// Convert to config
		config, err := r.ReadFromIssue(issue)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to parse task %s: %w", issue.ID, err))
			continue
		}

		configs = append(configs, config)
	}

	// Report any errors but continue
	if len(errs) > 0 {
		r.logger.Warnf("Found %d invalid tasks during batch parsing", len(errs))
	}

	return configs, nil
}

// BatchMetadata summarizes a batch of tasks.
type BatchMetadata struct {
	TotalTasks       int
	ByPriority       map[int]int    // Count by priority
	ByType           map[string]int  // Count by issue type
	AverageRetries   float64
	AverageReviewers float64
	EstimatedTokens  int
}

// GetBatchMetadata analyzes a set of issues.
func (r *BeadsTaskReader) GetBatchMetadata(issues []BeadsIssue) BatchMetadata {
	meta := BatchMetadata{
		TotalTasks:  len(issues),
		ByPriority:  make(map[int]int),
		ByType:      make(map[string]int),
	}

	totalRetries := 0
	totalReviewers := 0

	for _, issue := range issues {
		meta.ByPriority[issue.Priority]++
		meta.ByType[issue.Type]++
		meta.EstimatedTokens += issue.EstimatedTokens

		// Calculate expected retries and reviewers
		totalRetries += r.getRetriesFromPriority(issue.Priority)
		totalReviewers += r.getReviewerCountFromPriority(issue.Priority)
	}

	if meta.TotalTasks > 0 {
		meta.AverageRetries = float64(totalRetries) / float64(meta.TotalTasks)
		meta.AverageReviewers = float64(totalReviewers) / float64(meta.TotalTasks)
	}

	return meta
}
