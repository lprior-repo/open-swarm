package dag

import (
	"context"
	"fmt"
	"github.com/bitfield/script"
	"go.temporal.io/sdk/activity"
)

type ShellActivities struct{}

// RunDAGScript executes a shell command using bitfield/script
func (sa *ShellActivities) RunDAGScript(ctx context.Context, command string) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Executing shell command", "cmd", command)

	activity.RecordHeartbeat(ctx, "executing")

	p := script.Exec(command)
	output, err := p.String()
	if err != nil {
		logger.Error("Command failed", "error", err, "output", output)
		return output, fmt.Errorf("shell command failed: %w", err)
	}

	logger.Info("Command succeeded", "output", output)
	return output, nil
}
