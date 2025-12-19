// Run Enhanced TCR Workflow with real output
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"go.temporal.io/sdk/client"

	"open-swarm/internal/temporal"
)

func main() {
	// Parse command line flags
	taskID := flag.String("task", "demo-task-001", "Task ID to process")
	cellID := flag.String("cell", "cell-001", "Cell ID")
	branch := flag.String("branch", "main", "Git branch")
	description := flag.String("desc", "Demo implementation task", "Task description")
	criteria := flag.String("criteria", "Should pass all gates", "Acceptance criteria")
	maxRetries := flag.Int("retries", 2, "Max regeneration attempts")
	maxFixes := flag.Int("fixes", 5, "Max fix attempts per regeneration")
	reviewers := flag.Int("reviewers", 2, "Number of reviewers")
	flag.Parse()

	// Connect to Temporal server
	c, err := client.Dial(client.Options{
		HostPort: client.DefaultHostPort, // localhost:7233
	})
	if err != nil {
		log.Fatalln("❌ Unable to connect to Temporal server:", err)
	}
	defer c.Close()

	fmt.Println("\n" + "="*80)
	fmt.Println("🚀 Enhanced TCR Workflow - Real Execution")
	fmt.Println("="*80)
	fmt.Printf("Task ID:           %s\n", *taskID)
	fmt.Printf("Cell ID:           %s\n", *cellID)
	fmt.Printf("Branch:            %s\n", *branch)
	fmt.Printf("Description:       %s\n", *description)
	fmt.Printf("Max Retries:       %d\n", *maxRetries)
	fmt.Printf("Max Fix Attempts:  %d\n", *maxFixes)
	fmt.Printf("Reviewers:         %d\n", *reviewers)
	fmt.Println("="*80 + "\n")

	// Prepare workflow input
	input := temporal.EnhancedTCRInput{
		TaskID:             *taskID,
		CellID:             *cellID,
		Branch:             *branch,
		Description:        *description,
		AcceptanceCriteria: *criteria,
		MaxRetries:         *maxRetries,
		MaxFixAttempts:     *maxFixes,
		ReviewersCount:     *reviewers,
	}

	// Start workflow
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	fmt.Println("📋 Starting workflow execution...")
	workflowRun, err := c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID:        *taskID,
		TaskQueue: "reactor-task-queue",
	}, temporal.EnhancedTCRWorkflow, input)
	if err != nil {
		log.Fatalln("❌ Failed to start workflow:", err)
	}

	fmt.Printf("✅ Workflow started with ID: %s\n\n", workflowRun.GetID())

	// Wait for completion and get result
	fmt.Println("⏳ Waiting for workflow completion...")
	var result *temporal.EnhancedTCRResult
	err = workflowRun.Get(ctx, &result)
	if err != nil {
		log.Fatalln("❌ Workflow failed:", err)
	}

	// Display results
	fmt.Println("\n" + "="*80)
	fmt.Println("📊 Workflow Results")
	fmt.Println("="*80)
	fmt.Printf("Status:          %v\n", result.Success)
	fmt.Printf("Error:           %s\n", result.Error)
	fmt.Printf("Files Changed:   %d files\n", len(result.FilesChanged))
	fmt.Printf("Gate Results:    %d gates executed\n\n", len(result.GateResults))

	// Display gate-by-gate results
	fmt.Println("Gate Execution Details:")
	fmt.Println("-" * 80)
	for i, gate := range result.GateResults {
		status := "✅ PASS"
		if !gate.Passed {
			status = "❌ FAIL"
		}
		fmt.Printf("%d. %s %s (%.3fs)\n", i+1, gate.GateName, status, gate.Duration.Seconds())
		if !gate.Passed && gate.Error != "" {
			fmt.Printf("   Error: %s\n", gate.Error)
		}
	}

	fmt.Println("="*80)
	if result.Success {
		fmt.Println("🎉 Workflow completed successfully!")
	} else {
		fmt.Println("⚠️  Workflow failed - check errors above")
	}
	fmt.Println("="*80 + "\n")
}
