// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package opencode

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
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

// DefaultCodeGenerator implements CodeGenerator with Claude AI integration
type DefaultCodeGenerator struct {
	analyzer CodeAnalyzer
	apiKey   string
	apiURL   string
}

// NewCodeGenerator creates a new CodeGenerator with Claude AI support
func NewCodeGenerator(analyzer CodeAnalyzer) CodeGenerator {
	// Initialize from environment
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	apiURL := "https://api.anthropic.com/v1/messages"

	return &DefaultCodeGenerator{
		analyzer: analyzer,
		apiKey:   apiKey,
		apiURL:   apiURL,
	}
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
		// Call Claude API via HTTP
		resp, err := g.callClaudeAPI(ctx, prompt)
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

// callClaudeAPI makes an HTTP call to Claude API
func (g *DefaultCodeGenerator) callClaudeAPI(ctx context.Context, prompt string) (string, error) {
	// If no API key, return placeholder
	if g.apiKey == "" {
		return fmt.Sprintf("// Generated code (no API key configured)\n// Prompt: %s\npackage main\n", prompt), nil
	}

	// Prepare request payload
	payload := map[string]interface{}{
		"model":      "claude-3-5-sonnet-20241022",
		"max_tokens": 4096,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": prompt,
			},
		},
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", g.apiURL, bytes.NewReader(jsonPayload))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", g.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	// Make request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call API: %w", err)
	}
	defer resp.Body.Close()

	// Check status
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	// Parse response
	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	// Extract content
	if content, ok := response["content"].([]interface{}); ok && len(content) > 0 {
		if firstContent, ok := content[0].(map[string]interface{}); ok {
			if text, ok := firstContent["text"].(string); ok {
				return text, nil
			}
		}
	}

	return "", fmt.Errorf("no text content in response")
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
