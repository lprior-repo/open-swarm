// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the complete Open Swarm configuration
type Config struct {
	Project      ProjectConfig      `yaml:"project"`
	Model        ModelConfig        `yaml:"model"`
	MCPServers   MCPServersConfig   `yaml:"mcpServers"`
	Behavior     BehaviorConfig     `yaml:"behavior"`
	Coordination CoordinationConfig `yaml:"coordination"`
	Build        BuildConfig        `yaml:"build"`
}

// ProjectConfig holds project-level configuration
type ProjectConfig struct {
	Name             string `yaml:"name"`
	Description      string `yaml:"description"`
	WorkingDirectory string `yaml:"working_directory"`
}

// ModelConfig specifies model preferences
type ModelConfig struct {
	Default string            `yaml:"default"`
	Agents  map[string]string `yaml:"agents"`
}

// MCPServersConfig configures MCP server connections
type MCPServersConfig map[string]MCPServerConfig

// MCPServerConfig represents a single MCP server
type MCPServerConfig struct {
	Command     string `yaml:"command"`
	Description string `yaml:"description"`
	Enabled     bool   `yaml:"enabled"`
	AutoStart   bool   `yaml:"autostart"`
	Priority    int    `yaml:"priority"`
}

// BehaviorConfig controls agent behavior
type BehaviorConfig struct {
	AutoCoordinate    bool `yaml:"auto_coordinate"`
	CheckReservations bool `yaml:"check_reservations"`
	AutoRegister      bool `yaml:"auto_register"`
	PreserveThreads   bool `yaml:"preserve_threads"`
	UseTodos          bool `yaml:"use_todos"`
}

// CoordinationConfig manages coordination settings
type CoordinationConfig struct {
	Agent        AgentConfig        `yaml:"agent"`
	Messages     MessagesConfig     `yaml:"messages"`
	Reservations ReservationsConfig `yaml:"reservations"`
	Threads      ThreadsConfig      `yaml:"threads"`
}

// AgentConfig specifies agent identity
type AgentConfig struct {
	Program string `yaml:"program"`
	Model   string `yaml:"model"`
}

// MessagesConfig controls message handling
type MessagesConfig struct {
	AutoAck             bool   `yaml:"auto_ack"`
	CheckInterval       int    `yaml:"check_interval"`
	ImportanceThreshold string `yaml:"importance_threshold"`
}

// ReservationsConfig manages file reservations
type ReservationsConfig struct {
	DefaultTTL     int  `yaml:"default_ttl"`
	AutoRenew      bool `yaml:"auto_renew"`
	RenewThreshold int  `yaml:"renew_threshold"`
}

// ThreadsConfig controls thread behavior
type ThreadsConfig struct {
	AutoCreate    bool   `yaml:"auto_create"`
	SubjectPrefix string `yaml:"subject_prefix"`
}

// BuildConfig specifies build commands
type BuildConfig struct {
	Commands BuildCommands `yaml:"commands"`
	Slots    BuildSlots    `yaml:"slots"`
}

// BuildCommands are the available build commands
type BuildCommands struct {
	Test  string `yaml:"test"`
	Build string `yaml:"build"`
	Lint  string `yaml:"lint"`
	Fmt   string `yaml:"fmt"`
}

// BuildSlots defines build slot names
type BuildSlots struct {
	Test  string `yaml:"test"`
	Build string `yaml:"build"`
}

// Load loads the configuration from .claude/opencode.yaml
func Load() (*Config, error) {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	// Find config file
	configPath := filepath.Join(cwd, ".claude", "opencode.yaml")

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("configuration file not found: %s", configPath)
	}

	// Read file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Set working directory if not specified
	if cfg.Project.WorkingDirectory == "" {
		cfg.Project.WorkingDirectory = cwd
	}

	return &cfg, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Project.Name == "" {
		return fmt.Errorf("project name is required")
	}

	if c.Project.WorkingDirectory == "" {
		return fmt.Errorf("working directory is required")
	}

	if c.Coordination.Agent.Program == "" {
		return fmt.Errorf("agent program is required")
	}

	if c.Coordination.Agent.Model == "" {
		return fmt.Errorf("agent model is required")
	}

	return nil
}
