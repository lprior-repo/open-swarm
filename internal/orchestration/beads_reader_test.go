package orchestration

import (
	"testing"
)

// TestBeadsReaderReadFromIssue tests basic issue parsing
func TestBeadsReaderReadFromIssue(t *testing.T) {
	logger := &MockLogger{}
	reader := NewBeadsTaskReader(logger)

	issue := BeadsIssue{
		ID:          "open-swarm-001",
		Title:       "Implement user login",
		Description: "Implement authentication feature\n\nScenarios:\n- User enters credentials\n- System validates\n\nEdge Cases:\n- Invalid password\n- Account locked",
		Acceptance:  "All tests pass, authentication works",
		Type:        "feature",
		Status:      "open",
		Priority:    2,
		Labels:      []string{},
		Dependencies: []string{},
		AssignedTo:  "user1",
		EstimatedTokens: 500,
	}

	config, err := reader.ReadFromIssue(issue)
	if err != nil {
		t.Fatalf("ReadFromIssue failed: %v", err)
	}

	if config.TaskID != "open-swarm-001" {
		t.Fatalf("Expected open-swarm-001, got %s", config.TaskID)
	}

	if config.Title != "Implement user login" {
		t.Fatalf("Expected 'Implement user login', got %s", config.Title)
	}

	if config.ReviewersCount != 3 {
		t.Fatalf("Expected 3 reviewers for priority 2, got %d", config.ReviewersCount)
	}

	if len(config.Scenarios) == 0 {
		t.Fatal("Expected scenarios to be extracted")
	}

	if len(config.EdgeCases) == 0 {
		t.Fatal("Expected edge cases to be extracted")
	}
}

// TestBeadsReaderParseScenariosAndEdgeCases tests scenario and edge case extraction
func TestBeadsReaderParseScenariosAndEdgeCases(t *testing.T) {
	logger := &MockLogger{}
	reader := NewBeadsTaskReader(logger)

	description := `Task description

Scenarios:
- User enters valid credentials
- User enters invalid credentials
- User clicks remember me

Edge Cases:
- Account locked after 3 failed attempts
- Session timeout
- Network error during login`

	scenarios, edgeCases := reader.parseScenarios(description)

	if len(scenarios) != 3 {
		t.Fatalf("Expected 3 scenarios, got %d", len(scenarios))
	}

	if scenarios[0] != "User enters valid credentials" {
		t.Fatalf("Expected 'User enters valid credentials', got '%s'", scenarios[0])
	}

	if len(edgeCases) != 3 {
		t.Fatalf("Expected 3 edge cases, got %d", len(edgeCases))
	}

	if edgeCases[0] != "Account locked after 3 failed attempts" {
		t.Fatalf("Expected 'Account locked after 3 failed attempts', got '%s'", edgeCases[0])
	}
}

// TestBeadsReaderParseDependencies tests dependency parsing
func TestBeadsReaderParseDependencies(t *testing.T) {
	logger := &MockLogger{}
	reader := NewBeadsTaskReader(logger)

	deps := []string{"open-swarm-001", "open-swarm-002", "invalid-dep"}
	result := reader.parseDependencies(deps)

	if len(result) != 2 {
		t.Fatalf("Expected 2 valid dependencies, got %d", len(result))
	}

	if result[0] != "open-swarm-001" {
		t.Fatalf("Expected open-swarm-001, got %s", result[0])
	}
}

// TestBeadsReaderValidateTask tests task validation
func TestBeadsReaderValidateTask(t *testing.T) {
	logger := &MockLogger{}
	reader := NewBeadsTaskReader(logger)

	// Valid task
	validIssue := BeadsIssue{
		ID:          "open-swarm-001",
		Title:       "Test",
		Description: "Test description",
		Status:      "open",
	}

	err := reader.ValidateTask(validIssue)
	if err != nil {
		t.Fatalf("Valid task failed validation: %v", err)
	}

	// Missing ID
	noIDIssue := BeadsIssue{
		Title:       "Test",
		Description: "Test description",
	}

	err = reader.ValidateTask(noIDIssue)
	if err == nil {
		t.Fatal("Expected validation error for missing ID")
	}

	// Missing title
	noTitleIssue := BeadsIssue{
		ID:          "open-swarm-001",
		Description: "Test description",
	}

	err = reader.ValidateTask(noTitleIssue)
	if err == nil {
		t.Fatal("Expected validation error for missing title")
	}

	// Closed task
	closedIssue := BeadsIssue{
		ID:          "open-swarm-001",
		Title:       "Test",
		Description: "Test description",
		Status:      "closed",
	}

	err = reader.ValidateTask(closedIssue)
	if err == nil {
		t.Fatal("Expected validation error for closed task")
	}
}

// TestBeadsReaderCircularDependency tests circular dependency detection
func TestBeadsReaderCircularDependency(t *testing.T) {
	logger := &MockLogger{}
	reader := NewBeadsTaskReader(logger)

	// Circular: task1 depends on itself
	circular := BeadsIssue{
		ID:           "open-swarm-001",
		Title:        "Test",
		Description:  "Test",
		Dependencies: []string{"open-swarm-001"},
	}

	err := reader.ValidateTask(circular)
	if err == nil {
		t.Fatal("Expected circular dependency error")
	}
}

// TestBeadsReaderCreateBatch tests batch creation
func TestBeadsReaderCreateBatch(t *testing.T) {
	logger := &MockLogger{}
	reader := NewBeadsTaskReader(logger)

	issues := []BeadsIssue{
		{
			ID:          "open-swarm-001",
			Title:       "Task 1",
			Description: "Description 1",
			Status:      "open",
			Priority:    1,
		},
		{
			ID:          "open-swarm-002",
			Title:       "Task 2",
			Description: "Description 2",
			Status:      "open",
			Priority:    2,
		},
		{
			ID:          "open-swarm-003",
			Title:       "Task 3",
			Description: "Description 3",
			Status:      "closed", // This one is closed, should be skipped
			Priority:    3,
		},
	}

	configs, err := reader.CreateBatch(issues)
	if err != nil {
		t.Fatalf("CreateBatch failed: %v", err)
	}

	// Should only have 2 configs (closed task is skipped)
	if len(configs) != 2 {
		t.Fatalf("Expected 2 configs, got %d", len(configs))
	}
}

// TestBeadsReaderPriorityMapping tests reviewer and retry count mapping
func TestBeadsReaderPriorityMapping(t *testing.T) {
	logger := &MockLogger{}
	reader := NewBeadsTaskReader(logger)

	tests := []struct {
		priority        int
		expectedReviewers int
		expectedRetries  int
	}{
		{1, 5, 10}, // Critical
		{2, 3, 5},  // High
		{3, 1, 3},  // Medium
		{4, 1, 2},  // Low
		{5, 1, 2},  // Low
	}

	for _, test := range tests {
		reviewers := reader.getReviewerCountFromPriority(test.priority)
		if reviewers != test.expectedReviewers {
			t.Fatalf("Priority %d: expected %d reviewers, got %d", test.priority, test.expectedReviewers, reviewers)
		}

		retries := reader.getRetriesFromPriority(test.priority)
		if retries != test.expectedRetries {
			t.Fatalf("Priority %d: expected %d retries, got %d", test.priority, test.expectedRetries, retries)
		}
	}
}

// TestBeadsReaderLabelHandling tests label-based configuration
func TestBeadsReaderLabelHandling(t *testing.T) {
	logger := &MockLogger{}
	reader := NewBeadsTaskReader(logger)

	issue := BeadsIssue{
		ID:          "open-swarm-001",
		Title:       "Test",
		Description: "Test",
		Status:      "open",
		Priority:    3,
		Labels:      []string{"needs-parallel-review"},
	}

	config, _ := reader.ReadFromIssue(issue)

	if config.ReviewersCount != 3 {
		t.Fatalf("Expected 3 reviewers with needs-parallel-review label, got %d", config.ReviewersCount)
	}

	// Test high-complexity label
	issue.Labels = []string{"high-complexity"}
	config, _ = reader.ReadFromIssue(issue)

	if config.ReviewersCount != 5 {
		t.Fatalf("Expected 5 reviewers with high-complexity label, got %d", config.ReviewersCount)
	}

	if config.MaxRetries != 5 {
		t.Fatalf("Expected 5 max retries with high-complexity label, got %d", config.MaxRetries)
	}
}

// TestBeadsReaderHasLabel tests label detection
func TestBeadsReaderHasLabel(t *testing.T) {
	logger := &MockLogger{}
	reader := NewBeadsTaskReader(logger)

	labels := []string{"bug", "URGENT", "needs-review"}

	tests := []struct {
		target   string
		expected bool
	}{
		{"bug", true},
		{"BUG", true},
		{"urgent", true},
		{"URGENT", true},
		{"needs-review", true},
		{"NEEDS-REVIEW", true},
		{"notfound", false},
		{"review", false},
	}

	for _, test := range tests {
		result := reader.hasLabel(labels, test.target)
		if result != test.expected {
			t.Fatalf("hasLabel(%s): expected %v, got %v", test.target, test.expected, result)
		}
	}
}

// TestBeadsReaderGetBatchMetadata tests batch metadata calculation
func TestBeadsReaderGetBatchMetadata(t *testing.T) {
	logger := &MockLogger{}
	reader := NewBeadsTaskReader(logger)

	issues := []BeadsIssue{
		{ID: "1", Title: "T1", Description: "D1", Status: "open", Priority: 1, Type: "feature", EstimatedTokens: 100},
		{ID: "2", Title: "T2", Description: "D2", Status: "open", Priority: 2, Type: "bug", EstimatedTokens: 50},
		{ID: "3", Title: "T3", Description: "D3", Status: "open", Priority: 2, Type: "feature", EstimatedTokens: 75},
	}

	meta := reader.GetBatchMetadata(issues)

	if meta.TotalTasks != 3 {
		t.Fatalf("Expected 3 total tasks, got %d", meta.TotalTasks)
	}

	if meta.EstimatedTokens != 225 {
		t.Fatalf("Expected 225 total tokens, got %d", meta.EstimatedTokens)
	}

	if meta.ByPriority[1] != 1 {
		t.Fatalf("Expected 1 priority 1 task, got %d", meta.ByPriority[1])
	}

	if meta.ByPriority[2] != 2 {
		t.Fatalf("Expected 2 priority 2 tasks, got %d", meta.ByPriority[2])
	}

	if meta.ByType["feature"] != 2 {
		t.Fatalf("Expected 2 feature tasks, got %d", meta.ByType["feature"])
	}

	if meta.AverageReviewers <= 0 {
		t.Fatalf("Expected positive average reviewers, got %f", meta.AverageReviewers)
	}
}

// TestBeadsReaderRequirementConversion tests conversion to gates.Requirement
func TestBeadsReaderRequirementConversion(t *testing.T) {
	logger := &MockLogger{}
	reader := NewBeadsTaskReader(logger)

	issue := BeadsIssue{
		ID:          "open-swarm-001",
		Title:       "Auth Feature",
		Description: "Implement authentication\n\nScenarios:\n- User login\n\nEdge Cases:\n- Invalid token",
		Acceptance:  "Login works, sessions are secure",
		Status:      "open",
	}

	config, _ := reader.ReadFromIssue(issue)

	if config.RequirementsForGate == nil {
		t.Fatal("RequirementsForGate is nil")
	}

	req := config.RequirementsForGate
	if req.TaskID != "open-swarm-001" {
		t.Fatalf("Expected task ID, got %s", req.TaskID)
	}

	if req.Title != "Auth Feature" {
		t.Fatalf("Expected title, got %s", req.Title)
	}

	if req.Acceptance != "Login works, sessions are secure" {
		t.Fatalf("Expected acceptance criteria, got %s", req.Acceptance)
	}
}

// TestBeadsReaderEmptyDescription tests handling of empty descriptions
func TestBeadsReaderEmptyDescription(t *testing.T) {
	logger := &MockLogger{}
	reader := NewBeadsTaskReader(logger)

	issue := BeadsIssue{
		ID:          "open-swarm-001",
		Title:       "Task",
		Description: "", // Empty
		Status:      "open",
	}

	config, err := reader.ReadFromIssue(issue)
	if err != nil {
		t.Fatalf("ReadFromIssue failed: %v", err)
	}

	// Should have default scenario from description
	if len(config.Scenarios) != 1 {
		t.Fatalf("Expected 1 default scenario, got %d", len(config.Scenarios))
	}
}

// TestBeadsReaderDefaultValues tests default configuration values
func TestBeadsReaderDefaultValues(t *testing.T) {
	logger := &MockLogger{}
	reader := NewBeadsTaskReader(logger)

	issue := BeadsIssue{
		ID:          "open-swarm-001",
		Title:       "Task",
		Description: "Description",
		Status:      "open",
		Priority:    3, // Medium
	}

	config, _ := reader.ReadFromIssue(issue)

	if config.MaxRetries != 3 {
		t.Fatalf("Expected MaxRetries=3 for priority 3, got %d", config.MaxRetries)
	}

	if config.TimeoutSeconds != 300 {
		t.Fatalf("Expected TimeoutSeconds=300, got %d", config.TimeoutSeconds)
	}

	if config.ReviewersCount != 1 {
		t.Fatalf("Expected ReviewersCount=1 for priority 3, got %d", config.ReviewersCount)
	}
}
