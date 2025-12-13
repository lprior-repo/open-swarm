// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package planner

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBeadsCreator_GenerateIssueID(t *testing.T) {
	creator := NewBeadsCreator("test-project")

	id1 := creator.GenerateIssueID()
	id2 := creator.GenerateIssueID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
	assert.Contains(t, id1, "test-project-")
}

func TestBeadsCreator_FormatCreateCommand(t *testing.T) {
	creator := NewBeadsCreator("myproject")

	task := ParsedTask{
		Title:       "Implement authentication",
		Description: "Build user auth system",
		Priority:    1,
	}

	cmd := creator.FormatCreateCommand(task, "")

	assert.Contains(t, cmd, "bd create")
	assert.Contains(t, cmd, "Implement authentication")
	assert.Contains(t, cmd, "--priority=1")
	assert.Contains(t, cmd, "--description=")
}

func TestBeadsCreator_FormatCreateCommandWithParent(t *testing.T) {
	creator := NewBeadsCreator("myproject")

	task := ParsedTask{
		Title:    "Sub-task",
		Priority: 0,
	}

	cmd := creator.FormatCreateCommand(task, "myproject-abc")

	assert.Contains(t, cmd, "bd create")
	assert.Contains(t, cmd, "--parent=myproject-abc")
}

func TestBeadsCreator_CreatePlan(t *testing.T) {
	creator := NewBeadsCreator("test")

	tasks := []ParsedTask{
		{Title: "Setup", Priority: 0},
		{Title: "Build", Priority: 1, DependsOn: []int{0}},
		{Title: "Test", Priority: 1, DependsOn: []int{1}},
	}

	plan, err := creator.CreatePlan(tasks)

	require.NoError(t, err)
	assert.NotNil(t, plan)
	assert.Len(t, plan.Tasks, 3)
	assert.Len(t, plan.Commands, 3)
	assert.Len(t, plan.IDMap, 3)
}

func TestBeadsCreator_FormatPlanSummary(t *testing.T) {
	creator := NewBeadsCreator("test")

	tasks := []ParsedTask{
		{Title: "Task 1", Priority: 0},
		{Title: "Task 2", Priority: 1, DependsOn: []int{0}},
	}

	plan, err := creator.CreatePlan(tasks)
	require.NoError(t, err)

	summary := creator.FormatPlanSummary(plan)

	assert.Contains(t, summary, "Task 1")
	assert.Contains(t, summary, "Task 2")
	assert.Contains(t, summary, "depends on")
}
