// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package mergequeue

import (
	"context"
	"sync"
)

// Global registry for Branch Kill Notifiers (used for dependency injection)
// This follows the same pattern as the Temporal client registry
var (
	notifierRegistry = make(map[*Coordinator]BranchKillNotifier)
	notifierMu       sync.RWMutex
)

// SetNotifier sets the branch kill notifier for sending notifications.
// This allows dependency injection for testing and configuration.
func (c *Coordinator) SetNotifier(notifier BranchKillNotifier) {
	notifierMu.Lock()
	defer notifierMu.Unlock()
	notifierRegistry[c] = notifier
}

// getNotifier retrieves the branch kill notifier if configured.
func (c *Coordinator) getNotifier() BranchKillNotifier {
	notifierMu.RLock()
	defer notifierMu.RUnlock()
	return notifierRegistry[c]
}

// removeNotifier removes the branch kill notifier from the registry.
// Should be called when coordinator is stopped.
func (c *Coordinator) removeNotifier() {
	notifierMu.Lock()
	defer notifierMu.Unlock()
	delete(notifierRegistry, c)
}

// notifyBranchKilled sends a notification about a killed branch.
// This is called after a branch is marked as killed.
// Returns nil if no notifier is configured (graceful degradation).
func (c *Coordinator) notifyBranchKilled(ctx context.Context, branch *SpeculativeBranch, reason string) error {
	notifier := c.getNotifier()
	if notifier == nil {
		// No notifier configured, skip notification
		return nil
	}

	// Send notification (non-blocking - errors are logged but don't fail the kill)
	if err := notifier.NotifyBranchKilled(ctx, branch, reason); err != nil {
		// TODO: Log error
		// For now, we continue - notification failures shouldn't block kills
		_ = err
	}

	return nil
}
