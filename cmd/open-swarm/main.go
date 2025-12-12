// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package main

import (
	"fmt"
	"log"
	"os"

	"open-swarm/internal/config"
	"open-swarm/pkg/coordinator"
)

const version = "0.1.0"

func main() {
	fmt.Printf("Open Swarm v%s - Multi-Agent Coordination Framework\n", version)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize coordinator
	coord, err := coordinator.New(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize coordinator: %v", err)
	}

	fmt.Printf("\n✓ Project: %s\n", cfg.Project.Name)
	fmt.Printf("✓ Working Directory: %s\n", cfg.Project.WorkingDirectory)
	fmt.Printf("✓ Coordinator initialized\n\n")

	// Check command line arguments
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	command := os.Args[1]

	switch command {
	case "status":
		handleStatus(coord)
	case "agents":
		handleAgents(coord)
	case "sync":
		handleSync(coord)
	case "version":
		fmt.Printf("Open Swarm version %s\n", version)
	case "help":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		printUsage()
	}
}

func handleStatus(coord *coordinator.Coordinator) {
	fmt.Println("📊 Project Status")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	status := coord.GetStatus()

	fmt.Printf("\n🤖 Agents: %d active\n", status.ActiveAgents)
	fmt.Printf("📬 Messages: %d unread\n", status.UnreadMessages)
	fmt.Printf("📁 File Reservations: %d active\n", status.ActiveReservations)
	fmt.Printf("🧵 Threads: %d active\n", status.ActiveThreads)

	fmt.Println("\n✓ Status check complete")
}

func handleAgents(coord *coordinator.Coordinator) {
	fmt.Println("🤖 Active Agents")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	agents := coord.ListAgents()

	if len(agents) == 0 {
		fmt.Println("\nNo active agents found.")
		fmt.Println("Run with opencode to register an agent.")
		return
	}

	for _, agent := range agents {
		fmt.Printf("\n%s (%s, %s)\n", agent.Name, agent.Program, agent.Model)
		if agent.TaskDescription != "" {
			fmt.Printf("  Task: %s\n", agent.TaskDescription)
		}
		fmt.Printf("  Last active: %s\n", agent.LastActive)
	}

	fmt.Printf("\nTotal: %d agents\n", len(agents))
}

func handleSync(coord *coordinator.Coordinator) {
	fmt.Println("🔄 Synchronizing with coordination state...")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	if err := coord.Sync(); err != nil {
		log.Fatalf("Sync failed: %v", err)
	}

	fmt.Println("\n✓ Synchronization complete")
	fmt.Println("\nRun 'open-swarm status' for detailed status")
}

func printUsage() {
	fmt.Println("Usage: open-swarm <command>")
	fmt.Println("\nCommands:")
	fmt.Println("  status    Show project coordination status")
	fmt.Println("  agents    List all active agents")
	fmt.Println("  sync      Synchronize with coordination state")
	fmt.Println("  version   Show version information")
	fmt.Println("  help      Show this help message")
	fmt.Println("\nFor interactive agent coordination, use opencode with the")
	fmt.Println("slash commands defined in .claude/commands/")
	fmt.Println("\nExamples:")
	fmt.Println("  open-swarm status")
	fmt.Println("  open-swarm agents")
	fmt.Println("  open-swarm sync")
}
