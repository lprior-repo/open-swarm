// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(t *testing.T) string // Returns temp dir path
		cleanupFunc func(t *testing.T, dir string)
		wantErr     bool
		errContains string
		validate    func(t *testing.T, cfg *Config)
	}{
		{
			name: "valid configuration file",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				claudeDir := filepath.Join(tmpDir, ".claude")
				require.NoError(t, os.Mkdir(claudeDir, 0755))

				configContent := `
project:
  name: "test-project"
  description: "Test project description"
  working_directory: "/tmp/test"

model:
  default: "anthropic/claude-3-5-sonnet-20241022"
  agents:
    reviewer: "anthropic/claude-3-opus-20240229"

mcpServers:
  agent-mail:
    command: "python -m mcp_agent_mail.server"
    description: "Agent Mail MCP Server"
    enabled: true
    autostart: true
    priority: 1

behavior:
  auto_coordinate: true
  check_reservations: true
  auto_register: true
  preserve_threads: true
  use_todos: false

coordination:
  agent:
    program: "opencode"
    model: "anthropic/claude-3-5-sonnet-20241022"
  messages:
    auto_ack: false
    check_interval: 300
    importance_threshold: "normal"
  reservations:
    default_ttl: 3600
    auto_renew: true
    renew_threshold: 600
  threads:
    auto_create: true
    subject_prefix: "[open-swarm]"

build:
  commands:
    test: "go test ./..."
    build: "go build ./..."
    lint: "golangci-lint run"
    fmt: "gofmt -w ."
  slots:
    test: "test-slot"
    build: "build-slot"
`
				configPath := filepath.Join(claudeDir, "opencode.yaml")
				require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

				// Change to temp directory
				oldDir, err := os.Getwd()
				require.NoError(t, err)
				require.NoError(t, os.Chdir(tmpDir))

				// Store old dir for cleanup
				t.Cleanup(func() {
					os.Chdir(oldDir)
				})

				return tmpDir
			},
			wantErr: false,
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "test-project", cfg.Project.Name)
				assert.Equal(t, "Test project description", cfg.Project.Description)
				assert.Equal(t, "/tmp/test", cfg.Project.WorkingDirectory)
				assert.Equal(t, "anthropic/claude-3-5-sonnet-20241022", cfg.Model.Default)
				assert.Equal(t, "anthropic/claude-3-opus-20240229", cfg.Model.Agents["reviewer"])
				assert.True(t, cfg.Behavior.AutoCoordinate)
				assert.Equal(t, "opencode", cfg.Coordination.Agent.Program)
				assert.Equal(t, 3600, cfg.Coordination.Reservations.DefaultTTL)
				assert.Equal(t, "go test ./...", cfg.Build.Commands.Test)
			},
		},
		{
			name: "missing config file",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				oldDir, err := os.Getwd()
				require.NoError(t, err)
				require.NoError(t, os.Chdir(tmpDir))

				t.Cleanup(func() {
					os.Chdir(oldDir)
				})

				return tmpDir
			},
			wantErr:     true,
			errContains: "configuration file not found",
		},
		{
			name: "invalid yaml syntax",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				claudeDir := filepath.Join(tmpDir, ".claude")
				require.NoError(t, os.Mkdir(claudeDir, 0755))

				invalidYAML := `
project:
  name: "test"
  invalid yaml syntax here: [
`
				configPath := filepath.Join(claudeDir, "opencode.yaml")
				require.NoError(t, os.WriteFile(configPath, []byte(invalidYAML), 0644))

				oldDir, err := os.Getwd()
				require.NoError(t, err)
				require.NoError(t, os.Chdir(tmpDir))

				t.Cleanup(func() {
					os.Chdir(oldDir)
				})

				return tmpDir
			},
			wantErr:     true,
			errContains: "failed to parse config",
		},
		{
			name: "empty working directory defaults to cwd",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				claudeDir := filepath.Join(tmpDir, ".claude")
				require.NoError(t, os.Mkdir(claudeDir, 0755))

				configContent := `
project:
  name: "test-project"
  description: "Test"

coordination:
  agent:
    program: "opencode"
    model: "anthropic/claude-3-5-sonnet-20241022"
`
				configPath := filepath.Join(claudeDir, "opencode.yaml")
				require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

				oldDir, err := os.Getwd()
				require.NoError(t, err)
				require.NoError(t, os.Chdir(tmpDir))

				t.Cleanup(func() {
					os.Chdir(oldDir)
				})

				return tmpDir
			},
			wantErr: false,
			validate: func(t *testing.T, cfg *Config) {
				assert.NotEmpty(t, cfg.Project.WorkingDirectory)
				// Should be set to current working directory
				cwd, _ := os.Getwd()
				assert.Equal(t, cwd, cfg.Project.WorkingDirectory)
			},
		},
		{
			name: "minimal valid configuration",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				claudeDir := filepath.Join(tmpDir, ".claude")
				require.NoError(t, os.Mkdir(claudeDir, 0755))

				configContent := `
project:
  name: "minimal"

coordination:
  agent:
    program: "test"
    model: "test-model"
`
				configPath := filepath.Join(claudeDir, "opencode.yaml")
				require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

				oldDir, err := os.Getwd()
				require.NoError(t, err)
				require.NoError(t, os.Chdir(tmpDir))

				t.Cleanup(func() {
					os.Chdir(oldDir)
				})

				return tmpDir
			},
			wantErr: false,
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "minimal", cfg.Project.Name)
				assert.Equal(t, "test", cfg.Coordination.Agent.Program)
				assert.Equal(t, "test-model", cfg.Coordination.Agent.Model)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFunc != nil {
				tt.setupFunc(t)
			}

			cfg, err := Load()

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, cfg)

			if tt.validate != nil {
				tt.validate(t, cfg)
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		wantErr     bool
		errContains string
	}{
		{
			name: "valid configuration",
			config: &Config{
				Project: ProjectConfig{
					Name:             "test-project",
					WorkingDirectory: "/tmp/test",
				},
				Coordination: CoordinationConfig{
					Agent: AgentConfig{
						Program: "opencode",
						Model:   "claude-3-5-sonnet",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing project name",
			config: &Config{
				Project: ProjectConfig{
					Name:             "",
					WorkingDirectory: "/tmp/test",
				},
				Coordination: CoordinationConfig{
					Agent: AgentConfig{
						Program: "opencode",
						Model:   "claude-3-5-sonnet",
					},
				},
			},
			wantErr:     true,
			errContains: "project name is required",
		},
		{
			name: "missing working directory",
			config: &Config{
				Project: ProjectConfig{
					Name:             "test",
					WorkingDirectory: "",
				},
				Coordination: CoordinationConfig{
					Agent: AgentConfig{
						Program: "opencode",
						Model:   "claude-3-5-sonnet",
					},
				},
			},
			wantErr:     true,
			errContains: "working directory is required",
		},
		{
			name: "missing agent program",
			config: &Config{
				Project: ProjectConfig{
					Name:             "test",
					WorkingDirectory: "/tmp/test",
				},
				Coordination: CoordinationConfig{
					Agent: AgentConfig{
						Program: "",
						Model:   "claude-3-5-sonnet",
					},
				},
			},
			wantErr:     true,
			errContains: "agent program is required",
		},
		{
			name: "missing agent model",
			config: &Config{
				Project: ProjectConfig{
					Name:             "test",
					WorkingDirectory: "/tmp/test",
				},
				Coordination: CoordinationConfig{
					Agent: AgentConfig{
						Program: "opencode",
						Model:   "",
					},
				},
			},
			wantErr:     true,
			errContains: "agent model is required",
		},
		{
			name: "all fields empty",
			config: &Config{
				Project: ProjectConfig{},
				Coordination: CoordinationConfig{
					Agent: AgentConfig{},
				},
			},
			wantErr:     true,
			errContains: "project name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMCPServerConfig(t *testing.T) {
	tests := []struct {
		name   string
		config MCPServerConfig
		want   MCPServerConfig
	}{
		{
			name: "full configuration",
			config: MCPServerConfig{
				Command:     "python -m mcp_agent_mail.server",
				Description: "Agent Mail Server",
				Enabled:     true,
				AutoStart:   true,
				Priority:    1,
			},
			want: MCPServerConfig{
				Command:     "python -m mcp_agent_mail.server",
				Description: "Agent Mail Server",
				Enabled:     true,
				AutoStart:   true,
				Priority:    1,
			},
		},
		{
			name: "disabled server",
			config: MCPServerConfig{
				Command:     "test-command",
				Description: "Test Server",
				Enabled:     false,
				AutoStart:   false,
				Priority:    0,
			},
			want: MCPServerConfig{
				Command:     "test-command",
				Description: "Test Server",
				Enabled:     false,
				AutoStart:   false,
				Priority:    0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want.Command, tt.config.Command)
			assert.Equal(t, tt.want.Description, tt.config.Description)
			assert.Equal(t, tt.want.Enabled, tt.config.Enabled)
			assert.Equal(t, tt.want.AutoStart, tt.config.AutoStart)
			assert.Equal(t, tt.want.Priority, tt.config.Priority)
		})
	}
}

func TestBehaviorConfig(t *testing.T) {
	tests := []struct {
		name   string
		config BehaviorConfig
		want   BehaviorConfig
	}{
		{
			name: "all enabled",
			config: BehaviorConfig{
				AutoCoordinate:    true,
				CheckReservations: true,
				AutoRegister:      true,
				PreserveThreads:   true,
				UseTodos:          true,
			},
			want: BehaviorConfig{
				AutoCoordinate:    true,
				CheckReservations: true,
				AutoRegister:      true,
				PreserveThreads:   true,
				UseTodos:          true,
			},
		},
		{
			name: "all disabled",
			config: BehaviorConfig{
				AutoCoordinate:    false,
				CheckReservations: false,
				AutoRegister:      false,
				PreserveThreads:   false,
				UseTodos:          false,
			},
			want: BehaviorConfig{
				AutoCoordinate:    false,
				CheckReservations: false,
				AutoRegister:      false,
				PreserveThreads:   false,
				UseTodos:          false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.config)
		})
	}
}

func TestCoordinationConfig(t *testing.T) {
	tests := []struct {
		name   string
		config CoordinationConfig
		want   CoordinationConfig
	}{
		{
			name: "complete coordination config",
			config: CoordinationConfig{
				Agent: AgentConfig{
					Program: "opencode",
					Model:   "claude-3-5-sonnet",
				},
				Messages: MessagesConfig{
					AutoAck:             true,
					CheckInterval:       300,
					ImportanceThreshold: "high",
				},
				Reservations: ReservationsConfig{
					DefaultTTL:     3600,
					AutoRenew:      true,
					RenewThreshold: 600,
				},
				Threads: ThreadsConfig{
					AutoCreate:    true,
					SubjectPrefix: "[test]",
				},
			},
			want: CoordinationConfig{
				Agent: AgentConfig{
					Program: "opencode",
					Model:   "claude-3-5-sonnet",
				},
				Messages: MessagesConfig{
					AutoAck:             true,
					CheckInterval:       300,
					ImportanceThreshold: "high",
				},
				Reservations: ReservationsConfig{
					DefaultTTL:     3600,
					AutoRenew:      true,
					RenewThreshold: 600,
				},
				Threads: ThreadsConfig{
					AutoCreate:    true,
					SubjectPrefix: "[test]",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.config)
		})
	}
}

func TestBuildConfig(t *testing.T) {
	tests := []struct {
		name   string
		config BuildConfig
		want   BuildConfig
	}{
		{
			name: "complete build config",
			config: BuildConfig{
				Commands: BuildCommands{
					Test:  "go test ./...",
					Build: "go build ./...",
					Lint:  "golangci-lint run",
					Fmt:   "gofmt -w .",
				},
				Slots: BuildSlots{
					Test:  "test-slot",
					Build: "build-slot",
				},
			},
			want: BuildConfig{
				Commands: BuildCommands{
					Test:  "go test ./...",
					Build: "go build ./...",
					Lint:  "golangci-lint run",
					Fmt:   "gofmt -w .",
				},
				Slots: BuildSlots{
					Test:  "test-slot",
					Build: "build-slot",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.config)
		})
	}
}
