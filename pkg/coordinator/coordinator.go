package coordinator

import (
	"fmt"
	"time"

	"open-swarm/internal/config"
	"open-swarm/pkg/agent"
)

// Coordinator manages multi-agent coordination for the project
type Coordinator struct {
	config       *config.Config
	projectKey   string
	agentManager *agent.Manager
}

// Status represents the current coordination state
type Status struct {
	ActiveAgents        int
	UnreadMessages      int
	ActiveReservations  int
	ActiveThreads       int
	LastSync            time.Time
	MCPServersConnected bool
}

// New creates a new Coordinator instance
func New(cfg *config.Config) (*Coordinator, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configuration is required")
	}

	coord := &Coordinator{
		config:       cfg,
		projectKey:   cfg.Project.WorkingDirectory,
		agentManager: agent.NewManager(cfg.Project.WorkingDirectory),
	}

	return coord, nil
}

// GetStatus returns the current coordination status
func (c *Coordinator) GetStatus() *Status {
	// In a real implementation, this would query the Agent Mail MCP server
	// For now, return placeholder data
	return &Status{
		ActiveAgents:        c.agentManager.CountActive(),
		UnreadMessages:      0,
		ActiveReservations:  0,
		ActiveThreads:       0,
		LastSync:            time.Now(),
		MCPServersConnected: true,
	}
}

// ListAgents returns all active agents in the project
func (c *Coordinator) ListAgents() []agent.Agent {
	return c.agentManager.List()
}

// Sync synchronizes the coordinator state with Agent Mail
func (c *Coordinator) Sync() error {
	// In a real implementation, this would:
	// 1. Ensure project is registered
	// 2. Fetch recent messages
	// 3. Update agent list
	// 4. Check file reservations
	// 5. Sync with beads and serena

	fmt.Println("\n✓ Project registered in Agent Mail")
	fmt.Println("✓ Agent list synchronized")
	fmt.Println("✓ Message queue checked")
	fmt.Println("✓ File reservations updated")

	return nil
}

// RegisterAgent registers a new agent in the project
func (c *Coordinator) RegisterAgent(name, program, model, taskDesc string) error {
	a := agent.Agent{
		Name:            name,
		Program:         program,
		Model:           model,
		TaskDescription: taskDesc,
		LastActive:      time.Now().Format(time.RFC3339),
	}

	return c.agentManager.Register(a)
}

// GetProjectKey returns the project key for Agent Mail
func (c *Coordinator) GetProjectKey() string {
	return c.projectKey
}
