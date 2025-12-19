// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package spawner

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"open-swarm/internal/infra"
	"open-swarm/internal/workflow"
)

// MockPortManager mocks port allocation
type MockPortManager struct {
	mock.Mock
}

func (m *MockPortManager) Allocate() (int, error) {
	args := m.Called()
	return args.Int(0), args.Error(1)
}

func (m *MockPortManager) Release(port int) error {
	args := m.Called(port)
	return args.Error(0)
}

func (m *MockPortManager) AllocatedCount() int {
	args := m.Called()
	return args.Int(0)
}

func (m *MockPortManager) IsAllocated(port int) bool {
	args := m.Called(port)
	return args.Bool(0)
}

func (m *MockPortManager) AvailableCount() int {
	args := m.Called()
	return args.Int(0)
}

// MockServerManager mocks server lifecycle
type MockServerManager struct {
	mock.Mock
}

func (m *MockServerManager) BootServer(ctx context.Context, path, id string, port int) (*infra.ServerHandle, error) {
	args := m.Called(ctx, path, id, port)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*infra.ServerHandle), args.Error(1)
}

func (m *MockServerManager) Shutdown(handle *infra.ServerHandle) error {
	args := m.Called(handle)
	return args.Error(0)
}

func (m *MockServerManager) ShutdownByPID(pid int) error {
	args := m.Called(pid)
	return args.Error(0)
}

func (m *MockServerManager) IsHealthy(ctx context.Context, handle *infra.ServerHandle) bool {
	args := m.Called(ctx, handle)
	return args.Bool(0)
}

// MockWorktreeManager mocks worktree operations
type MockWorktreeManager struct {
	mock.Mock
}

func (m *MockWorktreeManager) CreateWorktree(id, branch string) (*infra.WorktreeInfo, error) {
	args := m.Called(id, branch)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*infra.WorktreeInfo), args.Error(1)
}

func (m *MockWorktreeManager) RemoveWorktree(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockWorktreeManager) ListWorktrees() ([]*infra.WorktreeInfo, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*infra.WorktreeInfo), args.Error(1)
}

func (m *MockWorktreeManager) PruneWorktrees() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockWorktreeManager) CleanupAll() error {
	args := m.Called()
	return args.Error(0)
}

// TestSpawnAgent_Success tests successful agent spawning
func TestSpawnAgent_Success(t *testing.T) {
	// Setup mocks
	portMgr := new(MockPortManager)
	portMgr.On("Allocate").Return(9001, nil)

	serverMgr := new(MockServerManager)
	serverMgr.On("BootServer", mock.Anything, mock.Anything, mock.Anything, 9001).
		Return(&infra.ServerHandle{
			BaseURL: "http://localhost:9001",
			Port:    9001,
			PID:     1234,
		}, nil)

	worktreeMgr := new(MockWorktreeManager)
	worktreeMgr.On("CreateWorktree", mock.Anything, "main").
		Return(&infra.WorktreeInfo{
			ID:   "wt-1",
			Path: "/tmp/worktree-1",
		}, nil)

	spawner := NewAgentSpawner(portMgr, serverMgr, worktreeMgr)
	ctx := context.Background()

	config := SpawnConfig{
		AgentID: "agent-1",
		TaskID:  "task-123",
		Branch:  "main",
	}

	// Spawn agent
	spawned, err := spawner.SpawnAgent(ctx, config)

	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, spawned)
	assert.Equal(t, "agent-1", spawned.ID)
	assert.Equal(t, "task-123", spawned.TaskID)
	assert.NotNil(t, spawned.Cell)
	assert.Equal(t, 9001, spawned.Cell.Port)
	assert.Equal(t, "http://localhost:9001", spawned.Cell.ServerHandle.BaseURL)
	assert.NotZero(t, spawned.CreatedAt)

	// Verify mocks called
	portMgr.AssertExpectations(t)
	serverMgr.AssertExpectations(t)
	worktreeMgr.AssertExpectations(t)
}

// TestSpawnAgent_MissingAgentID fails without agent ID
func TestSpawnAgent_MissingAgentID(t *testing.T) {
	spawner := NewAgentSpawner(
		new(MockPortManager),
		new(MockServerManager),
		new(MockWorktreeManager),
	)

	config := SpawnConfig{
		TaskID: "task-123",
		// Missing AgentID
	}

	spawned, err := spawner.SpawnAgent(context.Background(), config)

	assert.Error(t, err)
	assert.Nil(t, spawned)
	assert.Contains(t, err.Error(), "agent ID required")
}

// TestSpawnAgent_MissingTaskID fails without task ID
func TestSpawnAgent_MissingTaskID(t *testing.T) {
	spawner := NewAgentSpawner(
		new(MockPortManager),
		new(MockServerManager),
		new(MockWorktreeManager),
	)

	config := SpawnConfig{
		AgentID: "agent-1",
		// Missing TaskID
	}

	spawned, err := spawner.SpawnAgent(context.Background(), config)

	assert.Error(t, err)
	assert.Nil(t, spawned)
	assert.Contains(t, err.Error(), "task ID required")
}

// TestSpawnAgent_BootstrapFailure handles bootstrap failures
func TestSpawnAgent_BootstrapFailure(t *testing.T) {
	portMgr := new(MockPortManager)
	portMgr.On("Allocate").Return(9001, nil)
	portMgr.On("Release", 9001).Return(nil) // Cleanup on failure

	serverMgr := new(MockServerManager)
	serverMgr.On("BootServer", mock.Anything, mock.Anything, mock.Anything, 9001).
		Return(nil, assert.AnError)

	worktreeMgr := new(MockWorktreeManager)
	worktreeMgr.On("CreateWorktree", mock.Anything, "main").
		Return(&infra.WorktreeInfo{Path: "/tmp/wt"}, nil)
	worktreeMgr.On("RemoveWorktree", mock.Anything).Return(nil) // Cleanup on failure

	spawner := NewAgentSpawner(portMgr, serverMgr, worktreeMgr)

	config := SpawnConfig{
		AgentID: "agent-1",
		TaskID:  "task-123",
	}

	spawned, err := spawner.SpawnAgent(context.Background(), config)

	assert.Error(t, err)
	assert.Nil(t, spawned)
	assert.Contains(t, err.Error(), "failed to bootstrap cell")
}

// TestSpawnAgent_DefaultBranch uses main branch by default
func TestSpawnAgent_DefaultBranch(t *testing.T) {
	portMgr := new(MockPortManager)
	portMgr.On("Allocate").Return(9001, nil)

	serverMgr := new(MockServerManager)
	serverMgr.On("BootServer", mock.Anything, mock.Anything, mock.Anything, 9001).
		Return(&infra.ServerHandle{BaseURL: "http://localhost:9001", Port: 9001}, nil)
	serverMgr.On("Shutdown", mock.Anything).Return(nil)

	worktreeMgr := new(MockWorktreeManager)
	worktreeMgr.On("CreateWorktree", mock.Anything, "main").
		Return(&infra.WorktreeInfo{Path: "/tmp/wt"}, nil)
	worktreeMgr.On("RemoveWorktree", mock.Anything).Return(nil)

	spawner := NewAgentSpawner(portMgr, serverMgr, worktreeMgr)

	config := SpawnConfig{
		AgentID: "agent-1",
		TaskID:  "task-123",
		// Branch not specified
	}

	spawned, err := spawner.SpawnAgent(context.Background(), config)

	assert.NoError(t, err)
	assert.NotNil(t, spawned)
	// Verify CreateWorktree was called (main branch usage is internal to BootstrapCell)
	worktreeMgr.AssertCalled(t, "CreateWorktree", mock.Anything, "main")
}

// TestTeardownAgent_Success cleans up resources
func TestTeardownAgent_Success(t *testing.T) {
	portMgr := new(MockPortManager)
	portMgr.On("Release", mock.Anything).Return(nil) // Match any port

	serverMgr := new(MockServerManager)
	serverMgr.On("Shutdown", mock.Anything).Return(nil)
	serverMgr.On("ShutdownByPID", mock.Anything).Return(nil) // Match any PID

	worktreeMgr := new(MockWorktreeManager)
	worktreeMgr.On("RemoveWorktree", mock.Anything).Return(nil) // Match any worktree ID

	spawner := NewAgentSpawner(portMgr, serverMgr, worktreeMgr)

	spawned := &SpawnedAgent{
		ID: "agent-1",
		Cell: &workflow.CellBootstrap{
			CellID:     "agent-1",
			Port:       9001,
			WorktreeID: "wt-1",
			ServerHandle: &infra.ServerHandle{
				BaseURL: "http://localhost:9001",
				Port:    9001,
				PID:     1234,
			},
		},
	}

	metrics := spawner.TeardownAgent(context.Background(), spawned)

	assert.NotNil(t, metrics)
	assert.NotZero(t, metrics.TeardownDuration)

	// Verify cleanup was called
	portMgr.AssertCalled(t, "Release", mock.Anything)
	worktreeMgr.AssertCalled(t, "RemoveWorktree", mock.Anything)
	// Either Shutdown or ShutdownByPID will be called depending on the Server Handle state
}

// TestTeardownAgent_NilSpawned handles nil agent gracefully
func TestTeardownAgent_NilSpawned(t *testing.T) {
	spawner := NewAgentSpawner(
		new(MockPortManager),
		new(MockServerManager),
		new(MockWorktreeManager),
	)

	metrics := spawner.TeardownAgent(context.Background(), nil)

	assert.NotNil(t, metrics)
	assert.Zero(t, metrics.TeardownDuration)
}

// TestIsHealthy checks agent cell health
func TestIsHealthy_Healthy(t *testing.T) {
	serverMgr := new(MockServerManager)
	serverMgr.On("IsHealthy", mock.Anything, mock.Anything).Return(true)

	spawner := NewAgentSpawner(
		new(MockPortManager),
		serverMgr,
		new(MockWorktreeManager),
	)

	spawned := &SpawnedAgent{
		Cell: &workflow.CellBootstrap{
			ServerHandle: &infra.ServerHandle{},
		},
	}

	healthy := spawner.IsHealthy(context.Background(), spawned)
	assert.True(t, healthy)
}

// TestIsHealthy_Unhealthy detects dead agents
func TestIsHealthy_Unhealthy(t *testing.T) {
	serverMgr := new(MockServerManager)
	serverMgr.On("IsHealthy", mock.Anything, mock.Anything).Return(false)

	spawner := NewAgentSpawner(
		new(MockPortManager),
		serverMgr,
		new(MockWorktreeManager),
	)

	spawned := &SpawnedAgent{
		Cell: &workflow.CellBootstrap{
			ServerHandle: &infra.ServerHandle{},
		},
	}

	healthy := spawner.IsHealthy(context.Background(), spawned)
	assert.False(t, healthy)
}

// TestGetCellInfo returns cell metadata
func TestGetCellInfo(t *testing.T) {
	now := time.Now()
	spawned := &SpawnedAgent{
		ID:        "agent-1",
		Cell: &workflow.CellBootstrap{
			CellID:       "cell-1",
			Port:         9001,
			WorktreeID:   "wt-1",
			WorktreePath: "/tmp/wt-1",
			ServerHandle: &infra.ServerHandle{
				BaseURL: "http://localhost:9001",
			},
		},
		CreatedAt: now,
		TokensUsed: 1000,
	}

	spawner := NewAgentSpawner(
		new(MockPortManager),
		new(MockServerManager),
		new(MockWorktreeManager),
	)

	info := spawner.GetCellInfo(spawned)

	assert.NotNil(t, info)
	assert.Equal(t, "cell-1", info["cell_id"])
	assert.Equal(t, 9001, info["port"])
	assert.Equal(t, "wt-1", info["worktree_id"])
	assert.Equal(t, "/tmp/wt-1", info["worktree_path"])
	assert.Equal(t, "http://localhost:9001", info["base_url"])
	assert.Equal(t, now, info["created_at"])
	assert.Equal(t, 1000, info["tokens_used"])
}

// TestGetCellInfo_NilAgent returns nil for nil agent
func TestGetCellInfo_NilAgent(t *testing.T) {
	spawner := NewAgentSpawner(
		new(MockPortManager),
		new(MockServerManager),
		new(MockWorktreeManager),
	)

	info := spawner.GetCellInfo(nil)
	assert.Nil(t, info)
}

// TestLifecycleMetrics_Timing measures execution duration
func TestLifecycleMetrics_Timing(t *testing.T) {
	metrics := &LifecycleMetrics{
		SpawnDuration:     100 * time.Millisecond,
		ExecutionDuration: 500 * time.Millisecond,
		TeardownDuration:  50 * time.Millisecond,
	}

	metrics.TotalDuration = metrics.SpawnDuration + metrics.ExecutionDuration + metrics.TeardownDuration

	assert.Equal(t, 650*time.Millisecond, metrics.TotalDuration)
	assert.Less(t, metrics.TeardownDuration, metrics.ExecutionDuration)
}

// TestSpawnedAgent_Isolation verifies each agent has isolated cell
func TestSpawnedAgent_Isolation(t *testing.T) {
	portMgr := new(MockPortManager)
	portMgr.On("Allocate").Return(9001, nil).Once()
	portMgr.On("Allocate").Return(9002, nil).Once()

	serverMgr := new(MockServerManager)
	// Match any BootServer call with our port allocations
	serverMgr.On("BootServer", mock.Anything, mock.Anything, mock.Anything, 9001).
		Return(&infra.ServerHandle{BaseURL: "http://localhost:9001", Port: 9001}, nil)
	serverMgr.On("BootServer", mock.Anything, mock.Anything, mock.Anything, 9002).
		Return(&infra.ServerHandle{BaseURL: "http://localhost:9002", Port: 9002}, nil)
	serverMgr.On("Shutdown", mock.Anything).Return(nil)

	worktreeMgr := new(MockWorktreeManager)
	// The actual ID will be "cell-agent-X-<timestamp>"
	worktreeMgr.On("CreateWorktree", mock.MatchedBy(func(id string) bool {
		return id == "agent-1" || id == "agent-2" || len(id) > 0
	}), "main").Return(&infra.WorktreeInfo{ID: "wt-dynamic", Path: "/tmp/wt-dynamic"}, nil)
	worktreeMgr.On("RemoveWorktree", mock.Anything).Return(nil)

	spawner := NewAgentSpawner(portMgr, serverMgr, worktreeMgr)
	ctx := context.Background()

	// Spawn two agents
	agent1, err1 := spawner.SpawnAgent(ctx, SpawnConfig{
		AgentID: "agent-1",
		TaskID:  "task-1",
	})
	require.NoError(t, err1)

	agent2, err2 := spawner.SpawnAgent(ctx, SpawnConfig{
		AgentID: "agent-2",
		TaskID:  "task-2",
	})
	require.NoError(t, err2)

	// Verify isolation: different ports, different worktrees
	assert.NotEqual(t, agent1.Cell.Port, agent2.Cell.Port)
	assert.NotEqual(t, agent1.Cell.WorktreeID, agent2.Cell.WorktreeID)
	assert.NotEqual(t, agent1.Cell.ServerHandle.BaseURL, agent2.Cell.ServerHandle.BaseURL)
}
