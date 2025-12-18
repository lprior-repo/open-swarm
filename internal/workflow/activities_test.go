package workflow

import (
	"context"
	"errors"
	"strings"
	"testing"

	"open-swarm/internal/agent"
	"open-swarm/internal/infra"

	"github.com/sst/opencode-sdk-go"
)

// Mock implementations for testing
type mockPortManager struct {
	allocateFunc    func() (int, error)
	releaseFunc     func(port int) error
	allocatedCount  int
	availableCount  int
	isAllocatedFunc func(port int) bool
}

func (m *mockPortManager) Allocate() (int, error) {
	if m.allocateFunc != nil {
		return m.allocateFunc()
	}
	return 8080, nil
}

func (m *mockPortManager) Release(port int) error {
	if m.releaseFunc != nil {
		return m.releaseFunc(port)
	}
	return nil
}

func (m *mockPortManager) AllocatedCount() int {
	return m.allocatedCount
}

func (m *mockPortManager) AvailableCount() int {
	return m.availableCount
}

func (m *mockPortManager) IsAllocated(port int) bool {
	if m.isAllocatedFunc != nil {
		return m.isAllocatedFunc(port)
	}
	return false
}

type mockServerManager struct {
	bootFunc          func(ctx context.Context, path, id string, port int) (*infra.ServerHandle, error)
	shutdownFunc      func(handle *infra.ServerHandle) error
	shutdownByPIDFunc func(pid int) error
	healthyFunc       func(ctx context.Context, handle *infra.ServerHandle) bool
}

func (m *mockServerManager) BootServer(ctx context.Context, path, id string, port int) (*infra.ServerHandle, error) {
	if m.bootFunc != nil {
		return m.bootFunc(ctx, path, id, port)
	}
	return &infra.ServerHandle{
		Port:    port,
		BaseURL: "http://localhost:8080",
		PID:     12345,
	}, nil
}

func (m *mockServerManager) Shutdown(handle *infra.ServerHandle) error {
	if m.shutdownFunc != nil {
		return m.shutdownFunc(handle)
	}
	return nil
}

func (m *mockServerManager) ShutdownByPID(pid int) error {
	if m.shutdownByPIDFunc != nil {
		return m.shutdownByPIDFunc(pid)
	}
	return nil
}

func (m *mockServerManager) IsHealthy(ctx context.Context, handle *infra.ServerHandle) bool {
	if m.healthyFunc != nil {
		return m.healthyFunc(ctx, handle)
	}
	return true
}

type mockWorktreeManager struct {
	createFunc  func(id, branch string) (*infra.WorktreeInfo, error)
	removeFunc  func(id string) error
	listFunc    func() ([]*infra.WorktreeInfo, error)
	pruneFunc   func() error
	cleanupFunc func() error
}

func (m *mockWorktreeManager) CreateWorktree(id, branch string) (*infra.WorktreeInfo, error) {
	if m.createFunc != nil {
		return m.createFunc(id, branch)
	}
	return &infra.WorktreeInfo{
		ID:   id,
		Path: "/tmp/worktrees/" + id,
	}, nil
}

func (m *mockWorktreeManager) RemoveWorktree(id string) error {
	if m.removeFunc != nil {
		return m.removeFunc(id)
	}
	return nil
}

func (m *mockWorktreeManager) ListWorktrees() ([]*infra.WorktreeInfo, error) {
	if m.listFunc != nil {
		return m.listFunc()
	}
	return []*infra.WorktreeInfo{}, nil
}

func (m *mockWorktreeManager) PruneWorktrees() error {
	if m.pruneFunc != nil {
		return m.pruneFunc()
	}
	return nil
}

func (m *mockWorktreeManager) CleanupAll() error {
	if m.cleanupFunc != nil {
		return m.cleanupFunc()
	}
	return nil
}

type mockClient struct {
	executePromptFunc  func(ctx context.Context, prompt string, opts *agent.PromptOptions) (*agent.PromptResult, error)
	executeCommandFunc func(ctx context.Context, sessionID string, command string, args []string) (*agent.PromptResult, error)
	getFileStatusFunc  func(ctx context.Context) ([]opencode.File, error)
	baseURL            string
	port               int
}

func (m *mockClient) ExecutePrompt(ctx context.Context, prompt string, opts *agent.PromptOptions) (*agent.PromptResult, error) {
	if m.executePromptFunc != nil {
		return m.executePromptFunc(ctx, prompt, opts)
	}
	return &agent.PromptResult{
		SessionID: "test-session",
		MessageID: "test-message",
		Parts: []agent.ResultPart{
			{Type: "text", Text: "success"},
		},
	}, nil
}

func (m *mockClient) ExecuteCommand(ctx context.Context, sessionID string, command string, args []string) (*agent.PromptResult, error) {
	if m.executeCommandFunc != nil {
		return m.executeCommandFunc(ctx, sessionID, command, args)
	}
	return &agent.PromptResult{
		SessionID: "test-session",
		MessageID: "test-message",
		Parts: []agent.ResultPart{
			{Type: "text", Text: "success"},
		},
	}, nil
}

func (m *mockClient) GetFileStatus(ctx context.Context) ([]opencode.File, error) {
	if m.getFileStatusFunc != nil {
		return m.getFileStatusFunc(ctx)
	}
	return []opencode.File{}, nil
}

func (m *mockClient) GetBaseURL() string {
	if m.baseURL != "" {
		return m.baseURL
	}
	return "http://localhost:8080"
}

func (m *mockClient) GetPort() int {
	if m.port != 0 {
		return m.port
	}
	return 8080
}

// Tests for BootstrapCell
func TestBootstrapCell_Success(t *testing.T) {
	portMgr := &mockPortManager{}
	serverMgr := &mockServerManager{}
	worktreeMgr := &mockWorktreeManager{}

	activities := NewActivities(portMgr, serverMgr, worktreeMgr)

	cell, err := activities.BootstrapCell(context.Background(), "test-cell", "main")
	if err != nil {
		t.Fatalf("BootstrapCell failed: %v", err)
	}

	if cell.CellID != "test-cell" {
		t.Errorf("Expected CellID 'test-cell', got %s", cell.CellID)
	}
	if cell.Port == 0 {
		t.Error("Port should be allocated")
	}
	if cell.WorktreeID == "" {
		t.Error("WorktreeID should be set")
	}
	if cell.WorktreePath == "" {
		t.Error("WorktreePath should be set")
	}
	if cell.ServerHandle == nil {
		t.Error("ServerHandle should not be nil")
	}
	if cell.Client == nil {
		t.Error("Client should not be nil")
	}
}

func TestBootstrapCell_PortAllocationFailure(t *testing.T) {
	portMgr := &mockPortManager{
		allocateFunc: func() (int, error) {
			return 0, errors.New("no ports available")
		},
	}
	serverMgr := &mockServerManager{}
	worktreeMgr := &mockWorktreeManager{}

	activities := NewActivities(portMgr, serverMgr, worktreeMgr)

	_, err := activities.BootstrapCell(context.Background(), "test-cell", "main")
	if err == nil {
		t.Fatal("Expected error when port allocation fails")
	}
	if err.Error() != "failed to allocate port: no ports available" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestBootstrapCell_WorktreeCreationFailure(t *testing.T) {
	portReleased := false
	portMgr := &mockPortManager{
		releaseFunc: func(_ int) error {
			portReleased = true
			return nil
		},
	}
	serverMgr := &mockServerManager{}
	worktreeMgr := &mockWorktreeManager{
		createFunc: func(_ string, _ string) (*infra.WorktreeInfo, error) {
			return nil, errors.New("git worktree creation failed")
		},
	}

	activities := NewActivities(portMgr, serverMgr, worktreeMgr)

	_, err := activities.BootstrapCell(context.Background(), "test-cell", "main")
	if err == nil {
		t.Fatal("Expected error when worktree creation fails")
	}
	if !portReleased {
		t.Error("Port should be released on cleanup")
	}
}

func TestBootstrapCell_ServerBootFailure(t *testing.T) {
	portReleased := false
	worktreeRemoved := false

	portMgr := &mockPortManager{
		releaseFunc: func(_ int) error {
			portReleased = true
			return nil
		},
	}
	serverMgr := &mockServerManager{
		bootFunc: func(_ context.Context, _ string, _ string, _ int) (*infra.ServerHandle, error) {
			return nil, errors.New("server boot failed")
		},
	}
	worktreeMgr := &mockWorktreeManager{
		removeFunc: func(_ string) error {
			worktreeRemoved = true
			return nil
		},
	}

	activities := NewActivities(portMgr, serverMgr, worktreeMgr)

	_, err := activities.BootstrapCell(context.Background(), "test-cell", "main")
	if err == nil {
		t.Fatal("Expected error when server boot fails")
	}
	if !portReleased {
		t.Error("Port should be released on cleanup")
	}
	if !worktreeRemoved {
		t.Error("Worktree should be removed on cleanup")
	}
}

// Tests for TeardownCell
func TestTeardownCell_Success(t *testing.T) {
	serverShutdown := false
	worktreeRemoved := false
	portReleased := false

	portMgr := &mockPortManager{
		releaseFunc: func(_ int) error {
			portReleased = true
			return nil
		},
	}
	serverMgr := &mockServerManager{
		shutdownByPIDFunc: func(_ int) error {
			serverShutdown = true
			return nil
		},
	}
	worktreeMgr := &mockWorktreeManager{
		removeFunc: func(_ string) error {
			worktreeRemoved = true
			return nil
		},
	}

	activities := NewActivities(portMgr, serverMgr, worktreeMgr)

	cell := &CellBootstrap{
		CellID:       "test-cell",
		Port:         8080,
		WorktreeID:   "worktree-1",
		WorktreePath: "/tmp/worktrees/worktree-1",
		ServerHandle: &infra.ServerHandle{Port: 8080, BaseURL: "http://localhost:8080", PID: 12345},
		Client:       &mockClient{},
	}

	err := activities.TeardownCell(context.Background(), cell)
	if err != nil {
		t.Fatalf("TeardownCell failed: %v", err)
	}

	if !serverShutdown {
		t.Error("Server should be shut down")
	}
	if !worktreeRemoved {
		t.Error("Worktree should be removed")
	}
	if !portReleased {
		t.Error("Port should be released")
	}
}

func TestTeardownCell_PartialFailure(t *testing.T) {
	worktreeRemoved := false
	portReleased := false

	portMgr := &mockPortManager{
		releaseFunc: func(_ int) error {
			portReleased = true
			return nil
		},
	}
	serverMgr := &mockServerManager{
		shutdownByPIDFunc: func(_ int) error {
			return errors.New("shutdown failed")
		},
	}
	worktreeMgr := &mockWorktreeManager{
		removeFunc: func(_ string) error {
			worktreeRemoved = true
			return nil
		},
	}

	activities := NewActivities(portMgr, serverMgr, worktreeMgr)

	cell := &CellBootstrap{
		CellID:       "test-cell",
		Port:         8080,
		WorktreeID:   "worktree-1",
		ServerHandle: &infra.ServerHandle{PID: 12345},
	}

	err := activities.TeardownCell(context.Background(), cell)
	if err == nil {
		t.Fatal("Expected error when shutdown fails")
	}

	// Should still attempt other cleanup steps
	if !worktreeRemoved {
		t.Error("Worktree should still be removed despite server shutdown failure")
	}
	if !portReleased {
		t.Error("Port should still be released despite server shutdown failure")
	}
}

// Tests for ExecuteTask
func TestExecuteTask_Success(t *testing.T) {
	client := &mockClient{
		executePromptFunc: func(_ context.Context, _ string, _ *agent.PromptOptions) (*agent.PromptResult, error) {
			return &agent.PromptResult{
				SessionID: "test-session",
				Parts: []agent.ResultPart{
					{Type: "text", Text: "Task completed successfully"},
				},
			}, nil
		},
		getFileStatusFunc: func(_ context.Context) ([]opencode.File, error) {
			return []opencode.File{}, nil
		},
	}

	portMgr := &mockPortManager{}
	serverMgr := &mockServerManager{
		healthyFunc: func(_ context.Context, _ *infra.ServerHandle) bool {
			return true
		},
	}
	worktreeMgr := &mockWorktreeManager{}

	activities := NewActivities(portMgr, serverMgr, worktreeMgr)

	cell := &CellBootstrap{
		CellID:       "test-cell",
		ServerHandle: &infra.ServerHandle{},
		Client:       client,
	}

	task := &agent.TaskContext{
		Prompt: "Test task",
	}

	result, err := activities.ExecuteTask(context.Background(), cell, task)
	if err != nil {
		t.Fatalf("ExecuteTask failed: %v", err)
	}

	if !result.Success {
		t.Error("Task should succeed")
	}
	if result.Output == "" {
		t.Error("Output should not be empty")
	}
}

func TestExecuteTask_UnhealthyServer(t *testing.T) {
	portMgr := &mockPortManager{}
	serverMgr := &mockServerManager{
		healthyFunc: func(_ context.Context, _ *infra.ServerHandle) bool {
			return false
		},
	}
	worktreeMgr := &mockWorktreeManager{}

	activities := NewActivities(portMgr, serverMgr, worktreeMgr)

	cell := &CellBootstrap{
		CellID:       "test-cell",
		ServerHandle: &infra.ServerHandle{},
		Client:       &mockClient{},
	}

	task := &agent.TaskContext{
		Prompt: "Test task",
	}

	_, err := activities.ExecuteTask(context.Background(), cell, task)
	if err == nil {
		t.Fatal("Expected error when server is unhealthy")
	}
	if err.Error() != "server is not healthy" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

// Tests for RunTests
func TestRunTests_Pass(t *testing.T) {
	client := &mockClient{
		executeCommandFunc: func(ctx context.Context, sessionID string, command string, args []string) (*agent.PromptResult, error) {
			return &agent.PromptResult{
				Parts: []agent.ResultPart{
					{Type: "text", Text: "PASS\nok  \tpackage\t0.001s"},
				},
			}, nil
		},
	}

	portMgr := &mockPortManager{}
	serverMgr := &mockServerManager{}
	worktreeMgr := &mockWorktreeManager{}

	activities := NewActivities(portMgr, serverMgr, worktreeMgr)

	cell := &CellBootstrap{
		CellID: "test-cell",
		Client: client,
	}

	passed, err := activities.RunTests(context.Background(), cell)
	if err != nil {
		t.Fatalf("RunTests failed: %v", err)
	}

	if !passed {
		t.Error("Tests should pass")
	}
}

func TestRunTests_Fail(t *testing.T) {
	client := &mockClient{
		executePromptFunc: func(ctx context.Context, prompt string, opts *agent.PromptOptions) (*agent.PromptResult, error) {
			return &agent.PromptResult{
				Parts: []agent.ResultPart{
					{Type: "text", Text: "FAIL\nFAIL\tpackage\t0.001s"},
				},
			}, nil
		},
	}

	portMgr := &mockPortManager{}
	serverMgr := &mockServerManager{}
	worktreeMgr := &mockWorktreeManager{}

	activities := NewActivities(portMgr, serverMgr, worktreeMgr)

	cell := &CellBootstrap{
		CellID: "test-cell",
		Client: client,
	}

	passed, err := activities.RunTests(context.Background(), cell)
	if err != nil {
		t.Fatalf("RunTests failed: %v", err)
	}

	if passed {
		t.Error("Tests should fail")
	}
}

// Tests for CommitChanges
func TestCommitChanges_Success(t *testing.T) {
	commitCalled := false
	client := &mockClient{
		executePromptFunc: func(ctx context.Context, prompt string, opts *agent.PromptOptions) (*agent.PromptResult, error) {
			if strings.Contains(prompt, "git") {
				commitCalled = true
			}
			return &agent.PromptResult{
				Parts: []agent.ResultPart{
					{Type: "text", Text: "Changes committed"},
				},
			}, nil
		},
	}

	portMgr := &mockPortManager{}
	serverMgr := &mockServerManager{}
	worktreeMgr := &mockWorktreeManager{}

	activities := NewActivities(portMgr, serverMgr, worktreeMgr)

	cell := &CellBootstrap{
		CellID: "test-cell",
		Client: client,
	}

	err := activities.CommitChanges(context.Background(), cell, "Test commit")
	if err != nil {
		t.Fatalf("CommitChanges failed: %v", err)
	}

	if !commitCalled {
		t.Error("Git commit should be called")
	}
}

// Tests for RevertChanges
func TestRevertChanges_Success(t *testing.T) {
	revertCalled := false
	client := &mockClient{
		executePromptFunc: func(ctx context.Context, prompt string, opts *agent.PromptOptions) (*agent.PromptResult, error) {
			if strings.Contains(prompt, "git") {
				revertCalled = true
			}
			return &agent.PromptResult{
				Parts: []agent.ResultPart{
					{Type: "text", Text: "Changes reverted"},
				},
			}, nil
		},
	}

	portMgr := &mockPortManager{}
	serverMgr := &mockServerManager{}
	worktreeMgr := &mockWorktreeManager{}

	activities := NewActivities(portMgr, serverMgr, worktreeMgr)

	cell := &CellBootstrap{
		CellID: "test-cell",
		Client: client,
	}

	err := activities.RevertChanges(context.Background(), cell)
	if err != nil {
		t.Fatalf("RevertChanges failed: %v", err)
	}

	if !revertCalled {
		t.Error("Git revert should be called")
	}
}
