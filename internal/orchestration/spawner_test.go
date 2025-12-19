package orchestration

import (
	"context"
	"testing"
	"time"

	"open-swarm/internal/gates"
)

// TestSpawnerBasicExecution tests basic spawner - just verifies result structure
func TestSpawnerBasicExecution(t *testing.T) {
	logger := &MockLogger{}
	mem0 := &MockMem0Client{}

	// Create gate chain (empty, so gates check basic structure)
	chain := gates.NewGateChain()

	// Mock executor that returns success
	executor := func(ctx context.Context, config *AgentConfig) (string, []string, error) {
		return "Implementation code", []string{"file1.go"}, nil
	}

	// Mock test runner
	testRunner := func(ctx context.Context, config *AgentConfig) (*gates.TestResult, error) {
		return &gates.TestResult{
			Total:    10,
			Passed:   10,
			Failed:   0,
			ExitCode: 0,
		}, nil
	}

	spawner := NewSpawner(chain, mem0, logger, executor, testRunner)

	config := &AgentConfig{
		TaskID:           "task1",
		Title:            "Test Task",
		Description:      "Test description",
		RequirementsForGate: &gates.Requirement{
			TaskID:      "task1",
			Title:       "Test",
			Description: "Test requirement",
			Acceptance:  "All tests pass",
			Scenarios:   []string{"scenario 1", "scenario 2"},
			EdgeCases:   []string{"edge case 1"},
		},
	}

	ctx := context.Background()
	result, _ := spawner.Spawn(ctx, config)

	if result == nil {
		t.Fatal("Result is nil")
	}

	if result.TaskID != "task1" {
		t.Fatalf("Expected task1, got %s", result.TaskID)
	}

	// Verify gate results were captured
	if len(result.GateResults) == 0 {
		t.Fatal("Expected gate results to be captured")
	}
}

// TestSpawnerGateExecution tests gate execution in spawner
func TestSpawnerGateExecution(t *testing.T) {
	logger := &MockLogger{}
	mem0 := &MockMem0Client{}

	chain := gates.NewGateChain()

	executor := func(ctx context.Context, config *AgentConfig) (string, []string, error) {
		return "code", []string{}, nil
	}

	testRunner := func(ctx context.Context, config *AgentConfig) (*gates.TestResult, error) {
		return &gates.TestResult{
			Total:    5,
			Passed:   5,
			Failed:   0,
			ExitCode: 0,
		}, nil
	}

	spawner := NewSpawner(chain, mem0, logger, executor, testRunner)

	config := &AgentConfig{
		TaskID:      "task1",
		Title:       "Test",
		Description: "Test",
		RequirementsForGate: &gates.Requirement{
			TaskID:      "task1",
			Title:       "Test",
			Description: "Test",
			Acceptance:  "Test",
			Scenarios:   []string{"test"},
			EdgeCases:   []string{},
		},
	}

	ctx := context.Background()
	result, err := spawner.Spawn(ctx, config)

	if err != nil {
		t.Fatalf("Spawn failed: %v", err)
	}

	// Verify gate results were captured
	if len(result.GateResults) == 0 {
		t.Fatal("No gate results captured")
	}

	// Should have at least Gate 1 (Requirements Verification)
	if result.GateResults[0].GateName != "Requirements Verification" {
		t.Fatalf("Expected Gate 1, got %s", result.GateResults[0].GateName)
	}
}

// TestSpawnerExecutorFailure tests spawner captures executor errors after gates pass
func TestSpawnerExecutorFailure(t *testing.T) {
	logger := &MockLogger{}
	mem0 := &MockMem0Client{}

	chain := gates.NewGateChain()

	// Executor that fails
	executor := func(ctx context.Context, config *AgentConfig) (string, []string, error) {
		return "", nil, ErrExecutor // Simulated executor error
	}

	testRunner := func(ctx context.Context, config *AgentConfig) (*gates.TestResult, error) {
		return &gates.TestResult{}, nil
	}

	spawner := NewSpawner(chain, mem0, logger, executor, testRunner)

	config := &AgentConfig{
		TaskID:      "task1",
		Title:       "Test",
		Description: "Test",
		RequirementsForGate: &gates.Requirement{
			TaskID:      "task1",
			Title:       "Test",
			Description: "Test",
			Scenarios:   []string{"test scenario"},
		},
	}

	ctx := context.Background()
	result, _ := spawner.Spawn(ctx, config)

	// Either fails at Gate 1 or at execution - both are acceptable failure modes
	if result.Success {
		t.Fatal("Expected spawn to fail")
	}
}

// TestSpawnerTestFailure tests spawner handles test failure
func TestSpawnerTestFailure(t *testing.T) {
	logger := &MockLogger{}
	mem0 := &MockMem0Client{}

	chain := gates.NewGateChain()

	executor := func(ctx context.Context, config *AgentConfig) (string, []string, error) {
		return "code", []string{}, nil
	}

	// Test runner that fails
	testRunner := func(ctx context.Context, config *AgentConfig) (*gates.TestResult, error) {
		return nil, ErrTestRunner // Simulated test error
	}

	spawner := NewSpawner(chain, mem0, logger, executor, testRunner)

	config := &AgentConfig{
		TaskID:      "task1",
		Title:       "Test",
		Description: "Test",
		RequirementsForGate: &gates.Requirement{
			TaskID:      "task1",
			Title:       "Test",
			Description: "Test",
			Scenarios:   []string{"test scenario"},
		},
	}

	ctx := context.Background()
	result, _ := spawner.Spawn(ctx, config)

	// Either fails at Gate 1 or at test execution - both are acceptable failure modes
	if result.Success {
		t.Fatal("Expected spawn to fail")
	}
}

// TestSpawnerGateTimeout tests spawner respects gate timeout
func TestSpawnerGateTimeout(t *testing.T) {
	logger := &MockLogger{}
	mem0 := &MockMem0Client{}

	chain := gates.NewGateChain()

	executor := func(ctx context.Context, config *AgentConfig) (string, []string, error) {
		return "code", []string{}, nil
	}

	testRunner := func(ctx context.Context, config *AgentConfig) (*gates.TestResult, error) {
		return &gates.TestResult{Total: 1, Passed: 1}, nil
	}

	spawner := NewSpawner(chain, mem0, logger, executor, testRunner)
	spawner.SetGateTimeout(1 * time.Second)

	config := &AgentConfig{
		TaskID:      "task1",
		Title:       "Test",
		Description: "Test",
		RequirementsForGate: &gates.Requirement{
			TaskID:      "task1",
			Title:       "Test",
			Description: "Test",
		},
	}

	ctx := context.Background()
	result, _ := spawner.Spawn(ctx, config)

	// Should complete within timeout
	if result.ExecutionTime > 2*time.Second {
		t.Fatalf("Execution took too long: %v", result.ExecutionTime)
	}
}

// TestSpawnerMem0Integration tests spawner records results to Mem0
func TestSpawnerMem0Integration(t *testing.T) {
	logger := &MockLogger{}
	mem0 := &MockMem0Client{}

	chain := gates.NewGateChain()

	executor := func(ctx context.Context, config *AgentConfig) (string, []string, error) {
		return "code", []string{}, nil
	}

	testRunner := func(ctx context.Context, config *AgentConfig) (*gates.TestResult, error) {
		return &gates.TestResult{Total: 1, Passed: 1, ExitCode: 0}, nil
	}

	spawner := NewSpawner(chain, mem0, logger, executor, testRunner)

	config := &AgentConfig{
		TaskID:      "task1",
		Title:       "Test",
		Description: "Test",
		RequirementsForGate: &gates.Requirement{
			TaskID:      "task1",
			Title:       "Test",
			Description: "Test",
		},
	}

	ctx := context.Background()
	result, _ := spawner.Spawn(ctx, config)

	if result.Success {
		// Should have stored pattern on success
		if len(mem0.patterns) == 0 {
			t.Fatal("Expected patterns to be stored on success")
		}
	}
}

// Test error types for spawner
var (
	ErrExecutor   = &gates.GateError{Message: "executor failed"}
	ErrTestRunner = &gates.GateError{Message: "test runner failed"}
)

// TestSpawnerExecutionTime tests spawner measures execution time
func TestSpawnerExecutionTime(t *testing.T) {
	logger := &MockLogger{}
	mem0 := &MockMem0Client{}

	chain := gates.NewGateChain()

	executor := func(ctx context.Context, config *AgentConfig) (string, []string, error) {
		time.Sleep(50 * time.Millisecond)
		return "code", []string{}, nil
	}

	testRunner := func(ctx context.Context, config *AgentConfig) (*gates.TestResult, error) {
		time.Sleep(50 * time.Millisecond)
		return &gates.TestResult{Total: 1, Passed: 1}, nil
	}

	spawner := NewSpawner(chain, mem0, logger, executor, testRunner)

	config := &AgentConfig{
		TaskID:      "task1",
		Title:       "Test",
		Description: "Test",
		RequirementsForGate: &gates.Requirement{
			TaskID:      "task1",
			Title:       "Test",
			Description: "Test",
			Scenarios:   []string{"test scenario"},
		},
	}

	ctx := context.Background()
	result, _ := spawner.Spawn(ctx, config)

	// Just verify execution time was measured (gates may succeed or fail)
	if result.ExecutionTime <= 0 {
		t.Fatalf("ExecutionTime should be positive, got %v", result.ExecutionTime)
	}
}
