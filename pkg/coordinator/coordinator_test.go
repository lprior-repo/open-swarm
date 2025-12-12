// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package coordinator

import (
	"testing"
	"time"

	"open-swarm/internal/config"
	"open-swarm/pkg/agent"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.Config
		wantErr     bool
		errContains string
		validate    func(t *testing.T, c *Coordinator)
	}{
		{
			name: "valid configuration",
			config: &config.Config{
				Project: config.ProjectConfig{
					Name:             "test-project",
					WorkingDirectory: "/tmp/test",
				},
				Coordination: config.CoordinationConfig{
					Agent: config.AgentConfig{
						Program: "opencode",
						Model:   "claude-3-5-sonnet",
					},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, c *Coordinator) {
				assert.NotNil(t, c)
				assert.NotNil(t, c.config)
				assert.NotNil(t, c.agentManager)
				assert.Equal(t, "/tmp/test", c.projectKey)
				assert.Equal(t, "test-project", c.config.Project.Name)
			},
		},
		{
			name:        "nil configuration",
			config:      nil,
			wantErr:     true,
			errContains: "configuration is required",
		},
		{
			name: "empty working directory",
			config: &config.Config{
				Project: config.ProjectConfig{
					Name:             "test",
					WorkingDirectory: "",
				},
			},
			wantErr: false,
			validate: func(t *testing.T, c *Coordinator) {
				assert.NotNil(t, c)
				assert.Equal(t, "", c.projectKey)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coord, err := New(tt.config)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				assert.Nil(t, coord)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, coord)

			if tt.validate != nil {
				tt.validate(t, coord)
			}
		})
	}
}

func TestCoordinator_GetStatus(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) *Coordinator
		validate func(t *testing.T, status *Status)
	}{
		{
			name: "empty coordinator",
			setup: func(t *testing.T) *Coordinator {
				cfg := &config.Config{
					Project: config.ProjectConfig{
						Name:             "test",
						WorkingDirectory: "/tmp/test",
					},
				}
				coord, err := New(cfg)
				require.NoError(t, err)
				return coord
			},
			validate: func(t *testing.T, status *Status) {
				assert.NotNil(t, status)
				assert.Equal(t, 0, status.ActiveAgents)
				assert.Equal(t, 0, status.UnreadMessages)
				assert.Equal(t, 0, status.ActiveReservations)
				assert.Equal(t, 0, status.ActiveThreads)
				assert.True(t, status.MCPServersConnected)
				assert.WithinDuration(t, time.Now(), status.LastSync, 5*time.Second)
			},
		},
		{
			name: "coordinator with registered agents",
			setup: func(t *testing.T) *Coordinator {
				cfg := &config.Config{
					Project: config.ProjectConfig{
						Name:             "test",
						WorkingDirectory: "/tmp/test",
					},
				}
				coord, err := New(cfg)
				require.NoError(t, err)

				// Register some agents
				err = coord.RegisterAgent("agent1", "opencode", "claude-3-5-sonnet", "Task 1")
				require.NoError(t, err)
				err = coord.RegisterAgent("agent2", "opencode", "claude-3-opus", "Task 2")
				require.NoError(t, err)

				return coord
			},
			validate: func(t *testing.T, status *Status) {
				assert.NotNil(t, status)
				assert.Equal(t, 2, status.ActiveAgents)
				assert.True(t, status.MCPServersConnected)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coord := tt.setup(t)
			status := coord.GetStatus()

			if tt.validate != nil {
				tt.validate(t, status)
			}
		})
	}
}

func TestCoordinator_RegisterAgent(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T) *Coordinator
		agentName   string
		program     string
		model       string
		taskDesc    string
		wantErr     bool
		errContains string
		validate    func(t *testing.T, c *Coordinator)
	}{
		{
			name: "register valid agent",
			setup: func(t *testing.T) *Coordinator {
				cfg := &config.Config{
					Project: config.ProjectConfig{
						Name:             "test",
						WorkingDirectory: "/tmp/test",
					},
				}
				coord, err := New(cfg)
				require.NoError(t, err)
				return coord
			},
			agentName: "test-agent",
			program:   "opencode",
			model:     "claude-3-5-sonnet",
			taskDesc:  "Test task",
			wantErr:   false,
			validate: func(t *testing.T, c *Coordinator) {
				agents := c.ListAgents()
				assert.Len(t, agents, 1)
				assert.Equal(t, "test-agent", agents[0].Name)
				assert.Equal(t, "opencode", agents[0].Program)
				assert.Equal(t, "claude-3-5-sonnet", agents[0].Model)
				assert.Equal(t, "Test task", agents[0].TaskDescription)
				assert.NotEmpty(t, agents[0].LastActive)
			},
		},
		{
			name: "register multiple agents",
			setup: func(t *testing.T) *Coordinator {
				cfg := &config.Config{
					Project: config.ProjectConfig{
						Name:             "test",
						WorkingDirectory: "/tmp/test",
					},
				}
				coord, err := New(cfg)
				require.NoError(t, err)
				return coord
			},
			agentName: "agent2",
			program:   "opencode",
			model:     "claude-3-opus",
			taskDesc:  "Second task",
			wantErr:   false,
			validate: func(t *testing.T, c *Coordinator) {
				// Register first agent
				err := c.RegisterAgent("agent1", "opencode", "claude-3-5-sonnet", "First task")
				require.NoError(t, err)

				agents := c.ListAgents()
				assert.Len(t, agents, 2)
			},
		},
		{
			name: "register agent with empty name",
			setup: func(t *testing.T) *Coordinator {
				cfg := &config.Config{
					Project: config.ProjectConfig{
						Name:             "test",
						WorkingDirectory: "/tmp/test",
					},
				}
				coord, err := New(cfg)
				require.NoError(t, err)
				return coord
			},
			agentName:   "",
			program:     "opencode",
			model:       "claude-3-5-sonnet",
			taskDesc:    "Test",
			wantErr:     true,
			errContains: "agent name is required",
		},
		{
			name: "register agent with same name updates",
			setup: func(t *testing.T) *Coordinator {
				cfg := &config.Config{
					Project: config.ProjectConfig{
						Name:             "test",
						WorkingDirectory: "/tmp/test",
					},
				}
				coord, err := New(cfg)
				require.NoError(t, err)

				// Register initial agent
				err = coord.RegisterAgent("duplicate", "opencode", "claude-3-5-sonnet", "Initial task")
				require.NoError(t, err)

				return coord
			},
			agentName: "duplicate",
			program:   "opencode",
			model:     "claude-3-opus",
			taskDesc:  "Updated task",
			wantErr:   false,
			validate: func(t *testing.T, c *Coordinator) {
				agents := c.ListAgents()
				assert.Len(t, agents, 1)
				assert.Equal(t, "duplicate", agents[0].Name)
				assert.Equal(t, "claude-3-opus", agents[0].Model)
				assert.Equal(t, "Updated task", agents[0].TaskDescription)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coord := tt.setup(t)

			err := coord.RegisterAgent(tt.agentName, tt.program, tt.model, tt.taskDesc)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)

			if tt.validate != nil {
				tt.validate(t, coord)
			}
		})
	}
}

func TestCoordinator_ListAgents(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) *Coordinator
		validate func(t *testing.T, agents []agent.Agent)
	}{
		{
			name: "empty agent list",
			setup: func(t *testing.T) *Coordinator {
				cfg := &config.Config{
					Project: config.ProjectConfig{
						Name:             "test",
						WorkingDirectory: "/tmp/test",
					},
				}
				coord, err := New(cfg)
				require.NoError(t, err)
				return coord
			},
			validate: func(t *testing.T, agents []agent.Agent) {
				assert.NotNil(t, agents)
				assert.Len(t, agents, 0)
			},
		},
		{
			name: "single agent",
			setup: func(t *testing.T) *Coordinator {
				cfg := &config.Config{
					Project: config.ProjectConfig{
						Name:             "test",
						WorkingDirectory: "/tmp/test",
					},
				}
				coord, err := New(cfg)
				require.NoError(t, err)

				err = coord.RegisterAgent("agent1", "opencode", "claude-3-5-sonnet", "Task 1")
				require.NoError(t, err)

				return coord
			},
			validate: func(t *testing.T, agents []agent.Agent) {
				assert.Len(t, agents, 1)
				assert.Equal(t, "agent1", agents[0].Name)
			},
		},
		{
			name: "multiple agents",
			setup: func(t *testing.T) *Coordinator {
				cfg := &config.Config{
					Project: config.ProjectConfig{
						Name:             "test",
						WorkingDirectory: "/tmp/test",
					},
				}
				coord, err := New(cfg)
				require.NoError(t, err)

				err = coord.RegisterAgent("agent1", "opencode", "claude-3-5-sonnet", "Task 1")
				require.NoError(t, err)
				err = coord.RegisterAgent("agent2", "opencode", "claude-3-opus", "Task 2")
				require.NoError(t, err)
				err = coord.RegisterAgent("agent3", "opencode", "gpt-4", "Task 3")
				require.NoError(t, err)

				return coord
			},
			validate: func(t *testing.T, agents []agent.Agent) {
				assert.Len(t, agents, 3)
				names := make(map[string]bool)
				for _, a := range agents {
					names[a.Name] = true
				}
				assert.True(t, names["agent1"])
				assert.True(t, names["agent2"])
				assert.True(t, names["agent3"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coord := tt.setup(t)
			agents := coord.ListAgents()

			if tt.validate != nil {
				tt.validate(t, agents)
			}
		})
	}
}

func TestCoordinator_GetProjectKey(t *testing.T) {
	tests := []struct {
		name        string
		workingDir  string
		expectedKey string
	}{
		{
			name:        "standard working directory",
			workingDir:  "/tmp/test-project",
			expectedKey: "/tmp/test-project",
		},
		{
			name:        "root directory",
			workingDir:  "/",
			expectedKey: "/",
		},
		{
			name:        "nested directory",
			workingDir:  "/home/user/projects/my-project",
			expectedKey: "/home/user/projects/my-project",
		},
		{
			name:        "empty directory",
			workingDir:  "",
			expectedKey: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Project: config.ProjectConfig{
					Name:             "test",
					WorkingDirectory: tt.workingDir,
				},
			}

			coord, err := New(cfg)
			require.NoError(t, err)

			projectKey := coord.GetProjectKey()
			assert.Equal(t, tt.expectedKey, projectKey)
		})
	}
}

func TestCoordinator_Sync(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) *Coordinator
		wantErr bool
	}{
		{
			name: "successful sync",
			setup: func(t *testing.T) *Coordinator {
				cfg := &config.Config{
					Project: config.ProjectConfig{
						Name:             "test",
						WorkingDirectory: "/tmp/test",
					},
				}
				coord, err := New(cfg)
				require.NoError(t, err)
				return coord
			},
			wantErr: false,
		},
		{
			name: "sync with registered agents",
			setup: func(t *testing.T) *Coordinator {
				cfg := &config.Config{
					Project: config.ProjectConfig{
						Name:             "test",
						WorkingDirectory: "/tmp/test",
					},
				}
				coord, err := New(cfg)
				require.NoError(t, err)

				err = coord.RegisterAgent("agent1", "opencode", "claude-3-5-sonnet", "Task 1")
				require.NoError(t, err)

				return coord
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coord := tt.setup(t)
			err := coord.Sync()

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestStatus(t *testing.T) {
	tests := []struct {
		name   string
		status Status
		want   Status
	}{
		{
			name: "empty status",
			status: Status{
				ActiveAgents:        0,
				UnreadMessages:      0,
				ActiveReservations:  0,
				ActiveThreads:       0,
				LastSync:            time.Time{},
				MCPServersConnected: false,
			},
			want: Status{
				ActiveAgents:        0,
				UnreadMessages:      0,
				ActiveReservations:  0,
				ActiveThreads:       0,
				LastSync:            time.Time{},
				MCPServersConnected: false,
			},
		},
		{
			name: "active status",
			status: Status{
				ActiveAgents:        5,
				UnreadMessages:      10,
				ActiveReservations:  3,
				ActiveThreads:       2,
				LastSync:            time.Now(),
				MCPServersConnected: true,
			},
			want: Status{
				ActiveAgents:        5,
				UnreadMessages:      10,
				ActiveReservations:  3,
				ActiveThreads:       2,
				MCPServersConnected: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want.ActiveAgents, tt.status.ActiveAgents)
			assert.Equal(t, tt.want.UnreadMessages, tt.status.UnreadMessages)
			assert.Equal(t, tt.want.ActiveReservations, tt.status.ActiveReservations)
			assert.Equal(t, tt.want.ActiveThreads, tt.status.ActiveThreads)
			assert.Equal(t, tt.want.MCPServersConnected, tt.status.MCPServersConnected)
		})
	}
}

func TestCoordinator_Integration(t *testing.T) {
	// Integration test: Full workflow
	t.Run("complete workflow", func(t *testing.T) {
		// Create configuration
		cfg := &config.Config{
			Project: config.ProjectConfig{
				Name:             "integration-test",
				Description:      "Integration test project",
				WorkingDirectory: "/tmp/integration-test",
			},
			Coordination: config.CoordinationConfig{
				Agent: config.AgentConfig{
					Program: "opencode",
					Model:   "claude-3-5-sonnet",
				},
			},
		}

		// Create coordinator
		coord, err := New(cfg)
		require.NoError(t, err)
		require.NotNil(t, coord)

		// Verify initial state
		status := coord.GetStatus()
		assert.Equal(t, 0, status.ActiveAgents)

		// Register first agent
		err = coord.RegisterAgent("agent1", "opencode", "claude-3-5-sonnet", "Backend development")
		require.NoError(t, err)

		// Verify agent registered
		agents := coord.ListAgents()
		assert.Len(t, agents, 1)
		assert.Equal(t, "agent1", agents[0].Name)

		// Register second agent
		err = coord.RegisterAgent("agent2", "opencode", "claude-3-opus", "Frontend development")
		require.NoError(t, err)

		// Verify both agents
		agents = coord.ListAgents()
		assert.Len(t, agents, 2)

		// Check status
		status = coord.GetStatus()
		assert.Equal(t, 2, status.ActiveAgents)

		// Sync
		err = coord.Sync()
		require.NoError(t, err)

		// Verify project key
		projectKey := coord.GetProjectKey()
		assert.Equal(t, "/tmp/integration-test", projectKey)
	})
}
