package temporal

import (
	"fmt"
	"time"

	"github.com/gammazero/toposort"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
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

// runDag executes tasks in topological order with parallelism
func runDag(ctx workflow.Context, tasks []Task) error {
	logger := workflow.GetLogger(ctx)

	// 1. Build Graph & Toposort (The Math)
	taskMap := make(map[string]Task)
	edges := make([]toposort.Edge, 0)

	for _, t := range tasks {
		taskMap[t.Name] = t
	}

	// Build edges: [dependency, dependent]
	for _, t := range tasks {
		for _, dep := range t.Deps {
			edges = append(edges, toposort.Edge{dep, t.Name})
		}
	}

	// Perform topological sort
	sortedNodes, err := toposort.Toposort(edges)
	if err != nil {
		return fmt.Errorf("cycle detected in DAG: %w", err)
	}

	// Convert sorted nodes to task names
	flatOrder := make([]string, 0, len(sortedNodes))
	for _, node := range sortedNodes {
		flatOrder = append(flatOrder, node.(string))
	}

	logger.Info("DAG Toposort Complete", "order", flatOrder)

	// 2. Execution Loop with Parallel Task Launching
	completed := make(map[string]bool)
	pendingFutures := make(map[string]workflow.Future)
	failedTasks := make([]string, 0)

	// Activity options
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		HeartbeatTimeout:    30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    1 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    30 * time.Second,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	shellActivities := &ShellActivities{}

	for len(completed) < len(tasks) {
		// Check what is runnable
		for _, taskName := range flatOrder {
			if completed[taskName] || pendingFutures[taskName] != nil {
				continue
			}

			// Are all dependencies met?
			canRun := true
			for _, dep := range taskMap[taskName].Deps {
				if !completed[dep] {
					canRun = false
					break
				}
			}

			if canRun {
				// START ACTIVITY - Execute shell command
				logger.Info("Starting task", "name", taskName)
				cmd := taskMap[taskName].Command
				f := workflow.ExecuteActivity(ctx, shellActivities.RunScript, cmd)
				pendingFutures[taskName] = f
			}
		}

		// Wait for next completion using selector
		selector := workflow.NewSelector(ctx)

		for name := range pendingFutures {
			taskName := name
			taskFuture := pendingFutures[taskName]

			selector.AddFuture(taskFuture, func(f workflow.Future) {
				var output string
				err := f.Get(ctx, &output)

				if err != nil {
					logger.Error("Task failed", "name", taskName, "error", err)
					// Track failed task but don't break yet - let other pending tasks complete
					failedTasks = append(failedTasks, taskName)
				} else {
					logger.Info("Task completed", "name", taskName, "output", output)
					completed[taskName] = true
				}

				delete(pendingFutures, taskName)
			})
		}

		if len(pendingFutures) > 0 {
			selector.Select(ctx)

			// Check if any task failed - if so, abort DAG execution
			if len(failedTasks) > 0 {
				return fmt.Errorf("tasks failed: %v", failedTasks)
			}
		} else if len(completed) < len(tasks) {
			// Deadlock check (shouldn't happen with valid toposort)
			return fmt.Errorf("DAG stalled - no tasks runnable")
		}
	}

	logger.Info("DAG Execution Complete", "tasksCompleted", len(completed))
	return nil
}
