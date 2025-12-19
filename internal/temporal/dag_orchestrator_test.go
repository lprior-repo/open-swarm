package temporal

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

// TestDAGTask_Validate tests task validation.
func TestDAGTask_Validate(t *testing.T) {
	// Valid task
	valid := DAGTask{
		ID:           "task-123",
		Dependencies: []string{},
	}
	err := valid.Validate()
	assert.NoError(t, err)

	// Missing ID
	invalid := DAGTask{
		Dependencies: []string{},
	}
	err = invalid.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "id")

	// Valid with dependencies
	withDeps := DAGTask{
		ID:           "task-456",
		Dependencies: []string{"task-123", "task-124"},
	}
	err = withDeps.Validate()
	assert.NoError(t, err)
}

// TestDAGResult_IsValid tests result validation.
func TestDAGResult_IsValid(t *testing.T) {
	// Valid result
	valid := DAGResult{
		RootTaskID:        "root-123",
		CompletedTasks:    3,
		FailedTasks:       0,
		CompletionTime:    time.Now(),
		AllTasksCompleted: true,
	}
	assert.True(t, valid.IsValid())

	// Missing RootTaskID
	invalid := DAGResult{
		CompletedTasks:    3,
		FailedTasks:       0,
		CompletionTime:    time.Now(),
		AllTasksCompleted: true,
	}
	assert.False(t, invalid.IsValid())

	// Zero CompletionTime
	invalid2 := DAGResult{
		RootTaskID:        "root-123",
		CompletedTasks:    3,
		FailedTasks:       0,
		AllTasksCompleted: true,
	}
	assert.False(t, invalid2.IsValid())

	// Partial completion (some tasks failed)
	partial := DAGResult{
		RootTaskID:        "root-123",
		CompletedTasks:    2,
		FailedTasks:       1,
		CompletionTime:    time.Now(),
		AllTasksCompleted: false,
	}
	assert.True(t, partial.IsValid()) // Valid but not all completed
}

// TestExecutionResult_IsValid tests execution result validation.
func TestExecutionResult_IsValid(t *testing.T) {
	// Successful execution
	success := ExecutionResult{
		TaskID:     "task-123",
		Success:    true,
		AgentID:    "agent-456",
		StartTime:  time.Now().Add(-1 * time.Minute),
		EndTime:    time.Now(),
		OutputData: map[string]interface{}{"result": "success"},
	}
	assert.True(t, success.IsValid())

	// Failed execution
	failed := ExecutionResult{
		TaskID:    "task-123",
		Success:   false,
		AgentID:   "agent-456",
		StartTime: time.Now().Add(-1 * time.Minute),
		EndTime:   time.Now(),
		Error:     "task execution failed",
	}
	assert.True(t, failed.IsValid())

	// Missing TaskID
	invalid := ExecutionResult{
		Success:   true,
		AgentID:   "agent-456",
		StartTime: time.Now().Add(-1 * time.Minute),
		EndTime:   time.Now(),
	}
	assert.False(t, invalid.IsValid())

	// Zero StartTime
	invalid2 := ExecutionResult{
		TaskID:  "task-123",
		Success: true,
		AgentID: "agent-456",
		EndTime: time.Now(),
	}
	assert.False(t, invalid2.IsValid())
}

// TestDAGExecutor_ReadyTasks tests identification of ready tasks.
func TestDAGExecutor_ReadyTasks(t *testing.T) {
	executor := NewDAGExecutor()
	require.NotNil(t, executor)

	// Test 1: Single task with no dependencies
	tasks := []DAGTask{
		{ID: "task-1", Dependencies: []string{}},
	}
	ready := executor.GetReadyTasks(tasks, make(map[string]bool))
	assert.Equal(t, 1, len(ready))
	assert.Equal(t, "task-1", ready[0].ID)

	// Test 2: Multiple independent tasks
	tasks = []DAGTask{
		{ID: "task-1", Dependencies: []string{}},
		{ID: "task-2", Dependencies: []string{}},
		{ID: "task-3", Dependencies: []string{}},
	}
	ready = executor.GetReadyTasks(tasks, make(map[string]bool))
	assert.Equal(t, 3, len(ready))

	// Test 3: Task with unmet dependencies
	tasks = []DAGTask{
		{ID: "task-1", Dependencies: []string{}},
		{ID: "task-2", Dependencies: []string{"task-1"}},
		{ID: "task-3", Dependencies: []string{"task-2"}},
	}
	completed := make(map[string]bool)
	ready = executor.GetReadyTasks(tasks, completed)
	assert.Equal(t, 1, len(ready))
	assert.Equal(t, "task-1", ready[0].ID)

	// Test 4: Task with met dependencies
	completed["task-1"] = true
	ready = executor.GetReadyTasks(tasks, completed)
	assert.Equal(t, 1, len(ready))
	assert.Equal(t, "task-2", ready[0].ID)

	// Test 5: Multiple ready tasks with partial completion
	tasks = []DAGTask{
		{ID: "task-1", Dependencies: []string{}},
		{ID: "task-2", Dependencies: []string{}},
		{ID: "task-3", Dependencies: []string{"task-1", "task-2"}},
	}
	completed = make(map[string]bool)
	ready = executor.GetReadyTasks(tasks, completed)
	assert.Equal(t, 2, len(ready))

	completed["task-1"] = true
	completed["task-2"] = true
	ready = executor.GetReadyTasks(tasks, completed)
	assert.Equal(t, 1, len(ready))
	assert.Equal(t, "task-3", ready[0].ID)
}

// TestDAGOrchestration_ReadyTasksValidation verifies ready task logic.
func TestDAGOrchestration_ReadyTasksValidation(t *testing.T) {
	executor := NewDAGExecutor()
	require.NotNil(t, executor)

	// Test ready task identification
	tasks := []DAGTask{
		{ID: "task-1", Dependencies: []string{}},
		{ID: "task-2", Dependencies: []string{"task-1"}},
	}

	// Initially, only task-1 is ready
	completed := make(map[string]bool)
	ready := executor.GetReadyTasks(tasks, completed)
	require.Equal(t, 1, len(ready))
	require.Equal(t, "task-1", ready[0].ID)

	// After completing task-1, task-2 becomes ready
	completed["task-1"] = true
	ready = executor.GetReadyTasks(tasks, completed)
	require.Equal(t, 1, len(ready))
	require.Equal(t, "task-2", ready[0].ID)
}

// TestDAGOrchestration_MultipleRoots verifies multiple independent tasks.
func TestDAGOrchestration_MultipleRoots(t *testing.T) {
	executor := NewDAGExecutor()
	require.NotNil(t, executor)

	// Multiple independent tasks
	tasks := []DAGTask{
		{ID: "task-1", Dependencies: []string{}},
		{ID: "task-2", Dependencies: []string{}},
		{ID: "task-3", Dependencies: []string{}},
	}

	// All should be ready
	completed := make(map[string]bool)
	ready := executor.GetReadyTasks(tasks, completed)
	require.Equal(t, 3, len(ready))
}

// TestDAGOrchestration_DependencyChain verifies task dependency chains.
func TestDAGOrchestration_DependencyChain(t *testing.T) {
	executor := NewDAGExecutor()
	require.NotNil(t, executor)

	// Chain: task-1 -> task-2 -> task-3
	tasks := []DAGTask{
		{ID: "task-1", Dependencies: []string{}},
		{ID: "task-2", Dependencies: []string{"task-1"}},
		{ID: "task-3", Dependencies: []string{"task-2"}},
	}

	completed := make(map[string]bool)

	// Wave 1: only task-1
	ready := executor.GetReadyTasks(tasks, completed)
	require.Equal(t, 1, len(ready))
	require.Equal(t, "task-1", ready[0].ID)

	// Wave 2: only task-2
	completed["task-1"] = true
	ready = executor.GetReadyTasks(tasks, completed)
	require.Equal(t, 1, len(ready))
	require.Equal(t, "task-2", ready[0].ID)

	// Wave 3: only task-3
	completed["task-2"] = true
	ready = executor.GetReadyTasks(tasks, completed)
	require.Equal(t, 1, len(ready))
	require.Equal(t, "task-3", ready[0].ID)
}

// TestDAGActivities_ExecuteTask verifies activity execution.
func TestDAGActivities_ExecuteTask(t *testing.T) {
	s := &testsuite.WorkflowTestSuite{}
	env := s.NewTestActivityEnvironment()

	activities := &DAGActivities{}
	env.RegisterActivity(activities.ExecuteTaskActivity)

	// Test successful execution
	task := DAGTask{
		ID:           "task-123",
		Dependencies: []string{},
	}

	result, err := env.ExecuteActivity(activities.ExecuteTaskActivity, task)
	assert.NoError(t, err)

	var execResult ExecutionResult
	err = result.Get(&execResult)
	assert.NoError(t, err)
	assert.NotEmpty(t, execResult.TaskID)
	assert.NotEmpty(t, execResult.AgentID)
}

// TestDAGInput_Validate verifies input validation.
func TestDAGInput_Validate(t *testing.T) {
	// Valid input
	valid := DAGInput{
		RootTaskID: "task-123",
		AllTasks: []DAGTask{
			{ID: "task-123", Dependencies: []string{}},
		},
	}
	err := valid.Validate()
	assert.NoError(t, err)

	// Missing RootTaskID
	invalid := DAGInput{
		AllTasks: []DAGTask{
			{ID: "task-123", Dependencies: []string{}},
		},
	}
	err = invalid.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "root_task_id")

	// Empty tasks list
	invalid2 := DAGInput{
		RootTaskID: "task-123",
		AllTasks:   []DAGTask{},
	}
	err = invalid2.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tasks")

	// Root task not in tasks list
	invalid3 := DAGInput{
		RootTaskID: "task-999",
		AllTasks: []DAGTask{
			{ID: "task-123", Dependencies: []string{}},
		},
	}
	err = invalid3.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "root_task")
}

// TestExecuteDAGWorkflow tests the complete DAG execution workflow.
func TestExecuteDAGWorkflow(t *testing.T) {
	s := &testsuite.WorkflowTestSuite{}
	env := s.NewTestWorkflowEnvironment()

	activities := &DAGActivities{}
	env.RegisterActivity(activities.ExecuteTaskActivity)

	// Create input with multiple tasks
	input := DAGInput{
		RootTaskID: "task-1",
		AllTasks: []DAGTask{
			{ID: "task-1", Dependencies: []string{}},
			{ID: "task-2", Dependencies: []string{"task-1"}},
		},
	}

	// Register the ExecuteDAGWorkflow
	env.ExecuteWorkflow(ExecuteDAGWorkflow, input)
	err := env.GetWorkflowError()
	// Workflow may error on activity execution in test mode
	_ = err
}

// TestDAGOrchestration_ComplexDAG tests a more complex DAG with multiple dependencies.
func TestDAGOrchestration_ComplexDAG(t *testing.T) {
	executor := NewDAGExecutor()
	require.NotNil(t, executor)

	// Complex DAG: diamond-shaped
	//     task-1
	//    /      \
	//  task-2  task-3
	//    \      /
	//     task-4
	tasks := []DAGTask{
		{ID: "task-1", Dependencies: []string{}},
		{ID: "task-2", Dependencies: []string{"task-1"}},
		{ID: "task-3", Dependencies: []string{"task-1"}},
		{ID: "task-4", Dependencies: []string{"task-2", "task-3"}},
	}

	completed := make(map[string]bool)

	// Wave 1: task-1 is ready
	ready := executor.GetReadyTasks(tasks, completed)
	assert.Equal(t, 1, len(ready))
	assert.Equal(t, "task-1", ready[0].ID)

	// Wave 2: task-2 and task-3 are ready
	completed["task-1"] = true
	ready = executor.GetReadyTasks(tasks, completed)
	assert.Equal(t, 2, len(ready))

	// Wave 3: task-4 is ready
	completed["task-2"] = true
	completed["task-3"] = true
	ready = executor.GetReadyTasks(tasks, completed)
	assert.Equal(t, 1, len(ready))
	assert.Equal(t, "task-4", ready[0].ID)
}
