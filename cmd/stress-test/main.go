// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"go.temporal.io/sdk/client"

	"open-swarm/internal/temporal"
)

// AgentResult contains the result of an agent execution.
type AgentResult struct {
	AgentID  int
	Success  bool
	Duration time.Duration
	Error    error
	Output   string
}

const (
	defaultNumAgents        = 60
	defaultTaskType         = "shell"
	defaultSleepDuration    = 0.1
	oneHour                 = 1 * time.Hour
	reportLineWidth         = 60
	reportMinusLineWidth    = 60
	percentMultiplier       = 100.0
)

func main() {
	numAgents := flag.Int("agents", defaultNumAgents, "Number of parallel agents")
	taskType := flag.String("task", defaultTaskType, "Task type: shell, tcr")
	flag.Parse()

	log.Printf("🚀 Starting stress test with %d parallel agents", *numAgents)
	log.Printf("📋 Task type: %s", *taskType)

	// Connect to Temporal
	c, err := client.Dial(client.Options{
		HostPort: client.DefaultHostPort,
	})
	if err != nil {
		log.Fatalf("❌ Unable to connect to Temporal: %v", err)
	}
	defer c.Close()

	log.Println("✅ Connected to Temporal")

	// Track results
	results := make([]AgentResult, *numAgents)
	var wg sync.WaitGroup
	startTime := time.Now()

	// Launch all agents in parallel
	for i := 0; i < *numAgents; i++ {
		wg.Add(1)
		go func(agentID int) {
			defer wg.Done()
			results[agentID] = runAgent(c, agentID, *taskType)
		}(i)
	}

	// Wait for all agents to complete
	log.Printf("⏳ Waiting for %d agents to complete...", *numAgents)
	wg.Wait()
	totalDuration := time.Since(startTime)

	// Analyze results
	printResults(results, totalDuration)
}

func runAgent(c client.Client, agentID int, taskType string) AgentResult {
	ctx := context.Background()
	startTime := time.Now()

	workflowID := fmt.Sprintf("stress-test-agent-%d-%d", agentID, time.Now().Unix())
	workflowOptions := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: "reactor-task-queue",
	}

	var we client.WorkflowRun
	var err error

	switch taskType {
	case "shell":
		// Simple shell command task
		input := temporal.DAGWorkflowInput{
			WorkflowID: workflowID,
			Branch:     "main",
			Tasks: []temporal.Task{
				{
					Name:    fmt.Sprintf("agent-%d-task", agentID),
					Command: fmt.Sprintf("echo 'Agent %d processing...' && sleep %g && echo 'Complete'", agentID, defaultSleepDuration),
					Deps:    []string{},
				},
			},
		}
		we, err = c.ExecuteWorkflow(ctx, workflowOptions, temporal.TddDagWorkflow, input)

	case "tcr":
		// TCR workflow task
		input := temporal.TCRWorkflowInput{
			CellID:      fmt.Sprintf("cell-%d", agentID),
			Branch:      "main",
			TaskID:      fmt.Sprintf("stress-%d", agentID),
			Description: fmt.Sprintf("Stress test agent %d", agentID),
			Prompt:      fmt.Sprintf("Agent %d: Analyze codebase structure", agentID),
		}
		we, err = c.ExecuteWorkflow(ctx, workflowOptions, temporal.TCRWorkflow, input)

	default:
		return AgentResult{
			AgentID:  agentID,
			Success:  false,
			Duration: time.Since(startTime),
			Error:    fmt.Errorf("unknown task type: %s", taskType),
		}
	}

	if err != nil {
		return AgentResult{
			AgentID:  agentID,
			Success:  false,
			Duration: time.Since(startTime),
			Error:    fmt.Errorf("failed to start workflow: %w", err),
		}
	}

	// Wait for workflow completion
	var result interface{}
	err = we.Get(ctx, &result)
	duration := time.Since(startTime)

	if err != nil {
		return AgentResult{
			AgentID:  agentID,
			Success:  false,
			Duration: duration,
			Error:    fmt.Errorf("workflow failed: %w", err),
		}
	}

	return AgentResult{
		AgentID:  agentID,
		Success:  true,
		Duration: duration,
		Output:   fmt.Sprintf("%v", result),
	}
}

func printResults(results []AgentResult, totalDuration time.Duration) {
	successCount := 0
	failCount := 0
	var totalAgentTime time.Duration
	var minDuration time.Duration = oneHour
	var maxDuration time.Duration

	for i, r := range results {
		if r.Success {
			successCount++
		} else {
			failCount++
			log.Printf("❌ Agent %d failed: %v", i, r.Error)
		}

		totalAgentTime += r.Duration
		if r.Duration < minDuration {
			minDuration = r.Duration
		}
		if r.Duration > maxDuration {
			maxDuration = r.Duration
		}
	}

	avgDuration := totalAgentTime / time.Duration(len(results))

	fmt.Println("\n" + strings.Repeat("═", reportLineWidth))
	fmt.Println("📊 STRESS TEST RESULTS")
	fmt.Println(strings.Repeat("═", reportLineWidth))
	fmt.Printf("Total agents:        %d\n", len(results))
	fmt.Printf("✅ Successful:       %d (%.1f%%)\n", successCount, float64(successCount)/float64(len(results))*percentMultiplier)
	fmt.Printf("❌ Failed:           %d (%.1f%%)\n", failCount, float64(failCount)/float64(len(results))*percentMultiplier)
	fmt.Println(strings.Repeat("─", reportMinusLineWidth))
	fmt.Printf("⏱️  Total wall time:  %v\n", totalDuration)
	fmt.Printf("📈 Avg agent time:   %v\n", avgDuration)
	fmt.Printf("⚡ Min agent time:   %v\n", minDuration)
	fmt.Printf("🐌 Max agent time:   %v\n", maxDuration)
	fmt.Printf("🔥 Throughput:       %.2f agents/sec\n", float64(len(results))/totalDuration.Seconds())
	fmt.Println(strings.Repeat("═", reportLineWidth) + "\n")

	if successCount == len(results) {
		fmt.Println("🎉 ALL AGENTS COMPLETED SUCCESSFULLY!")
	} else {
		fmt.Printf("⚠️  %d agents failed - check logs above\n", failCount)
	}
}
