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
	log.SetFlags(log.Ltime)

	log.Println("🤖 Open Swarm - AI Agent Automation Demo")
	log.Println("==========================================")
	log.Println("")
	log.Println("This demo shows a complete AI agent workflow:")
	log.Println("  1. Multi-agent coordination")
	log.Println("  2. Parallel task execution")
	log.Println("  3. TDD workflow (Test-Commit-Revert)")
	log.Println("  4. DAG-based dependency management")
	log.Println("")

	// Connect to Temporal
	c, err := client.Dial(client.Options{
		HostPort: "localhost:7233",
	})
	if err != nil {
		log.Fatalln("❌ Unable to connect to Temporal:", err)
	}
	defer c.Close()

	log.Println("✅ Connected to Temporal server")
	log.Println("🌐 Watch live: http://localhost:8233")
	log.Println("")

	ctx := context.Background()

	// Demo 1: AI Agent Build Pipeline
	log.Println("═══════════════════════════════════════")
	log.Println("📊 DEMO 1: AI Agent Build Pipeline")
	log.Println("═══════════════════════════════════════")
	runBuildPipeline(ctx, c)

	time.Sleep(2 * time.Second)

	// Demo 2: Multi-Agent Feature Development
	log.Println("")
	log.Println("═══════════════════════════════════════")
	log.Println("👥 DEMO 2: Multi-Agent Feature Development")
	log.Println("═══════════════════════════════════════")
	runMultiAgentFeature(ctx, c)

	time.Sleep(2 * time.Second)

	// Demo 3: TDD Workflow with Agent
	log.Println("")
	log.Println("═══════════════════════════════════════")
	log.Println("🧪 DEMO 3: TDD Workflow with AI Agent")
	log.Println("═══════════════════════════════════════")
	runTDDWorkflow(ctx, c)

	log.Println("")
	log.Println("═══════════════════════════════════════")
	log.Println("✅ All demos complete!")
	log.Println("═══════════════════════════════════════")
	log.Println("")
	log.Println("📊 View all workflows: http://localhost:8233")
	log.Println("")
}

// runBuildPipeline simulates an AI agent build pipeline with parallel execution
func runBuildPipeline(ctx context.Context, c client.Client) {
	workflowID := fmt.Sprintf("ai-build-pipeline-%d", time.Now().Unix())

	tasks := []temporal.Task{
		// Setup phase
		{
			Name:    "init-workspace",
			Command: "echo '🔧 [Agent] Initializing workspace...' && sleep 1",
			Deps:    []string{},
		},
		{
			Name:    "install-deps",
			Command: "echo '📦 [Agent] Installing dependencies...' && sleep 2",
			Deps:    []string{"init-workspace"},
		},

		// Parallel build phase (3 agents working simultaneously)
		{
			Name:    "agent-1-backend",
			Command: "echo '⚙️  [Agent 1] Building backend API...' && sleep 3",
			Deps:    []string{"install-deps"},
		},
		{
			Name:    "agent-2-frontend",
			Command: "echo '🎨 [Agent 2] Building frontend UI...' && sleep 3",
			Deps:    []string{"install-deps"},
		},
		{
			Name:    "agent-3-worker",
			Command: "echo '🔄 [Agent 3] Building background worker...' && sleep 3",
			Deps:    []string{"install-deps"},
		},

		// Test phase (parallel tests)
		{
			Name:    "test-backend",
			Command: "echo '✅ [Agent 1] Running backend tests...' && sleep 2",
			Deps:    []string{"agent-1-backend"},
		},
		{
			Name:    "test-frontend",
			Command: "echo '✅ [Agent 2] Running frontend tests...' && sleep 2",
			Deps:    []string{"agent-2-frontend"},
		},
		{
			Name:    "test-integration",
			Command: "echo '🔗 [Agent 3] Running integration tests...' && sleep 2",
			Deps:    []string{"agent-1-backend", "agent-2-frontend", "agent-3-worker"},
		},

		// Deploy phase (waits for all tests)
		{
			Name:    "deploy-all",
			Command: "echo '🚀 [Coordinator] Deploying all services...' && sleep 2",
			Deps:    []string{"test-backend", "test-frontend", "test-integration"},
		},
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

	we, err := c.ExecuteWorkflow(ctx, workflowOptions, temporal.TddDagWorkflow, input)
	if err != nil {
		log.Printf("❌ Failed to start build pipeline: %v", err)
		return
	}

	log.Printf("✅ Build pipeline started: %s", we.GetID())
	log.Println("   👀 Watch: http://localhost:8233/namespaces/default/workflows/" + workflowID)

	// Send signal after tasks complete (simulating all tests passed)
	go func() {
		time.Sleep(18 * time.Second)
		log.Println("   📢 All tests passed! Sending completion signal...")
		c.SignalWorkflow(ctx, workflowID, "", "FixApplied", "Build pipeline successful")
	}()

	// Wait with timeout
	ctxTimeout, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	err = we.Get(ctxTimeout, nil)
	if err != nil {
		log.Printf("   ⚠️  Pipeline: %v", err)
	} else {
		log.Println("   ✅ Build pipeline completed!")
	}
}

// runMultiAgentFeature simulates multiple agents collaborating on a feature
func runMultiAgentFeature(ctx context.Context, c client.Client) {
	workflowID := fmt.Sprintf("multi-agent-feature-%d", time.Now().Unix())

	tasks := []temporal.Task{
		// Design phase
		{
			Name:    "design-architecture",
			Command: "echo '📐 [Architect Agent] Designing system architecture...' && sleep 2",
			Deps:    []string{},
		},

		// Parallel implementation (4 agents)
		{
			Name:    "impl-auth",
			Command: "echo '🔐 [Auth Agent] Implementing authentication...' && sleep 3",
			Deps:    []string{"design-architecture"},
		},
		{
			Name:    "impl-database",
			Command: "echo '💾 [DB Agent] Setting up database schema...' && sleep 3",
			Deps:    []string{"design-architecture"},
		},
		{
			Name:    "impl-api",
			Command: "echo '🌐 [API Agent] Building REST endpoints...' && sleep 3",
			Deps:    []string{"design-architecture"},
		},
		{
			Name:    "impl-ui",
			Command: "echo '🎨 [UI Agent] Creating user interface...' && sleep 3",
			Deps:    []string{"design-architecture"},
		},

		// Integration (coordinator agent)
		{
			Name:    "integrate-all",
			Command: "echo '🔗 [Coordinator] Integrating all components...' && sleep 2",
			Deps:    []string{"impl-auth", "impl-database", "impl-api", "impl-ui"},
		},

		// QA
		{
			Name:    "qa-validation",
			Command: "echo '✅ [QA Agent] Running validation tests...' && sleep 2",
			Deps:    []string{"integrate-all"},
		},
	}

	input := temporal.DAGWorkflowInput{
		WorkflowID: workflowID,
		Branch:     "feature/user-management",
		Tasks:      tasks,
	}

	workflowOptions := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: "open-swarm-demo",
	}

	we, err := c.ExecuteWorkflow(ctx, workflowOptions, temporal.TddDagWorkflow, input)
	if err != nil {
		log.Printf("❌ Failed to start multi-agent feature: %v", err)
		return
	}

	log.Printf("✅ Multi-agent feature started: %s", we.GetID())
	log.Println("   👀 Watch: http://localhost:8233/namespaces/default/workflows/" + workflowID)

	// Send signal after feature is complete
	go func() {
		time.Sleep(18 * time.Second)
		log.Println("   📢 Feature complete! Sending signal...")
		c.SignalWorkflow(ctx, workflowID, "", "FixApplied", "Feature development complete")
	}()

	ctxTimeout, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	err = we.Get(ctxTimeout, nil)
	if err != nil {
		log.Printf("   ⚠️  Feature: %v", err)
	} else {
		log.Println("   ✅ Multi-agent feature completed!")
	}
}

// runTDDWorkflow simulates a Test-Driven Development workflow with AI agent
func runTDDWorkflow(ctx context.Context, c client.Client) {
	workflowID := fmt.Sprintf("tdd-agent-%d", time.Now().Unix())

	// Simulate TDD cycle: Write test → Run (RED) → Implement → Run (GREEN) → Refactor
	tasks := []temporal.Task{
		{
			Name:    "write-test",
			Command: "echo '📝 [Agent] Writing test case...' && sleep 2",
			Deps:    []string{},
		},
		{
			Name:    "run-test-red",
			Command: "echo '🔴 [Agent] Running test (expecting failure)...' && sleep 1",
			Deps:    []string{"write-test"},
		},
		{
			Name:    "implement-feature",
			Command: "echo '⚙️  [Agent] Implementing feature to pass test...' && sleep 3",
			Deps:    []string{"run-test-red"},
		},
		{
			Name:    "run-test-green",
			Command: "echo '🟢 [Agent] Running test (expecting success)...' && sleep 1",
			Deps:    []string{"implement-feature"},
		},
		{
			Name:    "refactor-code",
			Command: "echo '♻️  [Agent] Refactoring for clean code...' && sleep 2",
			Deps:    []string{"run-test-green"},
		},
		{
			Name:    "commit-changes",
			Command: "echo '✅ [Agent] Committing changes to Git...' && sleep 1",
			Deps:    []string{"refactor-code"},
		},
	}

	input := temporal.DAGWorkflowInput{
		WorkflowID: workflowID,
		Branch:     "feature/tdd-example",
		Tasks:      tasks,
	}

	workflowOptions := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: "open-swarm-demo",
	}

	we, err := c.ExecuteWorkflow(ctx, workflowOptions, temporal.TddDagWorkflow, input)
	if err != nil {
		log.Printf("❌ Failed to start TDD workflow: %v", err)
		return
	}

	log.Printf("✅ TDD workflow started: %s", we.GetID())
	log.Println("   👀 Watch: http://localhost:8233/namespaces/default/workflows/" + workflowID)

	// Send signal after TDD cycle
	go func() {
		time.Sleep(12 * time.Second)
		log.Println("   📢 TDD cycle complete! Sending signal...")
		c.SignalWorkflow(ctx, workflowID, "", "FixApplied", "TDD cycle successful")
	}()

	ctxTimeout, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	err = we.Get(ctxTimeout, nil)
	if err != nil {
		log.Printf("   ⚠️  TDD: %v", err)
	} else {
		log.Println("   ✅ TDD workflow completed!")
	}
}
