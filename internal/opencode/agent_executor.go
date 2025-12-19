// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

// Package opencode provides the core agent execution library.
package opencode

import (
	"context"
	"fmt"
	"time"
)

// AgentExecutor is the main interface agents use to accomplish their tasks.
// It provides access to all opencode capabilities: code generation, testing, analysis.
type AgentExecutor interface {
	// GenerateCode creates code to meet requirements
	GenerateCode(ctx context.Context, task *CodeGenerationTask) (*GenerationResult, error)

	// RunTests executes tests and returns results
	RunTests(ctx context.Context, workDir string) (*TestRunResult, error)

	// AnalyzeFile analyzes code structure and complexity
	AnalyzeFile(ctx context.Context, filePath string) (*CodeAnalysis, error)

	// CompleteTask is a high-level method that orchestrates generation, testing, and verification
	CompleteTask(ctx context.Context, beadsTask *BeadsTaskSpec) (*TaskCompletion, error)
}

// BeadsTaskSpec represents a Beads task that an agent needs to complete
type BeadsTaskSpec struct {
	ID              string // Beads task ID
	Title           string // Task title
	Description     string // Detailed description
	AcceptanceCriteria string // What success looks like
	Dependencies    []string // Task IDs this depends on
	WorkDirectory   string // Where to work
}

// TaskCompletion represents the completion status of a task
type TaskCompletion struct {
	Success         bool          // Whether task completed successfully
	TaskID          string        // Beads task ID
	Duration        time.Duration // Total time taken
	FilesCreated    []string      // Files created
	FilesModified   []string      // Files modified
	TestsGenerated  int           // Number of tests generated
	TestsPassed     int           // Number of tests passing
	CodeGenerated   bool          // Whether code was generated
	ErrorMessage    string        // Any errors encountered
	Details         string        // Detailed completion notes
}

// DefaultAgentExecutor implements AgentExecutor
type DefaultAgentExecutor struct {
	testRunner    TestRunner
	codeGenerator CodeGenerator
	analyzer      CodeAnalyzer
}

// NewAgentExecutor creates a new AgentExecutor
func NewAgentExecutor(
	testRunner TestRunner,
	codeGenerator CodeGenerator,
	analyzer CodeAnalyzer,
) AgentExecutor {
	return &DefaultAgentExecutor{
		testRunner:    testRunner,
		codeGenerator: codeGenerator,
		analyzer:      analyzer,
	}
}

// GenerateCode generates code
func (a *DefaultAgentExecutor) GenerateCode(ctx context.Context, task *CodeGenerationTask) (*GenerationResult, error) {
	return a.codeGenerator.GenerateCode(ctx, task)
}

// RunTests runs tests
func (a *DefaultAgentExecutor) RunTests(ctx context.Context, workDir string) (*TestRunResult, error) {
	return a.testRunner.RunTests(ctx, workDir)
}

// AnalyzeFile analyzes code
func (a *DefaultAgentExecutor) AnalyzeFile(ctx context.Context, filePath string) (*CodeAnalysis, error) {
	return a.analyzer.AnalyzeFile(ctx, filePath)
}

// CompleteTask orchestrates the full process of completing a Beads task
func (a *DefaultAgentExecutor) CompleteTask(ctx context.Context, beadsTask *BeadsTaskSpec) (*TaskCompletion, error) {
	startTime := time.Now()
	completion := &TaskCompletion{
		TaskID:   beadsTask.ID,
		Duration: 0,
	}

	// Step 1: Generate test code (TDD RED phase)
	testTask := &CodeGenerationTask{
		TaskID:       beadsTask.ID + "-tests",
		Description:  "Generate tests for: " + beadsTask.Title,
		Requirements: beadsTask.AcceptanceCriteria,
		Language:     "go",
		MaxRetries:   2,
	}

	testResult, err := a.GenerateCode(ctx, testTask)
	if err != nil {
		completion.Success = false
		completion.ErrorMessage = fmt.Sprintf("failed to generate tests: %v", err)
		completion.Duration = time.Since(startTime)
		return completion, err
	}

	completion.TestsGenerated = 1
	completion.FilesCreated = append(completion.FilesCreated, testResult.FilesCreated...)
	completion.FilesModified = append(completion.FilesModified, testResult.FilesModified...)

	// Step 2: Run tests (should fail - RED state)
	testRunResult, err := a.RunTests(ctx, beadsTask.WorkDirectory)
	if err != nil {
		// Tests failing at RED phase is expected
		completion.TestsGenerated = testRunResult.TotalTests
	} else {
		completion.TestsGenerated = testRunResult.TotalTests
		if testRunResult.Success {
			// Tests passing when they should fail is a problem
			completion.ErrorMessage = "Tests passed unexpectedly in RED phase - implementation may already exist"
		}
	}

	// Step 3: Generate implementation code (TDD GREEN phase)
	implTask := &CodeGenerationTask{
		TaskID:       beadsTask.ID + "-implementation",
		Description:  "Generate implementation for: " + beadsTask.Title,
		Requirements: fmt.Sprintf("Make these failing tests pass:\n%s\n\nRequirements:\n%s",
			testResult.GeneratedCode, beadsTask.AcceptanceCriteria),
		Language:   "go",
		MaxRetries: 2,
	}

	implResult, err := a.GenerateCode(ctx, implTask)
	if err != nil {
		completion.Success = false
		completion.ErrorMessage = fmt.Sprintf("failed to generate implementation: %v", err)
		completion.Duration = time.Since(startTime)
		return completion, err
	}

	completion.CodeGenerated = true
	completion.FilesCreated = append(completion.FilesCreated, implResult.FilesCreated...)
	completion.FilesModified = append(completion.FilesModified, implResult.FilesModified...)

	// Step 4: Verify tests pass (GREEN state)
	testRunResult, err = a.RunTests(ctx, beadsTask.WorkDirectory)
	if err != nil {
		completion.Success = false
		completion.ErrorMessage = fmt.Sprintf("tests failed after implementation: %v", err)
		completion.TestsPassed = testRunResult.PassedTests
		completion.Duration = time.Since(startTime)
		return completion, err
	}

	completion.TestsPassed = testRunResult.PassedTests

	if !testRunResult.Success {
		completion.Success = false
		completion.ErrorMessage = fmt.Sprintf("%d tests still failing", testRunResult.FailedTests)
		completion.Duration = time.Since(startTime)
		return completion, fmt.Errorf("tests failed: %d failures", testRunResult.FailedTests)
	}

	// Step 5: Analyze generated code (optional)
	if len(completion.FilesCreated) > 0 {
		analysis, err := a.AnalyzeFile(ctx, completion.FilesCreated[0])
		if err == nil && analysis != nil {
			completion.Details = fmt.Sprintf("Generated %d files with %d functions, complexity: %d",
				len(completion.FilesModified), analysis.Complexity.Functions, analysis.Complexity.CyclomaticComplexity)
		}
	}

	// Task complete
	completion.Success = true
	completion.Duration = time.Since(startTime)

	return completion, nil
}
