// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

// Package spawner provides ephemeral agent spawning and lifecycle management.
// Each agent gets an isolated OpenCode server (cell) for bounded execution.
package spawner

import (
	"context"
	"fmt"
	"time"

	agentinternal "open-swarm/internal/agent"
	"open-swarm/internal/infra"
	"open-swarm/internal/workflow"
	agentpkg "open-swarm/pkg/agent"
)

// SpawnConfig contains configuration for spawning a new ephemeral agent
type SpawnConfig struct {
	AgentID     string        // Unique agent identifier
	TaskID      string        // Beads task ID to work on
	Branch      string        // Git branch for worktree (default: main)
	Timeout     time.Duration // Max execution time (default: 30min)
	TokenBudget int           // Token budget for agent (default: 50000)
}

// AgentSpawner manages ephemeral agent lifecycle (spawn → execute → teardown)
type AgentSpawner struct {
	portManager     infra.PortManagerInterface
	serverManager   infra.ServerManagerInterface
	worktreeManager infra.WorktreeManagerInterface
	activities      *workflow.Activities
}

// NewAgentSpawner creates a new agent spawner
func NewAgentSpawner(
	portMgr infra.PortManagerInterface,
	serverMgr infra.ServerManagerInterface,
	worktreeMgr infra.WorktreeManagerInterface,
) *AgentSpawner {
	return &AgentSpawner{
		portManager:     portMgr,
		serverManager:   serverMgr,
		worktreeManager: worktreeMgr,
		activities:      workflow.NewActivities(portMgr, serverMgr, worktreeMgr),
	}
}

// SpawnedAgent represents an ephemeral agent instance
type SpawnedAgent struct {
	ID        string                  // Agent identifier
	TaskID    string                  // Beads task being executed
	Cell      *workflow.CellBootstrap // Isolated execution environment
	CreatedAt time.Time               // Spawn timestamp
	Agent     *agentpkg.Agent         // Agent metadata
	TokensUsed int                    // Tokens consumed so far
}

// LifecycleMetrics tracks agent execution metrics
type LifecycleMetrics struct {
	SpawnDuration    time.Duration
	ExecutionDuration time.Duration
	TeardownDuration  time.Duration
	TotalDuration    time.Duration
}

// SpawnAgent creates a new ephemeral agent with isolated execution environment
// Returns a SpawnedAgent ready for execution, or error if spawn fails
func (as *AgentSpawner) SpawnAgent(ctx context.Context, config SpawnConfig) (*SpawnedAgent, error) {
	// Validate config
	if config.AgentID == "" {
		return nil, fmt.Errorf("agent ID required")
	}
	if config.TaskID == "" {
		return nil, fmt.Errorf("task ID required")
	}

	// Set defaults
	if config.Branch == "" {
		config.Branch = "main"
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Minute
	}
	if config.TokenBudget == 0 {
		config.TokenBudget = 50000
	}

	// Bootstrap isolated cell (port + worktree + server)
	cell, err := as.activities.BootstrapCell(ctx, config.AgentID, config.Branch)
	if err != nil {
		return nil, fmt.Errorf("failed to bootstrap cell for agent %s: %w", config.AgentID, err)
	}

	// Create spawned agent instance
	spawned := &SpawnedAgent{
		ID:        config.AgentID,
		TaskID:    config.TaskID,
		Cell:      cell,
		CreatedAt: time.Now(),
		Agent: &agentpkg.Agent{
			Name:            config.AgentID,
			Program:         "temporal-workflow",
			Model:           "claude-opus-4.5",
			TaskDescription: config.TaskID,
			ProjectKey:      "open-swarm",
		},
		TokensUsed: 0,
	}

	return spawned, nil
}

// ExecuteTask runs a task within the spawned agent's cell
func (as *AgentSpawner) ExecuteTask(
	ctx context.Context,
	spawned *SpawnedAgent,
	prompt string,
) (*agentinternal.ExecutionResult, error) {
	if spawned == nil || spawned.Cell == nil {
		return nil, fmt.Errorf("invalid spawned agent or cell")
	}

	// Create task context
	taskCtx := &agentinternal.TaskContext{
		TaskID: spawned.TaskID,
		Prompt: prompt,
	}

	// Execute via cell activities
	result, err := as.activities.ExecuteTask(ctx, spawned.Cell, taskCtx)
	if err != nil {
		return nil, fmt.Errorf("task execution failed for agent %s: %w", spawned.ID, err)
	}

	return result, nil
}

// RunTests executes tests in the agent's cell
func (as *AgentSpawner) RunTests(ctx context.Context, spawned *SpawnedAgent) (bool, error) {
	if spawned == nil || spawned.Cell == nil {
		return false, fmt.Errorf("invalid spawned agent or cell")
	}

	passed, err := as.activities.RunTests(ctx, spawned.Cell)
	if err != nil {
		return false, fmt.Errorf("test execution failed for agent %s: %w", spawned.ID, err)
	}

	return passed, nil
}

// CommitChanges commits modifications in the agent's cell
func (as *AgentSpawner) CommitChanges(
	ctx context.Context,
	spawned *SpawnedAgent,
	message string,
) error {
	if spawned == nil || spawned.Cell == nil {
		return fmt.Errorf("invalid spawned agent or cell")
	}

	err := as.activities.CommitChanges(ctx, spawned.Cell, message)
	if err != nil {
		return fmt.Errorf("commit failed for agent %s: %w", spawned.ID, err)
	}

	return nil
}

// RevertChanges reverts all modifications in the agent's cell
func (as *AgentSpawner) RevertChanges(ctx context.Context, spawned *SpawnedAgent) error {
	if spawned == nil || spawned.Cell == nil {
		return fmt.Errorf("invalid spawned agent or cell")
	}

	err := as.activities.RevertChanges(ctx, spawned.Cell)
	if err != nil {
		return fmt.Errorf("revert failed for agent %s: %w", spawned.ID, err)
	}

	return nil
}

// TeardownAgent destroys an ephemeral agent and releases all resources
// This is critical - must be called to prevent resource leaks
func (as *AgentSpawner) TeardownAgent(ctx context.Context, spawned *SpawnedAgent) *LifecycleMetrics {
	startTime := time.Now()
	metrics := &LifecycleMetrics{}

	if spawned == nil || spawned.Cell == nil {
		return metrics
	}

	// Cleanup cell resources (port, worktree, server)
	if err := as.activities.TeardownCell(ctx, spawned.Cell); err != nil {
		// Log but continue cleanup (best effort)
		fmt.Printf("warning: teardown error for agent %s: %v\n", spawned.ID, err)
	}

	metrics.TeardownDuration = time.Since(startTime)
	return metrics
}

// IsHealthy checks if an agent's cell is still operational
func (as *AgentSpawner) IsHealthy(ctx context.Context, spawned *SpawnedAgent) bool {
	if spawned == nil || spawned.Cell == nil {
		return false
	}

	return as.serverManager.IsHealthy(ctx, spawned.Cell.ServerHandle)
}

// GetCellInfo returns information about an agent's isolated cell
func (as *AgentSpawner) GetCellInfo(spawned *SpawnedAgent) map[string]interface{} {
	if spawned == nil || spawned.Cell == nil {
		return nil
	}

	return map[string]interface{}{
		"cell_id":       spawned.Cell.CellID,
		"port":          spawned.Cell.Port,
		"worktree_id":   spawned.Cell.WorktreeID,
		"worktree_path": spawned.Cell.WorktreePath,
		"base_url":      spawned.Cell.ServerHandle.BaseURL,
		"created_at":    spawned.CreatedAt,
		"tokens_used":   spawned.TokensUsed,
	}
}
