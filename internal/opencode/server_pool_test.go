// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package opencode

import (
	"context"
	"testing"
	"time"
)

func TestNewServerPool_ValidConfig(t *testing.T) {
	ctx := context.Background()
	pool, err := NewServerPool(ctx, 5, 8000, 9000)
	if err != nil {
		t.Fatalf("NewServerPool failed: %v", err)
	}

	if pool.TotalServerCount() != 5 {
		t.Errorf("expected 5 servers, got %d", pool.TotalServerCount())
	}

	if pool.AvailableServerCount() != 5 {
		t.Errorf("expected 5 available servers, got %d", pool.AvailableServerCount())
	}
}

func TestNewServerPool_InvalidCount(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		count     int
		minPort   int
		maxPort   int
		shouldErr bool
	}{
		{"zero servers", 0, 8000, 9000, true},
		{"too many servers", 101, 8000, 9000, true},
		{"port range too small", 5, 8000, 8002, true},
		{"valid max", 100, 8000, 8100, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewServerPool(ctx, tt.count, tt.minPort, tt.maxPort)
			if (err != nil) != tt.shouldErr {
				t.Errorf("expected error=%v, got error=%v", tt.shouldErr, err != nil)
			}
		})
	}
}

func TestGetAvailableServer(t *testing.T) {
	ctx := context.Background()
	pool, _ := NewServerPool(ctx, 3, 8000, 9000)

	// Get first server
	server1, err := pool.GetAvailableServer(ctx, "agent-1")
	if err != nil {
		t.Fatalf("GetAvailableServer failed: %v", err)
	}

	if server1.AgentID != "agent-1" {
		t.Errorf("expected agent-1, got %s", server1.AgentID)
	}

	if server1.Status != StatusBusy {
		t.Errorf("expected StatusBusy, got %s", server1.Status)
	}

	// Get second server
	server2, err := pool.GetAvailableServer(ctx, "agent-2")
	if err != nil {
		t.Fatalf("GetAvailableServer failed: %v", err)
	}

	if server2.ID == server1.ID {
		t.Errorf("expected different servers, got same: %s", server1.ID)
	}

	// Verify we have 1 server left
	if pool.AvailableServerCount() != 1 {
		t.Errorf("expected 1 available server, got %d", pool.AvailableServerCount())
	}
}

func TestReleaseServer(t *testing.T) {
	ctx := context.Background()
	pool, _ := NewServerPool(ctx, 3, 8000, 9000)

	// Get server
	server, _ := pool.GetAvailableServer(ctx, "agent-1")
	if pool.AvailableServerCount() != 2 {
		t.Errorf("expected 2 available servers, got %d", pool.AvailableServerCount())
	}

	// Release server
	err := pool.ReleaseServer(server.ID)
	if err != nil {
		t.Fatalf("ReleaseServer failed: %v", err)
	}

	if pool.AvailableServerCount() != 3 {
		t.Errorf("expected 3 available servers, got %d", pool.AvailableServerCount())
	}

	// Verify server state
	updated, _ := pool.GetServerByID(server.ID)
	if updated.AgentID != "" {
		t.Errorf("expected empty AgentID, got %s", updated.AgentID)
	}

	if updated.Status != StatusRunning {
		t.Errorf("expected StatusRunning, got %s", updated.Status)
	}
}

func TestHealthCheck(t *testing.T) {
	ctx := context.Background()
	pool, _ := NewServerPool(ctx, 1, 8000, 9000)

	server, _ := pool.GetAvailableServer(ctx, "agent-1")

	// Update LastSeen to current time
	_ = pool.UpdateLastSeen(server.ID)

	// Perform health check
	health, err := pool.HealthCheck(ctx, server.ID)
	if err != nil {
		t.Fatalf("HealthCheck failed: %v", err)
	}

	if !health.IsHealthy {
		t.Errorf("expected healthy server, got error: %s", health.Error)
	}
}

func TestStopServer(t *testing.T) {
	ctx := context.Background()
	pool, _ := NewServerPool(ctx, 2, 8000, 9000)

	server, _ := pool.GetAvailableServer(ctx, "agent-1")
	serverID := server.ID

	// Stop the server
	err := pool.StopServer(serverID)
	if err != nil {
		t.Fatalf("StopServer failed: %v", err)
	}

	// Verify status
	updated, _ := pool.GetServerByID(serverID)
	if updated.Status != StatusStopped {
		t.Errorf("expected StatusStopped, got %s", updated.Status)
	}

	// Try to stop again - should fail
	err = pool.StopServer(serverID)
	if err == nil {
		t.Errorf("expected error when stopping already-stopped server")
	}
}

func TestRestartServer(t *testing.T) {
	ctx := context.Background()
	pool, _ := NewServerPool(ctx, 2, 8000, 9000)

	server, _ := pool.GetAvailableServer(ctx, "agent-1")
	serverID := server.ID

	// Stop the server
	_ = pool.StopServer(serverID)

	// Restart the server
	err := pool.RestartServer(ctx, serverID)
	if err != nil {
		t.Fatalf("RestartServer failed: %v", err)
	}

	// Verify server is available again
	available, err := pool.GetAvailableServer(ctx, "agent-2")
	if err != nil {
		t.Fatalf("GetAvailableServer failed: %v", err)
	}

	if available.ID != serverID {
		t.Errorf("expected restarted server ID, got %s", available.ID)
	}
}

func TestGetStatus(t *testing.T) {
	ctx := context.Background()
	pool, _ := NewServerPool(ctx, 3, 8000, 9000)

	server1, _ := pool.GetAvailableServer(ctx, "agent-1")
	server2, _ := pool.GetAvailableServer(ctx, "agent-2")

	status := pool.GetStatus()

	if len(status) != 3 {
		t.Errorf("expected 3 servers in status, got %d", len(status))
	}

	// Verify status entries
	if status[server1.ID].AgentID != "agent-1" {
		t.Errorf("expected agent-1, got %s", status[server1.ID].AgentID)
	}

	if status[server2.ID].Status != StatusBusy {
		t.Errorf("expected StatusBusy, got %s", status[server2.ID].Status)
	}
}

func TestParallelAllocation(t *testing.T) {
	ctx := context.Background()
	pool, _ := NewServerPool(ctx, 5, 8000, 9000)

	// Simulate 5 agents requesting servers in parallel
	servers := make([]*ServerInstance, 5)
	errors := make([]error, 5)

	for i := 0; i < 5; i++ {
		go func(index int) {
			agentID := "agent-" + string(rune(48+index))
			server, err := pool.GetAvailableServer(ctx, agentID)
			servers[index] = server
			errors[index] = err
		}(i)
	}

	// Wait for goroutines
	time.Sleep(100 * time.Millisecond)

	// Verify all allocations succeeded
	for i, err := range errors {
		if err != nil {
			t.Errorf("allocation %d failed: %v", i, err)
		}
	}

	// Verify all servers are unique
	seenIDs := make(map[string]bool)
	for i, server := range servers {
		if server == nil {
			t.Errorf("server %d is nil", i)
			continue
		}
		if seenIDs[server.ID] {
			t.Errorf("server %s allocated twice", server.ID)
		}
		seenIDs[server.ID] = true
	}

	if len(seenIDs) != 5 {
		t.Errorf("expected 5 unique servers, got %d", len(seenIDs))
	}

	// Verify no servers available
	if pool.AvailableServerCount() != 0 {
		t.Errorf("expected 0 available servers, got %d", pool.AvailableServerCount())
	}
}

func TestContextCancellation(t *testing.T) {
	ctx := context.Background()
	pool, _ := NewServerPool(ctx, 1, 8000, 9000)

	// Allocate the only server
	_, _ = pool.GetAvailableServer(ctx, "agent-1")

	// Try to allocate another with cancelled context
	cancelCtx, cancel := context.WithCancel(ctx)
	cancel()

	_, err := pool.GetAvailableServer(cancelCtx, "agent-2")
	if err == nil {
		t.Errorf("expected context cancelled error, got nil")
	}
}

func TestShutdown(t *testing.T) {
	ctx := context.Background()
	pool, _ := NewServerPool(ctx, 3, 8000, 9000)

	// Allocate some servers
	_, _ = pool.GetAvailableServer(ctx, "agent-1")
	_, _ = pool.GetAvailableServer(ctx, "agent-2")

	// Shutdown the pool
	err := pool.Shutdown(ctx)
	if err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	// Verify all servers are stopped
	status := pool.GetStatus()
	for _, server := range status {
		if server.Status != StatusStopped {
			t.Errorf("expected StatusStopped, got %s for server %s", server.Status, server.ID)
		}
	}
}

func TestServerPortAllocation(t *testing.T) {
	ctx := context.Background()
	pool, _ := NewServerPool(ctx, 5, 8100, 8110)

	status := pool.GetStatus()

	// Verify port allocation
	expectedPorts := []int{8100, 8101, 8102, 8103, 8104}
	foundPorts := make(map[int]bool)

	for _, server := range status {
		foundPorts[server.Port] = true
	}

	for _, expectedPort := range expectedPorts {
		if !foundPorts[expectedPort] {
			t.Errorf("expected port %d not found in pool", expectedPort)
		}
	}
}

func TestUpdateLastSeen(t *testing.T) {
	ctx := context.Background()
	pool, _ := NewServerPool(ctx, 1, 8000, 9000)

	server, _ := pool.GetAvailableServer(ctx, "agent-1")

	// Set LastSeen to old time
	pool.mu.Lock()
	pool.instances[server.ID].LastSeen = time.Now().Add(-10 * time.Minute)
	pool.mu.Unlock()

	// Update LastSeen
	err := pool.UpdateLastSeen(server.ID)
	if err != nil {
		t.Fatalf("UpdateLastSeen failed: %v", err)
	}

	// Verify LastSeen is updated
	updated, _ := pool.GetServerByID(server.ID)
	if time.Since(updated.LastSeen) > 1*time.Second {
		t.Errorf("LastSeen not updated properly")
	}
}

func TestReleaseInvalidServer(t *testing.T) {
	ctx := context.Background()
	pool, _ := NewServerPool(ctx, 1, 8000, 9000)

	err := pool.ReleaseServer("nonexistent-server")
	if err == nil {
		t.Errorf("expected error when releasing nonexistent server")
	}
}

func TestGetServerByIDNotFound(t *testing.T) {
	ctx := context.Background()
	pool, _ := NewServerPool(ctx, 1, 8000, 9000)

	_, err := pool.GetServerByID("nonexistent-server")
	if err == nil {
		t.Errorf("expected error when getting nonexistent server")
	}
}
