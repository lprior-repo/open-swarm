// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package agent

import (
	"testing"
)

func TestNewManager(t *testing.T) {
	projectKey := "/test/project"
	manager := NewManager(projectKey)

	if manager == nil {
		t.Fatal("NewManager returned nil")
	}

	if manager.projectKey != projectKey {
		t.Errorf("Expected project key %s, got %s", projectKey, manager.projectKey)
	}

	if manager.agents == nil {
		t.Error("Expected agents map to be initialized")
	}
}

func TestRegisterAgent(t *testing.T) {
	manager := NewManager("/test/project")

	agent := Agent{
		Name:            "TestAgent",
		Program:         "opencode",
		Model:           "sonnet-4.5",
		TaskDescription: "Testing",
		LastActive:      "2025-12-12T00:00:00Z",
	}

	err := manager.Register(agent)
	if err != nil {
		t.Fatalf("Failed to register agent: %v", err)
	}

	// Verify agent was registered
	retrieved, exists := manager.Get("TestAgent")
	if !exists {
		t.Error("Agent not found after registration")
	}

	if retrieved.Name != agent.Name {
		t.Errorf("Expected name %s, got %s", agent.Name, retrieved.Name)
	}

	if retrieved.ProjectKey != "/test/project" {
		t.Errorf("Expected project key to be set to /test/project, got %s", retrieved.ProjectKey)
	}
}

func TestRegisterAgentNoName(t *testing.T) {
	manager := NewManager("/test/project")

	agent := Agent{
		Program: "opencode",
		Model:   "sonnet-4.5",
	}

	err := manager.Register(agent)
	if err == nil {
		t.Error("Expected error when registering agent without name")
	}
}

func TestListAgents(t *testing.T) {
	manager := NewManager("/test/project")

	// Register multiple agents
	agents := []Agent{
		{Name: "Agent1", Program: "opencode", Model: "sonnet-4.5"},
		{Name: "Agent2", Program: "codex-cli", Model: "gpt5-codex"},
		{Name: "Agent3", Program: "opencode", Model: "opus-4.1"},
	}

	for _, agent := range agents {
		if err := manager.Register(agent); err != nil {
			t.Fatalf("Failed to register agent: %v", err)
		}
	}

	// List agents
	listed := manager.List()

	if len(listed) != len(agents) {
		t.Errorf("Expected %d agents, got %d", len(agents), len(listed))
	}
}

func TestCountActive(t *testing.T) {
	manager := NewManager("/test/project")

	if count := manager.CountActive(); count != 0 {
		t.Errorf("Expected 0 active agents, got %d", count)
	}

	// Register agents
	for i := 1; i <= 3; i++ {
		agent := Agent{
			Name:    "Agent" + string(rune('0'+i)),
			Program: "opencode",
			Model:   "sonnet-4.5",
		}
		manager.Register(agent)
	}

	if count := manager.CountActive(); count != 3 {
		t.Errorf("Expected 3 active agents, got %d", count)
	}
}

func TestRemoveAgent(t *testing.T) {
	manager := NewManager("/test/project")

	agent := Agent{
		Name:    "TestAgent",
		Program: "opencode",
		Model:   "sonnet-4.5",
	}

	manager.Register(agent)

	// Verify agent exists
	if _, exists := manager.Get("TestAgent"); !exists {
		t.Error("Agent should exist before removal")
	}

	// Remove agent
	err := manager.Remove("TestAgent")
	if err != nil {
		t.Fatalf("Failed to remove agent: %v", err)
	}

	// Verify agent is gone
	if _, exists := manager.Get("TestAgent"); exists {
		t.Error("Agent should not exist after removal")
	}
}

func TestUpdateAgent(t *testing.T) {
	manager := NewManager("/test/project")

	agent := Agent{
		Name:            "TestAgent",
		Program:         "opencode",
		Model:           "sonnet-4.5",
		TaskDescription: "Original task",
	}

	manager.Register(agent)

	// Update agent
	agent.TaskDescription = "Updated task"
	err := manager.Update(agent)
	if err != nil {
		t.Fatalf("Failed to update agent: %v", err)
	}

	// Verify update
	retrieved, _ := manager.Get("TestAgent")
	if retrieved.TaskDescription != "Updated task" {
		t.Errorf("Expected task description 'Updated task', got '%s'", retrieved.TaskDescription)
	}
}
