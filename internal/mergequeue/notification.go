// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package mergequeue

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// BranchKillNotifier sends notifications when branches are killed
type BranchKillNotifier interface {
	NotifyBranchKilled(ctx context.Context, branch *SpeculativeBranch, reason string) error
}

// AgentMailNotifier implements BranchKillNotifier using Agent Mail MCP server
type AgentMailNotifier struct {
	baseURL    string
	httpClient *http.Client
}

// NewAgentMailNotifier creates a new Agent Mail notifier
func NewAgentMailNotifier(baseURL string) *AgentMailNotifier {
	if baseURL == "" {
		baseURL = "http://localhost:8765"
	}

	return &AgentMailNotifier{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// NotifyBranchKilled sends notifications to all agents whose changes are in the killed branch
func (n *AgentMailNotifier) NotifyBranchKilled(ctx context.Context, branch *SpeculativeBranch, reason string) error {
	if branch == nil {
		return fmt.Errorf("branch cannot be nil")
	}

	// Send notification to each agent
	for _, change := range branch.Changes {
		if change.ID == "" {
			continue
		}

		if err := n.sendMessage(ctx, change.ID, branch.ID, reason, branch.Depth, len(branch.Changes)); err != nil {
			// Return error for first failure
			return fmt.Errorf("failed to notify agent %s: %w", change.ID, err)
		}
	}

	return nil
}

// sendMessage sends a single message via Agent Mail MCP
func (n *AgentMailNotifier) sendMessage(ctx context.Context, agentID, branchID, reason string, depth, changeCount int) error {
	// Prepare MCP request for Agent Mail send_message tool
	mcpRequest := map[string]interface{}{
		"method": "tools/call",
		"params": map[string]interface{}{
			"name": "send_message",
			"arguments": map[string]interface{}{
				"to":      agentID,
				"subject": fmt.Sprintf("Branch %s killed", branchID),
				"body": fmt.Sprintf(
					"Your speculative branch %s has been killed.\n\nReason: %s\n\nBranch depth: %d\nChanges included: %d",
					branchID,
					reason,
					depth,
					changeCount,
				),
				"thread": branchID,
			},
		},
	}

	jsonData, err := json.Marshal(mcpRequest)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", n.baseURL+"/mcp", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// NoOpNotifier is a notifier that does nothing (for when notifications are disabled)
type NoOpNotifier struct{}

// NotifyBranchKilled does nothing
func (n *NoOpNotifier) NotifyBranchKilled(_ context.Context, _ *SpeculativeBranch, _ string) error {
	return nil
}
