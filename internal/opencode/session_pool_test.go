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

func TestNewSessionPool(t *testing.T) {
	pool := NewSessionPool(30*time.Minute, 50)

	if pool.Size() != 0 {
		t.Errorf("expected empty pool, got size %d", pool.Size())
	}

	if pool.sessionTTL != 30*time.Minute {
		t.Errorf("expected TTL 30m, got %v", pool.sessionTTL)
	}

	if pool.maxTurns != 50 {
		t.Errorf("expected maxTurns 50, got %d", pool.maxTurns)
	}
}

func TestNewSessionPool_Defaults(t *testing.T) {
	pool := NewSessionPool(0, 0)

	if pool.sessionTTL != 30*time.Minute {
		t.Errorf("expected default TTL 30m, got %v", pool.sessionTTL)
	}

	if pool.maxTurns != 50 {
		t.Errorf("expected default maxTurns 50, got %d", pool.maxTurns)
	}
}

func TestGetOrCreateSessionForTask_NewSession(t *testing.T) {
	ctx := context.Background()
	pool := NewSessionPool(30*time.Minute, 50)

	sessionID, err := pool.GetOrCreateSessionForTask(ctx, "agent-1", "task-1")
	if err != nil {
		t.Fatalf("GetOrCreateSessionForTask failed: %v", err)
	}

	if sessionID != "" {
		t.Errorf("expected empty sessionID for new session, got %s", sessionID)
	}
}

func TestGetOrCreateSessionForTask_ExistingSession(t *testing.T) {
	ctx := context.Background()
	pool := NewSessionPool(30*time.Minute, 50)

	// Register a session
	testSessionID := "session-123"
	err := pool.RegisterSession("agent-1", "task-1", testSessionID)
	if err != nil {
		t.Fatalf("RegisterSession failed: %v", err)
	}

	// Try to get it
	sessionID, err := pool.GetOrCreateSessionForTask(ctx, "agent-1", "task-1")
	if err != nil {
		t.Fatalf("GetOrCreateSessionForTask failed: %v", err)
	}

	if sessionID != testSessionID {
		t.Errorf("expected sessionID %s, got %s", testSessionID, sessionID)
	}
}

func TestGetOrCreateSessionForTask_InvalidInput(t *testing.T) {
	ctx := context.Background()
	pool := NewSessionPool(30*time.Minute, 50)

	tests := []struct {
		name    string
		agentID string
		taskID  string
	}{
		{"empty agentID", "", "task-1"},
		{"empty taskID", "agent-1", ""},
		{"both empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := pool.GetOrCreateSessionForTask(ctx, tt.agentID, tt.taskID)
			if err == nil {
				t.Errorf("expected error for invalid input")
			}
		})
	}
}

func TestRegisterSession(t *testing.T) {
	pool := NewSessionPool(30*time.Minute, 50)

	err := pool.RegisterSession("agent-1", "task-1", "session-123")
	if err != nil {
		t.Fatalf("RegisterSession failed: %v", err)
	}

	if pool.Size() != 1 {
		t.Errorf("expected pool size 1, got %d", pool.Size())
	}
}

func TestRegisterSession_InvalidInput(t *testing.T) {
	pool := NewSessionPool(30*time.Minute, 50)

	tests := []struct {
		name      string
		agentID   string
		taskID    string
		sessionID string
	}{
		{"empty agentID", "", "task-1", "session-1"},
		{"empty taskID", "agent-1", "", "session-1"},
		{"empty sessionID", "agent-1", "task-1", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := pool.RegisterSession(tt.agentID, tt.taskID, tt.sessionID)
			if err == nil {
				t.Errorf("expected error for invalid input")
			}
		})
	}
}

func TestRecordTurn(t *testing.T) {
	pool := NewSessionPool(30*time.Minute, 50)

	// Register session
	pool.RegisterSession("agent-1", "task-1", "session-123")

	// Record a turn
	err := pool.RecordTurn("agent-1", "task-1")
	if err != nil {
		t.Fatalf("RecordTurn failed: %v", err)
	}

	// Check turn count
	info, _ := pool.GetSessionInfo("agent-1", "task-1")
	if info.TurnCount != 1 {
		t.Errorf("expected TurnCount 1, got %d", info.TurnCount)
	}
}

func TestRecordTurn_MultipleIncrements(t *testing.T) {
	pool := NewSessionPool(30*time.Minute, 50)
	pool.RegisterSession("agent-1", "task-1", "session-123")

	for i := 1; i <= 5; i++ {
		pool.RecordTurn("agent-1", "task-1")
	}

	info, _ := pool.GetSessionInfo("agent-1", "task-1")
	if info.TurnCount != 5 {
		t.Errorf("expected TurnCount 5, got %d", info.TurnCount)
	}
}

func TestRecordTurn_InvalidInput(t *testing.T) {
	pool := NewSessionPool(30*time.Minute, 50)

	err := pool.RecordTurn("", "task-1")
	if err == nil {
		t.Errorf("expected error for empty agentID")
	}

	err = pool.RecordTurn("agent-1", "")
	if err == nil {
		t.Errorf("expected error for empty taskID")
	}
}

func TestRecordTurn_NonexistentSession(t *testing.T) {
	pool := NewSessionPool(30*time.Minute, 50)

	err := pool.RecordTurn("agent-1", "task-1")
	if err == nil {
		t.Errorf("expected error for nonexistent session")
	}
}

func TestGetSessionInfo(t *testing.T) {
	pool := NewSessionPool(30*time.Minute, 50)

	pool.RegisterSession("agent-1", "task-1", "session-123")
	pool.RecordTurn("agent-1", "task-1")

	info, err := pool.GetSessionInfo("agent-1", "task-1")
	if err != nil {
		t.Fatalf("GetSessionInfo failed: %v", err)
	}

	if info.ID != "session-123" {
		t.Errorf("expected ID session-123, got %s", info.ID)
	}

	if info.AgentID != "agent-1" {
		t.Errorf("expected AgentID agent-1, got %s", info.AgentID)
	}

	if info.TaskID != "task-1" {
		t.Errorf("expected TaskID task-1, got %s", info.TaskID)
	}

	if info.TurnCount != 1 {
		t.Errorf("expected TurnCount 1, got %d", info.TurnCount)
	}
}

func TestGetSessionInfo_Nonexistent(t *testing.T) {
	pool := NewSessionPool(30*time.Minute, 50)

	_, err := pool.GetSessionInfo("agent-1", "task-1")
	if err == nil {
		t.Errorf("expected error for nonexistent session")
	}
}

func TestRemoveSessionForTask(t *testing.T) {
	pool := NewSessionPool(30*time.Minute, 50)

	pool.RegisterSession("agent-1", "task-1", "session-123")
	if pool.Size() != 1 {
		t.Errorf("expected pool size 1 after register")
	}

	pool.RemoveSessionForTask("agent-1", "task-1")
	if pool.Size() != 0 {
		t.Errorf("expected pool size 0 after remove")
	}
}

func TestCleanup_ExpiredSessions(t *testing.T) {
	pool := NewSessionPool(100*time.Millisecond, 50)

	// Register session
	pool.RegisterSession("agent-1", "task-1", "session-123")
	if pool.Size() != 1 {
		t.Errorf("expected pool size 1")
	}

	// Wait for expiry
	time.Sleep(150 * time.Millisecond)

	// Cleanup
	pool.Cleanup()

	if pool.Size() != 0 {
		t.Errorf("expected pool size 0 after cleanup, got %d", pool.Size())
	}
}

func TestCleanup_FreshSessions(t *testing.T) {
	pool := NewSessionPool(1*time.Hour, 50)

	pool.RegisterSession("agent-1", "task-1", "session-123")
	pool.Cleanup()

	if pool.Size() != 1 {
		t.Errorf("expected pool size 1 (fresh session not expired)")
	}
}

func TestSessionTTL_SessionExpiration(t *testing.T) {
	pool := NewSessionPool(100*time.Millisecond, 50)

	pool.RegisterSession("agent-1", "task-1", "session-123")

	// Wait for expiry
	time.Sleep(150 * time.Millisecond)

	// Try to get session - should return empty string (signal to create new)
	sessionID, _ := pool.GetOrCreateSessionForTask(context.Background(), "agent-1", "task-1")
	if sessionID != "" {
		t.Errorf("expected empty sessionID for expired session, got %s", sessionID)
	}

	// Session should be removed from pool
	if pool.Size() != 0 {
		t.Errorf("expected expired session to be removed from pool")
	}
}

func TestSessionMaxTurns_TurnsExceeded(t *testing.T) {
	pool := NewSessionPool(30*time.Minute, 2) // Very low max turns for testing

	pool.RegisterSession("agent-1", "task-1", "session-123")

	// Record turns up to max
	pool.RecordTurn("agent-1", "task-1")
	pool.RecordTurn("agent-1", "task-1")

	// Try to get session - should return empty string because max turns reached
	sessionID, _ := pool.GetOrCreateSessionForTask(context.Background(), "agent-1", "task-1")
	if sessionID != "" {
		t.Errorf("expected empty sessionID when max turns exceeded, got %s", sessionID)
	}

	// Session should be removed
	if pool.Size() != 0 {
		t.Errorf("expected session to be removed after max turns exceeded")
	}
}

func TestMultipleSessions_DifferentAgents(t *testing.T) {
	pool := NewSessionPool(30*time.Minute, 50)

	pool.RegisterSession("agent-1", "task-1", "session-1")
	pool.RegisterSession("agent-2", "task-1", "session-2")

	if pool.Size() != 2 {
		t.Errorf("expected pool size 2, got %d", pool.Size())
	}

	// Each agent should get their own session
	id1, _ := pool.GetOrCreateSessionForTask(context.Background(), "agent-1", "task-1")
	id2, _ := pool.GetOrCreateSessionForTask(context.Background(), "agent-2", "task-1")

	if id1 != "session-1" {
		t.Errorf("expected session-1 for agent-1, got %s", id1)
	}

	if id2 != "session-2" {
		t.Errorf("expected session-2 for agent-2, got %s", id2)
	}
}

func TestMultipleSessions_SameAgentDifferentTasks(t *testing.T) {
	pool := NewSessionPool(30*time.Minute, 50)

	pool.RegisterSession("agent-1", "task-1", "session-1")
	pool.RegisterSession("agent-1", "task-2", "session-2")

	if pool.Size() != 2 {
		t.Errorf("expected pool size 2, got %d", pool.Size())
	}

	// Same agent with different tasks should get different sessions
	id1, _ := pool.GetOrCreateSessionForTask(context.Background(), "agent-1", "task-1")
	id2, _ := pool.GetOrCreateSessionForTask(context.Background(), "agent-1", "task-2")

	if id1 != "session-1" {
		t.Errorf("expected session-1 for task-1, got %s", id1)
	}

	if id2 != "session-2" {
		t.Errorf("expected session-2 for task-2, got %s", id2)
	}
}

func TestConcurrentSessionAccess(t *testing.T) {
	pool := NewSessionPool(30*time.Minute, 50)

	// Register session
	pool.RegisterSession("agent-1", "task-1", "session-123")

	// Concurrent reads
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			_, _ = pool.GetSessionInfo("agent-1", "task-1")
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	if pool.Size() != 1 {
		t.Errorf("expected pool size 1 after concurrent reads")
	}
}

func TestConcurrentSessionWrites(t *testing.T) {
	pool := NewSessionPool(30*time.Minute, 50)

	pool.RegisterSession("agent-1", "task-1", "session-123")

	// Concurrent turn records
	done := make(chan bool, 5)
	for i := 0; i < 5; i++ {
		go func() {
			pool.RecordTurn("agent-1", "task-1")
			done <- true
		}()
	}

	for i := 0; i < 5; i++ {
		<-done
	}

	info, _ := pool.GetSessionInfo("agent-1", "task-1")
	if info.TurnCount != 5 {
		t.Errorf("expected TurnCount 5, got %d", info.TurnCount)
	}
}
