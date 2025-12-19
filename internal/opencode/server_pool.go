// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package opencode

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ServerInstance represents a single OpenCode server instance.
type ServerInstance struct {
	ID       string        // Unique identifier (agent-N)
	Port     int           // Port number (8000-9000)
	URL      string        // HTTP endpoint
	Status   ServerStatus  // Current status
	AgentID  string        // Current agent using this server (or empty if free)
	Health   HealthStatus  // Last health check result
	LastSeen time.Time     // Last successful interaction
}

// ServerStatus represents the current state of a server instance.
type ServerStatus string

const (
	StatusStarting ServerStatus = "starting"
	StatusRunning  ServerStatus = "running"
	StatusBusy     ServerStatus = "busy"
	StatusUnhealthy ServerStatus = "unhealthy"
	StatusStopped  ServerStatus = "stopped"
)

// HealthStatus represents health check results.
type HealthStatus struct {
	IsHealthy bool
	Error     string
	Timestamp time.Time
}

// ServerPool manages a pool of OpenCode server instances for parallel agent execution.
type ServerPool struct {
	mu                 sync.RWMutex
	instances          map[string]*ServerInstance
	availableServers   chan *ServerInstance
	minPort            int
	maxPort            int
	healthCheckTimeout time.Duration
	startTimeout       time.Duration
}

// NewServerPool creates a new OpenCode server pool.
// serverCount: number of parallel servers (1-100)
// minPort: starting port (default 8000)
// maxPort: ending port (default 9000)
func NewServerPool(ctx context.Context, serverCount int, minPort, maxPort int) (*ServerPool, error) {
	if serverCount < 1 || serverCount > 100 {
		return nil, fmt.Errorf("serverCount must be between 1 and 100, got %d", serverCount)
	}

	if maxPort-minPort+1 < serverCount {
		return nil, fmt.Errorf("insufficient port range: need %d ports, have %d available",
			serverCount, maxPort-minPort+1)
	}

	pool := &ServerPool{
		instances:          make(map[string]*ServerInstance, serverCount),
		availableServers:   make(chan *ServerInstance, serverCount),
		minPort:            minPort,
		maxPort:            maxPort,
		healthCheckTimeout: 5 * time.Second,
		startTimeout:       30 * time.Second,
	}

	// Initialize server instances
	for i := 0; i < serverCount; i++ {
		port := minPort + i
		instance := &ServerInstance{
			ID:     fmt.Sprintf("opencode-%d", i),
			Port:   port,
			URL:    fmt.Sprintf("http://localhost:%d", port),
			Status: StatusStarting,
		}
		pool.instances[instance.ID] = instance
		pool.availableServers <- instance // Add to available queue
	}

	// Start background health checker
	go pool.healthCheckLoop(ctx)

	return pool, nil
}

// GetAvailableServer allocates an available server from the pool.
// Returns error if no servers are available or pool is exhausted.
func (p *ServerPool) GetAvailableServer(ctx context.Context, agentID string) (*ServerInstance, error) {
	select {
	case server := <-p.availableServers:
		p.mu.Lock()
		server.AgentID = agentID
		server.Status = StatusBusy
		p.mu.Unlock()
		return server, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("context cancelled while waiting for available server")
	}
}

// ReleaseServer returns a server to the available pool.
func (p *ServerPool) ReleaseServer(serverID string) error {
	p.mu.Lock()
	server, exists := p.instances[serverID]
	if !exists {
		p.mu.Unlock()
		return fmt.Errorf("server %s not found in pool", serverID)
	}

	if server.Status == StatusStopped {
		p.mu.Unlock()
		return fmt.Errorf("cannot release stopped server %s", serverID)
	}

	server.AgentID = ""
	server.Status = StatusRunning
	p.mu.Unlock()

	// Return to available queue
	select {
	case p.availableServers <- server:
		return nil
	default:
		return fmt.Errorf("failed to return server %s to pool (pool may be full)", serverID)
	}
}

// HealthCheck performs a health check on a server.
func (p *ServerPool) HealthCheck(ctx context.Context, serverID string) (HealthStatus, error) {
	p.mu.RLock()
	server, exists := p.instances[serverID]
	if !exists {
		p.mu.RUnlock()
		return HealthStatus{}, fmt.Errorf("server %s not found in pool", serverID)
	}
	p.mu.RUnlock()

	// Perform health check (ping endpoint)
	ctx, cancel := context.WithTimeout(ctx, p.healthCheckTimeout)
	defer cancel()

	// In a real implementation, this would make an HTTP request to the server
	// For now, we simulate a health check based on LastSeen time
	p.mu.Lock()
	isHealthy := server.Status != StatusStopped && time.Since(server.LastSeen) < 2*time.Minute
	health := HealthStatus{
		IsHealthy: isHealthy,
		Timestamp: time.Now(),
	}
	if !isHealthy {
		health.Error = "server not responding or timed out"
		server.Status = StatusUnhealthy
	}
	server.Health = health
	p.mu.Unlock()

	return health, nil
}

// UpdateLastSeen updates the LastSeen timestamp for a server.
func (p *ServerPool) UpdateLastSeen(serverID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	server, exists := p.instances[serverID]
	if !exists {
		return fmt.Errorf("server %s not found in pool", serverID)
	}

	server.LastSeen = time.Now()
	if server.Status == StatusUnhealthy {
		server.Status = StatusRunning
	}

	return nil
}

// GetStatus returns the current status of all servers in the pool.
func (p *ServerPool) GetStatus() map[string]*ServerInstance {
	p.mu.RLock()
	defer p.mu.RUnlock()

	status := make(map[string]*ServerInstance)
	for id, instance := range p.instances {
		// Create a copy to avoid exposing internal state
		copy := *instance
		status[id] = &copy
	}
	return status
}

// StopServer stops a specific server instance.
func (p *ServerPool) StopServer(serverID string) error {
	p.mu.Lock()
	server, exists := p.instances[serverID]
	if !exists {
		p.mu.Unlock()
		return fmt.Errorf("server %s not found in pool", serverID)
	}

	if server.Status == StatusStopped {
		p.mu.Unlock()
		return fmt.Errorf("server %s is already stopped", serverID)
	}

	server.Status = StatusStopped
	server.AgentID = ""
	p.mu.Unlock()

	// Try to drain from available queue if server was in it
	select {
	case <-p.availableServers:
		// Successfully removed from available queue
	default:
		// Server was not in available queue (may have been allocated)
	}

	return nil
}

// RestartServer restarts a specific server instance.
func (p *ServerPool) RestartServer(ctx context.Context, serverID string) error {
	p.mu.Lock()
	server, exists := p.instances[serverID]
	if !exists {
		p.mu.Unlock()
		return fmt.Errorf("server %s not found in pool", serverID)
	}

	// Mark as stopped
	server.Status = StatusStopped
	server.AgentID = ""
	p.mu.Unlock()

	// Try to drain from available queue if server was in it
	select {
	case <-p.availableServers:
		// Successfully removed from available queue
	default:
		// Server was not in available queue (may have been allocated)
	}

	// Wait a bit
	select {
	case <-time.After(1 * time.Second):
	case <-ctx.Done():
		return ctx.Err()
	}

	// Start the server again
	p.mu.Lock()
	server.Status = StatusStarting
	server.LastSeen = time.Now()
	p.mu.Unlock()

	// Return to available queue
	select {
	case p.availableServers <- server:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Shutdown gracefully shuts down all servers in the pool.
func (p *ServerPool) Shutdown(ctx context.Context) error {
	p.mu.Lock()
	for _, server := range p.instances {
		server.Status = StatusStopped
	}
	p.mu.Unlock()

	// Drain the available servers channel
	drainLoop:
	for {
		select {
		case <-p.availableServers:
			// Continue draining
		case <-time.After(100 * time.Millisecond):
			// No more servers in queue
			break drainLoop
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

// healthCheckLoop periodically checks server health in the background.
func (p *ServerPool) healthCheckLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.performHealthChecks(ctx)
		case <-ctx.Done():
			return
		}
	}
}

// performHealthChecks runs health checks on all servers.
func (p *ServerPool) performHealthChecks(ctx context.Context) {
	p.mu.RLock()
	serverIDs := make([]string, 0, len(p.instances))
	for id := range p.instances {
		serverIDs = append(serverIDs, id)
	}
	p.mu.RUnlock()

	for _, serverID := range serverIDs {
		_, _ = p.HealthCheck(ctx, serverID)
	}
}

// GetServerByID returns a specific server instance by ID.
func (p *ServerPool) GetServerByID(serverID string) (*ServerInstance, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	server, exists := p.instances[serverID]
	if !exists {
		return nil, fmt.Errorf("server %s not found in pool", serverID)
	}

	return server, nil
}

// AvailableServerCount returns the number of currently available servers.
func (p *ServerPool) AvailableServerCount() int {
	return len(p.availableServers)
}

// TotalServerCount returns the total number of servers in the pool.
func (p *ServerPool) TotalServerCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.instances)
}
