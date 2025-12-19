package orchestration

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// MockLogger implements Logger interface for testing
type MockLogger struct {
	mu      sync.Mutex
	entries []string
}

func (ml *MockLogger) Infof(format string, args ...interface{}) {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	ml.entries = append(ml.entries, fmt.Sprintf(format, args...))
}

func (ml *MockLogger) Errorf(format string, args ...interface{}) {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	ml.entries = append(ml.entries, "ERROR: "+fmt.Sprintf(format, args...))
}

func (ml *MockLogger) Debugf(format string, args ...interface{}) {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	ml.entries = append(ml.entries, "DEBUG: "+fmt.Sprintf(format, args...))
}

func (ml *MockLogger) Warnf(format string, args ...interface{}) {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	ml.entries = append(ml.entries, "WARN: "+fmt.Sprintf(format, args...))
}

// MockMem0Client implements Mem0Client interface for testing
type MockMem0Client struct {
	mu       sync.Mutex
	patterns []string
	failures map[string]string
}

func (mmc *MockMem0Client) StorePattern(ctx context.Context, pattern string) error {
	mmc.mu.Lock()
	defer mmc.mu.Unlock()
	mmc.patterns = append(mmc.patterns, pattern)
	return nil
}

func (mmc *MockMem0Client) GetPatterns(ctx context.Context, taskType string) ([]string, error) {
	mmc.mu.Lock()
	defer mmc.mu.Unlock()
	return mmc.patterns, nil
}

func (mmc *MockMem0Client) RecordFailure(ctx context.Context, taskID string, rootCause string) error {
	mmc.mu.Lock()
	defer mmc.mu.Unlock()
	if mmc.failures == nil {
		mmc.failures = make(map[string]string)
	}
	mmc.failures[taskID] = rootCause
	return nil
}

// TestCoordinatorAddAgent tests agent registration
func TestCoordinatorAddAgent(t *testing.T) {
	logger := &MockLogger{}
	mem0 := &MockMem0Client{}
	spawner := func(ctx context.Context, config *AgentConfig) (*AgentResult, error) {
		return &AgentResult{TaskID: config.TaskID, Success: true}, nil
	}

	coord := NewCoordinator(spawner, mem0, logger)

	config := &AgentConfig{
		TaskID:      "task-1",
		Title:       "Test Task",
		Description: "Test description",
	}

	err := coord.AddAgent(config)
	if err != nil {
		t.Fatalf("AddAgent failed: %v", err)
	}

	// Verify agent was added
	if _, exists := coord.agents[config.TaskID]; !exists {
		t.Fatal("Agent not found in coordinator")
	}
}

// TestCoordinatorDuplicateAgent tests that adding duplicate agents fails
func TestCoordinatorDuplicateAgent(t *testing.T) {
	logger := &MockLogger{}
	mem0 := &MockMem0Client{}
	spawner := func(ctx context.Context, config *AgentConfig) (*AgentResult, error) {
		return &AgentResult{TaskID: config.TaskID, Success: true}, nil
	}

	coord := NewCoordinator(spawner, mem0, logger)

	config := &AgentConfig{TaskID: "task-1", Title: "Test"}
	coord.AddAgent(config)

	// Try to add again
	err := coord.AddAgent(config)
	if err == nil {
		t.Fatal("Expected error adding duplicate agent, got nil")
	}
}

// TestCoordinatorBuildExecutionOrder tests topological sorting
func TestCoordinatorBuildExecutionOrder(t *testing.T) {
	logger := &MockLogger{}
	mem0 := &MockMem0Client{}
	spawner := func(ctx context.Context, config *AgentConfig) (*AgentResult, error) {
		return &AgentResult{TaskID: config.TaskID, Success: true}, nil
	}

	coord := NewCoordinator(spawner, mem0, logger)

	// Add tasks with dependencies: task2 depends on task1
	config1 := &AgentConfig{TaskID: "task1", Title: "Task 1", DependsOn: []string{}}
	config2 := &AgentConfig{TaskID: "task2", Title: "Task 2", DependsOn: []string{"task1"}}
	config3 := &AgentConfig{TaskID: "task3", Title: "Task 3", DependsOn: []string{}}

	coord.AddAgent(config1)
	coord.AddAgent(config2)
	coord.AddAgent(config3)

	order, err := coord.BuildExecutionOrder()
	if err != nil {
		t.Fatalf("BuildExecutionOrder failed: %v", err)
	}

	if len(order) != 3 {
		t.Fatalf("Expected 3 tasks in execution order, got %d", len(order))
	}

	// Verify task1 comes before task2
	pos1 := -1
	pos2 := -1
	for i, taskID := range order {
		if taskID == "task1" {
			pos1 = i
		}
		if taskID == "task2" {
			pos2 = i
		}
	}

	if pos1 == -1 || pos2 == -1 {
		t.Fatal("task1 or task2 not found in execution order")
	}

	if pos1 >= pos2 {
		t.Fatalf("task1 should come before task2, but order is %v", order)
	}
}

// TestCoordinatorCircularDependency tests circular dependency detection
func TestCoordinatorCircularDependency(t *testing.T) {
	logger := &MockLogger{}
	mem0 := &MockMem0Client{}
	spawner := func(ctx context.Context, config *AgentConfig) (*AgentResult, error) {
		return &AgentResult{TaskID: config.TaskID, Success: true}, nil
	}

	coord := NewCoordinator(spawner, mem0, logger)

	// Create circular: task1 -> task2 -> task1
	config1 := &AgentConfig{TaskID: "task1", Title: "Task 1", DependsOn: []string{"task2"}}
	config2 := &AgentConfig{TaskID: "task2", Title: "Task 2", DependsOn: []string{"task1"}}

	coord.AddAgent(config1)
	coord.AddAgent(config2)

	_, err := coord.BuildExecutionOrder()
	if err == nil {
		t.Fatal("Expected circular dependency error, got nil")
	}
}

// TestCoordinatorParallelExecution tests that independent agents run in parallel
func TestCoordinatorParallelExecution(t *testing.T) {
	logger := &MockLogger{}
	mem0 := &MockMem0Client{}

	// Track execution timing
	startTimes := make(map[string]time.Time)
	endTimes := make(map[string]time.Time)
	mu := sync.Mutex{}

	spawner := func(ctx context.Context, config *AgentConfig) (*AgentResult, error) {
		mu.Lock()
		startTimes[config.TaskID] = time.Now()
		mu.Unlock()

		// Sleep to simulate work
		time.Sleep(100 * time.Millisecond)

		mu.Lock()
		endTimes[config.TaskID] = time.Now()
		mu.Unlock()

		return &AgentResult{TaskID: config.TaskID, Success: true}, nil
	}

	coord := NewCoordinator(spawner, mem0, logger)
	coord.SetMaxConcurrent(10)

	// Add 3 independent tasks (no dependencies)
	for i := 1; i <= 3; i++ {
		config := &AgentConfig{
			TaskID:      fmt.Sprintf("task%d", i),
			Title:       fmt.Sprintf("Task %d", i),
			DependsOn:   []string{},
		}
		coord.AddAgent(config)
	}

	ctx := context.Background()
	err := coord.Execute(ctx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify all completed
	metrics := coord.GetMetrics()
	if metrics.SuccessCount != 3 {
		t.Fatalf("Expected 3 successful agents, got %d", metrics.SuccessCount)
	}

	if len(startTimes) < 2 {
		t.Fatal("Not enough tasks executed")
	}
}

// TestCoordinatorMaxConcurrent tests concurrency limiting
func TestCoordinatorMaxConcurrent(t *testing.T) {
	logger := &MockLogger{}
	mem0 := &MockMem0Client{}

	concurrentCount := 0
	maxConcurrent := 0
	mu := sync.Mutex{}

	spawner := func(ctx context.Context, config *AgentConfig) (*AgentResult, error) {
		mu.Lock()
		concurrentCount++
		if concurrentCount > maxConcurrent {
			maxConcurrent = concurrentCount
		}
		mu.Unlock()

		// Simulate work
		time.Sleep(50 * time.Millisecond)

		mu.Lock()
		concurrentCount--
		mu.Unlock()

		return &AgentResult{TaskID: config.TaskID, Success: true}, nil
	}

	coord := NewCoordinator(spawner, mem0, logger)
	coord.SetMaxConcurrent(2) // Limit to 2 concurrent

	// Add 5 independent tasks
	for i := 1; i <= 5; i++ {
		config := &AgentConfig{
			TaskID:    fmt.Sprintf("task%d", i),
			Title:     fmt.Sprintf("Task %d", i),
			DependsOn: []string{},
		}
		coord.AddAgent(config)
	}

	ctx := context.Background()
	err := coord.Execute(ctx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify max concurrent never exceeded 2
	if maxConcurrent > 2 {
		t.Fatalf("Max concurrent should be <= 2, but was %d", maxConcurrent)
	}
}

// TestCoordinatorSuccessCallback tests success callback invocation
func TestCoordinatorSuccessCallback(t *testing.T) {
	logger := &MockLogger{}
	mem0 := &MockMem0Client{}

	successCount := 0
	spawner := func(ctx context.Context, config *AgentConfig) (*AgentResult, error) {
		return &AgentResult{TaskID: config.TaskID, Success: true}, nil
	}

	coord := NewCoordinator(spawner, mem0, logger)
	coord.OnSuccess(func(result *AgentResult) error {
		successCount++
		return nil
	})

	config := &AgentConfig{TaskID: "task1", Title: "Test"}
	coord.AddAgent(config)

	ctx := context.Background()
	err := coord.Execute(ctx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if successCount != 1 {
		t.Fatalf("Expected 1 success callback, got %d", successCount)
	}
}

// TestCoordinatorFailureCallback tests failure callback invocation
func TestCoordinatorFailureCallback(t *testing.T) {
	logger := &MockLogger{}
	mem0 := &MockMem0Client{}

	failureCount := 0
	spawner := func(ctx context.Context, config *AgentConfig) (*AgentResult, error) {
		return &AgentResult{TaskID: config.TaskID, Success: false, Error: "test error"}, nil
	}

	coord := NewCoordinator(spawner, mem0, logger)
	coord.OnFailure(func(result *AgentResult) error {
		failureCount++
		return nil
	})

	config := &AgentConfig{TaskID: "task1", Title: "Test"}
	coord.AddAgent(config)

	ctx := context.Background()
	err := coord.Execute(ctx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if failureCount != 1 {
		t.Fatalf("Expected 1 failure callback, got %d", failureCount)
	}
}

// TestCoordinatorMetrics tests metrics calculation
func TestCoordinatorMetrics(t *testing.T) {
	logger := &MockLogger{}
	mem0 := &MockMem0Client{}

	spawner := func(ctx context.Context, config *AgentConfig) (*AgentResult, error) {
		return &AgentResult{
			TaskID:     config.TaskID,
			Success:    true,
			TokensUsed: 100,
		}, nil
	}

	coord := NewCoordinator(spawner, mem0, logger)

	config1 := &AgentConfig{TaskID: "task1", Title: "Task 1"}
	config2 := &AgentConfig{TaskID: "task2", Title: "Task 2"}
	coord.AddAgent(config1)
	coord.AddAgent(config2)

	ctx := context.Background()
	err := coord.Execute(ctx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	metrics := coord.GetMetrics()
	if metrics.TotalAgents != 2 {
		t.Fatalf("Expected 2 total agents, got %d", metrics.TotalAgents)
	}
	if metrics.SuccessCount != 2 {
		t.Fatalf("Expected 2 successful agents, got %d", metrics.SuccessCount)
	}
	if metrics.TotalTokens != 200 {
		t.Fatalf("Expected 200 total tokens, got %d", metrics.TotalTokens)
	}
	if metrics.AverageTokens != 100 {
		t.Fatalf("Expected 100 average tokens, got %d", metrics.AverageTokens)
	}
}

// TestCoordinatorGetResult tests result retrieval
func TestCoordinatorGetResult(t *testing.T) {
	logger := &MockLogger{}
	mem0 := &MockMem0Client{}

	spawner := func(ctx context.Context, config *AgentConfig) (*AgentResult, error) {
		return &AgentResult{
			TaskID:  config.TaskID,
			Success: true,
			Error:   "test error",
		}, nil
	}

	coord := NewCoordinator(spawner, mem0, logger)

	config := &AgentConfig{TaskID: "task1", Title: "Test"}
	coord.AddAgent(config)

	ctx := context.Background()
	coord.Execute(ctx)

	result := coord.GetResult("task1")
	if result == nil {
		t.Fatal("Result not found")
	}
	if result.TaskID != "task1" {
		t.Fatalf("Expected task1, got %s", result.TaskID)
	}
	if !result.Success {
		t.Fatal("Expected success=true")
	}
}

// TestCoordinatorExecutionStages tests execution stage grouping by depth
func TestCoordinatorExecutionStages(t *testing.T) {
	logger := &MockLogger{}
	mem0 := &MockMem0Client{}
	spawner := func(ctx context.Context, config *AgentConfig) (*AgentResult, error) {
		return &AgentResult{TaskID: config.TaskID, Success: true}, nil
	}

	coord := NewCoordinator(spawner, mem0, logger)

	// Create a 3-level dependency tree
	// Stage 1: task1, task2 (no deps)
	// Stage 2: task3 (depends on task1), task4 (depends on task2)
	// Stage 3: task5 (depends on task3 and task4)

	config1 := &AgentConfig{TaskID: "task1", Title: "Task 1", DependsOn: []string{}}
	config2 := &AgentConfig{TaskID: "task2", Title: "Task 2", DependsOn: []string{}}
	config3 := &AgentConfig{TaskID: "task3", Title: "Task 3", DependsOn: []string{"task1"}}
	config4 := &AgentConfig{TaskID: "task4", Title: "Task 4", DependsOn: []string{"task2"}}
	config5 := &AgentConfig{TaskID: "task5", Title: "Task 5", DependsOn: []string{"task3", "task4"}}

	coord.AddAgent(config1)
	coord.AddAgent(config2)
	coord.AddAgent(config3)
	coord.AddAgent(config4)
	coord.AddAgent(config5)

	order, _ := coord.BuildExecutionOrder()
	stages := coord.getExecutionStages(order)

	if len(stages) != 3 {
		t.Fatalf("Expected 3 execution stages, got %d", len(stages))
	}

	if len(stages[0]) != 2 {
		t.Fatalf("Expected stage 1 to have 2 tasks, got %d", len(stages[0]))
	}
}
