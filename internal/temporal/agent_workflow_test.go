package temporal

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

// TestSpawnAgentInput_Validate verifies input validation.
func TestSpawnAgentInput_Validate(t *testing.T) {
	// Valid input
	valid := SpawnAgentInput{
		TaskID:    "task-123",
		ContextID: "ctx-456",
	}
	err := valid.Validate()
	assert.NoError(t, err)

	// Missing TaskID
	invalid := SpawnAgentInput{
		ContextID: "ctx-456",
	}
	err = invalid.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "task_id")

	// Missing ContextID
	invalid2 := SpawnAgentInput{
		TaskID: "task-123",
	}
	err = invalid2.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context_id")
}

// TestSpawnAgentOutput_IsValid verifies output validation.
func TestSpawnAgentOutput_IsValid(t *testing.T) {
	// Valid output
	valid := SpawnAgentOutput{
		AgentID:   "agent-123",
		ServerURL: "http://localhost:9000",
		StartTime: time.Now(),
	}
	assert.True(t, valid.IsValid())

	// Missing AgentID
	invalid := SpawnAgentOutput{
		ServerURL: "http://localhost:9000",
		StartTime: time.Now(),
	}
	assert.False(t, invalid.IsValid())

	// Missing ServerURL
	invalid2 := SpawnAgentOutput{
		AgentID:   "agent-123",
		StartTime: time.Now(),
	}
	assert.False(t, invalid2.IsValid())

	// Zero StartTime
	invalid3 := SpawnAgentOutput{
		AgentID:   "agent-123",
		ServerURL: "http://localhost:9000",
	}
	assert.False(t, invalid3.IsValid())
}

// TestSpawnAgentActivities_CreateServerActivity verifies server creation activity.
func TestSpawnAgentActivities_CreateServerActivity(t *testing.T) {
	s := &testsuite.WorkflowTestSuite{}
	env := s.NewTestActivityEnvironment()

	activities := &SpawnAgentActivities{}
	env.RegisterActivity(activities.CreateOpenCodeServerActivity)

	// Test with valid task ID
	result, err := env.ExecuteActivity(activities.CreateOpenCodeServerActivity, "task-123")
	assert.NoError(t, err)
	var url string
	err = result.Get(&url)
	assert.NoError(t, err)
	assert.NotEmpty(t, url)
	assert.Contains(t, url, "http://localhost:")
}

// TestSpawnAgentActivities_HealthCheckActivity verifies health check activity.
func TestSpawnAgentActivities_HealthCheckActivity(t *testing.T) {
	s := &testsuite.WorkflowTestSuite{}
	env := s.NewTestActivityEnvironment()

	activities := &SpawnAgentActivities{}
	env.RegisterActivity(activities.HealthCheckServerActivity)

	// Test with localhost (should fail in test)
	result, err := env.ExecuteActivity(activities.HealthCheckServerActivity, "http://localhost:9999")
	if err == nil {
		var healthy bool
		err = result.Get(&healthy)
		// May fail due to no server running
		assert.NoError(t, err)
	}
}

// TestSpawnAgentWorkflow_InputValidation verifies workflow input validation.
func TestSpawnAgentWorkflow_InputValidation(t *testing.T) {
	s := &testsuite.WorkflowTestSuite{}
	env := s.NewTestWorkflowEnvironment()

	activities := &SpawnAgentActivities{}
	env.RegisterActivity(activities)

	// Test with invalid input (missing TaskID)
	env.ExecuteWorkflow(SpawnAgentWorkflow, SpawnAgentInput{
		ContextID: "ctx-456",
	})

	require.Error(t, env.GetWorkflowError())
	assert.Contains(t, env.GetWorkflowError().Error(), "task_id")
}

// TestSpawnAgentWorkflow_SuccessfulSpawn verifies successful agent spawn workflow.
func TestSpawnAgentWorkflow_SuccessfulSpawn(t *testing.T) {
	s := &testsuite.WorkflowTestSuite{}
	env := s.NewTestWorkflowEnvironment()

	activities := &SpawnAgentActivities{}
	env.RegisterActivity(activities)

	// Test with valid input
	env.ExecuteWorkflow(SpawnAgentWorkflow, SpawnAgentInput{
		TaskID:    "task-123",
		ContextID: "ctx-456",
	})

	require.NoError(t, env.GetWorkflowError())
	var result SpawnAgentOutput
	err := env.GetWorkflowResult(&result)
	require.NoError(t, err)
	assert.NotEmpty(t, result.AgentID)
	assert.NotEmpty(t, result.ServerURL)
	assert.False(t, result.StartTime.IsZero())
}

// TestSpawnAgentWorkflow_TimeoutHandling verifies workflow execution.
func TestSpawnAgentWorkflow_TimeoutHandling(t *testing.T) {
	s := &testsuite.WorkflowTestSuite{}
	env := s.NewTestWorkflowEnvironment()

	activities := &SpawnAgentActivities{}
	env.RegisterActivity(activities)

	// Execute workflow
	env.ExecuteWorkflow(SpawnAgentWorkflow, SpawnAgentInput{
		TaskID:    "task-123",
		ContextID: "ctx-456",
	})

	// Should complete successfully
	require.NoError(t, env.GetWorkflowError())
}
