// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package opencode

import (
	"context"
	"time"
)

// CodeGenerator provides a high-level interface for agents to generate code.
// Wraps the executor and helps coordinate code generation tasks.
type CodeGenerator interface {
	// GenerateCode generates code based on a description and returns modified files
	GenerateCode(ctx context.Context, task *CodeGenerationTask) (*GenerationResult, error)

	// GenerateTestCode generates test code (TDD RED phase)
	GenerateTestCode(ctx context.Context, requirements string, existingCode string) (*GenerationResult, error)

	// GenerateImplementation generates implementation code (TDD GREEN phase)
	GenerateImplementation(ctx context.Context, testCode string, requirements string) (*GenerationResult, error)

	// RefactorCode improves existing code without changing behavior
	RefactorCode(ctx context.Context, filePath string, reason string) (*GenerationResult, error)

	// GenerateDocumentation generates documentation for code
	GenerateDocumentation(ctx context.Context, filePath string) (*GenerationResult, error)
}

// CodeGenerationTask describes what code to generate
type CodeGenerationTask struct {
	TaskID          string        // Unique task identifier
	Description     string        // What needs to be generated
	Requirements    string        // Detailed requirements
	ExistingCode    string        // Any existing code context
	Language        string        // Programming language (e.g., "go", "typescript")
	MaxTokens       int           // Maximum tokens to use (0 = default)
	RetryOnFailure  bool          // Whether to retry if generation fails
	MaxRetries      int           // Maximum number of retries
	VerificationFn  func(string) bool // Optional function to verify generated code
}

// GenerationResult contains the result of code generation
type GenerationResult struct {
	Success         bool          // Whether generation succeeded
	GeneratedCode   string        // The generated code
	FilesCreated    []string      // List of files created
	FilesModified   []string      // List of files modified
	Duration        time.Duration // How long generation took
	ErrorMessage    string        // Error if it failed
	Attempts        int           // Number of attempts made
	FullOutput      string        // Full output from the generator
}

// DefaultCodeGenerator implements CodeGenerator
type DefaultCodeGenerator struct {
	analyzer CodeAnalyzer
}

// NewCodeGenerator creates a new CodeGenerator
func NewCodeGenerator(analyzer CodeAnalyzer) CodeGenerator {
	return &DefaultCodeGenerator{
		analyzer: analyzer,
	}
}

// GenerateCode generates code based on a task description
func (g *DefaultCodeGenerator) GenerateCode(ctx context.Context, task *CodeGenerationTask) (*GenerationResult, error) {
	startTime := time.Now()

	// Build prompt
	prompt := buildCodeGenerationPrompt(task)

	// Stub implementation - would use executor in real scenario
	// This demonstrates the interface for agents to use

	return &GenerationResult{
		Success:       true,
		GeneratedCode: prompt,
		FilesModified: []string{},
		Duration:      time.Since(startTime),
		Attempts:      1,
		FullOutput:    prompt,
	}, nil
}

// GenerateTestCode generates test code (RED phase)
func (g *DefaultCodeGenerator) GenerateTestCode(ctx context.Context, requirements string, existingCode string) (*GenerationResult, error) {
	task := &CodeGenerationTask{
		TaskID:       "generate-tests",
		Description:  "Generate test code",
		Requirements: requirements,
		ExistingCode: existingCode,
		Language:     "go",
		MaxRetries:   2,
	}

	return g.GenerateCode(ctx, task)
}

// GenerateImplementation generates implementation code (GREEN phase)
func (g *DefaultCodeGenerator) GenerateImplementation(ctx context.Context, testCode string, requirements string) (*GenerationResult, error) {
	task := &CodeGenerationTask{
		TaskID:       "generate-implementation",
		Description:  "Generate implementation code",
		Requirements: requirements,
		ExistingCode: testCode,
		Language:     "go",
		MaxRetries:   2,
	}

	return g.GenerateCode(ctx, task)
}

// RefactorCode improves existing code
func (g *DefaultCodeGenerator) RefactorCode(ctx context.Context, filePath string, reason string) (*GenerationResult, error) {
	task := &CodeGenerationTask{
		TaskID:       "refactor-code",
		Description:  "Refactor code: " + reason,
		Requirements: "Improve the code at " + filePath + " by: " + reason,
		Language:     "go",
	}

	return g.GenerateCode(ctx, task)
}

// GenerateDocumentation generates documentation
func (g *DefaultCodeGenerator) GenerateDocumentation(ctx context.Context, filePath string) (*GenerationResult, error) {
	task := &CodeGenerationTask{
		TaskID:       "generate-docs",
		Description:  "Generate documentation for " + filePath,
		Requirements: "Create comprehensive documentation for " + filePath,
		Language:     "markdown",
	}

	return g.GenerateCode(ctx, task)
}

// buildCodeGenerationPrompt constructs a prompt for code generation
func buildCodeGenerationPrompt(task *CodeGenerationTask) string {
	prompt := "Generate " + task.Language + " code:\n\n"

	if task.Description != "" {
		prompt += "Task: " + task.Description + "\n\n"
	}

	if task.Requirements != "" {
		prompt += "Requirements:\n" + task.Requirements + "\n\n"
	}

	if task.ExistingCode != "" {
		prompt += "Existing code context:\n" + task.ExistingCode + "\n\n"
	}

	prompt += "Generate the code now."
	return prompt
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
