// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package opencode

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"open-swarm/internal/agent"
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

// DefaultCodeGenerator implements CodeGenerator with OpenCode SDK for Claude/Copilot
type DefaultCodeGenerator struct {
	analyzer CodeAnalyzer
	client   agent.ClientInterface
}

// NewCodeGenerator creates a new CodeGenerator with OpenCode SDK support
// Uses OpenCode for authentication and model selection (Claude or Copilot)
func NewCodeGenerator(analyzer CodeAnalyzer) CodeGenerator {
	// Note: Client is nil by default. Must be set via SetClient() for real code generation.
	// This allows testing without OpenCode server running.
	return &DefaultCodeGenerator{
		analyzer: analyzer,
		client:   nil,
	}
}

// NewCodeGeneratorWithClient creates a CodeGenerator with an existing OpenCode client
// Use this for production with real Claude/Copilot integration
func NewCodeGeneratorWithClient(analyzer CodeAnalyzer, client agent.ClientInterface) CodeGenerator {
	return &DefaultCodeGenerator{
		analyzer: analyzer,
		client:   client,
	}
}

// SetClient sets or updates the OpenCode client for code generation
func (g *DefaultCodeGenerator) SetClient(client agent.ClientInterface) {
	g.client = client
}

// GenerateCode generates code based on a task description using Claude
func (g *DefaultCodeGenerator) GenerateCode(ctx context.Context, task *CodeGenerationTask) (*GenerationResult, error) {
	startTime := time.Now()
	result := &GenerationResult{
		FilesCreated:  []string{},
		FilesModified: []string{},
	}

	// Build prompt
	prompt := buildCodeGenerationPrompt(task)

	// Call Claude API
	var attempts int
	var lastErr error

	maxRetries := task.MaxRetries
	if maxRetries == 0 {
		maxRetries = 1
	}

	for attempts = 0; attempts < maxRetries; attempts++ {
		var resp string
		var err error

		// Use OpenCode SDK if client is configured
		if g.client != nil {
			resp, err = g.generateViaOpenCode(ctx, prompt, task)
		} else {
			// Fallback to placeholder for testing without OpenCode server
			resp = g.generatePlaceholder(prompt, task)
			err = nil
		}

		if err != nil {
			lastErr = err
			continue
		}

		if resp != "" {
			result.GeneratedCode = resp
			result.FullOutput = resp
			result.Success = true

			// Try to extract and create files from the generated code
			files := extractFilesFromGeneration(result.GeneratedCode, task.Language)
			result.FilesCreated = files

			result.Duration = time.Since(startTime)
			result.Attempts = attempts + 1
			return result, nil
		}
	}

	result.Success = false
	result.Duration = time.Since(startTime)
	result.Attempts = attempts
	if lastErr != nil {
		result.ErrorMessage = fmt.Sprintf("Failed to generate code after %d attempts: %v", attempts, lastErr)
		return result, lastErr
	}

	result.ErrorMessage = "No code generated from Claude"
	return result, fmt.Errorf("no code generated")
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
	// Read existing code
	content, err := os.ReadFile(filePath)
	if err != nil {
		return &GenerationResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to read file: %v", err),
		}, err
	}

	task := &CodeGenerationTask{
		TaskID:       "refactor-code",
		Description:  "Refactor code: " + reason,
		Requirements: "Improve the following code by: " + reason,
		ExistingCode: string(content),
		Language:     "go",
	}

	result, err := g.GenerateCode(context.Background(), task)
	if result.Success {
		result.FilesModified = []string{filePath}
	}
	return result, err
}

// GenerateDocumentation generates documentation
func (g *DefaultCodeGenerator) GenerateDocumentation(ctx context.Context, filePath string) (*GenerationResult, error) {
	// Read the code file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return &GenerationResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to read file: %v", err),
		}, err
	}

	task := &CodeGenerationTask{
		TaskID:       "generate-docs",
		Description:  "Generate documentation for " + filePath,
		Requirements: "Create comprehensive documentation and comments for the following code",
		ExistingCode: string(content),
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

// generateViaOpenCode calls Claude/Copilot via OpenCode SDK
// OpenCode SDK handles authentication automatically
func (g *DefaultCodeGenerator) generateViaOpenCode(ctx context.Context, prompt string, task *CodeGenerationTask) (string, error) {
	if g.client == nil {
		return "", fmt.Errorf("OpenCode client not configured")
	}

	// Execute prompt via OpenCode SDK
	// OpenCode handles model selection (Claude or Copilot) based on configuration
	result, err := g.client.ExecutePrompt(ctx, prompt, &agent.PromptOptions{
		Model: "claude-3-5-sonnet",
		Title: task.Description,
	})

	if err != nil {
		return "", fmt.Errorf("OpenCode execution failed: %w", err)
	}

	if result == nil || len(result.Parts) == 0 {
		return "", fmt.Errorf("no output from OpenCode")
	}

	// Extract text from first part of response
	for _, part := range result.Parts {
		if part.Type == "text" && part.Text != "" {
			return part.Text, nil
		}
	}

	return "", fmt.Errorf("no text content in response")
}

// generatePlaceholder generates placeholder code for testing
// Used when OpenCode client is not configured (testing mode)
func (g *DefaultCodeGenerator) generatePlaceholder(prompt string, task *CodeGenerationTask) string {
	// Return realistic placeholder code based on language
	ext := ".go"
	if task.Language == "python" {
		ext = ".py"
	} else if task.Language == "typescript" || task.Language == "javascript" {
		ext = ".ts"
	} else if task.Language == "markdown" {
		ext = ".md"
	}

	placeholder := fmt.Sprintf(`// filepath: generated%s
// This is placeholder code generated in testing mode (no OpenCode client)
// In production, use SetClient() or NewCodeGeneratorWithClient() with an OpenCode client
// Task: %s
// Requirements: %s

package main

func main() {
	// TODO: Implement based on requirements
}
`, ext, task.Description, task.Requirements)

	return placeholder
}

// extractFilesFromGeneration extracts file paths from generated code
// Looks for patterns like "// filepath: path/to/file.go" or "# filename: file.go"
func extractFilesFromGeneration(generatedCode string, language string) []string {
	var files []string
	lines := strings.Split(generatedCode, "\n")

	// Look for file path markers
	commentMarker := "//"
	if language == "python" {
		commentMarker = "#"
	} else if language == "markdown" {
		return []string{"generated-doc.md"}
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Look for patterns like "// filepath: ..." or "// filename: ..."
		if strings.Contains(trimmed, commentMarker+" filepath:") || strings.Contains(trimmed, commentMarker+" filename:") {
			// Extract the file path
			parts := strings.SplitAfterN(trimmed, ":", 2)
			if len(parts) == 2 {
				filePath := strings.TrimSpace(parts[1])
				if filePath != "" {
					files = append(files, filePath)
				}
			}
		}
	}

	// If no files were explicitly marked, return a generated filename
	if len(files) == 0 {
		ext := ".go"
		if language == "python" {
			ext = ".py"
		} else if language == "typescript" || language == "javascript" {
			ext = ".ts"
		} else if language == "markdown" {
			ext = ".md"
		}
		files = append(files, "generated"+ext)
	}

	return files
}
