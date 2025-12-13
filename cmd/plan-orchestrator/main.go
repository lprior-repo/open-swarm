// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"

	"open-swarm/internal/planner"
)

func main() {
	// CLI flags
	var (
		projectPrefix = flag.String("prefix", "open-swarm", "Project prefix for issue IDs")
		dryRun        = flag.Bool("dry-run", false, "Show what would be created without actually creating issues")
		inputFile     = flag.String("file", "", "Read plan from file instead of stdin")
		execute       = flag.Bool("execute", false, "Actually execute bd commands to create issues")
	)

	flag.Parse()

	log.Println("🎯 PLAN ORCHESTRATOR - Parse user plans and create Beads issues")
	log.Printf("📋 Project prefix: %s", *projectPrefix)

	// Read input plan
	var input string
	var err error

	if *inputFile != "" {
		log.Printf("📄 Reading plan from: %s", *inputFile)
		data, err := os.ReadFile(*inputFile)
		if err != nil {
			log.Fatalf("❌ Failed to read input file: %v", err)
		}
		input = string(data)
	} else {
		log.Println("📝 Reading plan from stdin (Ctrl+D to finish):")
		input, err = readStdin()
		if err != nil {
			log.Fatalf("❌ Failed to read stdin: %v", err)
		}
	}

	if strings.TrimSpace(input) == "" {
		log.Println("⚠️  No input provided. Exiting.")
		return
	}

	// Parse the plan
	parser := planner.NewPlanParser()
	tasks, err := parser.Parse(input)
	if err != nil {
		log.Fatalf("❌ Failed to parse plan: %v", err)
	}

	if len(tasks) == 0 {
		log.Println("⚠️  No tasks found in plan. Check your input format.")
		log.Println("\nSupported formats:")
		log.Println("  - Numbered lists: 1. Task name")
		log.Println("  - Bullet points: - Task name or * Task name")
		log.Println("  - Headers: ## Task N: Task name")
		log.Println("\nOptional annotations:")
		log.Println("  - Priority: [P0] to [P5]")
		log.Println("  - Dependencies: (depends on: 1, 2)")
		return
	}

	log.Printf("✅ Parsed %d tasks from plan\n", len(tasks))

	// Create execution plan
	creator := planner.NewBeadsCreator(*projectPrefix)
	plan, err := creator.CreatePlan(tasks)
	if err != nil {
		log.Fatalf("❌ Failed to create execution plan: %v", err)
	}

	// Display plan summary
	log.Println("\n" + strings.Repeat("═", 80))
	log.Println("📊 EXECUTION PLAN")
	log.Println(strings.Repeat("═", 80))
	fmt.Print(creator.FormatPlanSummary(plan))
	log.Println(strings.Repeat("═", 80))

	if *dryRun {
		log.Println("\n🔍 DRY RUN MODE - Showing commands that would be executed:")
		log.Println(strings.Repeat("─", 80))
		for i, cmd := range plan.Commands {
			fmt.Printf("%d. %s\n", i+1, cmd)
		}
		log.Println(strings.Repeat("─", 80))
		log.Println("\n✅ Run with --execute flag to actually create these issues")
		return
	}

	if !*execute {
		log.Println("\n⚠️  To create these issues, run with --execute flag")
		log.Println("   Or use --dry-run to see the exact commands")
		return
	}

	// Execute the plan
	log.Println("\n🚀 Creating Beads issues...")
	createdIDs, err := executeCommands(plan.Commands)
	if err != nil {
		log.Fatalf("❌ Failed to execute plan: %v", err)
	}

	// Print results
	log.Println("\n" + strings.Repeat("═", 80))
	log.Println("✅ SUCCESSFULLY CREATED ISSUES")
	log.Println(strings.Repeat("═", 80))
	for i, id := range createdIDs {
		fmt.Printf("%d. %s - %s\n", i+1, id, plan.Tasks[i].Title)
	}
	log.Println(strings.Repeat("═", 80))

	log.Printf("\n🎉 Created %d Beads issues!", len(createdIDs))
	log.Println("📝 Run 'bd list' to see all issues")
	log.Println("🔄 Run 'bd sync' to sync with git remote")
}

func readStdin() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	var builder strings.Builder

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				builder.WriteString(line)
				break
			}
			return "", err
		}
		builder.WriteString(line)
	}

	return builder.String(), nil
}

func executeCommands(commands []string) ([]string, error) {
	var createdIDs []string

	for i, cmdStr := range commands {
		log.Printf("  [%d/%d] Executing: %s", i+1, len(commands), cmdStr)

		// Parse the command (simple shell-like parsing)
		parts := parseCommand(cmdStr)
		if len(parts) < 2 {
			return nil, fmt.Errorf("invalid command: %s", cmdStr)
		}

		// Execute the bd command
		// #nosec G204 - command is constructed internally from trusted parser output
		cmd := exec.Command(parts[0], parts[1:]...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("    ❌ Failed: %v", err)
			log.Printf("    Output: %s", string(output))
			return nil, fmt.Errorf("command failed: %s - %w", cmdStr, err)
		}

		// Extract issue ID from output
		// bd create returns output like: "Created issue: open-swarm-abc"
		outputStr := string(output)
		issueID := extractIssueID(outputStr)
		if issueID != "" {
			createdIDs = append(createdIDs, issueID)
			log.Printf("    ✅ Created: %s", issueID)
		} else {
			log.Printf("    ✅ Success (output: %s)", strings.TrimSpace(outputStr))
		}
	}

	return createdIDs, nil
}

// parseCommand does simple command parsing (handles quoted strings)
func parseCommand(cmd string) []string {
	var parts []string
	var current strings.Builder
	inQuote := false

	for i, char := range cmd {
		switch char {
		case '"':
			if inQuote {
				// End of quoted string
				inQuote = false
			} else {
				// Start of quoted string
				inQuote = true
			}
		case ' ':
			if inQuote {
				current.WriteRune(char)
			} else if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(char)
		}

		// Add last part at end
		if i == len(cmd)-1 && current.Len() > 0 {
			parts = append(parts, current.String())
		}
	}

	return parts
}

// extractIssueID extracts the issue ID from bd create output
func extractIssueID(output string) string {
	// Look for patterns like "Created issue: open-swarm-abc" or just "open-swarm-abc"
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "Created") || strings.Contains(line, "issue") {
			// Extract ID-like patterns (prefix-xxxx)
			parts := strings.Fields(line)
			for _, part := range parts {
				if strings.Contains(part, "-") && len(part) > 3 {
					return part
				}
			}
		}
	}
	return ""
}
