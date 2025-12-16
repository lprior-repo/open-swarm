// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package temporal

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// BenchmarkStrategy defines which TCR workflow to benchmark
type BenchmarkStrategy string

const (
	StrategyBasic    BenchmarkStrategy = "basic"
	StrategyEnhanced BenchmarkStrategy = "enhanced"
)

// BenchmarkInput defines input for benchmark execution
type BenchmarkInput struct {
	Strategy    BenchmarkStrategy
	NumRuns     int
	Concurrency int
	Prompt      string
	Description string
	RepoBranch  string
}

// BenchmarkResult contains aggregated results from benchmark runs
type BenchmarkResult struct {
	Strategy      BenchmarkStrategy
	TotalRuns     int
	SuccessCount  int
	FailureCount  int
	TotalDuration time.Duration
	AvgDuration   time.Duration
	RunResults    []RunResult
}

// RunResult contains individual run results
type RunResult struct {
	RunID        int
	Success      bool
	Error        string
	Duration     time.Duration
	FilesChanged []string
}

// BenchmarkWorkflow executes N parallel runs of the specified TCR strategy
// and aggregates the results for performance comparison
func BenchmarkWorkflow(ctx workflow.Context, input BenchmarkInput) (*BenchmarkResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting Benchmark", "strategy", input.Strategy, "runs", input.NumRuns)

	startTime := workflow.Now(ctx)

	// Activity Options: Long timeout for LLM tasks
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Minute,
		HeartbeatTimeout:    1 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 1, // No retries in benchmarks
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Results container
	results := &BenchmarkResult{
		Strategy:   input.Strategy,
		TotalRuns:  input.NumRuns,
		RunResults: make([]RunResult, 0, input.NumRuns),
	}

	// Futures tracker
	var futures []workflow.Future

	// Launch runs in parallel (Temporal handles concurrency)
	for i := 0; i < input.NumRuns; i++ {
		runID := i + 1
		// Unique IDs for each child workflow to prevent collision
		cellID := fmt.Sprintf("bench-%s-%d-%d", input.Strategy, workflow.Now(ctx).Unix(), runID)
		taskID := fmt.Sprintf("TASK-%03d", runID)

		cwo := workflow.ChildWorkflowOptions{
			WorkflowID: fmt.Sprintf("%s-%s", cellID, input.Strategy),
		}
		childCtx := workflow.WithChildOptions(ctx, cwo)

		var f workflow.Future
		if input.Strategy == StrategyEnhanced {
			req := EnhancedTCRInput{
				CellID:             cellID,
				Branch:             input.RepoBranch,
				TaskID:             taskID,
				Description:        input.Description,
				AcceptanceCriteria: input.Prompt,
				ReviewersCount:     2, // 2 Judges for speed
			}
			f = workflow.ExecuteChildWorkflow(childCtx, EnhancedTCRWorkflow, req)
		} else {
			req := TCRWorkflowInput{
				CellID:      cellID,
				Branch:      input.RepoBranch,
				TaskID:      taskID,
				Description: input.Description,
				Prompt:      input.Prompt,
			}
			f = workflow.ExecuteChildWorkflow(childCtx, TCRWorkflow, req)
		}
		futures = append(futures, f)
	}

	// Gather Results
	for i, f := range futures {
		runStart := workflow.Now(ctx)
		var runRes RunResult
		runRes.RunID = i + 1

		// Helper to extract success/error from different result types
		if input.Strategy == StrategyEnhanced {
			var r EnhancedTCRResult
			err := f.Get(ctx, &r)
			if err != nil {
				runRes.Success = false
				runRes.Error = err.Error()
			} else {
				runRes.Success = r.Success
				runRes.Error = r.Error
				runRes.FilesChanged = r.FilesChanged
			}
		} else {
			var r TCRWorkflowResult
			err := f.Get(ctx, &r)
			if err != nil {
				runRes.Success = false
				runRes.Error = err.Error()
			} else {
				runRes.Success = r.Success
				runRes.Error = r.Error
				runRes.FilesChanged = r.FilesChanged
			}
		}

		runRes.Duration = workflow.Now(ctx).Sub(runStart)
		if runRes.Success {
			results.SuccessCount++
		} else {
			results.FailureCount++
		}
		results.RunResults = append(results.RunResults, runRes)
	}

	results.TotalDuration = workflow.Now(ctx).Sub(startTime)
	if results.TotalRuns > 0 {
		results.AvgDuration = results.TotalDuration / time.Duration(results.TotalRuns)
	}

	logger.Info("Benchmark Complete",
		"strategy", input.Strategy,
		"success", results.SuccessCount,
		"failure", results.FailureCount,
		"avgDuration", results.AvgDuration)

	return results, nil
}
