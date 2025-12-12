// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package main

import (
	"context"
	"log"
	"time"

	"go.temporal.io/sdk/client"

	"open-swarm/internal/temporal"
)

func main() {
	// Connect to Temporal
	c, err := client.Dial(client.Options{
		HostPort: client.DefaultHostPort,
	})
	if err != nil {
		log.Fatalln("❌ Unable to connect:", err)
	}
	defer c.Close()

	log.Println("✅ Connected to Temporal server")

	// Test 1: Simple DAG workflow with shell commands
	log.Println("\n🧪 Test 1: Simple DAG Workflow")
	testSimpleDAG(c)

	// Test 2: DAG with dependencies
	log.Println("\n🧪 Test 2: DAG with Dependencies")
	testDAGWithDependencies(c)

	// Test 3: Parallel execution
	log.Println("\n🧪 Test 3: Parallel DAG Execution")
	testParallelDAG(c)

	log.Println("\n✅ All workflow tests completed!")
}

func testSimpleDAG(c client.Client) {
	input := temporal.DAGWorkflowInput{
		WorkflowID: "test-simple-dag",
		Branch:     "main",
		Tasks: []temporal.Task{
			{Name: "task1", Command: "echo 'Task 1 complete'", Deps: []string{}},
			{Name: "task2", Command: "echo 'Task 2 complete'", Deps: []string{}},
		},
	}

	workflowOptions := client.StartWorkflowOptions{
		ID:        "test-simple-dag-" + time.Now().Format("20060102-150405"),
		TaskQueue: "reactor-task-queue",
	}

	we, err := c.ExecuteWorkflow(context.Background(), workflowOptions, temporal.TddDagWorkflow, input)
	if err != nil {
		log.Printf("❌ Failed to start workflow: %v", err)
		return
	}

	log.Printf("✅ Workflow started: %s", we.GetID())

	// Wait for result (with timeout)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = we.Get(ctx, nil)
	if err != nil {
		log.Printf("⚠️  Workflow failed: %v", err)
	} else {
		log.Println("✅ Workflow completed successfully")
	}
}

func testDAGWithDependencies(c client.Client) {
	input := temporal.DAGWorkflowInput{
		WorkflowID: "test-dag-deps",
		Branch:     "main",
		Tasks: []temporal.Task{
			{Name: "prepare", Command: "echo 'Preparing...'", Deps: []string{}},
			{Name: "build", Command: "echo 'Building...'", Deps: []string{"prepare"}},
			{Name: "test", Command: "echo 'Testing...'", Deps: []string{"build"}},
		},
	}

	workflowOptions := client.StartWorkflowOptions{
		ID:        "test-dag-deps-" + time.Now().Format("20060102-150405"),
		TaskQueue: "reactor-task-queue",
	}

	we, err := c.ExecuteWorkflow(context.Background(), workflowOptions, temporal.TddDagWorkflow, input)
	if err != nil {
		log.Printf("❌ Failed to start workflow: %v", err)
		return
	}

	log.Printf("✅ Workflow started: %s", we.GetID())

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = we.Get(ctx, nil)
	if err != nil {
		log.Printf("⚠️  Workflow failed: %v", err)
	} else {
		log.Println("✅ Workflow completed successfully")
	}
}

func testParallelDAG(c client.Client) {
	// Diamond dependency: prepare -> (build1, build2) -> deploy
	input := temporal.DAGWorkflowInput{
		WorkflowID: "test-parallel-dag",
		Branch:     "main",
		Tasks: []temporal.Task{
			{Name: "prepare", Command: "echo 'Preparing workspace'", Deps: []string{}},
			{Name: "build1", Command: "echo 'Building component 1'", Deps: []string{"prepare"}},
			{Name: "build2", Command: "echo 'Building component 2'", Deps: []string{"prepare"}},
			{Name: "deploy", Command: "echo 'Deploying all components'", Deps: []string{"build1", "build2"}},
		},
	}

	workflowOptions := client.StartWorkflowOptions{
		ID:        "test-parallel-dag-" + time.Now().Format("20060102-150405"),
		TaskQueue: "reactor-task-queue",
	}

	we, err := c.ExecuteWorkflow(context.Background(), workflowOptions, temporal.TddDagWorkflow, input)
	if err != nil {
		log.Printf("❌ Failed to start workflow: %v", err)
		return
	}

	log.Printf("✅ Workflow started: %s", we.GetID())

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = we.Get(ctx, nil)
	if err != nil {
		log.Printf("⚠️  Workflow failed: %v", err)
	} else {
		log.Println("✅ Workflow completed successfully")
	}
}
