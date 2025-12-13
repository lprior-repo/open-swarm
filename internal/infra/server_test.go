// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package infra

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServerManager(t *testing.T) {
	sm := NewServerManager()
	assert.NotNil(t, sm)
	assert.Equal(t, "opencode", sm.opencodeCommand)
	assert.Equal(t, 10*time.Second, sm.healthTimeout)
	assert.Equal(t, 200*time.Millisecond, sm.healthInterval)
}

func TestServerManager_SetOpencodeCommand(t *testing.T) {
	sm := NewServerManager()
	sm.SetOpencodeCommand("custom-opencode")
	assert.Equal(t, "custom-opencode", sm.opencodeCommand)
}

func TestServerManager_SetHealthTimeout(t *testing.T) {
	sm := NewServerManager()
	sm.SetHealthTimeout(5 * time.Second)
	assert.Equal(t, 5*time.Second, sm.healthTimeout)
}

func TestServerHandle_Fields(t *testing.T) {
	handle := &ServerHandle{
		Port:       8080,
		WorktreeID: "test-worktree",
		WorkDir:    "/tmp/test",
		BaseURL:    "http://localhost:8080",
		PID:        12345,
	}

	assert.Equal(t, 8080, handle.Port)
	assert.Equal(t, "test-worktree", handle.WorktreeID)
	assert.Equal(t, "/tmp/test", handle.WorkDir)
	assert.Equal(t, "http://localhost:8080", handle.BaseURL)
	assert.Equal(t, 12345, handle.PID)
}

// Integration test - requires opencode CLI installed
func TestServerManager_BootServer_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	sm := NewServerManager()
	portMgr := NewPortManager(8000, 8100)

	port, err := portMgr.Allocate()
	require.NoError(t, err)
	defer portMgr.Release(port)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Boot server
	handle, err := sm.BootServer(ctx, ".", "test-agent", port)
	if err != nil {
		t.Logf("Server boot failed (may be expected if opencode not installed): %v", err)
		t.Skip("Skipping server boot test - opencode may not be available")
		return
	}
	require.NotNil(t, handle)
	defer func() {
		_ = sm.Shutdown(handle)
	}()

	// Verify handle
	assert.Equal(t, port, handle.Port)
	assert.NotZero(t, handle.PID)
	assert.Contains(t, handle.BaseURL, "localhost")

	// Test health check
	healthy := sm.IsHealthy(ctx, handle)
	assert.True(t, healthy, "Server should be healthy after boot")
}

func TestServerManager_HealthTimeout(t *testing.T) {
	ctx := context.Background()
	serverMgr := NewServerManager()

	// Set unreasonably short timeout
	serverMgr.SetHealthTimeout(1 * time.Millisecond)

	// Use a command that won't respond to health checks
	// 'sleep 100' exists and will run but won't serve HTTP
	serverMgr.SetOpencodeCommand("sleep")

	cwd, err := os.Getwd()
	require.NoError(t, err)

	// This should fail due to health check timeout
	_, err = serverMgr.BootServer(ctx, cwd, "test-timeout", 8080)

	// Verify error occurred
	assert.Error(t, err, "BootServer should fail when health check times out")
	assert.Contains(t, err.Error(), "failed to become ready", "Error should mention health check failure")
}
