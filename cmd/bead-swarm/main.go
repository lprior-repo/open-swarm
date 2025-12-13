// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"sync"
	"time"

	"go.temporal.io/sdk/client"

	"open-swarm/internal/temporal"
)

// Bead represents a task/issue to be processed by an agent.
type Bead struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Status      string   `json:"status"`
	Priority    int      `json:"priority"`
	IssueType   string   `json:"issue_type"`
	Labels      []string `json:"labels"`
}

// BeadResult contains the result of processing a single bead.
type BeadResult struct {
	BeadID    string
	Success   bool
	Duration  time.Duration
	Error     error
	Committed bool
}

func main() {
	maxAgents := flag.Int("max", 10, "Maximum parallel agents")
	dryRun := flag.Bool("dry-run", false, "Don't actually run workflows, just show what would run")
	flag.Parse()

	log.Printf("🐝 BEAD SWARM - Multi-Agent Code Generation")
	log.Printf("📊 Max parallel agents: %d", *maxAgents)

	// Get ready beads from bd CLI
	beads, err := getReadyBeads()
	if err != nil {
		log.Fatalf("❌ Failed to get beads: %v", err)
	}

	// Filter out epics (too broad for automated agents)
	workableBeads := filterWorkableBeads(beads)

	log.Printf("📋 Found %d total ready beads", len(beads))
	log.Printf("✅ Filtered to %d workable tasks (excluding epics)", len(workableBeads))

	if len(workableBeads) == 0 {
		log.Println("⚠️  No workable beads found. Create some tasks with: bd create")
		return
	}

	// Limit to max agents
	if len(workableBeads) > *maxAgents {
		log.Printf("⚠️  Limiting to first %d beads", *maxAgents)
		workableBeads = workableBeads[:*maxAgents]
	}

	// Show what we're about to do
	log.Println("\n" + strings.Repeat("═", 60))
	log.Println("📝 BEADS TO PROCESS:")
	log.Println(strings.Repeat("═", 60))
	for i, bead := range workableBeads {
		log.Printf("%2d. [%s] %s", i+1, bead.ID, bead.Title)
	}
	log.Println(strings.Repeat("═", 60))

	if *dryRun {
		log.Println("🔍 DRY RUN - Would process these beads but not actually running")
		return
	}

	// Connect to Temporal
	c, err := client.Dial(client.Options{
		HostPort: client.DefaultHostPort,
	})
	if err != nil {
		log.Fatalf("❌ Unable to connect to Temporal: %v", err)
	}
	defer c.Close()

	log.Println("✅ Connected to Temporal")
	log.Println("🚀 Launching agents...")

	// Process beads in parallel
	results := processBeadsInParallel(c, workableBeads)

	// Print results
	printResults(results)
}

func getReadyBeads() ([]Bead, error) {
	cmd := exec.Command("bd", "ready", "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("bd command failed: %w", err)
	}

	var beads []Bead
	if err := json.Unmarshal(output, &beads); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return beads, nil
}

func filterWorkableBeads(beads []Bead) []Bead {
	workable := []Bead{}
	for _, bead := range beads {
		// Skip epics - they're too broad
		if bead.IssueType == "epic" {
			continue
		}
		// Skip already in-progress (someone else is working on it)
		if bead.Status == "in_progress" {
			continue
		}
		workable = append(workable, bead)
	}
	return workable
}

func processBeadsInParallel(c client.Client, beads []Bead) []BeadResult {
	results := make([]BeadResult, len(beads))
	var wg sync.WaitGroup

	for i, bead := range beads {
		wg.Add(1)
		go func(idx int, b Bead) {
			defer wg.Done()
			results[idx] = processOneBead(c, b)
		}(i, bead)
	}

	wg.Wait()
	return results
}

func processOneBead(c client.Client, bead Bead) BeadResult {
	ctx := context.Background()
	startTime := time.Now()

	log.Printf("🤖 Agent starting on: [%s] %s", bead.ID, bead.Title)

	// Mark bead as in_progress
	// #nosec G204 - bead.ID is from trusted bd CLI output
	updateCmd := exec.Command("bd", "update", bead.ID, "--status=in_progress")
	if err := updateCmd.Run(); err != nil {
		log.Printf("⚠️  Failed to mark %s as in_progress: %v", bead.ID, err)
	}

	// Create workflow input
	workflowID := fmt.Sprintf("bead-%s-%d", bead.ID, time.Now().Unix())
	prompt := buildPromptForBead(bead)

	input := temporal.TCRWorkflowInput{
		CellID:      fmt.Sprintf("bead-%s", bead.ID),
		Branch:      "main",
		TaskID:      bead.ID,
		Description: bead.Title,
		Prompt:      prompt,
	}

	workflowOptions := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: "reactor-task-queue",
	}

	// Execute workflow
	we, err := c.ExecuteWorkflow(ctx, workflowOptions, temporal.TCRWorkflow, input)
	if err != nil {
		return BeadResult{
			BeadID:   bead.ID,
			Success:  false,
			Duration: time.Since(startTime),
			Error:    fmt.Errorf("failed to start workflow: %w", err),
		}
	}

	// Wait for completion
	var result temporal.TCRWorkflowResult
	err = we.Get(ctx, &result)
	duration := time.Since(startTime)

	if err != nil {
		// Mark as failed
		// #nosec G204 - bead.ID is from trusted bd CLI output
		closeCmd := exec.Command("bd", "update", bead.ID, "--status=open")
		if err := closeCmd.Run(); err != nil {
			log.Printf("⚠️  Failed to update bead status: %v", err)
		}

		log.Printf("❌ Agent failed on: [%s] %s - %v", bead.ID, bead.Title, err)
		return BeadResult{
			BeadID:   bead.ID,
			Success:  false,
			Duration: duration,
			Error:    err,
		}
	}

	if result.Success {
		// Close the bead
		// #nosec G204 - bead.ID is from trusted bd CLI output
		closeCmd := exec.Command("bd", "close", bead.ID, "--reason=Completed by automated agent swarm")
		if err := closeCmd.Run(); err != nil {
			log.Printf("⚠️  Failed to close %s: %v", bead.ID, err)
		}

		log.Printf("✅ Agent completed: [%s] %s (%v)", bead.ID, bead.Title, duration)
		return BeadResult{
			BeadID:    bead.ID,
			Success:   true,
			Duration:  duration,
			Committed: true,
		}
	}

	// Tests failed - revert
	closeCmd := exec.Command("bd", "update", bead.ID, "--status=open")
	if err := closeCmd.Run(); err != nil {
		log.Printf("⚠️  Failed to update bead status: %v", err)
	}

	log.Printf("⚠️  Agent tests failed: [%s] %s", bead.ID, bead.Title)
	return BeadResult{
		BeadID:   bead.ID,
		Success:  false,
		Duration: duration,
		Error:    fmt.Errorf("tests failed: %s", result.Error),
	}
}

func buildPromptForBead(bead Bead) string {
	prompt := fmt.Sprintf(`Task: %s

Description: %s

Requirements:
- Implement the feature described above
- Write comprehensive tests
- Follow Go best practices
- Ensure all tests pass
- Use TDD approach (write tests first)

Labels: %s
`, bead.Title, bead.Description, strings.Join(bead.Labels, ", "))

	return prompt
}

func printResults(results []BeadResult) {
	successCount := 0
	failCount := 0
	committedCount := 0
	var totalDuration time.Duration

	fmt.Println("\n" + strings.Repeat("═", 80))
	fmt.Println("📊 BEAD SWARM RESULTS")
	fmt.Println(strings.Repeat("═", 80))

	for _, r := range results {
		totalDuration += r.Duration
		if r.Success {
			successCount++
			if r.Committed {
				committedCount++
			}
			fmt.Printf("✅ [%s] Success in %v (COMMITTED)\n", r.BeadID, r.Duration)
		} else {
			failCount++
			fmt.Printf("❌ [%s] Failed in %v - %v\n", r.BeadID, r.Duration, r.Error)
		}
	}

	avgDuration := totalDuration / time.Duration(len(results))

	fmt.Println(strings.Repeat("─", 80))
	fmt.Printf("Total beads processed: %d\n", len(results))
	fmt.Printf("✅ Successful:         %d (%.1f%%)\n", successCount, float64(successCount)/float64(len(results))*100)
	fmt.Printf("💾 Committed to git:   %d\n", committedCount)
	fmt.Printf("❌ Failed:             %d (%.1f%%)\n", failCount, float64(failCount)/float64(len(results))*100)
	fmt.Printf("⏱️  Average duration:   %v\n", avgDuration)
	fmt.Println(strings.Repeat("═", 80))

	if committedCount > 0 {
		fmt.Printf("\n🎉 Successfully generated and committed code for %d beads!\n", committedCount)
		fmt.Println("📝 Run 'git log' to see the commits")
		fmt.Println("🔍 Run 'git diff HEAD~" + fmt.Sprintf("%d", committedCount) + "..HEAD' to see all changes")
	}
}
