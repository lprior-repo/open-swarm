// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package agent

import (
	"fmt"
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
		return fmt.Errorf("agent name is required")
	}

	a.ProjectKey = m.projectKey
	m.agents[a.Name] = a

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

	if _, exists := m.agents[name]; !exists {
		return fmt.Errorf("agent %s not found", name)
	}

	delete(m.agents, name)
	return nil
}

// Update updates an agent's information
func (m *Manager) Update(a Agent) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.agents[a.Name]; !exists {
		return fmt.Errorf("agent %s not found", a.Name)
	}

	a.ProjectKey = m.projectKey
	m.agents[a.Name] = a

	return nil
}
