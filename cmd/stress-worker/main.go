// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package main

import (
	"log"
	"os/exec"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

// SimpleAgentWorkflow - minimal workflow for stress testing
func SimpleAgentWorkflow(ctx workflow.Context, agentID int, command string) (string, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Agent started", "agentID", agentID)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 60000000000, // 1 minute in nanoseconds
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var result string
	err := workflow.ExecuteActivity(ctx, RunShellCommand, command).Get(ctx, &result)

	if err != nil {
		logger.Error("Agent failed", "agentID", agentID, "error", err)
		return "", err
	}

	logger.Info("Agent completed", "agentID", agentID)
	return result, nil
}

// RunShellCommand activity
func RunShellCommand(command string) (string, error) {
	cmd := exec.Command("bash", "-c", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func main() {
	// Connect to Temporal
	c, err := client.Dial(client.Options{
		HostPort: client.DefaultHostPort,
	})
	if err != nil {
		log.Fatalln("❌ Unable to connect:", err)
	}
	defer c.Close()

	log.Println("✅ Connected to Temporal")

	// Create worker
	w := worker.New(c, "stress-test-queue", worker.Options{})

	// Register workflow and activities
	w.RegisterWorkflow(SimpleAgentWorkflow)
	w.RegisterActivity(RunShellCommand)

	log.Println("⚙️  Worker listening on task queue: stress-test-queue")

	// Start worker
	err = w.Run(worker.InterruptCh())
	if err != nil {
		log.Fatalln("❌ Worker failed:", err)
	}
}
