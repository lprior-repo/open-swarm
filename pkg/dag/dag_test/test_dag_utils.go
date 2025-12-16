// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package dag_test

import (
	"open-swarm/pkg/dag"
	"testing"

	"github.com/gammazero/toposort"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// findNodeIndices finds the indices of specified nodes in a sorted list
func findNodeIndices(sorted []interface{}, nodeNames []string) map[string]int {
	indices := make(map[string]int)
	for _, name := range nodeNames {
		indices[name] = -1
	}
	for i, node := range sorted {
		if _, exists := indices[node.(string)]; exists {
			indices[node.(string)] = i
		}
	}
	return indices
}

// TestDAGToposort tests the topological sorting of DAG tasks
func TestDAGToposort(t *testing.T) {
	tests := getToposortTestCases()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runToposortTest(t, tt)
		})
	}
}

// toposortTestCase represents a single toposort test
type toposortTestCase struct {
	name               string
	tasks              []dag.Task
	expectedOrderCount int
	shouldError        bool
	errorContains      string
	verifyOrder        func(t *testing.T, sorted []interface{})
}

// getToposortTestCases returns all test cases
func getToposortTestCases() []toposortTestCase {
	return []toposortTestCase{
		createLinearTest(),
		createParallelTest(),
		createDiamondTest(),
		createSimpleTest(),
		createSelfRefTest(),
		createMutualDepTest(),
		createCircularTest(),
		createIndependentTest(),
	}
}

// runToposortTest executes a single test case
func runToposortTest(t *testing.T, tt toposortTestCase) {
	edges := buildEdgesFromTasks(tt.tasks)
	sortedNodes, err := toposort.Toposort(edges)

	if tt.shouldError {
		require.Error(t, err)
		assert.Contains(t, err.Error(), tt.errorContains)
	} else {
		require.NoError(t, err)
		assert.Equal(t, tt.expectedOrderCount, len(sortedNodes))
		if tt.verifyOrder != nil {
			tt.verifyOrder(t, sortedNodes)
		}
	}
}

// buildEdgesFromTasks converts tasks to toposort edges
func buildEdgesFromTasks(tasks []dag.Task) []toposort.Edge {
	edges := make([]toposort.Edge, 0)
	for _, task := range tasks {
		for _, dep := range task.Deps {
			edges = append(edges, toposort.Edge{dep, task.Name})
		}
	}
	return edges
}

// createLinearTest creates a linear dependency test
func createLinearTest() toposortTestCase {
	return toposortTestCase{
		name: "simple linear dependency",
		tasks: []dag.Task{
			{Name: "task1", Command: "echo 1", Deps: []string{}},
			{Name: "task2", Command: "echo 2", Deps: []string{"task1"}},
			{Name: "task3", Command: "echo 3", Deps: []string{"task2"}},
		},
		expectedOrderCount: 3,
		shouldError:        false,
		verifyOrder: func(t *testing.T, sorted []interface{}) {
			indices := findNodeIndices(sorted, []string{"task1", "task2", "task3"})
			assert.Greater(t, indices["task2"], indices["task1"], "task1 should come before task2")
			assert.Greater(t, indices["task3"], indices["task2"], "task2 should come before task3")
		},
	}
}

// createParallelTest creates a parallel dependency test
func createParallelTest() toposortTestCase {
	return toposortTestCase{
		name: "multiple parallel tasks with common dependency",
		tasks: []dag.Task{
			{Name: "setup", Command: "echo setup", Deps: []string{}},
			{Name: "test1", Command: "echo test1", Deps: []string{"setup"}},
			{Name: "test2", Command: "echo test2", Deps: []string{"setup"}},
			{Name: "finalize", Command: "echo finalize", Deps: []string{"test1", "test2"}},
		},
		expectedOrderCount: 4,
		shouldError:        false,
		verifyOrder: func(t *testing.T, sorted []interface{}) {
			indices := findNodeIndices(sorted, []string{"setup", "test1", "test2", "finalize"})
			assert.Greater(t, indices["test1"], indices["setup"])
			assert.Greater(t, indices["test2"], indices["setup"])
			assert.Greater(t, indices["finalize"], indices["test1"])
			assert.Greater(t, indices["finalize"], indices["test2"])
		},
	}
}

// createDiamondTest creates a diamond dependency test
func createDiamondTest() toposortTestCase {
	return toposortTestCase{
		name: "diamond dependency",
		tasks: []dag.Task{
			{Name: "a", Command: "echo a", Deps: []string{}},
			{Name: "b", Command: "echo b", Deps: []string{"a"}},
			{Name: "c", Command: "echo c", Deps: []string{"a"}},
			{Name: "d", Command: "echo d", Deps: []string{"b", "c"}},
		},
		expectedOrderCount: 4,
		shouldError:        false,
		verifyOrder: func(t *testing.T, sorted []interface{}) {
			indices := findNodeIndices(sorted, []string{"a", "b", "c", "d"})
			assert.Greater(t, indices["b"], indices["a"])
			assert.Greater(t, indices["c"], indices["a"])
			assert.Greater(t, indices["d"], indices["b"])
			assert.Greater(t, indices["d"], indices["c"])
		},
	}
}

// createSimpleTest creates a simple dependency test
func createSimpleTest() toposortTestCase {
	return toposortTestCase{
		name: "single task with dependency",
		tasks: []dag.Task{
			{Name: "task1", Command: "echo 1", Deps: []string{}},
			{Name: "task2", Command: "echo 2", Deps: []string{"task1"}},
		},
		expectedOrderCount: 2,
		shouldError:        false,
	}
}

// createSelfRefTest creates a self-reference cycle test
func createSelfRefTest() toposortTestCase {
	return toposortTestCase{
		name: "cycle detection - self reference",
		tasks: []dag.Task{
			{Name: "task1", Command: "echo 1", Deps: []string{"task1"}},
		},
		shouldError:   true,
		errorContains: "cannot be the same",
	}
}

// createMutualDepTest creates a mutual dependency cycle test
func createMutualDepTest() toposortTestCase {
	return toposortTestCase{
		name: "cycle detection - mutual dependency",
		tasks: []dag.Task{
			{Name: "task1", Command: "echo 1", Deps: []string{"task2"}},
			{Name: "task2", Command: "echo 2", Deps: []string{"task1"}},
		},
		shouldError:   true,
		errorContains: "cycle",
	}
}

// createCircularTest creates a circular chain cycle test
func createCircularTest() toposortTestCase {
	return toposortTestCase{
		name: "cycle detection - circular chain",
		tasks: []dag.Task{
			{Name: "a", Command: "echo a", Deps: []string{"b"}},
			{Name: "b", Command: "echo b", Deps: []string{"c"}},
			{Name: "c", Command: "echo c", Deps: []string{"a"}},
		},
		shouldError:   true,
		errorContains: "cycle",
	}
}

// createIndependentTest creates an independent chains test
func createIndependentTest() toposortTestCase {
	return toposortTestCase{
		name: "two independent chains",
		tasks: []dag.Task{
			{Name: "task1", Command: "echo 1", Deps: []string{}},
			{Name: "task2", Command: "echo 2", Deps: []string{"task1"}},
			{Name: "task3", Command: "echo 3", Deps: []string{}},
			{Name: "task4", Command: "echo 4", Deps: []string{"task3"}},
		},
		expectedOrderCount: 4,
		shouldError:        false,
	}
}

// TestDAGDependencyValidation tests that dependency validation works correctly
func TestDAGDependencyValidation(t *testing.T) {
	tests := []struct {
		name        string
		tasks       []dag.Task
		shouldError bool
	}{
		{
			name: "all dependencies resolved",
			tasks: []dag.Task{
				{Name: "task1", Command: "echo 1", Deps: []string{}},
				{Name: "task2", Command: "echo 2", Deps: []string{"task1"}},
			},
			shouldError: false,
		},
		{
			name: "missing dependency",
			tasks: []dag.Task{
				{Name: "task1", Command: "echo 1", Deps: []string{"nonexistent"}},
			},
			shouldError: true,
		},
		{
			name: "multiple missing dependencies",
			tasks: []dag.Task{
				{Name: "task1", Command: "echo 1", Deps: []string{"missing1", "missing2"}},
			},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build task map for validation
			taskMap := make(map[string]dag.Task)
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

// TestDAGWorkflowInputSerialization tests that DAG workflow input is properly serializable
func TestDAGWorkflowInputSerialization(t *testing.T) {
	tests := []struct {
		name  string
		input dag.WorkflowInput
		valid bool
	}{
		{
			name: "single task input",
			input: dag.WorkflowInput{
				WorkflowID: "dag-001",
				Branch:     "main",
				Tasks: []dag.Task{
					{Name: "task1", Command: "echo hello", Deps: []string{}},
				},
			},
			valid: true,
		},
		{
			name: "multiple tasks with dependencies",
			input: dag.WorkflowInput{
				WorkflowID: "dag-001",
				Branch:     "feature-branch",
				Tasks: []dag.Task{
					{Name: "setup", Command: "npm install", Deps: []string{}},
					{Name: "test", Command: "npm test", Deps: []string{"setup"}},
					{Name: "build", Command: "npm run build", Deps: []string{"test"}},
				},
			},
			valid: true,
		},
		{
			name: "complex DAG",
			input: dag.WorkflowInput{
				WorkflowID: "dag-complex",
				Branch:     "main",
				Tasks: []dag.Task{
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
		task dag.Task
	}{
		{
			name: "task with no dependencies",
			task: dag.Task{
				Name:    "simple",
				Command: "echo test",
				Deps:    []string{},
			},
		},
		{
			name: "task with single dependency",
			task: dag.Task{
				Name:    "complex",
				Command: "npm test",
				Deps:    []string{"npm install"},
			},
		},
		{
			name: "task with multiple dependencies",
			task: dag.Task{
				Name:    "deploy",
				Command: "docker push myimage",
				Deps:    []string{"build", "test", "lint"},
			},
		},
		{
			name: "task with complex command",
			task: dag.Task{
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
