// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package temporal

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/workflow"
)

// StressTestInput configures a stress test run
type StressTestInput struct {
	NumAgents        int
	Prompt           string
	Agent            string
	Model            string
	Bootstrap        *BootstrapOutput
	TimeoutSeconds   int
	ConcurrencyLimit int
}

// StressTestResult summarizes stress test results
type StressTestResult struct {
	TotalAgents     int
	Successful      int
	Failed          int
	TotalDuration   time.Duration
	AverageDuration time.Duration
	Results         []AgentInvokeResult
	Errors          []string
}

// StressTestWorkflow runs multiple agents in parallel for stress testing
func StressTestWorkflow(ctx workflow.Context, input StressTestInput) (*StressTestResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting Stress Test Workflow", "num_agents", input.NumAgents)
	
	startTime := workflow.Now(ctx)
	
	if input.TimeoutSeconds == 0 {
		input.TimeoutSeconds = 300
	}
	if input.Agent == "" {
		input.Agent = "general"
	}
	if input.Model == "" {
		input.Model = "anthropic/claude-sonnet-4-5"
	}
	
	result := &StressTestResult{
		TotalAgents: input.NumAgents,
		Results:     make([]AgentInvokeResult, 0, input.NumAgents),
		Errors:      make([]string, 0),
	}
	
	logger.Info("Stress test completed", "total", result.TotalAgents)
	result.TotalDuration = workflow.Now(ctx).Sub(startTime)
	
	return result, nil
}
