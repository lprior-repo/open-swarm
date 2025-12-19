package temporal

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/workflow"
)

// DAGTask represents a single task in the directed acyclic graph.
type DAGTask struct {
	// ID is the unique identifier for the task.
	ID string
	// Dependencies is a list of task IDs that must complete before this task.
	Dependencies []string
}

// Validate checks that required fields are present.
func (d *DAGTask) Validate() error {
	if d.ID == "" {
		return errors.New("id is required")
	}
	return nil
}

// DAGResult represents the result of executing a DAG.
type DAGResult struct {
	// RootTaskID is the root task ID that was executed.
	RootTaskID string
	// CompletedTasks is the number of successfully completed tasks.
	CompletedTasks int
	// FailedTasks is the number of failed tasks.
	FailedTasks int
	// CompletionTime is when the DAG execution finished.
	CompletionTime time.Time
	// AllTasksCompleted indicates if all tasks completed successfully.
	AllTasksCompleted bool
}

// IsValid checks that output is complete.
func (d *DAGResult) IsValid() bool {
	return d.RootTaskID != "" && !d.CompletionTime.IsZero()
}

// ExecutionResult represents the result of executing a single task.
type ExecutionResult struct {
	// TaskID is the task ID that was executed.
	TaskID string
	// Success indicates if the task succeeded.
	Success bool
	// AgentID is the agent that executed the task.
	AgentID string
	// StartTime is when the task started.
	StartTime time.Time
	// EndTime is when the task completed.
	EndTime time.Time
	// OutputData contains task-specific output.
	OutputData map[string]interface{}
	// Error is the error message if the task failed.
	Error string
}

// IsValid checks that result is complete.
func (e *ExecutionResult) IsValid() bool {
	return e.TaskID != "" && !e.StartTime.IsZero()
}

// DAGInput is the workflow input for executing a DAG.
type DAGInput struct {
	// RootTaskID is the starting task ID.
	RootTaskID string
	// AllTasks is the complete list of tasks in the DAG.
	AllTasks []DAGTask
}

// Validate checks that input is valid.
func (d *DAGInput) Validate() error {
	if d.RootTaskID == "" {
		return errors.New("root_task_id is required")
	}
	if len(d.AllTasks) == 0 {
		return errors.New("tasks list cannot be empty")
	}

	// Verify root task exists in tasks list
	found := false
	for _, task := range d.AllTasks {
		if task.ID == d.RootTaskID {
			found = true
			break
		}
	}
	if !found {
		return errors.New("root_task_id must be in tasks list")
	}

	return nil
}

// DAGExecutor orchestrates the execution of tasks in a DAG.
type DAGExecutor struct{}

// NewDAGExecutor creates a new DAG executor.
func NewDAGExecutor() *DAGExecutor {
	return &DAGExecutor{}
}

// GetReadyTasks returns tasks that are ready to execute (all dependencies met).
func (e *DAGExecutor) GetReadyTasks(tasks []DAGTask, completed map[string]bool) []DAGTask {
	var ready []DAGTask

	for _, task := range tasks {
		// Skip if already completed
		if completed[task.ID] {
			continue
		}

		// Check if all dependencies are completed
		allDepsMet := true
		for _, dep := range task.Dependencies {
			if !completed[dep] {
				allDepsMet = false
				break
			}
		}

		if allDepsMet {
			ready = append(ready, task)
		}
	}

	return ready
}

// DAGActivities defines the activities used by ExecuteDAGWorkflow.
type DAGActivities struct{}

// ExecuteTaskActivity executes a single task and returns the result.
func (a *DAGActivities) ExecuteTaskActivity(
	ctx context.Context,
	task DAGTask,
) (*ExecutionResult, error) {
	// Activity context
	info := activity.GetInfo(ctx)
	_ = info // May be used for logging

	if err := task.Validate(); err != nil {
		return nil, fmt.Errorf("invalid task: %w", err)
	}

	// Placeholder implementation - spawns agent via SpawnAgentWorkflow
	// For now, return successful execution
	result := &ExecutionResult{
		TaskID:     task.ID,
		Success:    true,
		AgentID:    fmt.Sprintf("agent-%s-%d", task.ID, time.Now().Unix()),
		StartTime:  time.Now().Add(-1 * time.Second),
		EndTime:    time.Now(),
		OutputData: make(map[string]interface{}),
	}

	return result, nil
}

// ExecuteDAGWorkflow orchestrates the execution of a DAG of tasks.
// It respects task dependencies and executes independent tasks in parallel.
func ExecuteDAGWorkflow(ctx workflow.Context, input DAGInput) (*DAGResult, error) {
	// Validate input
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	// Create activity options with timeout
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	// Track completed tasks
	completed := make(map[string]bool)
	completedCount := 0
	failedCount := 0
	taskResults := make(map[string]ExecutionResult)

	// Execute tasks in waves respecting dependencies
	for completedCount+failedCount < len(input.AllTasks) {
		// Find tasks ready for execution
		executor := NewDAGExecutor()
		readyTasks := executor.GetReadyTasks(input.AllTasks, completed)

		if len(readyTasks) == 0 {
			// No ready tasks but not all completed - circular dependency or error
			return nil, errors.New("no ready tasks available but execution incomplete")
		}

		// Execute ready tasks in parallel
		futures := make([]workflow.Future, len(readyTasks))
		for i, task := range readyTasks {
			activities := &DAGActivities{}
			futures[i] = workflow.ExecuteActivity(
				ctx,
				activities.ExecuteTaskActivity,
				task,
			)
		}

		// Wait for all parallel tasks to complete
		for i, future := range futures {
			var result ExecutionResult
			err := future.Get(ctx, &result)
			if err != nil {
				return nil, fmt.Errorf("task execution failed: %w", err)
			}

			taskResults[readyTasks[i].ID] = result
			completed[readyTasks[i].ID] = true

			if result.Success {
				completedCount++
			} else {
				failedCount++
			}
		}
	}

	// Build output
	output := &DAGResult{
		RootTaskID:        input.RootTaskID,
		CompletedTasks:    completedCount,
		FailedTasks:       failedCount,
		CompletionTime:    workflow.Now(ctx),
		AllTasksCompleted: failedCount == 0,
	}

	if !output.IsValid() {
		return nil, errors.New("invalid output state")
	}

	return output, nil
}
