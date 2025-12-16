package dag

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// Configuration constants
const (
	StartToCloseTimeout     = 10 * time.Minute
	HeartbeatTimeout        = 30 * time.Second
	RetryBackoffCoefficient = 2.0
	RetryMaxAttempts        = 3
)

// Engine manages the execution of a DAG
type Engine struct {
	Scheduler *Scheduler
}

func NewEngine() *Engine {
	return &Engine{
		Scheduler: &Scheduler{},
	}
}

// Run executes the DAG tasks within the given workflow context.
// It manages the state and returns the final state and an error if any task fails.
// If prevState is provided, it will resume from that state.
func (e *Engine) Run(ctx workflow.Context, tasks []Task, prevState *State) (*State, error) {
	logger := workflow.GetLogger(ctx)

	var state *State
	if prevState != nil {
		logger.Info("Resuming DAG from previous state")
		state = prevState
		// Reset transient fields for the new run
		state.PendingFutures = make(map[string]workflow.Future)
		state.FailedTasks = make([]string, 0)
	} else {
		// 1. Plan
		logger.Info("Starting new DAG execution")
		flatOrder, err := e.Scheduler.BuildExecutionOrder(tasks)
		if err != nil {
			return nil, err
		}
		logger.Info("DAG Schedule", "order", flatOrder)
		// 2. Initialize State
		state = NewState(tasks, flatOrder)
	}

	// 3. Configure Activity Options
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: StartToCloseTimeout,
		HeartbeatTimeout:    HeartbeatTimeout,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    1 * time.Second,
			BackoffCoefficient: RetryBackoffCoefficient,
			MaximumInterval:    HeartbeatTimeout,
			MaximumAttempts:    RetryMaxAttempts,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// 4. Execute Loop
	shellActivities := &ShellActivities{}

	for len(state.Completed) < len(tasks) {
		e.scheduleRunnableTasks(ctx, logger, state, shellActivities)

		// Wait logic
		if len(state.PendingFutures) > 0 {
			if err := e.waitForTaskCompletion(ctx, logger, state); err != nil {
				return state, err // Return state on failure
			}
		} else if len(state.Completed) < len(tasks) {
			// This condition indicates a stall, which is a failure.
			return state, fmt.Errorf("DAG stalled - no tasks runnable (check dependencies)")
		}
	}

	logger.Info("DAG Execution Complete", "tasksCompleted", len(state.Completed))
	return state, nil
}

func (e *Engine) scheduleRunnableTasks(ctx workflow.Context, logger log.Logger, state *State, activities *ShellActivities) {
	for _, taskName := range state.FlatOrder {
		// Skip if done or running
		if state.Completed[taskName] || state.PendingFutures[taskName] != nil {
			continue
		}

		if e.allDependenciesMet(state, taskName) {
			logger.Info("Starting task", "name", taskName)
			cmd := state.TaskMap[taskName].Command

			// Execute
			f := workflow.ExecuteActivity(ctx, activities.RunDAGScript, cmd)
			state.PendingFutures[taskName] = f
		}
	}
}

func (e *Engine) allDependenciesMet(state *State, taskName string) bool {
	for _, dep := range state.TaskMap[taskName].Deps {
		if !state.Completed[dep] {
			return false
		}
	}
	return true
}

func (e *Engine) waitForTaskCompletion(ctx workflow.Context, logger log.Logger, state *State) error {
	selector := workflow.NewSelector(ctx)

	for name, future := range state.PendingFutures {
		taskName := name // Capture for closure
		selector.AddFuture(future, func(f workflow.Future) {
			var output string
			err := f.Get(ctx, &output)

			if err != nil {
				logger.Error("Task failed", "name", taskName, "error", err)
				state.FailedTasks = append(state.FailedTasks, taskName)
			} else {
				logger.Info("Task completed", "name", taskName, "output", output)
				state.Completed[taskName] = true
			}

			// Remove from pending
			delete(state.PendingFutures, taskName)
		})
	}

	selector.Select(ctx)

	if len(state.FailedTasks) > 0 {
		return fmt.Errorf("tasks failed: %v", state.FailedTasks)
	}
	return nil
}
