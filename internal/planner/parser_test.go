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

func TestParsePlan_SimpleList(t *testing.T) {
	input := `
1. Create user authentication endpoint
2. Add password hashing
3. Implement JWT token generation
`
	parser := NewPlanParser()
	tasks, err := parser.Parse(input)

	require.NoError(t, err)
	assert.Len(t, tasks, 3)
	assert.Equal(t, "Create user authentication endpoint", tasks[0].Title)
	assert.Equal(t, "Add password hashing", tasks[1].Title)
	assert.Equal(t, "Implement JWT token generation", tasks[2].Title)
}

func TestParsePlan_WithDependencies(t *testing.T) {
	input := `
1. Create database schema
2. Build user model (depends on: 1)
3. Add authentication middleware (depends on: 2)
`
	parser := NewPlanParser()
	tasks, err := parser.Parse(input)

	require.NoError(t, err)
	assert.Len(t, tasks, 3)

	assert.Empty(t, tasks[0].DependsOn)
	assert.Equal(t, []int{0}, tasks[1].DependsOn)
	assert.Equal(t, []int{1}, tasks[2].DependsOn)
}

func TestParsePlan_BulletPoints(t *testing.T) {
	input := `
- Setup project structure
- Configure dependencies
- Write initial tests
`
	parser := NewPlanParser()
	tasks, err := parser.Parse(input)

	require.NoError(t, err)
	assert.Len(t, tasks, 3)
	assert.Equal(t, "Setup project structure", tasks[0].Title)
	assert.Equal(t, "Configure dependencies", tasks[1].Title)
	assert.Equal(t, "Write initial tests", tasks[2].Title)
}

func TestParsePlan_EmptyInput(t *testing.T) {
	parser := NewPlanParser()
	tasks, err := parser.Parse("")

	require.NoError(t, err)
	assert.Empty(t, tasks)
}

func TestParsePlan_WithPriorities(t *testing.T) {
	input := `
1. Fix critical security bug [P0]
2. Add new feature [P1]
3. Refactor code [P2]
`
	parser := NewPlanParser()
	tasks, err := parser.Parse(input)

	require.NoError(t, err)
	assert.Len(t, tasks, 3)
	assert.Equal(t, 0, tasks[0].Priority)
	assert.Equal(t, 1, tasks[1].Priority)
	assert.Equal(t, 2, tasks[2].Priority)
}

func TestParsePlan_WithDescriptions(t *testing.T) {
	input := `
## Task 1: Create user authentication endpoint
Description: Build REST API endpoint for user login

## Task 2: Add password hashing
Description: Use bcrypt for secure password storage
`
	parser := NewPlanParser()
	tasks, err := parser.Parse(input)

	require.NoError(t, err)
	assert.Len(t, tasks, 2)

	assert.Equal(t, "Create user authentication endpoint", tasks[0].Title)
	assert.Equal(t, "Build REST API endpoint for user login", tasks[0].Description)

	assert.Equal(t, "Add password hashing", tasks[1].Title)
	assert.Equal(t, "Use bcrypt for secure password storage", tasks[1].Description)
}
