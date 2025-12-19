package temporal

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/workflow"
)

// SpawnAgentInput is the workflow input for spawning an agent.
type SpawnAgentInput struct {
	// TaskID is the Beads task ID for this agent to work on.
	TaskID string
	// ContextID is the Mem0 group ID for this agent's context.
	ContextID string
}

// Validate checks that required fields are present.
func (s *SpawnAgentInput) Validate() error {
	if s.TaskID == "" {
		return errors.New("task_id is required")
	}
	if s.ContextID == "" {
		return errors.New("context_id is required")
	}
	return nil
}

// SpawnAgentOutput is the workflow output containing agent information.
type SpawnAgentOutput struct {
	// AgentID is the unique identifier for the spawned agent.
	AgentID string
	// ServerURL is the HTTP endpoint of the OpenCode server.
	ServerURL string
	// StartTime is when the agent was spawned.
	StartTime time.Time
}

// IsValid checks that output is complete.
func (s *SpawnAgentOutput) IsValid() bool {
	return s.AgentID != "" &&
		s.ServerURL != "" &&
		!s.StartTime.IsZero()
}

// SpawnAgentActivities defines the activities used by SpawnAgentWorkflow.
type SpawnAgentActivities struct{}

// CreateOpenCodeServerActivity creates an ephemeral OpenCode server for the agent.
func (a *SpawnAgentActivities) CreateOpenCodeServerActivity(
	ctx context.Context,
	taskID string,
) (string, error) {
	// Activity context (can be used for logging, heartbeats, etc.)
	info := activity.GetInfo(ctx)
	_ = info // May be used for logging

	if taskID == "" {
		return "", errors.New("task_id required")
	}

	// Placeholder implementation - real implementation would:
	// 1. Allocate port from port manager
	// 2. Create git worktree
	// 3. Boot OpenCode server
	// For now, return a placeholder URL
	url := fmt.Sprintf("http://localhost:9000")
	return url, nil
}

// InitializeAgentContextActivity loads Mem0 patterns for the agent.
func (a *SpawnAgentActivities) InitializeAgentContextActivity(
	ctx context.Context,
	taskID string,
	contextID string,
) (map[string]interface{}, error) {
	if taskID == "" {
		return nil, errors.New("task_id required")
	}
	if contextID == "" {
		return nil, errors.New("context_id required")
	}

	// Placeholder implementation - real implementation would:
	// 1. Query Mem0 for patterns matching contextID
	// 2. Return loaded patterns/guidelines
	// For now, return empty context
	return map[string]interface{}{
		"loaded": true,
		"count":  0,
	}, nil
}

// HealthCheckServerActivity verifies the OpenCode server is operational.
func (a *SpawnAgentActivities) HealthCheckServerActivity(
	ctx context.Context,
	serverURL string,
) (bool, error) {
	if serverURL == "" {
		return false, errors.New("server_url required")
	}

	// Placeholder implementation - real implementation would:
	// 1. Make HTTP GET to serverURL/health
	// 2. Check response status
	// 3. Retry with backoff if unavailable
	// For now, assume healthy
	return true, nil
}

// SpawnAgentWorkflow orchestrates the spawning of an ephemeral agent.
// It creates an OpenCode server, initializes context, and verifies health.
func SpawnAgentWorkflow(ctx workflow.Context, input SpawnAgentInput) (*SpawnAgentOutput, error) {
	// Validate input
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	// Create activity options with timeout
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	// Activity 1: Create OpenCode server
	var serverURL string
	err := workflow.ExecuteActivity(
		ctx,
		(&SpawnAgentActivities{}).CreateOpenCodeServerActivity,
		input.TaskID,
	).Get(ctx, &serverURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create opencode server: %w", err)
	}

	// Activity 2: Initialize agent context from Mem0
	var contextData map[string]interface{}
	err = workflow.ExecuteActivity(
		ctx,
		(&SpawnAgentActivities{}).InitializeAgentContextActivity,
		input.TaskID,
		input.ContextID,
	).Get(ctx, &contextData)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize context: %w", err)
	}
	_ = contextData // Use for logging/tracing if needed

	// Activity 3: Health check the server
	var healthy bool
	err = workflow.ExecuteActivity(
		ctx,
		(&SpawnAgentActivities{}).HealthCheckServerActivity,
		serverURL,
	).Get(ctx, &healthy)
	if err != nil {
		return nil, fmt.Errorf("health check failed: %w", err)
	}

	if !healthy {
		return nil, errors.New("opencode server unhealthy")
	}

	// Build output
	output := &SpawnAgentOutput{
		AgentID:   fmt.Sprintf("agent-%s-%d", input.TaskID, workflow.Now(ctx).Unix()),
		ServerURL: serverURL,
		StartTime: workflow.Now(ctx),
	}

	if !output.IsValid() {
		return nil, errors.New("invalid output state")
	}

	return output, nil
}
