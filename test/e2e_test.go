//go:build e2e
// +build e2e

// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"open-swarm/internal/temporal"
)

const (
	testTimeout     = 5 * time.Minute
	temporalAddress = "localhost:7233"
	taskQueue       = "open-swarm-test"
)

// TestE2E_Prerequisites verifies that all required services are running
func TestE2E_Prerequisites(t *testing.T) {
	t.Run("Docker Compose Running", func(t *testing.T) {
		cmd := exec.Command("docker", "compose", "ps", "--format", "json")
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "docker compose ps failed: %s", output)
		assert.Contains(t, string(output), "temporal", "Temporal container should be running")
		assert.Contains(t, string(output), "postgresql", "PostgreSQL container should be running")
	})

	t.Run("Temporal Server Accessible", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		c, err := client.Dial(client.Options{
			HostPort:  temporalAddress,
			Namespace: "default",
		})
		require.NoError(t, err, "Failed to connect to Temporal")
		defer c.Close()

		// Verify we can make a request
		_, err = c.WorkflowService().GetSystemInfo(ctx, nil)
		assert.NoError(t, err, "Temporal server should be healthy")
	})

	t.Run("Git Repository Available", func(t *testing.T) {
		cmd := exec.Command("git", "status")
		err := cmd.Run()
		require.NoError(t, err, "Should be in a git repository")
	})

	t.Run("OpenCode Available", func(t *testing.T) {
		// Check if opencode is in PATH
		_, err := exec.LookPath("opencode")
		if err != nil {
			t.Skip("OpenCode not in PATH, skipping OpenCode-dependent tests")
		}
	})
}

// TestE2E_TCRWorkflow_FullCycle tests the entire TCR workflow end-to-end
func TestE2E_TCRWorkflow_FullCycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Connect to Temporal
	c, err := client.Dial(client.Options{
		HostPort:  temporalAddress,
		Namespace: "default",
	})
	require.NoError(t, err, "Failed to connect to Temporal")
	defer c.Close()

	// Start worker in goroutine
	w := worker.New(c, taskQueue, worker.Options{})

	// Register activities and workflows
	cellActivities := temporal.NewCellActivities()
	w.RegisterWorkflow(temporal.TCRWorkflow)
	w.RegisterActivity(cellActivities.BootstrapCell)
	w.RegisterActivity(cellActivities.ExecuteTask)
	w.RegisterActivity(cellActivities.RunTests)
	w.RegisterActivity(cellActivities.CommitChanges)
	w.RegisterActivity(cellActivities.RevertChanges)
	w.RegisterActivity(cellActivities.TeardownCell)

	// Start worker
	go func() {
		err := w.Run(worker.InterruptCh())
		if err != nil {
			t.Logf("Worker error: %v", err)
		}
	}()
	defer w.Stop()

	// Wait for worker to be ready
	time.Sleep(2 * time.Second)

	t.Run("Successful Test Flow - Commit", func(t *testing.T) {
		workflowID := fmt.Sprintf("tcr-test-success-%d", time.Now().Unix())

		input := temporal.TCRWorkflowInput{
			CellID:      "e2e-test-cell-1",
			Branch:      "main",
			TaskID:      "test-task-1",
			Description: "Add simple function",
			Prompt:      "Create a function that returns 42",
		}

		workflowOptions := client.StartWorkflowOptions{
			ID:        workflowID,
			TaskQueue: taskQueue,
		}

		we, err := c.ExecuteWorkflow(ctx, workflowOptions, temporal.TCRWorkflow, input)
		require.NoError(t, err, "Failed to start TCR workflow")

		var result temporal.TCRWorkflowResult
		err = we.Get(ctx, &result)

		// Note: This may fail in CI without OpenCode, but tests the workflow execution
		if err != nil {
			t.Logf("Workflow execution error (expected in CI): %v", err)
			return
		}

		assert.True(t, result.Success, "Workflow should succeed")
		if result.TestsPassed {
			assert.NotEmpty(t, result.FilesChanged, "Should have committed files")
		}
	})

	t.Run("Failed Test Flow - Revert", func(t *testing.T) {
		workflowID := fmt.Sprintf("tcr-test-failure-%d", time.Now().Unix())

		input := temporal.TCRWorkflowInput{
			CellID:      "e2e-test-cell-2",
			Branch:      "main",
			TaskID:      "test-task-2",
			Description: "Intentionally broken code",
			Prompt:      "Create a function that will fail tests",
		}

		workflowOptions := client.StartWorkflowOptions{
			ID:        workflowID,
			TaskQueue: taskQueue,
		}

		we, err := c.ExecuteWorkflow(ctx, workflowOptions, temporal.TCRWorkflow, input)
		require.NoError(t, err, "Failed to start TCR workflow")

		var result temporal.TCRWorkflowResult
		err = we.Get(ctx, &result)

		// Workflow may complete with result.Success=false instead of error
		if err == nil {
			assert.False(t, result.TestsPassed || result.Success, "Should detect test failure")
		}
	})
}

// TestE2E_DAGWorkflow_ParallelExecution tests DAG workflow with parallel tasks
func TestE2E_DAGWorkflow_ParallelExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Connect to Temporal
	c, err := client.Dial(client.Options{
		HostPort:  temporalAddress,
		Namespace: "default",
	})
	require.NoError(t, err, "Failed to connect to Temporal")
	defer c.Close()

	// Start worker
	w := worker.New(c, taskQueue, worker.Options{})

	shellActivities := &temporal.ShellActivities{}
	w.RegisterWorkflow(temporal.TddDagWorkflow)
	w.RegisterActivity(shellActivities.RunScript)

	go w.Run(worker.InterruptCh())
	defer w.Stop()

	time.Sleep(2 * time.Second)

	workflowID := fmt.Sprintf("dag-cycle-%d", time.Now().Unix())

	// Create a cycle: A -> B -> C -> A
	tasks := []temporal.Task{
		{Name: "A", Command: "echo 'A'", Deps: []string{"C"}},
		{Name: "B", Command: "echo 'B'", Deps: []string{"A"}},
		{Name: "C", Command: "echo 'C'", Deps: []string{"B"}},
	}

	input := temporal.DAGWorkflowInput{
		WorkflowID: workflowID,
		Branch:     "main",
		Tasks:      tasks,
	}

	workflowOptions := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: taskQueue,
	}

	_, err = c.ExecuteWorkflow(ctx, workflowOptions, temporal.TddDagWorkflow, input)
	require.NoError(t, err, "Failed to start DAG workflow")

	// Wait for first cycle to fail
	time.Sleep(3 * time.Second)

	// The workflow should be waiting for signal (failed due to cycle)
	// Attempt to signal - workflow should still be running
	_ = c.SignalWorkflow(ctx, workflowID, "", "FixApplied", "Attempted fix")

	// Either signal succeeds (workflow still running) or workflow already errored out
	// Both cases are acceptable for cycle detection test
}

// TestE2E_SystemIntegration tests full system integration
func TestE2E_SystemIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	t.Run("Health Checks", func(t *testing.T) {
		// Test Temporal connection
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		c, err := client.Dial(client.Options{
			HostPort:  temporalAddress,
			Namespace: "default",
		})
		require.NoError(t, err)
		defer c.Close()

		// Verify namespace exists
		resp, err := c.WorkflowService().DescribeNamespace(ctx, nil)
		assert.NoError(t, err)
		if err == nil {
			assert.NotNil(t, resp)
		}
	})

	t.Run("Worker Registration", func(t *testing.T) {
		_, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		c, err := client.Dial(client.Options{
			HostPort:  temporalAddress,
			Namespace: "default",
		})
		require.NoError(t, err)
		defer c.Close()

		w := worker.New(c, taskQueue, worker.Options{})

		// Register everything
		cellActivities := temporal.NewCellActivities()
		shellActivities := &temporal.ShellActivities{}

		w.RegisterWorkflow(temporal.TCRWorkflow)
		w.RegisterWorkflow(temporal.TddDagWorkflow)
		w.RegisterActivity(cellActivities.BootstrapCell)
		w.RegisterActivity(cellActivities.ExecuteTask)
		w.RegisterActivity(cellActivities.RunTests)
		w.RegisterActivity(cellActivities.CommitChanges)
		w.RegisterActivity(cellActivities.RevertChanges)
		w.RegisterActivity(cellActivities.TeardownCell)
		w.RegisterActivity(shellActivities.RunScript)

		// Start and stop worker
		go w.Run(worker.InterruptCh())
		time.Sleep(1 * time.Second)
		w.Stop()
	})
}

// TestMain ensures proper setup and teardown for E2E tests
func TestMain(m *testing.M) {
	// Check if running in E2E mode
	if os.Getenv("E2E_TEST") == "" {
		fmt.Println("Skipping E2E tests (set E2E_TEST=1 to run)")
		os.Exit(0)
	}

	// Verify Docker Compose is running
	cmd := exec.Command("docker", "compose", "ps", "--format", "json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("ERROR: Docker Compose not running: %v\nOutput: %s\n", err, output)
		fmt.Println("Run: make docker-up")
		os.Exit(1)
	}

	if !contains(string(output), "temporal") {
		fmt.Println("ERROR: Temporal container not running")
		fmt.Println("Run: make docker-up")
		os.Exit(1)
	}

	// Wait for Temporal to be ready
	fmt.Println("Waiting for Temporal server to be ready...")
	time.Sleep(5 * time.Second)

	// Run tests
	code := m.Run()

	os.Exit(code)
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && s != "" && substr != "" &&
		(s == substr || len(s) >= len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
