// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package temporal

import (
	"context"
	"fmt"

	"github.com/bitfield/script"
	"go.temporal.io/sdk/activity"
)

// ShellActivities handles shell command execution using bitfield/script
// This provides clean, elegant shell operations without os/exec verbosity
type ShellActivities struct{}

// RunScript executes a shell command using bitfield/script
// Heartbeats are recorded for long-running commands to enable cancellation
func (sa *ShellActivities) RunScript(ctx context.Context, command string) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Executing shell command", "cmd", command)

	// Heartbeat for long-running commands
	activity.RecordHeartbeat(ctx, "executing")

	// Execute with script (handles pipes elegantly)
	p := script.Exec(command)

	output, err := p.String()
	if err != nil {
		logger.Error("Command failed", "error", err, "output", output)
		return output, fmt.Errorf("shell command failed: %w", err)
	}

	logger.Info("Command succeeded", "output", output)
	return output, nil
}

// RunScriptInDir executes a shell command in a specific directory
// The directory is changed before executing the command
func (sa *ShellActivities) RunScriptInDir(ctx context.Context, dir, command string) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Executing shell command", "dir", dir, "cmd", command)

	activity.RecordHeartbeat(ctx, "executing")

	// Change to directory and execute
	p := script.Exec("cd " + dir + " && " + command)

	output, err := p.String()
	if err != nil {
		logger.Error("Command failed", "error", err, "output", output)
		return output, fmt.Errorf("shell command failed: %w", err)
	}

	return output, nil
}
