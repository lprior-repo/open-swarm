package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"open-swarm/internal/gates"
	"open-swarm/internal/orchestration"
)

func main() {
	// Parse flags
	agentCount := flag.Int("agents", 24, "Number of concurrent agents to spawn")
	taskLimit := flag.Int("tasks", 30, "Max tasks to process")
	timeout := flag.Duration("timeout", 5*time.Minute, "Timeout per agent execution")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	// Create logger
	logger := &SimpleLogger{}

	// Create coordinator
	coordinator := orchestration.NewCoordinator(
		mockSpawner(logger),
		&MockMem0Client{},
		logger,
	)

	// Configure for 24 agents
	coordinator.SetMaxConcurrent(*agentCount)
	logger.Infof("🚀 Starting 24-Agent Swarm Orchestration")
	logger.Infof("Max concurrent agents: %d", *agentCount)
	logger.Infof("Task limit: %d", *taskLimit)

	// Setup callbacks
	coordinator.OnSuccess(func(result *orchestration.AgentResult) error {
		logger.Infof("✅ SUCCESS [%s] - Execution time: %v", result.TaskID, result.ExecutionTime)
		return nil
	})

	coordinator.OnFailure(func(result *orchestration.AgentResult) error {
		logger.Errorf("❌ FAILED [%s] - %s", result.TaskID, result.FailureReason)
		return nil
	})

	// Load mock tasks (in real implementation, these would come from Beads)
	loadMockTasks(coordinator, *taskLimit)

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Execute agents
	go func() {
		if err := coordinator.Execute(ctx); err != nil {
			logger.Errorf("Execution error: %v", err)
		}
	}()

	// Wait for completion or signal
	<-sigChan
	logger.Infof("Received shutdown signal")
	cancel()

	// Print final metrics
	printMetrics(coordinator, logger)
}

// mockSpawner creates a mock executor function for testing
func mockSpawner(logger orchestration.Logger) orchestration.AgentSpawnerFunc {
	return func(ctx context.Context, config *orchestration.AgentConfig) (*orchestration.AgentResult, error) {
		logger.Infof("[%s] Executing agent for task: %s", config.TaskID, config.Title)

		result := &orchestration.AgentResult{
			TaskID:        config.TaskID,
			ExecutionTime: time.Duration(100+len(config.TaskID)%200) * time.Millisecond,
			TestsPassed:   true,
			Success:       true,
			FilesModified: []string{"main.go", "test_main.go"},
			TokensUsed:    2000 + len(config.TaskID)%1000,
			Timestamp:     time.Now(),
		}

		// Simulate 5 gates passing
		gateNames := []string{
			"Requirements Verification",
			"Test Immutability",
			"Empirical Honesty",
			"Hard Work Enforcement",
			"Requirement Drift Detection",
		}

		for _, gateName := range gateNames {
			result.GateResults = append(result.GateResults, orchestration.GateCheckResult{
				GateName: gateName,
				Passed:   true,
				Message:  "Gate passed",
				Duration: 50 * time.Millisecond,
			})
		}

		result.TestResult = &gates.TestResult{
			Total:    10,
			Passed:   10,
			Failed:   0,
			Output:   "All tests passed",
			ExitCode: 0,
		}

		return result, nil
	}
}

// loadMockTasks adds mock tasks to the coordinator
func loadMockTasks(c *orchestration.Coordinator, count int) {
	tasks := []struct {
		id    string
		title string
		deps  []string
	}{
		{"open-swarm-m0b2", "Implement Test Immutability Lock", []string{}},
		{"open-swarm-wh0l", "Implement Empirical Honesty Output", []string{"open-swarm-m0b2"}},
		{"open-swarm-5v70", "Implement Hard Work Enforcement", []string{"open-swarm-m0b2"}},
		{"open-swarm-24jq", "Implement Requirement Drift Detection", []string{"open-swarm-m0b2"}},
		{"open-swarm-o8pt", "Temporal Agent Spawning & Lifecycle", []string{}},
		{"open-swarm-b08", "Build complete test generation prompt", []string{}},
		{"open-swarm-ura", "Build basic impl prompt with test contents", []string{"open-swarm-b08"}},
		{"open-swarm-inw", "Build impl prompt with output path", []string{"open-swarm-b08"}},
		{"open-swarm-gso", "Build impl prompt with test failures", []string{"open-swarm-b08"}},
		{"open-swarm-qs9", "Build impl prompt with review feedback", []string{"open-swarm-b08"}},
		{"open-swarm-lex", "Build complete implementation prompt", []string{"open-swarm-ura", "open-swarm-inw", "open-swarm-gso", "open-swarm-qs9"}},
		{"open-swarm-7nh", "Build base review prompt structure", []string{}},
		{"open-swarm-mh4", "Create test-generator.yaml", []string{"open-swarm-b08"}},
		{"open-swarm-h3f", "Create implementation.yaml", []string{"open-swarm-lex"}},
		{"open-swarm-jww", "Create reviewer-testing.yaml", []string{"open-swarm-7nh"}},
		{"open-swarm-ta5", "Create reviewer-functional.yaml", []string{"open-swarm-7nh"}},
		{"open-swarm-vzk", "Create reviewer-architecture.yaml", []string{"open-swarm-7nh"}},
		{"open-swarm-3iv", "InvokeAgent: basic SDK call", []string{}},
		{"open-swarm-115", "InvokeAgent: agent selection", []string{"open-swarm-3iv"}},
		{"open-swarm-b4u", "InvokeAgent: result parsing", []string{"open-swarm-3iv"}},
		{"open-swarm-y3m", "InvokeAgent: error handling", []string{"open-swarm-3iv"}},
		{"open-swarm-t3s", "RunLint: basic execution", []string{}},
		{"open-swarm-ah6", "RunLint: output parsing", []string{"open-swarm-t3s"}},
		{"open-swarm-28a", "RunLint: error handling", []string{"open-swarm-t3s"}},
		{"open-swarm-dp6", "RunTests: basic execution", []string{}},
		{"open-swarm-9ll", "RunTests: pass detection", []string{"open-swarm-dp6"}},
	}

	for i, task := range tasks {
		if i >= count {
			break
		}

		config := &orchestration.AgentConfig{
			TaskID:             task.id,
			Title:              task.title,
			Description:        fmt.Sprintf("Implement: %s", task.title),
			AcceptanceCriteria: "Implementation must pass all tests and gates",
			DependsOn:          task.deps,
			MaxRetries:         2,
			TimeoutSeconds:     30,
		}

		if err := c.AddAgent(config); err != nil {
			log.Printf("Failed to add agent %s: %v", task.id, err)
		}
	}
}

// printMetrics outputs execution metrics
func printMetrics(c *orchestration.Coordinator, logger orchestration.Logger) {
	metrics := c.GetMetrics()

	logger.Infof("")
	logger.Infof("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	logger.Infof("📊 Execution Metrics")
	logger.Infof("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	logger.Infof("Total Agents:        %d", metrics.TotalAgents)
	logger.Infof("Successful:          %d (%.1f%%)", metrics.SuccessCount, float64(metrics.SuccessCount)*100/float64(metrics.TotalAgents))
	logger.Infof("Failed:              %d (%.1f%%)", metrics.FailureCount, float64(metrics.FailureCount)*100/float64(metrics.TotalAgents))
	logger.Infof("Total Time:          %v", metrics.TotalTime)
	logger.Infof("Average per Agent:   %v", metrics.AverageTime)
	logger.Infof("Total Tokens Used:   %d", metrics.TotalTokens)
	logger.Infof("Avg Tokens/Agent:    %d", metrics.AverageTokens)
	logger.Infof("Parallel Speedup:    %.2fx", metrics.ParallelFactor)
	logger.Infof("")

	if len(metrics.GatePassRate) > 0 {
		logger.Infof("Gate Pass Rates:")
		for gate, rate := range metrics.GatePassRate {
			logger.Infof("  %s: %.1f%%", gate, rate*100)
		}
		logger.Infof("")
	}
}

// SimpleLogger implements Logger interface
type SimpleLogger struct{}

func (l *SimpleLogger) Infof(format string, args ...interface{}) {
	fmt.Printf("[INFO]  "+format+"\n", args...)
}

func (l *SimpleLogger) Errorf(format string, args ...interface{}) {
	fmt.Printf("[ERROR] "+format+"\n", args...)
}

func (l *SimpleLogger) Debugf(format string, args ...interface{}) {
	fmt.Printf("[DEBUG] "+format+"\n", args...)
}

func (l *SimpleLogger) Warnf(format string, args ...interface{}) {
	fmt.Printf("[WARN]  "+format+"\n", args...)
}

// MockMem0Client implements Mem0Client interface
type MockMem0Client struct{}

func (m *MockMem0Client) StorePattern(ctx context.Context, pattern string) error {
	return nil
}

func (m *MockMem0Client) GetPatterns(ctx context.Context, taskType string) ([]string, error) {
	return []string{}, nil
}

func (m *MockMem0Client) RecordFailure(ctx context.Context, taskID string, rootCause string) error {
	return nil
}
