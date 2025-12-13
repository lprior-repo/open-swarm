// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package temporal

import (
	"context"
	"fmt"

	"go.temporal.io/sdk/activity"

	"open-swarm/internal/agent"
	"open-swarm/internal/infra"
	"open-swarm/internal/workflow"
)

// CellActivities handles OpenCode cell lifecycle
type CellActivities struct {
	activities *workflow.Activities
}

// NewCellActivities creates a new CellActivities instance using global managers
func NewCellActivities() *CellActivities {
	pm, sm, wm := GetManagers()
	return &CellActivities{
		activities: workflow.NewActivities(pm, sm, wm),
	}
}

// Serializable input/output types (no pointers to processes)

// BootstrapInput contains parameters for bootstrapping a cell
type BootstrapInput struct {
	CellID string
	Branch string
}

// BootstrapOutput contains the serializable cell information
type BootstrapOutput struct {
	CellID       string
	Port         int
	WorktreeID   string
	WorktreePath string
	BaseURL      string
	ServerPID    int
}

// TaskInput contains parameters for executing a task
type TaskInput struct {
	TaskID      string
	Description string
	Prompt      string
}

// TaskOutput contains the task execution result
type TaskOutput struct {
	Success       bool
	Output        string
	FilesModified []string
	ErrorMessage  string
}

// BootstrapCell creates an isolated OpenCode cell
// This activity allocates resources (port, worktree, server) and returns serializable output
func (ca *CellActivities) BootstrapCell(ctx context.Context, input BootstrapInput) (*BootstrapOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Bootstrapping cell", "cellID", input.CellID)

	activity.RecordHeartbeat(ctx, "allocating resources")

	// Call existing infrastructure
	cell, err := ca.activities.BootstrapCell(ctx, input.CellID, input.Branch)
	if err != nil {
		return nil, fmt.Errorf("failed to bootstrap cell %q: %w", input.CellID, err)
	}

	// Convert to serializable output
	return &BootstrapOutput{
		CellID:       cell.CellID,
		Port:         cell.Port,
		WorktreeID:   cell.WorktreeID,
		WorktreePath: cell.WorktreePath,
		BaseURL:      cell.ServerHandle.BaseURL,
		ServerPID:    cell.ServerHandle.PID,
	}, nil
}

// ExecuteTask runs a prompt in the cell
func (ca *CellActivities) ExecuteTask(ctx context.Context, bootstrap *BootstrapOutput, task TaskInput) (*TaskOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Executing task", "taskID", task.TaskID)

	activity.RecordHeartbeat(ctx, "executing prompt")

	// Reconstruct cell from bootstrap output
	cell := ca.reconstructCell(bootstrap)

	taskCtx := &agent.TaskContext{
		TaskID:      task.TaskID,
		Description: task.Description,
		Prompt:      task.Prompt,
	}

	result, err := ca.activities.ExecuteTask(ctx, cell, taskCtx)
	if err != nil {
		return &TaskOutput{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	return &TaskOutput{
		Success:       result.Success,
		Output:        result.Output,
		FilesModified: result.FilesModified,
		ErrorMessage:  result.ErrorMessage,
	}, nil
}

// RunTests executes tests in the cell
func (ca *CellActivities) RunTests(ctx context.Context, bootstrap *BootstrapOutput) (bool, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Running tests", "cellID", bootstrap.CellID)

	cell := ca.reconstructCell(bootstrap)
	passed, err := ca.activities.RunTests(ctx, cell)
	if err != nil {
		return false, fmt.Errorf("failed to run tests in cell %q: %w", bootstrap.CellID, err)
	}
	return passed, nil
}

// CommitChanges commits work in the cell
func (ca *CellActivities) CommitChanges(ctx context.Context, bootstrap *BootstrapOutput, message string) error {
	cell := ca.reconstructCell(bootstrap)
	if err := ca.activities.CommitChanges(ctx, cell, message); err != nil {
		return fmt.Errorf("failed to commit changes in cell %q: %w", bootstrap.CellID, err)
	}
	return nil
}

// RevertChanges reverts work in the cell
func (ca *CellActivities) RevertChanges(ctx context.Context, bootstrap *BootstrapOutput) error {
	cell := ca.reconstructCell(bootstrap)
	if err := ca.activities.RevertChanges(ctx, cell); err != nil {
		return fmt.Errorf("failed to revert changes in cell %q: %w", bootstrap.CellID, err)
	}
	return nil
}

// TeardownCell destroys the cell and releases resources
func (ca *CellActivities) TeardownCell(ctx context.Context, bootstrap *BootstrapOutput) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Tearing down cell", "cellID", bootstrap.CellID)

	cell := ca.reconstructCell(bootstrap)
	if err := ca.activities.TeardownCell(ctx, cell); err != nil {
		return fmt.Errorf("failed to teardown cell %q: %w", bootstrap.CellID, err)
	}
	return nil
}

// reconstructCell rebuilds runtime cell from serialized bootstrap
// Note: Cmd and process cannot be reconstructed - TeardownCell will use PID directly
func (ca *CellActivities) reconstructCell(bootstrap *BootstrapOutput) *workflow.CellBootstrap {
	// Reconstruct server handle
	serverHandle := &infra.ServerHandle{
		Port:    bootstrap.Port,
		BaseURL: bootstrap.BaseURL,
		PID:     bootstrap.ServerPID,
		// Note: Cmd and process cannot be reconstructed -
		// TeardownCell will use PID directly
	}

	return &workflow.CellBootstrap{
		CellID:       bootstrap.CellID,
		Port:         bootstrap.Port,
		WorktreeID:   bootstrap.WorktreeID,
		WorktreePath: bootstrap.WorktreePath,
		ServerHandle: serverHandle,
		Client:       agent.NewClient(bootstrap.BaseURL, bootstrap.Port),
	}
}
