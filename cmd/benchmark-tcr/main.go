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
	"strings"
	"time"

	"open-swarm/internal/temporal"

	"go.temporal.io/sdk/client"
)

func main() {
	strategy := flag.String("strategy", "basic", "Strategy: 'basic' or 'enhanced'")
	runs := flag.Int("runs", 3, "Number of parallel runs")
	prompt := flag.String("prompt", "", "Coding challenge prompt")
	branch := flag.String("branch", "main", "Git branch")
	concurrency := flag.Int("concurrency", 0, "Max concurrent runs (0 = unlimited)")

	flag.Parse()

	if *prompt == "" {
		log.Fatal("❌ Prompt required. Example: -prompt 'Implement a thread-safe LRU cache'")
	}

	if *strategy != "basic" && *strategy != "enhanced" {
		log.Fatal("❌ Strategy must be 'basic' or 'enhanced'")
	}

	if *concurrency == 0 {
		*concurrency = *runs
	}

	// 1. Connect to Temporal
	c, err := client.Dial(client.Options{})
	if err != nil {
		log.Fatalf("❌ Temporal connection failed: %v", err)
	}
	defer c.Close()

	// 2. Start Benchmark
	runID := fmt.Sprintf("bench-%s-%d", *strategy, time.Now().Unix())
	log.Printf("🚀 Starting %s Benchmark (%d runs)...", strings.ToUpper(*strategy), *runs)

	input := temporal.BenchmarkInput{
		Strategy:    temporal.BenchmarkStrategy(*strategy),
		NumRuns:     *runs,
		Concurrency: *concurrency,
		Prompt:      *prompt,
		Description: "Benchmark Evaluation",
		RepoBranch:  *branch,
	}

	we, err := c.ExecuteWorkflow(context.Background(), client.StartWorkflowOptions{
		ID:        runID,
		TaskQueue: "reactor-task-queue",
	}, temporal.BenchmarkWorkflow, input)

	if err != nil {
		log.Fatalf("❌ Execution failed: %v", err)
	}
	log.Printf("⏳ Workflow ID: %s", we.GetID())
	log.Printf("⏳ Run ID: %s", we.GetRunID())

	// 3. Wait & Report
	var res temporal.BenchmarkResult
	if err := we.Get(context.Background(), &res); err != nil {
		log.Fatalf("❌ Workflow failed: %v", err)
	}

	printReport(res)
}

func printReport(r temporal.BenchmarkResult) {
	fmt.Println("\n==========================================")
	fmt.Printf("📊 RESULTS: %s\n", strings.ToUpper(string(r.Strategy)))
	fmt.Println("==========================================")
	fmt.Printf("Runs:         %d\n", r.TotalRuns)
	fmt.Printf("✅ Success:   %d (%.1f%%)\n", r.SuccessCount, pct(r.SuccessCount, r.TotalRuns))
	fmt.Printf("❌ Failed:    %d (%.1f%%)\n", r.FailureCount, pct(r.FailureCount, r.TotalRuns))
	fmt.Printf("⏱️  Total Time: %s\n", r.TotalDuration.Round(time.Second))
	fmt.Printf("⏱️  Avg Time:  %s\n", r.AvgDuration.Round(time.Second))
	fmt.Println("------------------------------------------")

	// Detailed run results
	if len(r.RunResults) > 0 {
		fmt.Println("\n📋 Individual Run Results:")
		for _, run := range r.RunResults {
			status := "✅"
			if !run.Success {
				status = "❌"
			}
			fmt.Printf("  %s Run #%d: %s", status, run.RunID, run.Duration.Round(time.Second))
			if !run.Success && run.Error != "" {
				fmt.Printf(" - Error: %s", run.Error)
			}
			if len(run.FilesChanged) > 0 {
				fmt.Printf(" - Files: %d", len(run.FilesChanged))
			}
			fmt.Println()
		}
	}

	fmt.Println("==========================================")
}

func pct(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return float64(a) / float64(b) * 100
}
