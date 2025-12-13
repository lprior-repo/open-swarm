// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package temporal

import (
	"fmt"
	"time"

	"github.com/gammazero/toposort"
	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// Activity execution timeouts
const (
	// DAGStartToCloseTimeout is the maximum time for executing a DAG task
	DAGStartToCloseTimeout = 10 * time.Minute

	// DAGHeartbeatTimeout is the heartbeat timeout for DAG activities
	DAGHeartbeatTimeout = 30 * time.Second

	// DAGRetryBackoffCoefficient is the exponential backoff coefficient for DAG retries
	DAGRetryBackoffCoefficient = 2.0

	// DAGRetryMaxAttempts is the maximum number of retry attempts for DAG tasks
	DAGRetryMaxAttempts = 3
)

// Task represents a node in the DAG
type Task struct {
	Name    string
	Command string
	Deps    []string
}

// DAGWorkflowInput contains tasks to execute
type DAGWorkflowInput struct {
	WorkflowID string
	Branch     string
	Tasks      []Task
}

// TddDagWorkflow implements Test-Driven Development loop with DAG execution
// It keeps retrying the entire DAG until success or manual abort
func TddDagWorkflow(ctx workflow.Context, input DAGWorkflowInput) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting TDD DAG Workflow", "workflowID", input.WorkflowID)

	// TDD LOOP: Keep running until success
	attempt := 1
	for {
		logger.Info("TDD Cycle Start", "attempt", attempt)

		// 1. Run the DAG
		err := runDag(ctx, input.Tasks)

		if err == nil {
			logger.Info("TDD Cycle Succeeded!", "attempts", attempt)
			return nil // Success! Break the loop
		}

		// 2. Failure Handling - Wait for human intervention
		logger.Error("TDD Cycle Failed", "attempt", attempt, "error", err)
		logger.Info("Waiting for 'FixApplied' signal to retry...")

		// Block and wait for signal from human
		var signalVal string
		signalChan := workflow.GetSignalChannel(ctx, "FixApplied")
		signalChan.Receive(ctx, &signalVal)

		logger.Info("Received FixApplied signal", "message", signalVal)

		// Loop restarts with incremented attempt counter
		attempt++
	}
}

// dagState holds mutable state for DAG execution
type dagState struct {
	taskMap        map[string]Task
	flatOrder      []string
	completed      map[string]bool
	pendingFutures map[string]workflow.Future
	failedTasks    []string
}

// runDag executes tasks in topological order with parallelism
func runDag(ctx workflow.Context, tasks []Task) error {
	logger := workflow.GetLogger(ctx)

	flatOrder, err := buildDAGOrder(tasks)
	if err != nil {
		return err
	}

	logger.Info("DAG Toposort Complete", "order", flatOrder)

	state := &dagState{
		taskMap:        buildTaskMap(tasks),
		flatOrder:      flatOrder,
		completed:      make(map[string]bool),
		pendingFutures: make(map[string]workflow.Future),
		failedTasks:    make([]string, 0),
	}

	ctx = workflow.WithActivityOptions(ctx, buildActivityOptions())

	return executeDAG(ctx, logger, state, tasks)
}

// buildDAGOrder builds edges and performs topological sort
func buildDAGOrder(tasks []Task) ([]string, error) {
	edges := make([]toposort.Edge, 0)
	for _, t := range tasks {
		for _, dep := range t.Deps {
			edges = append(edges, toposort.Edge{dep, t.Name})
		}
	}

	sortedNodes, err := toposort.Toposort(edges)
	if err != nil {
		return nil, fmt.Errorf("cycle detected in DAG: %w", err)
	}

	flatOrder := make([]string, 0, len(sortedNodes))
	for _, node := range sortedNodes {
		flatOrder = append(flatOrder, node.(string))
	}
	return flatOrder, nil
}

// buildTaskMap creates a map of task names to tasks
func buildTaskMap(tasks []Task) map[string]Task {
	taskMap := make(map[string]Task)
	for _, t := range tasks {
		taskMap[t.Name] = t
	}
	return taskMap
}

// buildActivityOptions creates configured activity options
func buildActivityOptions() workflow.ActivityOptions {
	return workflow.ActivityOptions{
		StartToCloseTimeout: DAGStartToCloseTimeout,
		HeartbeatTimeout:    DAGHeartbeatTimeout,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    1 * time.Second,
			BackoffCoefficient: DAGRetryBackoffCoefficient,
			MaximumInterval:    DAGHeartbeatTimeout,
			MaximumAttempts:    DAGRetryMaxAttempts,
		},
	}
}

// executeDAG runs the main execution loop
func executeDAG(ctx workflow.Context, logger log.Logger, state *dagState, tasks []Task) error {
	shellActivities := &ShellActivities{}

	for len(state.completed) < len(tasks) {
		launchRunnableTasks(ctx, logger, state, shellActivities)

		if len(state.pendingFutures) > 0 {
			if err := waitForTaskCompletion(ctx, logger, state); err != nil {
				return err
			}
		} else if len(state.completed) < len(tasks) {
			return fmt.Errorf("DAG stalled - no tasks runnable")
		}
	}

	logger.Info("DAG Execution Complete", "tasksCompleted", len(state.completed))
	return nil
}

// launchRunnableTasks launches all tasks whose dependencies are met
func launchRunnableTasks(ctx workflow.Context, logger log.Logger, state *dagState, activities *ShellActivities) {
	for _, taskName := range state.flatOrder {
		if state.completed[taskName] || state.pendingFutures[taskName] != nil {
			continue
		}

		if allDependenciesCompleted(state, taskName) {
			logger.Info("Starting task", "name", taskName)
			cmd := state.taskMap[taskName].Command
			f := workflow.ExecuteActivity(ctx, activities.RunScript, cmd)
			state.pendingFutures[taskName] = f
		}
	}
}

// allDependenciesCompleted checks if all dependencies for a task are complete
func allDependenciesCompleted(state *dagState, taskName string) bool {
	for _, dep := range state.taskMap[taskName].Deps {
		if !state.completed[dep] {
			return false
		}
	}
	return true
}

// waitForTaskCompletion waits for at least one task to complete
func waitForTaskCompletion(ctx workflow.Context, logger log.Logger, state *dagState) error {
	selector := workflow.NewSelector(ctx)

	for name := range state.pendingFutures {
		taskName := name
		taskFuture := state.pendingFutures[taskName]

		selector.AddFuture(taskFuture, func(f workflow.Future) {
			handleTaskResult(logger, state, taskName, f, ctx)
		})
	}

	selector.Select(ctx)

	if len(state.failedTasks) > 0 {
		return fmt.Errorf("tasks failed: %v", state.failedTasks)
	}
	return nil
}

// handleTaskResult processes the result of a completed task
func handleTaskResult(logger log.Logger, state *dagState, taskName string, f workflow.Future, ctx workflow.Context) {
	var output string
	err := f.Get(ctx, &output)

	if err != nil {
		logger.Error("Task failed", "name", taskName, "error", err)
		state.failedTasks = append(state.failedTasks, taskName)
	} else {
		logger.Info("Task completed", "name", taskName, "output", output)
		state.completed[taskName] = true
	}

	delete(state.pendingFutures, taskName)
}

// StressTestInput configures a stress test run
type StressTestInput struct {
	NumAgents        int
	Prompt           string
	Agent            string
	Model            string
	Bootstrap        *BootstrapOutput
	TimeoutSeconds   int
	ConcurrencyLimit int
}

// StressTestResult summarizes stress test results
type StressTestResult struct {
	TotalAgents     int
	Successful      int
	Failed          int
	TotalDuration   time.Duration
	AverageDuration time.Duration
	Results         []AgentInvokeResult
	Errors          []string
}

// StressTestWorkflow runs multiple agents in parallel for stress testing
func StressTestWorkflow(ctx workflow.Context, input StressTestInput) (*StressTestResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting Stress Test Workflow",
		"num_agents", input.NumAgents,
		"agent", input.Agent,
		"model", input.Model,
		"concurrency_limit", input.ConcurrencyLimit)

	startTime := workflow.Now(ctx)

	// Set defaults
	if input.TimeoutSeconds == 0 {
		input.TimeoutSeconds = 300 // 5 minutes
	}
	if input.Agent == "" {
		input.Agent = "general"
	}
	if input.Model == "" {
		input.Model = "anthropic/claude-sonnet-4-5"
	}

	// Configure activity options for agent invocations
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: time.Duration(input.TimeoutSeconds) * time.Second,
		HeartbeatTimeout:    5 * time.Minute, // 5 minutes for LLM responses
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	// Initialize result
	result := &StressTestResult{
		TotalAgents: input.NumAgents,
		Results:     make([]AgentInvokeResult, 0, input.NumAgents),
		Errors:      make([]string, 0),
	}

	// Create agent activities
	agentActivities := NewAgentActivities()

	// Track pending futures
	type agentExecution struct {
		future workflow.Future
		index  int
	}

	pendingExecutions := make([]agentExecution, 0, input.NumAgents)

	// Launch agents
	if input.ConcurrencyLimit > 0 {
		// Controlled concurrency: launch in batches
		logger.Info("Launching agents with concurrency control", "limit", input.ConcurrencyLimit)

		for batch := 0; batch < input.NumAgents; batch += input.ConcurrencyLimit {
			batchSize := input.ConcurrencyLimit
			if batch+batchSize > input.NumAgents {
				batchSize = input.NumAgents - batch
			}

			batchFutures := make([]agentExecution, batchSize)

			// Launch batch
			for i := 0; i < batchSize; i++ {
				agentIndex := batch + i
				agentInput := &AgentInvokeInput{
					Bootstrap:      input.Bootstrap,
					Prompt:         fmt.Sprintf("%s (Agent %d/%d)", input.Prompt, agentIndex+1, input.NumAgents),
					Agent:          input.Agent,
					Model:          input.Model,
					Title:          fmt.Sprintf("Stress Test Agent %d", agentIndex+1),
					TimeoutSeconds: input.TimeoutSeconds,
					StreamOutput:   false,
				}

				future := workflow.ExecuteActivity(ctx, agentActivities.InvokeAgent, agentInput)
				batchFutures[i] = agentExecution{future: future, index: agentIndex}
			}

			// Wait for batch to complete
			for _, exec := range batchFutures {
				var agentResult *AgentInvokeResult
				err := exec.future.Get(ctx, &agentResult)

				if err != nil {
					result.Failed++
					result.Errors = append(result.Errors, fmt.Sprintf("Agent %d failed: %v", exec.index+1, err))
					logger.Error("Agent failed", "index", exec.index+1, "error", err)
				} else {
					if agentResult.Success {
						result.Successful++
					} else {
						result.Failed++
						result.Errors = append(result.Errors, fmt.Sprintf("Agent %d returned error: %s", exec.index+1, agentResult.Error))
					}
					result.Results = append(result.Results, *agentResult)
				}
			}

			logger.Info("Batch completed", "batch", batch/input.ConcurrencyLimit+1, "completed", result.Successful+result.Failed, "total", input.NumAgents)
		}
	} else {
		// Unlimited concurrency: launch all at once
		logger.Info("Launching all agents in parallel")

		for i := 0; i < input.NumAgents; i++ {
			agentInput := &AgentInvokeInput{
				Bootstrap:      input.Bootstrap,
				Prompt:         fmt.Sprintf("%s (Agent %d/%d)", input.Prompt, i+1, input.NumAgents),
				Agent:          input.Agent,
				Model:          input.Model,
				Title:          fmt.Sprintf("Stress Test Agent %d", i+1),
				TimeoutSeconds: input.TimeoutSeconds,
				StreamOutput:   false,
			}

			future := workflow.ExecuteActivity(ctx, agentActivities.InvokeAgent, agentInput)
			pendingExecutions = append(pendingExecutions, agentExecution{future: future, index: i})
		}

		// Wait for all to complete
		for _, exec := range pendingExecutions {
			var agentResult *AgentInvokeResult
			err := exec.future.Get(ctx, &agentResult)

			if err != nil {
				result.Failed++
				result.Errors = append(result.Errors, fmt.Sprintf("Agent %d failed: %v", exec.index+1, err))
				logger.Error("Agent failed", "index", exec.index+1, "error", err)
			} else {
				if agentResult.Success {
					result.Successful++
				} else {
					result.Failed++
					result.Errors = append(result.Errors, fmt.Sprintf("Agent %d returned error: %s", exec.index+1, agentResult.Error))
				}
				result.Results = append(result.Results, *agentResult)
			}
		}
	}

	// Calculate statistics
	result.TotalDuration = workflow.Now(ctx).Sub(startTime)
	if len(result.Results) > 0 {
		var totalDuration time.Duration
		for _, r := range result.Results {
			totalDuration += r.Duration
		}
		result.AverageDuration = totalDuration / time.Duration(len(result.Results))
	}

	logger.Info("Stress Test Workflow Completed",
		"total_agents", result.TotalAgents,
		"successful", result.Successful,
		"failed", result.Failed,
		"total_duration", result.TotalDuration,
		"average_duration", result.AverageDuration)

	return result, nil
}
