package dag

import (
	"go.temporal.io/sdk/workflow"
	"open-swarm/pkg/types"
)

// Re-export public types for convenience
type Task = types.Task
type WorkflowInput = types.DAGWorkflowInput

// State holds the mutable state of a running DAG.
// Exploring this out allows for checkpointing in the future.
type State struct {
	TaskMap        map[string]Task
	FlatOrder      []string
	Completed      map[string]bool
	PendingFutures map[string]workflow.Future
	FailedTasks    []string
}

func NewState(tasks []Task, order []string) *State {
	taskMap := make(map[string]Task)
	for _, t := range tasks {
		taskMap[t.Name] = t
	}

	return &State{
		TaskMap:        taskMap,
		FlatOrder:      order,
		Completed:      make(map[string]bool),
		PendingFutures: make(map[string]workflow.Future),
		FailedTasks:    make([]string, 0),
	}
}
