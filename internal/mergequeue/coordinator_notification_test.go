// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package mergequeue

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockBranchKillNotifier implements BranchKillNotifier for testing
type MockBranchKillNotifier struct {
	mu               sync.Mutex
	notifications    []*SpeculativeBranch
	notificationErrs []error
}

func (m *MockBranchKillNotifier) NotifyBranchKilled(_ context.Context, branch *SpeculativeBranch, _ string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.notifications = append(m.notifications, branch)
	if len(m.notificationErrs) > 0 {
		err := m.notificationErrs[0]
		m.notificationErrs = m.notificationErrs[1:]
		return err
	}
	return nil
}

func (m *MockBranchKillNotifier) GetNotifications() []*SpeculativeBranch {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.notifications
}

func TestCoordinator_SetNotifier(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{})
	notifier := &MockBranchKillNotifier{}

	coord.SetNotifier(notifier)

	retrieved := coord.getNotifier()
	require.NotNil(t, retrieved)
	assert.Equal(t, notifier, retrieved)
}

func TestCoordinator_RemoveNotifier(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{})
	notifier := &MockBranchKillNotifier{}

	coord.SetNotifier(notifier)
	assert.NotNil(t, coord.getNotifier())

	coord.removeNotifier()
	assert.Nil(t, coord.getNotifier())
}

func TestCoordinator_KillFailedBranch_SendsNotification(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{})
	notifier := &MockBranchKillNotifier{}
	coord.SetNotifier(notifier)

	// Add a branch to the coordinator
	branch := &SpeculativeBranch{
		ID:    "test-branch-1",
		Depth: 2,
		Changes: []ChangeRequest{
			{
				ID:            "agent-123",
				WorktreePath:  "/tmp/worktree1",
				FilesModified: []string{"file1.go"},
			},
		},
		Status: BranchStatusTesting,
	}

	coord.mu.Lock()
	coord.activeBranches["test-branch-1"] = branch
	coord.mu.Unlock()

	// Kill the branch
	err := coord.KillFailedBranchWithValidation(context.Background(), "test-branch-1", "tests failed")
	require.NoError(t, err)

	// Verify notification was sent
	notifications := notifier.GetNotifications()
	require.Len(t, notifications, 1)
	assert.Equal(t, "test-branch-1", notifications[0].ID)
}

func TestCoordinator_KillFailedBranch_NoNotifierConfigured(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{})
	// Don't set a notifier

	// Add a branch
	branch := &SpeculativeBranch{
		ID:    "test-branch-1",
		Depth: 1,
		Changes: []ChangeRequest{
			{
				ID:            "agent-123",
				WorktreePath:  "/tmp/worktree1",
				FilesModified: []string{"file1.go"},
			},
		},
		Status: BranchStatusTesting,
	}

	coord.mu.Lock()
	coord.activeBranches["test-branch-1"] = branch
	coord.mu.Unlock()

	// Kill should still succeed even without notifier (graceful degradation)
	err := coord.KillFailedBranchWithValidation(context.Background(), "test-branch-1", "tests failed")
	require.NoError(t, err)

	// Verify branch was killed
	coord.mu.Lock()
	killedBranch := coord.activeBranches["test-branch-1"]
	coord.mu.Unlock()

	assert.Equal(t, BranchStatusKilled, killedBranch.Status)
	assert.Equal(t, "tests failed", killedBranch.KillReason)
}

func TestCoordinator_NotifyBranchKilled_NotifierReturnsError(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{})
	notifier := &MockBranchKillNotifier{
		notificationErrs: []error{assert.AnError},
	}
	coord.SetNotifier(notifier)

	// Add a branch
	branch := &SpeculativeBranch{
		ID:    "test-branch-1",
		Depth: 1,
		Changes: []ChangeRequest{
			{ID: "agent-123", WorktreePath: "/tmp/w1", FilesModified: []string{"file1.go"}},
		},
		Status: BranchStatusTesting,
	}

	coord.mu.Lock()
	coord.activeBranches["test-branch-1"] = branch
	coord.mu.Unlock()

	// Kill should still succeed even if notification fails
	err := coord.KillFailedBranchWithValidation(context.Background(), "test-branch-1", "tests failed")
	require.NoError(t, err)

	// Verify branch was still killed
	coord.mu.Lock()
	killedBranch := coord.activeBranches["test-branch-1"]
	coord.mu.Unlock()

	assert.Equal(t, BranchStatusKilled, killedBranch.Status)
}

func TestCoordinator_KillDependentBranches_SendsNotifications(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{})
	notifier := &MockBranchKillNotifier{}
	coord.SetNotifier(notifier)

	// Create a parent branch with children
	parentBranch := &SpeculativeBranch{
		ID:          "parent-branch",
		Depth:       1,
		Changes:     []ChangeRequest{{ID: "agent-parent", WorktreePath: "/tmp/p", FilesModified: []string{"p.go"}}},
		Status:      BranchStatusFailed,
		ChildrenIDs: []string{"child-1", "child-2"},
	}

	child1 := &SpeculativeBranch{
		ID:       "child-1",
		Depth:    2,
		Changes:  []ChangeRequest{{ID: "agent-child1", WorktreePath: "/tmp/c1", FilesModified: []string{"c1.go"}}},
		Status:   BranchStatusTesting,
		ParentID: "parent-branch",
	}

	child2 := &SpeculativeBranch{
		ID:       "child-2",
		Depth:    2,
		Changes:  []ChangeRequest{{ID: "agent-child2", WorktreePath: "/tmp/c2", FilesModified: []string{"c2.go"}}},
		Status:   BranchStatusTesting,
		ParentID: "parent-branch",
	}

	coord.mu.Lock()
	coord.activeBranches["parent-branch"] = parentBranch
	coord.activeBranches["child-1"] = child1
	coord.activeBranches["child-2"] = child2
	coord.mu.Unlock()

	// Kill dependent branches
	err := coord.KillDependentBranchesWithValidation(context.Background(), "parent-branch")
	require.NoError(t, err)

	// Verify notifications were sent for both children
	notifications := notifier.GetNotifications()
	assert.GreaterOrEqual(t, len(notifications), 2, "Should have sent notifications for both child branches")
}
