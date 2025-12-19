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

// SessionContext represents a reusable session tied to an agent and task.
// It tracks session metadata and lifecycle information.
type SessionContext struct {
	ID        string    // OpenCode session ID
	Created   time.Time // When session was created
	LastUsed  time.Time // Last time session was used
	AgentID   string    // Agent using this session
	TaskID    string    // Task this session is for
	TurnCount int       // Number of turns executed
}

// SessionPool manages reusable sessions for agents.
// Instead of creating a new session for each prompt, sessions are reused
// within the same agent+task combination, improving context retention.
type SessionPool struct {
	mu       sync.RWMutex
	sessions map[string]*SessionContext

	// Configuration
	sessionTTL time.Duration // How long to keep idle sessions
	maxTurns   int           // Max turns per session before reset
}

// NewSessionPool creates a new session pool with default configuration.
// sessionTTL: how long to keep idle sessions (default: 30 minutes)
// maxTurns: maximum turns per session before creating new one (default: 50)
func NewSessionPool(sessionTTL time.Duration, maxTurns int) *SessionPool {
	if sessionTTL <= 0 {
		sessionTTL = 30 * time.Minute
	}
	if maxTurns <= 0 {
		maxTurns = 50
	}

	return &SessionPool{
		sessions:   make(map[string]*SessionContext),
		sessionTTL: sessionTTL,
		maxTurns:   maxTurns,
	}
}

// GetOrCreateSessionForTask returns an existing session for this agent+task,
// or creates a new one if needed. This enables context retention across
// multiple prompts within the same logical task.
func (p *SessionPool) GetOrCreateSessionForTask(
	ctx context.Context,
	agentID string,
	taskID string,
) (string, error) {
	if agentID == "" {
		return "", fmt.Errorf("agentID is required")
	}
	if taskID == "" {
		return "", fmt.Errorf("taskID is required")
	}

	sessionKey := fmt.Sprintf("%s:%s", agentID, taskID)

	p.mu.RLock()
	session, exists := p.sessions[sessionKey]
	p.mu.RUnlock()

	// Return existing session if still valid
	if exists {
		// Check if session hasn't expired
		if time.Since(session.LastUsed) < p.sessionTTL {
			// Check if we haven't exceeded max turns
			if session.TurnCount < p.maxTurns {
				p.updateLastUsed(sessionKey)
				return session.ID, nil
			}
		}
		// Session is stale or has too many turns - remove it
		p.removeSession(sessionKey)
	}

	// Create new session (caller will get actual session ID from OpenCode)
	// For now, return empty string to signal caller should create new session
	// The caller will then call RegisterSession() with the actual ID
	return "", nil
}

// RegisterSession records a newly created session in the pool.
// This should be called after creating a session through the OpenCode SDK.
func (p *SessionPool) RegisterSession(
	agentID string,
	taskID string,
	sessionID string,
) error {
	if agentID == "" {
		return fmt.Errorf("agentID is required")
	}
	if taskID == "" {
		return fmt.Errorf("taskID is required")
	}
	if sessionID == "" {
		return fmt.Errorf("sessionID is required")
	}

	sessionKey := fmt.Sprintf("%s:%s", agentID, taskID)

	p.mu.Lock()
	defer p.mu.Unlock()

	p.sessions[sessionKey] = &SessionContext{
		ID:       sessionID,
		Created:  time.Now(),
		LastUsed: time.Now(),
		AgentID:  agentID,
		TaskID:   taskID,
		TurnCount: 0,
	}

	return nil
}

// RecordTurn increments the turn count for a session.
// This helps track how many operations have been performed and know when to reset.
func (p *SessionPool) RecordTurn(agentID, taskID string) error {
	if agentID == "" || taskID == "" {
		return fmt.Errorf("agentID and taskID are required")
	}

	sessionKey := fmt.Sprintf("%s:%s", agentID, taskID)

	p.mu.Lock()
	defer p.mu.Unlock()

	session, exists := p.sessions[sessionKey]
	if !exists {
		return fmt.Errorf("session not found for %s", sessionKey)
	}

	session.TurnCount++
	session.LastUsed = time.Now()

	return nil
}

// GetSessionInfo returns information about a session (for debugging/monitoring).
func (p *SessionPool) GetSessionInfo(agentID, taskID string) (*SessionContext, error) {
	if agentID == "" || taskID == "" {
		return nil, fmt.Errorf("agentID and taskID are required")
	}

	sessionKey := fmt.Sprintf("%s:%s", agentID, taskID)

	p.mu.RLock()
	defer p.mu.RUnlock()

	session, exists := p.sessions[sessionKey]
	if !exists {
		return nil, fmt.Errorf("session not found for %s", sessionKey)
	}

	// Return a copy to avoid external modification
	copy := *session
	return &copy, nil
}

// RemoveSessionForTask explicitly removes a session when a task is complete.
func (p *SessionPool) RemoveSessionForTask(agentID, taskID string) {
	sessionKey := fmt.Sprintf("%s:%s", agentID, taskID)
	p.removeSession(sessionKey)
}

// Cleanup removes all expired sessions. Should be called periodically.
func (p *SessionPool) Cleanup() {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	for key, session := range p.sessions {
		if now.Sub(session.LastUsed) > p.sessionTTL {
			delete(p.sessions, key)
		}
	}
}

// Size returns the number of active sessions in the pool.
func (p *SessionPool) Size() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.sessions)
}

// Private helpers

func (p *SessionPool) updateLastUsed(sessionKey string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if session, exists := p.sessions[sessionKey]; exists {
		session.LastUsed = time.Now()
	}
}

func (p *SessionPool) removeSession(sessionKey string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.sessions, sessionKey)
}
