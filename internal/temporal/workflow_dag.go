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

// buildDAGOrder builds edges and performs topological sort.
// Ensures all tasks are included, even those without dependencies.
func buildDAGOrder(tasks []Task) ([]string, error) {
	if len(tasks) == 0 {
		return []string{}, nil
	}

	// Build edges from dependencies
	edges := make([]toposort.Edge, 0)
	for _, t := range tasks {
		for _, dep := range t.Deps {
			edges = append(edges, toposort.Edge{dep, t.Name})
		}
	}

	// If no edges (all tasks are independent), return tasks in order
	if len(edges) == 0 {
		flatOrder := make([]string, 0, len(tasks))
		for _, t := range tasks {
			flatOrder = append(flatOrder, t.Name)
		}
		return flatOrder, nil
	}

	sortedNodes, err := toposort.Toposort(edges)
	if err != nil {
		return nil, fmt.Errorf("cycle detected in DAG: %w", err)
	}

	// Build set of nodes in sorted output
	inSorted := make(map[string]bool, len(sortedNodes))
	flatOrder := make([]string, 0, len(tasks))
	for _, node := range sortedNodes {
		name := node.(string)
		inSorted[name] = true
		flatOrder = append(flatOrder, name)
	}

	// Add any tasks not included in toposort (root tasks with no dependents)
	for _, t := range tasks {
		if !inSorted[t.Name] {
			// Prepend root tasks so they run first
			flatOrder = append([]string{t.Name}, flatOrder...)
		}
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
