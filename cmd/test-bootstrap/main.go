package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/sst/opencode-sdk-go"
	"github.com/sst/opencode-sdk-go/option"

	"open-swarm/internal/agent"
	"open-swarm/internal/infra"
	"open-swarm/internal/temporal"
	"open-swarm/internal/workflow"
)

func main() {
	// Initialize global managers first
	temporal.InitializeGlobals(8000, 9000, ".", "./worktrees")

	pm, sm, wm := temporal.GetManagers()

	activities := workflow.NewActivities(pm, sm, wm)

	ctx := context.Background()

	// Bootstrap
	fmt.Println("Bootstrapping cell...")
	cell, err := activities.BootstrapCell(ctx, "test-cell", "main")
	if err != nil {
		log.Fatalf("Bootstrap failed: %v", err)
	}
	fmt.Printf("✓ Bootstrap succeeded: Port=%d, PID=%d, URL=%s\n", cell.Port, cell.ServerHandle.PID, cell.ServerHandle.BaseURL)

	// Check health immediately
	healthy := sm.IsHealthy(ctx, cell.ServerHandle)
	fmt.Printf("✓ Health check (immediate): %v\n", healthy)

	// Test SDK call immediately after health check passes
	fmt.Println("\nTesting SDK session creation immediately after health check...")
	client := opencode.NewClient(option.WithBaseURL(cell.ServerHandle.BaseURL))
	session, err := client.Session.New(ctx, opencode.SessionNewParams{
		Title: opencode.F("Test Session Immediate"),
	})
	if err != nil {
		fmt.Printf("⚠️  Immediate SDK call failed: %v\n", err)
	} else {
		fmt.Printf("✓ Immediate SDK call succeeded: SessionID=%s\n", session.ID)
	}

	// Wait additional time
	fmt.Println("\nWaiting 5 more seconds for full initialization...")
	time.Sleep(5 * time.Second)

	// Test SDK call after settling time
	fmt.Println("Testing SDK session creation after settling time...")
	session2, err := client.Session.New(ctx, opencode.SessionNewParams{
		Title: opencode.F("Test Session After Settling"),
	})
	if err != nil {
		fmt.Printf("⚠️  SDK call after settling failed: %v\n", err)
	} else {
		fmt.Printf("✓ SDK call after settling succeeded: SessionID=%s\n", session2.ID)
	}

	// Reconstruct without Cmd (simulate Temporal serialization)
	reconstructedHandle := &infra.ServerHandle{
		Port:    cell.Port,
		BaseURL: cell.ServerHandle.BaseURL,
		PID:     cell.ServerHandle.PID,
		// No Cmd!
	}

	healthyReconstructed := sm.IsHealthy(ctx, reconstructedHandle)
	fmt.Printf("✓ Health check (reconstructed): %v\n", healthyReconstructed)

	// Reconstruct cell (like ExecuteTask does)
	reconstructedCell := &workflow.CellBootstrap{
		CellID:       cell.CellID,
		Port:         cell.Port,
		WorktreeID:   cell.WorktreeID,
		WorktreePath: cell.WorktreePath,
		ServerHandle: reconstructedHandle,
		Client:       agent.NewClient(reconstructedHandle.BaseURL, reconstructedHandle.Port),
	}

	// Test ExecuteTask
	fmt.Println("\nTesting ExecuteTask...")
	taskCtx := &agent.TaskContext{
		TaskID:      "TEST-001",
		Description: "Test task",
		Prompt:      "List files in the current directory",
	}

	result, err := activities.ExecuteTask(ctx, reconstructedCell, taskCtx)
	if err != nil {
		fmt.Printf("⚠️  ExecuteTask failed: %v\n", err)
	} else {
		fmt.Printf("✓ ExecuteTask succeeded: Success=%v, Output=%s\n", result.Success, result.Output[:min(100, len(result.Output))])
	}

	// Teardown
	fmt.Println("\nTearing down...")
	if err := activities.TeardownCell(ctx, cell); err != nil {
		log.Fatalf("Teardown failed: %v", err)
	}

	fmt.Println("✓ Test completed successfully!")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
