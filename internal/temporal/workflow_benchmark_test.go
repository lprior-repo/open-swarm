// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package temporal

import (
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/testsuite"
)

type BenchmarkWorkflowTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite
	env *testsuite.TestWorkflowEnvironment
}

func (s *BenchmarkWorkflowTestSuite) SetupTest() {
	s.env = s.NewTestWorkflowEnvironment()
	s.env.SetWorkflowRunTimeout(30 * time.Minute)
}

func (s *BenchmarkWorkflowTestSuite) AfterTest(suiteName, testName string) {
	s.env.AssertExpectations(s.T())
}

func TestBenchmarkWorkflowTestSuite(t *testing.T) {
	suite.Run(t, new(BenchmarkWorkflowTestSuite))
}

// TestBenchmarkWorkflow_BasicStrategy_Success tests successful basic TCR benchmark
func (s *BenchmarkWorkflowTestSuite) TestBenchmarkWorkflow_BasicStrategy_Success() {
	input := BenchmarkInput{
		Strategy:    StrategyBasic,
		NumRuns:     3,
		Concurrency: 3,
		Prompt:      "Implement a function",
		Description: "Test task",
		RepoBranch:  "main",
	}

	// Mock child workflow executions
	for i := 0; i < input.NumRuns; i++ {
		mockResult := &TCRWorkflowResult{
			Success:      true,
			TestsPassed:  true,
			FilesChanged: []string{"file1.go", "file2.go"},
			Error:        "",
		}
		s.env.OnWorkflow(TCRWorkflow, mock.Anything, mock.Anything).Return(mockResult, nil).Once()
	}

	s.env.ExecuteWorkflow(BenchmarkWorkflow, input)

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	var result BenchmarkResult
	s.NoError(s.env.GetWorkflowResult(&result))

	// Assertions
	s.Equal(StrategyBasic, result.Strategy)
	s.Equal(3, result.TotalRuns)
	s.Equal(3, result.SuccessCount)
	s.Equal(0, result.FailureCount)
	s.Equal(3, len(result.RunResults))
	s.GreaterOrEqual(result.TotalDuration, time.Duration(0))
	s.GreaterOrEqual(result.AvgDuration, time.Duration(0))

	// Verify all runs were successful
	for _, run := range result.RunResults {
		s.True(run.Success)
		s.Empty(run.Error)
		s.NotEmpty(run.FilesChanged)
	}
}

// TestBenchmarkWorkflow_EnhancedStrategy_Success tests successful enhanced TCR benchmark
func (s *BenchmarkWorkflowTestSuite) TestBenchmarkWorkflow_EnhancedStrategy_Success() {
	input := BenchmarkInput{
		Strategy:    StrategyEnhanced,
		NumRuns:     2,
		Concurrency: 2,
		Prompt:      "Implement with TDD",
		Description: "Enhanced test task",
		RepoBranch:  "main",
	}

	// Mock enhanced workflow executions
	for i := 0; i < input.NumRuns; i++ {
		mockResult := &EnhancedTCRResult{
			Success:      true,
			GateResults:  []GateResult{{GateName: "GenTest", Passed: true}},
			FilesChanged: []string{"impl.go", "impl_test.go"},
			Error:        "",
		}
		s.env.OnWorkflow(EnhancedTCRWorkflow, mock.Anything, mock.Anything).Return(mockResult, nil).Once()
	}

	s.env.ExecuteWorkflow(BenchmarkWorkflow, input)

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	var result BenchmarkResult
	s.NoError(s.env.GetWorkflowResult(&result))

	// Assertions
	s.Equal(StrategyEnhanced, result.Strategy)
	s.Equal(2, result.TotalRuns)
	s.Equal(2, result.SuccessCount)
	s.Equal(0, result.FailureCount)
	s.Equal(2, len(result.RunResults))

	// Verify all runs were successful
	for _, run := range result.RunResults {
		s.True(run.Success)
		s.Empty(run.Error)
		s.NotEmpty(run.FilesChanged)
	}
}

// TestBenchmarkWorkflow_MixedResults tests benchmark with both success and failure
func (s *BenchmarkWorkflowTestSuite) TestBenchmarkWorkflow_MixedResults() {
	input := BenchmarkInput{
		Strategy:    StrategyBasic,
		NumRuns:     4,
		Concurrency: 4,
		Prompt:      "Complex task",
		Description: "Test task",
		RepoBranch:  "main",
	}

	// Mock 2 successes and 2 failures
	s.env.OnWorkflow(TCRWorkflow, mock.Anything, mock.Anything).Return(&TCRWorkflowResult{
		Success:      true,
		TestsPassed:  true,
		FilesChanged: []string{"file1.go"},
	}, nil).Once()

	s.env.OnWorkflow(TCRWorkflow, mock.Anything, mock.Anything).Return(&TCRWorkflowResult{
		Success:      false,
		TestsPassed:  false,
		FilesChanged: []string{},
		Error:        "Tests failed",
	}, nil).Once()

	s.env.OnWorkflow(TCRWorkflow, mock.Anything, mock.Anything).Return(&TCRWorkflowResult{
		Success:      true,
		TestsPassed:  true,
		FilesChanged: []string{"file2.go"},
	}, nil).Once()

	s.env.OnWorkflow(TCRWorkflow, mock.Anything, mock.Anything).Return(&TCRWorkflowResult{
		Success:      false,
		TestsPassed:  false,
		FilesChanged: []string{},
		Error:        "Compilation failed",
	}, nil).Once()

	s.env.ExecuteWorkflow(BenchmarkWorkflow, input)

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	var result BenchmarkResult
	s.NoError(s.env.GetWorkflowResult(&result))

	// Assertions
	s.Equal(4, result.TotalRuns)
	s.Equal(2, result.SuccessCount)
	s.Equal(2, result.FailureCount)
	s.Equal(4, len(result.RunResults))

	// Verify mixed results
	successCount := 0
	failureCount := 0
	for _, run := range result.RunResults {
		if run.Success {
			successCount++
			s.Empty(run.Error)
		} else {
			failureCount++
			s.NotEmpty(run.Error)
		}
	}
	s.Equal(2, successCount)
	s.Equal(2, failureCount)
}

// TestBenchmarkWorkflow_SingleRun tests benchmark with single run
func (s *BenchmarkWorkflowTestSuite) TestBenchmarkWorkflow_SingleRun() {
	input := BenchmarkInput{
		Strategy:    StrategyBasic,
		NumRuns:     1,
		Concurrency: 1,
		Prompt:      "Simple task",
		Description: "Single run test",
		RepoBranch:  "main",
	}

	mockResult := &TCRWorkflowResult{
		Success:      true,
		TestsPassed:  true,
		FilesChanged: []string{"single.go"},
		Error:        "",
	}
	s.env.OnWorkflow(TCRWorkflow, mock.Anything, mock.Anything).Return(mockResult, nil).Once()

	s.env.ExecuteWorkflow(BenchmarkWorkflow, input)

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	var result BenchmarkResult
	s.NoError(s.env.GetWorkflowResult(&result))

	// Assertions
	s.Equal(1, result.TotalRuns)
	s.Equal(1, result.SuccessCount)
	s.Equal(0, result.FailureCount)
	s.Equal(1, len(result.RunResults))
	s.Equal(result.TotalDuration, result.AvgDuration)
}

// TestBenchmarkWorkflow_AllFailures tests benchmark where all runs fail
func (s *BenchmarkWorkflowTestSuite) TestBenchmarkWorkflow_AllFailures() {
	input := BenchmarkInput{
		Strategy:    StrategyEnhanced,
		NumRuns:     3,
		Concurrency: 3,
		Prompt:      "Impossible task",
		Description: "Failure test",
		RepoBranch:  "main",
	}

	// Mock all failures
	for i := 0; i < input.NumRuns; i++ {
		mockResult := &EnhancedTCRResult{
			Success:      false,
			GateResults:  []GateResult{{GateName: "GenTest", Passed: false, Error: "Failed"}},
			FilesChanged: []string{},
			Error:        "Gate failed",
		}
		s.env.OnWorkflow(EnhancedTCRWorkflow, mock.Anything, mock.Anything).Return(mockResult, nil).Once()
	}

	s.env.ExecuteWorkflow(BenchmarkWorkflow, input)

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	var result BenchmarkResult
	s.NoError(s.env.GetWorkflowResult(&result))

	// Assertions
	s.Equal(3, result.TotalRuns)
	s.Equal(0, result.SuccessCount)
	s.Equal(3, result.FailureCount)
	s.Equal(3, len(result.RunResults))

	// Verify all runs failed
	for _, run := range result.RunResults {
		s.False(run.Success)
		s.NotEmpty(run.Error)
	}
}
