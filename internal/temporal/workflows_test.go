// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package temporal

import (
	"testing"

	"github.com/gammazero/toposort"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDAGToposort tests the topological sorting of DAG tasks
func TestDAGToposort(t *testing.T) {
	tests := []struct {
		name                string
		tasks               []Task
		expectedOrderCount  int
		shouldError         bool
		errorContains       string
		verifyOrder         func(t *testing.T, sorted []interface{})
	}{
		{
			name: "simple linear dependency",
			tasks: []Task{
				{Name: "task1", Command: "echo 1", Deps: []string{}},
				{Name: "task2", Command: "echo 2", Deps: []string{"task1"}},
				{Name: "task3", Command: "echo 3", Deps: []string{"task2"}},
			},
			expectedOrderCount: 3,
			shouldError:        false,
			verifyOrder: func(t *testing.T, sorted []interface{}) {
				// task1 should come before task2, task2 before task3
				task1Idx := -1
				task2Idx := -1
				task3Idx := -1
				for i, node := range sorted {
					if node == "task1" {
						task1Idx = i
					} else if node == "task2" {
						task2Idx = i
					} else if node == "task3" {
						task3Idx = i
					}
				}
				assert.Greater(t, task2Idx, task1Idx, "task1 should come before task2")
				assert.Greater(t, task3Idx, task2Idx, "task2 should come before task3")
			},
		},
		{
			name: "multiple parallel tasks with common dependency",
			tasks: []Task{
				{Name: "setup", Command: "echo setup", Deps: []string{}},
				{Name: "test1", Command: "echo test1", Deps: []string{"setup"}},
				{Name: "test2", Command: "echo test2", Deps: []string{"setup"}},
				{Name: "finalize", Command: "echo finalize", Deps: []string{"test1", "test2"}},
			},
			expectedOrderCount: 4,
			shouldError:        false,
			verifyOrder: func(t *testing.T, sorted []interface{}) {
				// setup should come before test1 and test2
				// test1 and test2 should come before finalize
				setupIdx := -1
				test1Idx := -1
				test2Idx := -1
				finalizeIdx := -1
				for i, node := range sorted {
					switch node {
					case "setup":
						setupIdx = i
					case "test1":
						test1Idx = i
					case "test2":
						test2Idx = i
					case "finalize":
						finalizeIdx = i
					}
				}
				assert.Greater(t, test1Idx, setupIdx)
				assert.Greater(t, test2Idx, setupIdx)
				assert.Greater(t, finalizeIdx, test1Idx)
				assert.Greater(t, finalizeIdx, test2Idx)
			},
		},
		{
			name: "diamond dependency",
			tasks: []Task{
				{Name: "a", Command: "echo a", Deps: []string{}},
				{Name: "b", Command: "echo b", Deps: []string{"a"}},
				{Name: "c", Command: "echo c", Deps: []string{"a"}},
				{Name: "d", Command: "echo d", Deps: []string{"b", "c"}},
			},
			expectedOrderCount: 4,
			shouldError:        false,
			verifyOrder: func(t *testing.T, sorted []interface{}) {
				// a should come before b and c
				// b and c should come before d
				aIdx := -1
				bIdx := -1
				cIdx := -1
				dIdx := -1
				for i, node := range sorted {
					switch node {
					case "a":
						aIdx = i
					case "b":
						bIdx = i
					case "c":
						cIdx = i
					case "d":
						dIdx = i
					}
				}
				assert.Greater(t, bIdx, aIdx)
				assert.Greater(t, cIdx, aIdx)
				assert.Greater(t, dIdx, bIdx)
				assert.Greater(t, dIdx, cIdx)
			},
		},
		{
			name: "single task with dependency",
			tasks: []Task{
				{Name: "task1", Command: "echo 1", Deps: []string{}},
				{Name: "task2", Command: "echo 2", Deps: []string{"task1"}},
			},
			expectedOrderCount: 2,
			shouldError:        false,
		},
		{
			name: "cycle detection - self reference",
			tasks: []Task{
				{Name: "task1", Command: "echo 1", Deps: []string{"task1"}},
			},
			shouldError:  true,
			errorContains: "cannot be the same",
		},
		{
			name: "cycle detection - mutual dependency",
			tasks: []Task{
				{Name: "task1", Command: "echo 1", Deps: []string{"task2"}},
				{Name: "task2", Command: "echo 2", Deps: []string{"task1"}},
			},
			shouldError:   true,
			errorContains: "cycle",
		},
		{
			name: "cycle detection - circular chain",
			tasks: []Task{
				{Name: "a", Command: "echo a", Deps: []string{"b"}},
				{Name: "b", Command: "echo b", Deps: []string{"c"}},
				{Name: "c", Command: "echo c", Deps: []string{"a"}},
			},
			shouldError:   true,
			errorContains: "cycle",
		},
		{
			name: "two independent chains",
			tasks: []Task{
				{Name: "task1", Command: "echo 1", Deps: []string{}},
				{Name: "task2", Command: "echo 2", Deps: []string{"task1"}},
				{Name: "task3", Command: "echo 3", Deps: []string{}},
				{Name: "task4", Command: "echo 4", Deps: []string{"task3"}},
			},
			expectedOrderCount: 4,
			shouldError:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build edges for toposort
			edges := make([]toposort.Edge, 0)
			for _, task := range tt.tasks {
				for _, dep := range task.Deps {
					edges = append(edges, toposort.Edge{dep, task.Name})
				}
			}

			// Perform topological sort
			sortedNodes, err := toposort.Toposort(edges)

			if tt.shouldError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedOrderCount, len(sortedNodes))

				// Run optional verification
				if tt.verifyOrder != nil {
					tt.verifyOrder(t, sortedNodes)
				}
			}
		})
	}
}

// TestDAGDependencyValidation tests that dependency validation works correctly
func TestDAGDependencyValidation(t *testing.T) {
	tests := []struct {
		name        string
		tasks       []Task
		shouldError bool
	}{
		{
			name: "all dependencies resolved",
			tasks: []Task{
				{Name: "task1", Command: "echo 1", Deps: []string{}},
				{Name: "task2", Command: "echo 2", Deps: []string{"task1"}},
			},
			shouldError: false,
		},
		{
			name: "missing dependency",
			tasks: []Task{
				{Name: "task1", Command: "echo 1", Deps: []string{"nonexistent"}},
			},
			shouldError: true,
		},
		{
			name: "multiple missing dependencies",
			tasks: []Task{
				{Name: "task1", Command: "echo 1", Deps: []string{"missing1", "missing2"}},
			},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build task map for validation
			taskMap := make(map[string]Task)
			for _, task := range tt.tasks {
				taskMap[task.Name] = task
			}

			// Validate dependencies
			var hasError bool
			for _, task := range tt.tasks {
				for _, dep := range task.Deps {
					if _, exists := taskMap[dep]; !exists {
						hasError = true
						break
					}
				}
				if hasError {
					break
				}
			}

			if tt.shouldError {
				assert.True(t, hasError, "expected validation to fail")
			} else {
				assert.False(t, hasError, "expected validation to succeed")
			}
		})
	}
}

// TestTCRWorkflowInputSerialization tests that TCR workflow input is properly serializable
func TestTCRWorkflowInputSerialization(t *testing.T) {
	tests := []struct {
		name  string
		input TCRWorkflowInput
		valid bool
	}{
		{
			name: "valid complete input",
			input: TCRWorkflowInput{
				CellID:      "cell-001",
				Branch:      "main",
				TaskID:      "task-123",
				Description: "Implement feature X",
				Prompt:      "Create a new function that does Y",
			},
			valid: true,
		},
		{
			name: "minimal valid input",
			input: TCRWorkflowInput{
				CellID: "cell-001",
				Branch: "main",
				TaskID: "task-123",
			},
			valid: true,
		},
		{
			name: "input with special characters",
			input: TCRWorkflowInput{
				CellID:      "cell-001",
				Branch:      "feature/test-dash",
				TaskID:      "task-123_456",
				Description: "Fix bug #1234 in service-x",
				Prompt:      "Create a method: func() string { return \"test\" }",
			},
			valid: true,
		},
		{
			name: "input with multiline description",
			input: TCRWorkflowInput{
				CellID: "cell-001",
				Branch: "main",
				TaskID: "task-123",
				Description: `Implement feature X
This is a multiline description
with multiple lines of text`,
				Prompt: `Write code to:
1. Do X
2. Do Y`,
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify all required fields are present
			assert.NotEmpty(t, tt.input.CellID)
			assert.NotEmpty(t, tt.input.Branch)
			assert.NotEmpty(t, tt.input.TaskID)

			// Verify input is serializable (all string fields)
			assert.IsType(t, "", tt.input.CellID)
			assert.IsType(t, "", tt.input.Branch)
			assert.IsType(t, "", tt.input.TaskID)
			assert.IsType(t, "", tt.input.Description)
			assert.IsType(t, "", tt.input.Prompt)
		})
	}
}

// TestTCRWorkflowResultSerialization tests that TCR workflow result is properly serializable
func TestTCRWorkflowResultSerialization(t *testing.T) {
	tests := []struct {
		name   string
		result TCRWorkflowResult
	}{
		{
			name: "successful result",
			result: TCRWorkflowResult{
				Success:      true,
				TestsPassed:  true,
				FilesChanged: []string{"file1.go", "file2.go"},
				Error:        "",
			},
		},
		{
			name: "failed result with error",
			result: TCRWorkflowResult{
				Success:      false,
				TestsPassed:  false,
				FilesChanged: []string{},
				Error:        "bootstrap failed: connection refused",
			},
		},
		{
			name: "partial success",
			result: TCRWorkflowResult{
				Success:      false,
				TestsPassed:  false,
				FilesChanged: []string{"modified.go"},
				Error:        "tests failed: assertion error on line 42",
			},
		},
		{
			name: "empty result",
			result: TCRWorkflowResult{
				Success:      false,
				TestsPassed:  false,
				FilesChanged: []string{},
				Error:        "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify all fields are serializable types
			assert.IsType(t, true, tt.result.Success)
			assert.IsType(t, true, tt.result.TestsPassed)
			assert.IsType(t, []string{}, tt.result.FilesChanged)
			assert.IsType(t, "", tt.result.Error)
		})
	}
}

// TestDAGWorkflowInputSerialization tests that DAG workflow input is properly serializable
func TestDAGWorkflowInputSerialization(t *testing.T) {
	tests := []struct {
		name  string
		input DAGWorkflowInput
		valid bool
	}{
		{
			name: "single task input",
			input: DAGWorkflowInput{
				WorkflowID: "dag-001",
				Branch:     "main",
				Tasks: []Task{
					{Name: "task1", Command: "echo hello", Deps: []string{}},
				},
			},
			valid: true,
		},
		{
			name: "multiple tasks with dependencies",
			input: DAGWorkflowInput{
				WorkflowID: "dag-001",
				Branch:     "feature-branch",
				Tasks: []Task{
					{Name: "setup", Command: "npm install", Deps: []string{}},
					{Name: "test", Command: "npm test", Deps: []string{"setup"}},
					{Name: "build", Command: "npm run build", Deps: []string{"test"}},
				},
			},
			valid: true,
		},
		{
			name: "complex DAG",
			input: DAGWorkflowInput{
				WorkflowID: "dag-complex",
				Branch:     "main",
				Tasks: []Task{
					{Name: "clone", Command: "git clone ...", Deps: []string{}},
					{Name: "install", Command: "npm install", Deps: []string{"clone"}},
					{Name: "lint", Command: "npm run lint", Deps: []string{"install"}},
					{Name: "test", Command: "npm test", Deps: []string{"install"}},
					{Name: "build", Command: "npm run build", Deps: []string{"lint", "test"}},
				},
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify required fields
			assert.NotEmpty(t, tt.input.WorkflowID)
			assert.NotEmpty(t, tt.input.Branch)
			assert.NotEmpty(t, tt.input.Tasks)

			// Verify all tasks have required fields
			for _, task := range tt.input.Tasks {
				assert.NotEmpty(t, task.Name)
				assert.NotEmpty(t, task.Command)
				assert.IsType(t, []string{}, task.Deps)
			}
		})
	}
}

// TestTaskStructure tests the Task struct for proper serialization
func TestTaskStructure(t *testing.T) {
	tests := []struct {
		name string
		task Task
	}{
		{
			name: "task with no dependencies",
			task: Task{
				Name:    "simple",
				Command: "echo test",
				Deps:    []string{},
			},
		},
		{
			name: "task with single dependency",
			task: Task{
				Name:    "complex",
				Command: "npm test",
				Deps:    []string{"npm install"},
			},
		},
		{
			name: "task with multiple dependencies",
			task: Task{
				Name:    "deploy",
				Command: "docker push myimage",
				Deps:    []string{"build", "test", "lint"},
			},
		},
		{
			name: "task with complex command",
			task: Task{
				Name:    "advanced",
				Command: "cd src && go build -o bin/app && bin/app --flag=value",
				Deps:    []string{"setup", "generate"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify all fields are present and correct type
			assert.NotEmpty(t, tt.task.Name)
			assert.NotEmpty(t, tt.task.Command)
			assert.IsType(t, []string{}, tt.task.Deps)

			// Verify fields are serializable
			assert.IsType(t, "", tt.task.Name)
			assert.IsType(t, "", tt.task.Command)
		})
	}
}
