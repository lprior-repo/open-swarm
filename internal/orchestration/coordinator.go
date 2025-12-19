// Package orchestration provides the core DAG orchestration engine for the 50-agent AI swarm system.
// Coordinates agent spawning, parallel execution, failure recovery, and Mem0 learning integration.
package orchestration

import (
	"context"
	"fmt"
	"sync"
	"time"

	"open-swarm/internal/gates"
)

// AgentConfig represents configuration for a single agent execution.
type AgentConfig struct {
	TaskID              string            // Beads task ID
	Title               string            // Task title
	Description         string            // Task description
	AcceptanceCriteria  string            // Acceptance criteria from Beads
	Scenarios           []string          // Test scenarios
	EdgeCases           []string          // Edge cases
	DependsOn           []string          // Task IDs this depends on
	MaxRetries          int               // Max retries for this task
	TimeoutSeconds      int               // Execution timeout
	ReviewersCount      int               // Parallel reviewers (default 1)
	RequirementsForGate *gates.Requirement // For gate verification
}

// AgentResult represents the outcome of agent execution.
type AgentResult struct {
	TaskID           string              // Beads task ID
	Success          bool                // Overall success
	ExecutionTime    time.Duration       // Total execution time
	TestsPassed      bool                // All tests passed
	TestResult       *gates.TestResult   // Raw test results
	GateResults      []GateCheckResult   // Per-gate results
	FilesModified    []string            // Changed files
	Error            string              // Error message if failed
	FailureReason    string              // Root cause analysis
	TokensUsed       int                 // LLM token consumption
	RetryAttempts    int                 // How many retries needed
	Timestamp        time.Time           // When execution completed
	Mem0Patterns     []string            // Learned patterns to store
	LearningValue    float64             // How much did we learn (0-1)
}

// GateCheckResult represents a single gate check result.
type GateCheckResult struct {
	GateName  string        // Gate type name
	GateType  gates.GateType // Gate type enum
	Passed    bool          // Did it pass?
	Message   string        // Gate message
	Details   string        // Detailed explanation
	Duration  time.Duration // Time taken
	Error     error         // If failed
}

// ExecutionMetrics tracks system-wide metrics.
type ExecutionMetrics struct {
	TotalAgents      int           // Total agents spawned
	SuccessCount     int           // Agents that succeeded
	FailureCount     int           // Agents that failed
	TotalTime        time.Duration // Total execution time
	AverageTime      time.Duration // Average per agent
	TotalTokens      int           // Total tokens used
	AverageTokens    int           // Average tokens per agent
	ParallelFactor   float64       // Speedup vs sequential
	GatePassRate     map[string]float64 // Pass rate per gate
	LearningCount    int           // Patterns learned
	MemoriesStored   int           // Mem0 entries created
}

// Coordinator orchestrates agent execution, manages dependencies, and coordinates failure recovery.
type Coordinator struct {
	mu                 sync.RWMutex
	agents             map[string]*AgentConfig    // Task ID -> config
	results            map[string]*AgentResult    // Task ID -> result
	dependencies       map[string][]string        // Task ID -> dependencies
	executionOrder     []string                   // Topologically sorted order
	metrics            ExecutionMetrics
	gateChain          *gates.GateChain
	failureCallback    func(*AgentResult) error   // Called on failure
	successCallback    func(*AgentResult) error   // Called on success
	spawnerFunc        AgentSpawnerFunc            // Function to spawn agents
	mem0Integration    Mem0Client                  // Mem0 integration
	maxConcurrent      int                        // Max parallel agents
	startTime          time.Time
	logger             Logger
}

// AgentSpawnerFunc is the function signature for spawning agents.
type AgentSpawnerFunc func(ctx context.Context, config *AgentConfig) (*AgentResult, error)

// Mem0Client interface for learning integration.
type Mem0Client interface {
	StorePattern(ctx context.Context, pattern string) error
	GetPatterns(ctx context.Context, taskType string) ([]string, error)
	RecordFailure(ctx context.Context, taskID string, rootCause string) error
}

// Logger interface for structured logging.
type Logger interface {
	Infof(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Debugf(format string, args ...interface{})
	Warnf(format string, args ...interface{})
}

// NewCoordinator creates a new orchestration coordinator.
func NewCoordinator(
	spawner AgentSpawnerFunc,
	mem0 Mem0Client,
	logger Logger,
) *Coordinator {
	return &Coordinator{
		agents:        make(map[string]*AgentConfig),
		results:       make(map[string]*AgentResult),
		dependencies:  make(map[string][]string),
		spawnerFunc:   spawner,
		mem0Integration: mem0,
		maxConcurrent: 10, // Default: 10 parallel agents
		logger:        logger,
		metrics: ExecutionMetrics{
			GatePassRate: make(map[string]float64),
		},
	}
}

// AddAgent registers an agent configuration to be executed.
func (c *Coordinator) AddAgent(config *AgentConfig) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.agents[config.TaskID]; exists {
		return fmt.Errorf("agent %s already registered", config.TaskID)
	}

	c.agents[config.TaskID] = config
	c.dependencies[config.TaskID] = config.DependsOn

	// Validate dependencies exist
	for _, dep := range config.DependsOn {
		if _, exists := c.agents[dep]; !exists {
			c.logger.Warnf("dependency %s for task %s not yet registered", dep, config.TaskID)
		}
	}

	return nil
}

// SetGateChain sets the gate chain to use for verification.
func (c *Coordinator) SetGateChain(chain *gates.GateChain) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.gateChain = chain
}

// SetMaxConcurrent sets the maximum concurrent agents.
func (c *Coordinator) SetMaxConcurrent(max int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if max > 0 {
		c.maxConcurrent = max
	}
}

// OnSuccess registers a callback for successful executions.
func (c *Coordinator) OnSuccess(callback func(*AgentResult) error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.successCallback = callback
}

// OnFailure registers a callback for failed executions.
func (c *Coordinator) OnFailure(callback func(*AgentResult) error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.failureCallback = callback
}

// BuildExecutionOrder creates a topologically sorted execution plan.
func (c *Coordinator) BuildExecutionOrder() ([]string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Kahn's algorithm for topological sort
	inDegree := make(map[string]int)
	adjacency := make(map[string][]string)

	// Initialize
	for taskID := range c.agents {
		inDegree[taskID] = 0
		adjacency[taskID] = []string{}
	}

	// Build graph
	for taskID, deps := range c.dependencies {
		inDegree[taskID] = len(deps)
		for _, dep := range deps {
			adjacency[dep] = append(adjacency[dep], taskID)
		}
	}

	// Topological sort
	queue := []string{}
	for taskID, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, taskID)
		}
	}

	result := []string{}
	for len(queue) > 0 {
		taskID := queue[0]
		queue = queue[1:]
		result = append(result, taskID)

		for _, dependent := range adjacency[taskID] {
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}

	if len(result) != len(c.agents) {
		return nil, fmt.Errorf("circular dependency detected in task graph")
	}

	c.executionOrder = result
	return result, nil
}

// Execute runs all agents respecting dependencies and parallelism constraints.
func (c *Coordinator) Execute(ctx context.Context) error {
	c.mu.Lock()
	c.startTime = time.Now()
	c.mu.Unlock()

	// Build execution order
	order, err := c.BuildExecutionOrder()
	if err != nil {
		return fmt.Errorf("failed to build execution order: %w", err)
	}

	c.logger.Infof("Executing %d agents in %d stages", len(order), len(c.getExecutionStages(order)))

	// Execute in waves respecting dependencies
	completed := make(map[string]bool)
	completionLock := sync.Mutex{}

	for {
		// Find all agents that can run now
		readyAgents := c.getReadyAgents(order, completed)
		if len(readyAgents) == 0 {
			break // No more agents to run
		}

		// Execute ready agents in parallel (up to maxConcurrent)
		if err := c.executeAgentWave(ctx, readyAgents, completed, &completionLock); err != nil {
			return fmt.Errorf("agent wave execution failed: %w", err)
		}
	}

	// Calculate metrics
	c.calculateMetrics()

	c.logger.Infof("Execution complete: %d success, %d failed",
		c.metrics.SuccessCount, c.metrics.FailureCount)

	return nil
}

// executeAgentWave executes a set of independent agents in parallel.
func (c *Coordinator) executeAgentWave(
	ctx context.Context,
	agentIDs []string,
	completed map[string]bool,
	completionLock *sync.Mutex,
) error {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, c.maxConcurrent)
	errChan := make(chan error, len(agentIDs))

	for _, taskID := range agentIDs {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Execute agent
			config := c.agents[id]
			c.logger.Infof("Starting agent: %s (%s)", id, config.Title)

			result, err := c.spawnerFunc(ctx, config)
			if err != nil {
				result = &AgentResult{
					TaskID: id,
					Success: false,
					Error: err.Error(),
					Timestamp: time.Now(),
				}
			}

			// Store result
			c.mu.Lock()
			c.results[id] = result
			c.mu.Unlock()

			// Mark completed
			completionLock.Lock()
			completed[id] = true
			completionLock.Unlock()

			// Handle success/failure callbacks
			if result.Success {
				c.mu.Lock()
				c.metrics.SuccessCount++
				c.mu.Unlock()
				if c.successCallback != nil {
					if err := c.successCallback(result); err != nil {
						c.logger.Errorf("success callback failed for %s: %v", id, err)
						errChan <- err
					}
				}
				c.logger.Infof("Agent succeeded: %s", id)
			} else {
				c.mu.Lock()
				c.metrics.FailureCount++
				c.mu.Unlock()
				if c.failureCallback != nil {
					if err := c.failureCallback(result); err != nil {
						c.logger.Errorf("failure callback failed for %s: %v", id, err)
						errChan <- err
					}
				}
				c.logger.Errorf("Agent failed: %s - %s", id, result.Error)
			}
		}(taskID)
	}

	wg.Wait()
	close(errChan)

	// Collect errors
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		return fmt.Errorf("errors during wave execution: %v", errs)
	}

	return nil
}

// getReadyAgents returns all agents whose dependencies are satisfied.
func (c *Coordinator) getReadyAgents(order []string, completed map[string]bool) []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var ready []string
	for _, taskID := range order {
		if completed[taskID] {
			continue // Already done
		}

		// Check if all dependencies are met
		deps := c.dependencies[taskID]
		allDone := true
		for _, dep := range deps {
			if !completed[dep] {
				allDone = false
				break
			}
		}

		if allDone {
			ready = append(ready, taskID)
		}
	}

	return ready
}

// getExecutionStages groups agents by their depth in the dependency graph.
func (c *Coordinator) getExecutionStages(order []string) [][]string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	depths := make(map[string]int)
	stages := make(map[int][]string)

	// Calculate depth of each task
	for _, taskID := range order {
		maxDepth := 0
		for _, dep := range c.dependencies[taskID] {
			if depths[dep] > maxDepth {
				maxDepth = depths[dep]
			}
		}
		depths[taskID] = maxDepth + 1
	}

	// Group by depth
	for taskID, depth := range depths {
		stages[depth] = append(stages[depth], taskID)
	}

	// Build result
	result := make([][]string, 0)
	for i := 1; len(stages[i]) > 0; i++ {
		result = append(result, stages[i])
	}

	return result
}

// GetResult returns the execution result for a specific task.
func (c *Coordinator) GetResult(taskID string) *AgentResult {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.results[taskID]
}

// GetMetrics returns aggregated execution metrics.
func (c *Coordinator) GetMetrics() ExecutionMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.metrics
}

// calculateMetrics computes final metrics.
func (c *Coordinator) calculateMetrics() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.metrics.TotalAgents = len(c.results)
	c.metrics.TotalTime = time.Since(c.startTime)

	if c.metrics.TotalAgents > 0 {
		c.metrics.AverageTime = c.metrics.TotalTime / time.Duration(c.metrics.TotalAgents)

		for _, result := range c.results {
			c.metrics.TotalTokens += result.TokensUsed
		}
		c.metrics.AverageTokens = c.metrics.TotalTokens / c.metrics.TotalAgents

		// Calculate gate pass rates
		gateCounts := make(map[string]int)
		gatePass := make(map[string]int)

		for _, result := range c.results {
			for _, gr := range result.GateResults {
				gateName := string(gr.GateType)
				gateCounts[gateName]++
				if gr.Passed {
					gatePass[gateName]++
				}
			}
		}

		for gate, total := range gateCounts {
			if total > 0 {
				c.metrics.GatePassRate[gate] = float64(gatePass[gate]) / float64(total)
			}
		}
	}

	// Estimate parallel speedup
	if c.metrics.TotalAgents > 1 && c.metrics.AverageTime > 0 {
		sequentialTime := c.metrics.AverageTime * time.Duration(c.metrics.TotalAgents)
		if sequentialTime > 0 {
			c.metrics.ParallelFactor = float64(sequentialTime) / float64(c.metrics.TotalTime)
		}
	}
}
