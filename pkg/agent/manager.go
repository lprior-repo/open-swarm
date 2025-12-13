// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package agent

import (
	"fmt"
	"log/slog"
	"sync"
)

// Agent represents an agent working in the project
type Agent struct {
	Name            string
	Program         string
	Model           string
	TaskDescription string
	LastActive      string
	ProjectKey      string
}

// Manager handles agent registration and tracking
type Manager struct {
	projectKey string
	agents     map[string]Agent
	mu         sync.RWMutex
}

// NewManager creates a new agent manager
func NewManager(projectKey string) *Manager {
	return &Manager{
		projectKey: projectKey,
		agents:     make(map[string]Agent),
	}
}

// Register registers a new agent
func (m *Manager) Register(a Agent) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if a.Name == "" {
		slog.Error("Agent registration failed: name is required")
		return fmt.Errorf("agent name is required")
	}

	// Check if agent already exists
	if existing, exists := m.agents[a.Name]; exists {
		slog.Info("Agent re-registration (updating existing agent)",
			"name", a.Name,
			"previous_task", existing.TaskDescription,
			"new_task", a.TaskDescription,
			"model", a.Model)
	} else {
		slog.Info("New agent registered",
			"name", a.Name,
			"program", a.Program,
			"model", a.Model,
			"task", a.TaskDescription,
			"project", m.projectKey)
	}

	a.ProjectKey = m.projectKey
	m.agents[a.Name] = a

	slog.Info("Active agents in project",
		"count", len(m.agents),
		"project", m.projectKey)

	return nil
}

// Get retrieves an agent by name
func (m *Manager) Get(name string) (Agent, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	agent, exists := m.agents[name]
	return agent, exists
}

// List returns all registered agents
func (m *Manager) List() []Agent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	agents := make([]Agent, 0, len(m.agents))
	for _, agent := range m.agents {
		agents = append(agents, agent)
	}

	return agents
}

// CountActive returns the count of active agents
func (m *Manager) CountActive() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.agents)
}

// Remove removes an agent from the registry
func (m *Manager) Remove(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	agent, exists := m.agents[name]
	if !exists {
		slog.Warn("Agent removal failed: agent not found",
			"name", name,
			"project", m.projectKey)
		return fmt.Errorf("agent %s not found", name)
	}

	slog.Info("Agent removed from project",
		"name", name,
		"task", agent.TaskDescription,
		"project", m.projectKey,
		"remaining_agents", len(m.agents)-1)

	delete(m.agents, name)
	return nil
}

// Update updates an agent's information
func (m *Manager) Update(a Agent) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, exists := m.agents[a.Name]
	if !exists {
		slog.Warn("Agent update failed: agent not found",
			"name", a.Name,
			"project", m.projectKey)
		return fmt.Errorf("agent %s not found", a.Name)
	}

	slog.Info("Agent updated",
		"name", a.Name,
		"previous_task", existing.TaskDescription,
		"new_task", a.TaskDescription,
		"last_active", a.LastActive,
		"project", m.projectKey)

	a.ProjectKey = m.projectKey
	m.agents[a.Name] = a

	return nil
}
