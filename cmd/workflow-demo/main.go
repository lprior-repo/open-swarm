// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.temporal.io/sdk/client"

	"open-swarm/internal/temporal"
)

func main() {
	// Connect to Temporal
	c, err := client.Dial(client.Options{
		HostPort: "localhost:7233",
	})
	if err != nil {
		log.Fatalln("❌ Unable to connect to Temporal:", err)
	}
	defer c.Close()

	log.Println("✅ Connected to Temporal server")
	log.Println("🌐 Open Temporal UI: http://localhost:8233")
	log.Println("")

	// Demo 1: Simple DAG Workflow
	log.Println("📊 Starting Demo DAG Workflow...")
	workflowID := fmt.Sprintf("demo-dag-%d", time.Now().Unix())

	tasks := []temporal.Task{
		{Name: "prepare", Command: "echo '🔧 Preparing workspace...' && sleep 2", Deps: []string{}},
		{Name: "build-frontend", Command: "echo '🎨 Building frontend...' && sleep 3", Deps: []string{"prepare"}},
		{Name: "build-backend", Command: "echo '⚙️  Building backend...' && sleep 3", Deps: []string{"prepare"}},
		{Name: "run-tests", Command: "echo '🧪 Running tests...' && sleep 2", Deps: []string{"build-frontend", "build-backend"}},
		{Name: "deploy", Command: "echo '🚀 Deploying...' && sleep 2", Deps: []string{"run-tests"}},
	}

	input := temporal.DAGWorkflowInput{
		WorkflowID: workflowID,
		Branch:     "main",
		Tasks:      tasks,
	}

	workflowOptions := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: "open-swarm-demo",
	}

	we, err := c.ExecuteWorkflow(context.Background(), workflowOptions, temporal.TddDagWorkflow, input)
	if err != nil {
		log.Fatalf("❌ Failed to start workflow: %v", err)
	}

	log.Printf("✅ Workflow started!")
	log.Printf("   ID: %s", we.GetID())
	log.Printf("   RunID: %s", we.GetRunID())
	log.Printf("")
	log.Printf("👀 Watch it live:")
	log.Printf("   1. Open: http://localhost:8233")
	log.Printf("   2. Click 'Workflows' → Search for: %s", workflowID)
	log.Printf("   3. Watch the DAG execute in real-time!")
	log.Printf("")
	log.Printf("⏳ Waiting for workflow to complete (or press Ctrl+C)...")
	log.Printf("")

	// Wait for completion (with timeout)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// The workflow will wait for signal since it's TDD mode
	// For demo purposes, let's send the signal after a delay
	go func() {
		time.Sleep(15 * time.Second)
		log.Println("")
		log.Println("📢 Sending 'FixApplied' signal to complete workflow...")
		err := c.SignalWorkflow(ctx, workflowID, "", "FixApplied", "Demo completed successfully")
		if err != nil {
			log.Printf("⚠️  Failed to send signal: %v", err)
		}
	}()

	err = we.Get(ctx, nil)
	if err != nil {
		log.Printf("⚠️  Workflow ended with error: %v", err)
	} else {
		log.Println("")
		log.Println("✅ Workflow completed successfully!")
		log.Printf("   View history: http://localhost:8233/namespaces/default/workflows/%s", workflowID)
	}
}
