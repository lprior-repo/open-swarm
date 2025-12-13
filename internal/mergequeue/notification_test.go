// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package mergequeue

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAgentMailNotifier(t *testing.T) {
	tests := []struct {
		name        string
		baseURL     string
		expectedURL string
	}{
		{
			name:        "custom base URL",
			baseURL:     "http://example.com:9000",
			expectedURL: "http://example.com:9000",
		},
		{
			name:        "empty base URL defaults to localhost",
			baseURL:     "",
			expectedURL: "http://localhost:8765",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notifier := NewAgentMailNotifier(tt.baseURL)
			require.NotNil(t, notifier)
			assert.Equal(t, tt.expectedURL, notifier.baseURL)
			assert.NotNil(t, notifier.httpClient)
		})
	}
}

func TestAgentMailNotifier_NotifyBranchKilled_Success(t *testing.T) {
	messageCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/mcp", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		messageCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := NewAgentMailNotifier(server.URL)
	branch := &SpeculativeBranch{
		ID:    "test-branch-1",
		Depth: 2,
		Changes: []ChangeRequest{
			{
				ID:            "agent-123",
				WorktreePath:  "/tmp/worktree1",
				FilesModified: []string{"file1.go"},
			},
			{
				ID:            "agent-456",
				WorktreePath:  "/tmp/worktree2",
				FilesModified: []string{"file2.go"},
			},
		},
		Status: BranchStatusTesting,
	}

	err := notifier.NotifyBranchKilled(context.Background(), branch, "tests failed")
	require.NoError(t, err)
	assert.Equal(t, 2, messageCount)
}

func TestAgentMailNotifier_NotifyBranchKilled_NilBranch(t *testing.T) {
	notifier := NewAgentMailNotifier("http://localhost:8765")
	err := notifier.NotifyBranchKilled(context.Background(), nil, "tests failed")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "branch cannot be nil")
}

func TestAgentMailNotifier_NotifyBranchKilled_EmptyChanges(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not send messages for empty changes")
	}))
	defer server.Close()

	notifier := NewAgentMailNotifier(server.URL)
	branch := &SpeculativeBranch{
		ID:      "test-branch-1",
		Depth:   1,
		Changes: []ChangeRequest{},
		Status:  BranchStatusTesting,
	}

	err := notifier.NotifyBranchKilled(context.Background(), branch, "tests failed")
	require.NoError(t, err)
}

func TestAgentMailNotifier_NotifyBranchKilled_SkipEmptyAgentID(t *testing.T) {
	messageCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		messageCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := NewAgentMailNotifier(server.URL)
	branch := &SpeculativeBranch{
		ID:    "test-branch-1",
		Depth: 1,
		Changes: []ChangeRequest{
			{
				ID:            "agent-123",
				WorktreePath:  "/tmp/worktree1",
				FilesModified: []string{"file1.go"},
			},
			{
				ID:            "", // Empty agent ID should be skipped
				WorktreePath:  "/tmp/worktree2",
				FilesModified: []string{"file2.go"},
			},
		},
		Status: BranchStatusTesting,
	}

	err := notifier.NotifyBranchKilled(context.Background(), branch, "tests failed")
	require.NoError(t, err)

	// Should have sent only one message (for agent-123)
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, 1, messageCount)
}

func TestAgentMailNotifier_NotifyBranchKilled_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	notifier := NewAgentMailNotifier(server.URL)
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

	err := notifier.NotifyBranchKilled(context.Background(), branch, "tests failed")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to notify agent")
}

func TestAgentMailNotifier_NotifyBranchKilled_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := NewAgentMailNotifier(server.URL)
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

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := notifier.NotifyBranchKilled(ctx, branch, "tests failed")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}

func TestAgentMailNotifier_NotifyBranchKilled_MultipleAgents(t *testing.T) {
	messageCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		messageCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := NewAgentMailNotifier(server.URL)
	branch := &SpeculativeBranch{
		ID:    "test-branch-1",
		Depth: 2,
		Changes: []ChangeRequest{
			{ID: "agent-1", WorktreePath: "/tmp/w1", FilesModified: []string{"a.go"}},
			{ID: "agent-2", WorktreePath: "/tmp/w2", FilesModified: []string{"b.go"}},
			{ID: "agent-3", WorktreePath: "/tmp/w3", FilesModified: []string{"c.go"}},
		},
		Status: BranchStatusTesting,
	}

	err := notifier.NotifyBranchKilled(context.Background(), branch, "tests failed")
	require.NoError(t, err)
	assert.Equal(t, 3, messageCount)
}

func TestNoOpNotifier_NotifyBranchKilled(t *testing.T) {
	notifier := &NoOpNotifier{}
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

	// Should not error
	err := notifier.NotifyBranchKilled(context.Background(), branch, "tests failed")
	require.NoError(t, err)
}
