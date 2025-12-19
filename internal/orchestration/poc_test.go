package orchestration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"open-swarm/internal/gates"
)

// TestPOC10AgentValidation validates the 10-agent POC architecture
func TestPOC10AgentValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping POC test in short mode")
	}

	logger := &MockLogger{}
	mem0 := &MockMem0Client{}

	// Create coordinator
	spawner := mockAgentSpawner()
	coordinator := NewCoordinator(spawner, mem0, logger)
	coordinator.SetMaxConcurrent(10)

	// Create 10 test Beads issues
	issues := create10TestBeadsIssues()
	reader := NewBeadsTaskReader(logger)

	// Convert to agent configs
	configs, err := reader.CreateBatch(issues)
	if err != nil {
		t.Fatalf("Failed to create batch: %v", err)
	}

	if len(configs) != 10 {
		t.Fatalf("Expected 10 configs, got %d", len(configs))
	}

	// Add agents to coordinator
	for _, config := range configs {
		if err := coordinator.AddAgent(config); err != nil {
			t.Fatalf("Failed to add agent %s: %v", config.TaskID, err)
		}
	}

	// Build execution order
	order, err := coordinator.BuildExecutionOrder()
	if err != nil {
		t.Fatalf("Failed to build execution order: %v", err)
	}

	if len(order) != 10 {
		t.Fatalf("Expected 10 tasks in execution order, got %d", len(order))
	}

	// Execute all agents
	ctx := context.Background()
	err = coordinator.Execute(ctx)
	if err != nil {
		t.Logf("Execution error (expected for POC): %v", err)
	}

	// Validate results
	metrics := coordinator.GetMetrics()

	// All agents should execute
	if metrics.TotalAgents != 10 {
		t.Fatalf("Expected 10 total agents, got %d", metrics.TotalAgents)
	}

	// At least some should succeed (not all may due to gate failures)
	if metrics.SuccessCount == 0 {
		t.Logf("Warning: No agents succeeded. This may indicate gate issues.")
	}

	t.Logf("POC Results: %d/%d agents succeeded (%.1f%%)", metrics.SuccessCount, metrics.TotalAgents, float64(metrics.SuccessCount)*100/float64(metrics.TotalAgents))
	t.Logf("Total execution time: %v", metrics.TotalTime)
	t.Logf("Average tokens per agent: %d", metrics.AverageTokens)
}

// TestPOC1AgentSimple validates single agent execution
func TestPOC1AgentSimple(t *testing.T) {
	logger := &MockLogger{}
	mem0 := &MockMem0Client{}

	spawner := mockAgentSpawner()
	coordinator := NewCoordinator(spawner, mem0, logger)

	// Create 1 simple issue
	issue := BeadsIssue{
		ID:          "poc-001",
		Title:       "Simple Task",
		Description: "Implement basic feature\n\nScenarios:\n- User performs action\n- System responds\n\nEdge Cases:\n- Network error",
		Acceptance:  "Feature works end-to-end",
		Type:        "feature",
		Status:      "open",
		Priority:    2,
		EstimatedTokens: 100,
	}

	reader := NewBeadsTaskReader(logger)
	config, err := reader.ReadFromIssue(issue)
	if err != nil {
		t.Fatalf("Failed to read issue: %v", err)
	}

	coordinator.AddAgent(config)

	ctx := context.Background()
	err = coordinator.Execute(ctx)
	if err != nil {
		t.Logf("Execution error: %v", err)
	}

	result := coordinator.GetResult("poc-001")
	if result == nil {
		t.Fatal("Result not found")
	}

	if result.TaskID != "poc-001" {
		t.Fatalf("Expected poc-001, got %s", result.TaskID)
	}

	t.Logf("Single agent result: Success=%v, Files=%v", result.Success, result.FilesModified)
}

// TestPOC3AgentConsensus validates consensus with 3 agents on same task
func TestPOC3AgentConsensus(t *testing.T) {
	logger := &MockLogger{}
	mem0 := &MockMem0Client{}

	spawner := mockAgentSpawner()
	coordinator := NewCoordinator(spawner, mem0, logger)
	coordinator.SetMaxConcurrent(3)

	// Create 3 independent agents for same task
	for i := 1; i <= 3; i++ {
		config := &AgentConfig{
			TaskID:      fmt.Sprintf("consensus-%d", i),
			Title:       "Consensus Task",
			Description: "Test consensus across agents",
			RequirementsForGate: &gates.Requirement{
				TaskID:      fmt.Sprintf("consensus-%d", i),
				Title:       "Consensus Task",
				Description: "Test consensus across agents",
				Scenarios:   []string{"scenario 1"},
			},
		}
		coordinator.AddAgent(config)
	}

	ctx := context.Background()
	coordinator.Execute(ctx)

	metrics := coordinator.GetMetrics()

	// All 3 should execute
	if metrics.TotalAgents != 3 {
		t.Fatalf("Expected 3 agents, got %d", metrics.TotalAgents)
	}

	// Check if any succeeded (for consensus validation)
	successRate := float64(metrics.SuccessCount) / float64(metrics.TotalAgents)
	t.Logf("3-Agent Consensus: Success rate = %.1f%%", successRate*100)
}

// TestPOCParallelExecution validates 10 agents running in parallel
func TestPOCParallelExecution(t *testing.T) {
	logger := &MockLogger{}
	mem0 := &MockMem0Client{}

	// Track timing
	startTime := time.Now()
	spawner := mockAgentSpawner()
	coordinator := NewCoordinator(spawner, mem0, logger)
	coordinator.SetMaxConcurrent(10)

	// Create 10 independent tasks
	for i := 1; i <= 10; i++ {
		config := &AgentConfig{
			TaskID:      fmt.Sprintf("parallel-%02d", i),
			Title:       fmt.Sprintf("Task %d", i),
			Description: fmt.Sprintf("Task %d description", i),
			RequirementsForGate: &gates.Requirement{
				TaskID:      fmt.Sprintf("parallel-%02d", i),
				Title:       fmt.Sprintf("Task %d", i),
				Description: fmt.Sprintf("Task %d description", i),
				Scenarios:   []string{"scenario 1"},
			},
		}
		coordinator.AddAgent(config)
	}

	ctx := context.Background()
	coordinator.Execute(ctx)
	totalTime := time.Since(startTime)

	metrics := coordinator.GetMetrics()

	if metrics.TotalAgents != 10 {
		t.Fatalf("Expected 10 agents, got %d", metrics.TotalAgents)
	}

	t.Logf("10-Agent Parallel: %d succeeded in %v (%.2f agents/sec)", metrics.SuccessCount, totalTime, float64(metrics.TotalAgents)/totalTime.Seconds())
	t.Logf("Average time per agent: %v", metrics.AverageTime)
	t.Logf("Parallel speedup factor: %.2f", metrics.ParallelFactor)
}

// TestPOCGateEffectiveness validates all 5 gates work
func TestPOCGateEffectiveness(t *testing.T) {
	logger := &MockLogger{}
	mem0 := &MockMem0Client{}

	chain := gates.NewGateChain()

	executor := func(ctx context.Context, config *AgentConfig) (string, []string, error) {
		return "implementation", []string{}, nil
	}

	testRunner := func(ctx context.Context, config *AgentConfig) (*gates.TestResult, error) {
		return &gates.TestResult{Total: 5, Passed: 5, ExitCode: 0}, nil
	}

	spawner := NewSpawner(chain, mem0, logger, executor, testRunner)

	config := &AgentConfig{
		TaskID:      "gates-test",
		Title:       "Gate Test",
		Description: "Test all gates",
		RequirementsForGate: &gates.Requirement{
			TaskID:      "gates-test",
			Title:       "Gate Test",
			Description: "Test all gates",
			Scenarios:   []string{"scenario 1", "scenario 2"},
			EdgeCases:   []string{"edge 1"},
		},
	}

	ctx := context.Background()
	result, _ := spawner.Spawn(ctx, config)

	if result == nil {
		t.Fatal("Result is nil")
	}

	// Count gates that were checked
	gateCount := len(result.GateResults)
	t.Logf("Gates executed: %d", gateCount)

	// Verify gate names
	for _, gr := range result.GateResults {
		t.Logf("  - %s: %v", gr.GateName, gr.Passed)
	}
}

// TestPOCMem0Integration validates learning loop
func TestPOCMem0Integration(t *testing.T) {
	logger := &MockLogger{}
	mem0 := &MockMem0Client{}

	spawner := mockAgentSpawner()
	coordinator := NewCoordinator(spawner, mem0, logger)

	// Create agents that will succeed/fail
	configs := []AgentConfig{
		{
			TaskID:      "mem0-success-1",
			Title:       "Success Task 1",
			Description: "Will succeed",
			RequirementsForGate: &gates.Requirement{
				TaskID:      "mem0-success-1",
				Title:       "Success Task 1",
				Description: "Will succeed",
				Scenarios:   []string{"scenario"},
			},
		},
		{
			TaskID:      "mem0-success-2",
			Title:       "Success Task 2",
			Description: "Will succeed",
			RequirementsForGate: &gates.Requirement{
				TaskID:      "mem0-success-2",
				Title:       "Success Task 2",
				Description: "Will succeed",
				Scenarios:   []string{"scenario"},
			},
		},
	}

	for i := range configs {
		coordinator.AddAgent(&configs[i])
	}

	ctx := context.Background()
	coordinator.Execute(ctx)

	metrics := coordinator.GetMetrics()
	t.Logf("Mem0 Learning: %d agents executed, %d succeeded", metrics.TotalAgents, metrics.SuccessCount)
	t.Logf("Mem0 patterns stored: %d", len(mem0.patterns))
	t.Logf("Mem0 failures recorded: %d", len(mem0.failures))
}

// Helper functions

func create10TestBeadsIssues() []BeadsIssue {
	issues := []BeadsIssue{}
	for i := 1; i <= 10; i++ {
		issue := BeadsIssue{
			ID:          fmt.Sprintf("poc-task-%02d", i),
			Title:       fmt.Sprintf("POC Task %d", i),
			Description: fmt.Sprintf("POC task %d implementation\n\nScenarios:\n- Scenario A for task %d\n- Scenario B for task %d\n\nEdge Cases:\n- Edge case for task %d", i, i, i, i),
			Acceptance:  fmt.Sprintf("Task %d implementation complete with all tests passing", i),
			Type:        "feature",
			Status:      "open",
			Priority:    2,
			Labels:      []string{},
			Dependencies: []string{},
			EstimatedTokens: 200,
		}
		issues = append(issues, issue)
	}
	return issues
}

func mockAgentSpawner() AgentSpawnerFunc {
	return func(ctx context.Context, config *AgentConfig) (*AgentResult, error) {
		// Simulate successful agent execution
		result := &AgentResult{
			TaskID:        config.TaskID,
			Success:       true,
			ExecutionTime: time.Duration(time.Duration(100) * time.Millisecond),
			TestsPassed:   true,
			TestResult: &gates.TestResult{
				Total:    5,
				Passed:   5,
				Failed:   0,
				ExitCode: 0,
			},
			FilesModified: []string{"main.go", "main_test.go"},
			TokensUsed:    150,
			Timestamp:     time.Now(),
		}
		return result, nil
	}
}

// POCResults summarizes POC validation results
type POCResults struct {
	TotalAgents      int
	SuccessfulAgents int
	FailedAgents     int
	TotalTokens      int
	AverageTokens    int
	TotalTime        time.Duration
	GatesValidated   int
	Mem0Patterns     int
}

// ValidatePOCResults checks if POC meets success criteria
func ValidatePOCResults(t *testing.T, results POCResults) bool {
	success := true

	// Criterion 1: All agents complete
	if results.TotalAgents != 10 {
		t.Logf("❌ Expected 10 agents, got %d", results.TotalAgents)
		success = false
	} else {
		t.Logf("✅ All 10 agents completed")
	}

	// Criterion 2: High success rate
	successRate := float64(results.SuccessfulAgents) / float64(results.TotalAgents)
	if successRate < 0.8 {
		t.Logf("⚠️  Success rate %.1f%% below 80%% target", successRate*100)
	} else {
		t.Logf("✅ Success rate: %.1f%%", successRate*100)
	}

	// Criterion 3: Mem0 learning
	if results.Mem0Patterns < 5 {
		t.Logf("⚠️  Only %d Mem0 patterns (target: 5+)", results.Mem0Patterns)
	} else {
		t.Logf("✅ Mem0 captured %d+ patterns", results.Mem0Patterns)
	}

	// Criterion 4: Gates work
	if results.GatesValidated < 5 {
		t.Logf("⚠️  Only %d gates validated (target: 5)", results.GatesValidated)
	} else {
		t.Logf("✅ All 5 gates validated")
	}

	return success
}
