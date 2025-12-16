// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"go.temporal.io/sdk/client"

	"open-swarm/internal/temporal"
)

func main() {
	// Command line flags
	numAgents := flag.Int("agents", 100, "Number of agents to spawn")
	prompt := flag.String("prompt", "Write a hello world function in Go", "Prompt to send to each agent")
	agentType := flag.String("agent", "general", "Agent type (build, plan, general)")
	model := flag.String("model", "anthropic/claude-sonnet-4-5", "Model to use")
	concurrency := flag.Int("concurrency", 10, "Concurrency limit (0 for unlimited)")
	timeout := flag.Int("timeout", 300, "Timeout in seconds per agent")
	repoPath := flag.String("repo", ".", "Repository path for bootstrap")
	branch := flag.String("branch", "main", "Git branch")

	flag.Parse()

	log.Printf("🧪 Stress Test Configuration:")
	log.Printf("   Agents: %d", *numAgents)
	log.Printf("   Prompt: %s", *prompt)
	log.Printf("   Agent Type: %s", *agentType)
	log.Printf("   Model: %s", *model)
	log.Printf("   Concurrency: %d", *concurrency)
	log.Printf("   Timeout: %ds", *timeout)
	log.Printf("   Repository: %s", *repoPath)
	log.Printf("   Branch: %s", *branch)

	// Connect to Temporal
	c, err := client.Dial(client.Options{
		HostPort: client.DefaultHostPort,
	})
	if err != nil {
		log.Fatalf("❌ Unable to create Temporal client: %v", err)
	}
	defer c.Close()

	log.Println("✅ Connected to Temporal server")

	// Create bootstrap output (simplified - in production you'd bootstrap properly)
	bootstrap := &temporal.BootstrapOutput{
		CellID:       fmt.Sprintf("stress-test-%d", time.Now().Unix()),
		Port:         8000,
		WorktreeID:   fmt.Sprintf("stress-test-%d", time.Now().Unix()),
		WorktreePath: *repoPath,
		BaseURL:      "http://localhost:8000",
		ServerPID:    0,
	}

	// Create stress test input
	input := temporal.StressTestInput{
		NumAgents:        *numAgents,
		Prompt:           *prompt,
		Agent:            *agentType,
		Model:            *model,
		Bootstrap:        bootstrap,
		TimeoutSeconds:   *timeout,
		ConcurrencyLimit: *concurrency,
	}

	// Start workflow
	workflowID := fmt.Sprintf("stress-test-%d", time.Now().Unix())
	workflowOptions := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: "reactor-task-queue",
	}

	log.Printf("🚀 Starting stress test workflow: %s", workflowID)
	startTime := time.Now()

	we, err := c.ExecuteWorkflow(context.Background(), workflowOptions, temporal.StressTestWorkflow, input)
	if err != nil {
		log.Fatalf("❌ Unable to execute workflow: %v", err)
	}

	log.Printf("⏳ Workflow started: %s", we.GetID())
	log.Printf("   Run ID: %s", we.GetRunID())
	log.Println("   Waiting for completion...")

	// Wait for workflow to complete
	var result temporal.StressTestResult
	err = we.Get(context.Background(), &result)
	if err != nil {
		log.Fatalf("❌ Workflow execution failed: %v", err)
	}

	duration := time.Since(startTime)

	// Print results
	separator := strings.Repeat("=", 80)
	log.Println("\n" + separator)
	log.Println("📊 STRESS TEST RESULTS")
	log.Println(separator)
	log.Printf("Total Agents:       %d", result.TotalAgents)
	log.Printf("Successful:         %d (%.1f%%)", result.Successful, float64(result.Successful)/float64(result.TotalAgents)*100)
	log.Printf("Failed:             %d (%.1f%%)", result.Failed, float64(result.Failed)/float64(result.TotalAgents)*100)
	log.Printf("Total Duration:     %v", result.TotalDuration)
	log.Printf("Average Duration:   %v", result.AverageDuration)
	log.Printf("Wall Clock Time:    %v", duration)
	log.Println(separator)

	if len(result.Errors) > 0 {
		log.Printf("\n⚠️  Errors (%d):", len(result.Errors))
		for i, errMsg := range result.Errors {
			if i < 10 { // Limit to first 10 errors
				log.Printf("   %d. %s", i+1, errMsg)
			}
		}
		if len(result.Errors) > 10 {
			log.Printf("   ... and %d more errors", len(result.Errors)-10)
		}
	}

	// Print detailed results as JSON
	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err == nil {
		log.Printf("\n📄 Full Results (JSON):\n%s\n", string(resultJSON))
	}

	if result.Failed > 0 {
		log.Println("\n❌ Stress test completed with failures")
	} else {
		log.Println("\n✅ Stress test completed successfully!")
	}
}
