// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package planner

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
)

// BeadsCreator creates Beads issues from parsed tasks
type BeadsCreator struct {
	projectPrefix string
}

// NewBeadsCreator creates a new Beads issue creator
func NewBeadsCreator(projectPrefix string) *BeadsCreator {
	return &BeadsCreator{
		projectPrefix: projectPrefix,
	}
}

// GenerateIssueID generates a unique issue ID
func (bc *BeadsCreator) GenerateIssueID() string {
	bytes := make([]byte, 2)
	_, _ = rand.Read(bytes)
	suffix := hex.EncodeToString(bytes)
	return fmt.Sprintf("%s-%s", bc.projectPrefix, suffix)
}

// FormatCreateCommand formats a bd create command for a task
func (bc *BeadsCreator) FormatCreateCommand(task ParsedTask, parentID string) string {
	var parts []string

	parts = append(parts, "bd create")
	parts = append(parts, fmt.Sprintf(`"%s"`, task.Title))

	if task.Description != "" {
		parts = append(parts, fmt.Sprintf(`--description="%s"`, task.Description))
	}

	parts = append(parts, fmt.Sprintf("--priority=%d", task.Priority))

	if parentID != "" {
		parts = append(parts, fmt.Sprintf("--parent=%s", parentID))
	}

	return strings.Join(parts, " ")
}

// ExecutionPlan represents a plan for creating Beads issues
type ExecutionPlan struct {
	Tasks    []ParsedTask
	Commands []string
	IDMap    map[int]string
}

// CreatePlan creates an execution plan for the given tasks
func (bc *BeadsCreator) CreatePlan(tasks []ParsedTask) (*ExecutionPlan, error) {
	var commands []string
	idMap := make(map[int]string)

	for i, task := range tasks {
		taskID := bc.GenerateIssueID()
		idMap[i] = taskID

		var parentID string
		if len(task.DependsOn) > 0 {
			parentIdx := task.DependsOn[0]
			if parentIdx < i {
				parentID = idMap[parentIdx]
			}
		}

		cmd := bc.FormatCreateCommand(task, parentID)
		commands = append(commands, cmd)
	}

	return &ExecutionPlan{
		Tasks:    tasks,
		Commands: commands,
		IDMap:    idMap,
	}, nil
}

// FormatPlanSummary creates a human-readable summary of the plan
func (bc *BeadsCreator) FormatPlanSummary(plan *ExecutionPlan) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("Plan to create %d Beads issues:\n\n", len(plan.Tasks)))

	for i, task := range plan.Tasks {
		builder.WriteString(fmt.Sprintf("%d. %s", i+1, task.Title))
		if id, ok := plan.IDMap[i]; ok {
			builder.WriteString(fmt.Sprintf(" [%s]", id))
		}
		builder.WriteString(fmt.Sprintf(" (P%d)", task.Priority))

		if len(task.DependsOn) > 0 {
			var deps []string
			for _, depIdx := range task.DependsOn {
				if depID, ok := plan.IDMap[depIdx]; ok {
					deps = append(deps, depID)
				}
			}
			if len(deps) > 0 {
				builder.WriteString(fmt.Sprintf(" - depends on: %s", strings.Join(deps, ", ")))
			}
		}
		builder.WriteString("\n")
	}

	return builder.String()
}
