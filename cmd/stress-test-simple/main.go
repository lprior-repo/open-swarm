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
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type AgentResult struct {
	AgentID  int
	Success  bool
	Duration time.Duration
	Error    error
	Output   string
}

// SimpleAgentWorkflow - minimal workflow for stress testing
func SimpleAgentWorkflow(ctx workflow.Context, agentID int, command string) (string, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Agent started", "agentID", agentID)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 1 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    1 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var result string
	err := workflow.ExecuteActivity(ctx, "RunShellCommand", command).Get(ctx, &result)

	if err != nil {
		logger.Error("Agent failed", "agentID", agentID, "error", err)
		return "", fmt.Errorf("agent %d failed: %w", agentID, err)
	}

	logger.Info("Agent completed", "agentID", agentID)
	return result, nil
}

func main() {
	numAgents := flag.Int("agents", 60, "Number of parallel agents")
	flag.Parse()

	log.Printf("🚀 Starting stress test with %d parallel agents", *numAgents)

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
	log.Printf("⏳ Launching %d agents...", *numAgents)
	for i := 0; i < *numAgents; i++ {
		wg.Add(1)
		go func(agentID int) {
			defer wg.Done()
			results[agentID] = runAgent(c, agentID)
		}(i)
	}

	// Wait for all agents to complete
	wg.Wait()
	totalDuration := time.Since(startTime)

	// Analyze results
	printResults(results, totalDuration)
}

func runAgent(c client.Client, agentID int) AgentResult {
	ctx := context.Background()
	startTime := time.Now()

	workflowID := fmt.Sprintf("stress-agent-%d-%d", agentID, time.Now().UnixNano())
	command := fmt.Sprintf("echo 'Agent %d: Processing task...' && sleep 0.1 && echo 'Agent %d: Complete!'", agentID, agentID)

	workflowOptions := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: "stress-test-queue",
	}

	we, err := c.ExecuteWorkflow(ctx, workflowOptions, "SimpleAgentWorkflow", agentID, command)
	if err != nil {
		return AgentResult{
			AgentID:  agentID,
			Success:  false,
			Duration: time.Since(startTime),
			Error:    fmt.Errorf("failed to start workflow: %w", err),
		}
	}

	// Wait for workflow completion
	var result string
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
		Output:   result,
	}
}

func printResults(results []AgentResult, totalDuration time.Duration) {
	successCount := 0
	failCount := 0
	var totalAgentTime time.Duration
	var minDuration time.Duration = time.Hour
	var maxDuration time.Duration

	fmt.Println("\n" + strings.Repeat("─", 60))
	fmt.Println("📊 INDIVIDUAL AGENT RESULTS:")
	fmt.Println(strings.Repeat("─", 60))

	for i, r := range results {
		if r.Success {
			successCount++
			fmt.Printf("✅ Agent %2d: %v\n", i, r.Duration)
		} else {
			failCount++
			fmt.Printf("❌ Agent %2d: %v - %v\n", i, r.Duration, r.Error)
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

	fmt.Println("\n" + strings.Repeat("═", 60))
	fmt.Println("📊 STRESS TEST SUMMARY")
	fmt.Println(strings.Repeat("═", 60))
	fmt.Printf("Total agents:        %d\n", len(results))
	fmt.Printf("✅ Successful:       %d (%.1f%%)\n", successCount, float64(successCount)/float64(len(results))*100)
	fmt.Printf("❌ Failed:           %d (%.1f%%)\n", failCount, float64(failCount)/float64(len(results))*100)
	fmt.Println(strings.Repeat("─", 60))
	fmt.Printf("⏱️  Total wall time:  %v\n", totalDuration)
	fmt.Printf("📈 Avg agent time:   %v\n", avgDuration)
	fmt.Printf("⚡ Min agent time:   %v\n", minDuration)
	fmt.Printf("🐌 Max agent time:   %v\n", maxDuration)
	fmt.Printf("🔥 Throughput:       %.2f agents/sec\n", float64(len(results))/totalDuration.Seconds())
	fmt.Printf("⚙️  Parallelism:      %.1fx (%.0f%% parallel efficiency)\n",
		totalAgentTime.Seconds()/totalDuration.Seconds(),
		(totalAgentTime.Seconds()/totalDuration.Seconds())/float64(len(results))*100)
	fmt.Println(strings.Repeat("═", 60) + "\n")

	if successCount == len(results) {
		fmt.Println("🎉 ALL AGENTS COMPLETED SUCCESSFULLY!")
		fmt.Println("✨ System handled 60 parallel agents without issues!")
	} else {
		fmt.Printf("⚠️  %d agents failed - check logs above\n", failCount)
	}
}
