// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"open-swarm/internal/config"
	"open-swarm/internal/conflict"
	"open-swarm/pkg/coordinator"
)

const (
	blueLakeReservationID    = 101
	redMountainReservationID = 102
	greenForestReservationID = 103
	oldAgentReservationID    = 104
	thirtyMinutes            = 30 * time.Minute
	fifteenMinutes           = 15 * time.Minute
	threeMinutes             = 3 * time.Minute
	tenMinutes               = 10 * time.Minute
)

func main() {
	// Configure structured logging with JSON output for production
	// or text output for development
	logFormat := os.Getenv("LOG_FORMAT")
	var handler slog.Handler
	if logFormat == "json" {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	}
	slog.SetDefault(slog.New(handler))

	fmt.Println("=== Open Swarm Logging Demo ===\n")
	fmt.Println("This demonstrates the comprehensive logging system that tracks")
	fmt.Println("agent interactions and conflict resolution in real-time.\n")

	// Demo 1: Agent Coordination
	fmt.Println("\n--- Demo 1: Agent Coordination ---")
	cfg := &config.Config{
		Project: config.ProjectConfig{
			WorkingDirectory: "/demo/project",
		},
	}

	coord, err := coordinator.New(cfg)
	if err != nil {
		fmt.Printf("Failed to create coordinator: %v\n", err)
		return
	}

	// Register multiple agents
	if err := coord.RegisterAgent("BlueLake", "opencode", "sonnet-4.5", "Implementing user authentication"); err != nil {
		fmt.Printf("Failed to register BlueLake agent: %v\n", err)
	}
	if err := coord.RegisterAgent("RedMountain", "opencode", "opus-4.5", "Writing tests for auth module"); err != nil {
		fmt.Printf("Failed to register RedMountain agent: %v\n", err)
	}
	if err := coord.RegisterAgent("GreenForest", "opencode", "haiku-4.5", "Refactoring database layer"); err != nil {
		fmt.Printf("Failed to register GreenForest agent: %v\n", err)
	}

	// Sync coordination state
	if err := coord.Sync(); err != nil {
		fmt.Printf("Failed to sync coordination state: %v\n", err)
	}

	// Demo 2: Conflict Detection
	fmt.Println("\n\n--- Demo 2: Conflict Detection ---")
	analyzer := conflict.NewAnalyzer("/demo/project")

	// Simulate existing reservations
	existingReservations := []conflict.Reservation{
		{
			ID:        blueLakeReservationID,
			AgentName: "BlueLake",
			Pattern:   "internal/auth/*.go",
			Exclusive: true,
			ExpiresAt: time.Now().Add(thirtyMinutes),
		},
		{
			ID:        redMountainReservationID,
			AgentName: "RedMountain",
			Pattern:   "internal/auth/*_test.go",
			Exclusive: false,
			ExpiresAt: time.Now().Add(fifteenMinutes),
		},
	}

	ctx := context.Background()

	// Scenario 1: No conflict (different patterns)
	fmt.Println("\nScenario 1: GreenForest requests internal/db/*.go (no conflict expected)")
	c1, _ := analyzer.CheckConflict(ctx, "GreenForest", "internal/db/*.go", true, existingReservations)
	if c1 == nil {
		fmt.Println("✓ No conflict - can proceed")
	}

	// Scenario 2: Conflict detected (overlapping exclusive patterns)
	fmt.Println("\nScenario 2: GreenForest requests internal/auth/*.go (conflict expected)")
	c2, _ := analyzer.CheckConflict(ctx, "GreenForest", "internal/auth/*.go", true, existingReservations)
	if c2 != nil {
		resolution := analyzer.SuggestResolution(c2)
		report := analyzer.FormatConflictReport(c2, resolution)
		fmt.Println(report)
	}

	// Scenario 3: Expiring reservation
	fmt.Println("\nScenario 3: Reservation expiring soon")
	expiringReservations := []conflict.Reservation{
		{
			ID:        greenForestReservationID,
			AgentName: "BlueLake",
			Pattern:   "internal/config/*.go",
			Exclusive: true,
			ExpiresAt: time.Now().Add(threeMinutes), // Expires in 3 minutes
		},
	}
	c3, _ := analyzer.CheckConflict(ctx, "GreenForest", "internal/config/*.go", true, expiringReservations)
	if c3 != nil {
		resolution := analyzer.SuggestResolution(c3)
		fmt.Printf("Resolution: %s (wait for expiration)\n", resolution)
	}

	// Scenario 4: Stale reservation
	fmt.Println("\nScenario 4: Stale (expired) reservation")
	staleReservations := []conflict.Reservation{
		{
			ID:        oldAgentReservationID,
			AgentName: "OldAgent",
			Pattern:   "internal/cache/*.go",
			Exclusive: true,
			ExpiresAt: time.Now().Add(-tenMinutes), // Expired 10 minutes ago
		},
	}
	c4, _ := analyzer.CheckConflict(ctx, "GreenForest", "internal/cache/*.go", true, staleReservations)
	if c4 != nil {
		resolution := analyzer.SuggestResolution(c4)
		fmt.Printf("Resolution: %s (force release stale reservation)\n", resolution)
	}

	fmt.Println("\n\n=== Demo Complete ===")
	fmt.Println("\nKey Takeaways:")
	fmt.Println("✓ All agent operations are logged with structured data")
	fmt.Println("✓ Conflicts are detected early with clear resolution strategies")
	fmt.Println("✓ Agents coordinate cooperatively (negotiate, wait, or force-release)")
	fmt.Println("✓ No hostile behavior - system promotes collaboration")
	fmt.Println("\nTo see JSON logs, run: LOG_FORMAT=json go run ./cmd/logging-demo")
}
