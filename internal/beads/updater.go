// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package beads

import (
	"fmt"
	"os/exec"
)

// UpdateTaskStatus updates the status of a task in Beads
// Supports statuses: open, in_progress, closed
func UpdateTaskStatus(taskID, status string) error {
	cmd := exec.Command("bd", "update", taskID, "--status", status)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to update task %s status: %w", taskID, err)
	}
	return nil
}

// AttachResult adds a workflow result as a comment to a task
func AttachResult(taskID, result string) error {
	cmd := exec.Command("bd", "comment", taskID, result)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to attach result to task %s: %w", taskID, err)
	}
	return nil
}

// ResolveDependencies marks a task as completed and unblocks dependent tasks
// This uses bd close to complete the task, which automatically unblocks dependents
func ResolveDependencies(taskID string) error {
	cmd := exec.Command("bd", "close", taskID)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to close task %s: %w", taskID, err)
	}
	return nil
}

// MarkInProgress updates a task to in_progress status with agent name in comment
func MarkInProgress(taskID, agentName string) error {
	if err := UpdateTaskStatus(taskID, "in_progress"); err != nil {
		return err
	}

	comment := fmt.Sprintf("Agent '%s' starting work", agentName)
	return AttachResult(taskID, comment)
}

// MarkCompleted marks a task as completed and records the result
func MarkCompleted(taskID, result string) error {
	if err := AttachResult(taskID, fmt.Sprintf("RESULT:\n%s", result)); err != nil {
		return err
	}

	return ResolveDependencies(taskID)
}

// MarkFailed marks a task as failed and records the error
func MarkFailed(taskID, errMsg string) error {
	if err := AttachResult(taskID, fmt.Sprintf("ERROR:\n%s", errMsg)); err != nil {
		return err
	}

	return UpdateTaskStatus(taskID, "open")
}

// UpdateProgress updates a task with progress information
func UpdateProgress(taskID, currentGate string) error {
	comment := fmt.Sprintf("Progress update: at gate %s", currentGate)
	return AttachResult(taskID, comment)
}
