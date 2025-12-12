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

	"go.temporal.io/sdk/client"

	"open-swarm/internal/temporal"
)

func main() {
	// Parse flags
	workflowType := flag.String("workflow", "tcr", "Workflow type: tcr or dag")
	taskID := flag.String("task", "", "Task ID")
	prompt := flag.String("prompt", "", "Task prompt")
	desc := flag.String("desc", "", "Task description")
	branch := flag.String("branch", "main", "Git branch")
	flag.Parse()

	if *taskID == "" || *prompt == "" {
		log.Fatal("❌ --task and --prompt are required")
	}

	// Connect to Temporal
	c, err := client.Dial(client.Options{
		HostPort: client.DefaultHostPort,
	})
	if err != nil {
		log.Fatalln("❌ Unable to connect:", err)
	}
	defer c.Close()

	// Submit workflow
	workflowID := fmt.Sprintf("reactor-%s", *taskID)
	workflowOptions := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: "reactor-task-queue",
	}

	var we client.WorkflowRun

	switch *workflowType {
	case "tcr":
		input := temporal.TCRWorkflowInput{
			CellID:      "primary",
			Branch:      *branch,
			TaskID:      *taskID,
			Description: *desc,
			Prompt:      *prompt,
		}
		we, err = c.ExecuteWorkflow(context.Background(), workflowOptions, temporal.TCRWorkflow, input)

	case "dag":
		// Example: Parse tasks from JSON file or flags
		log.Fatal("❌ DAG workflow not implemented in CLI yet")

	default:
		log.Fatalf("❌ Unknown workflow type: %s", *workflowType)
	}

	if err != nil {
		log.Fatalln("❌ Failed to start workflow:", err)
	}

	log.Printf("✅ Workflow started")
	log.Printf("   ID: %s", we.GetID())
	log.Printf("   RunID: %s", we.GetRunID())
	log.Printf("   Web UI: http://localhost:8233/namespaces/default/workflows/%s", workflowID)

	// Wait for result
	log.Println("⏳ Waiting for workflow to complete...")

	var result temporal.TCRWorkflowResult
	err = we.Get(context.Background(), &result)
	if err != nil {
		log.Fatalln("❌ Workflow failed:", err)
	}

	if result.Success {
		log.Println("✅ Workflow succeeded!")
		log.Printf("   Tests: PASSED")
		log.Printf("   Files changed: %v", result.FilesChanged)
	} else {
		log.Println("❌ Workflow failed")
		log.Printf("   Error: %s", result.Error)
	}
}
