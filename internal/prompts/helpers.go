package prompts

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LoadFileContent loads the content of a file and returns a CodeContext
func LoadFileContent(filePath string) (CodeContext, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return CodeContext{}, fmt.Errorf("failed to read file: %w", err)
	}

	ext := filepath.Ext(filePath)
	lang := getLanguageFromExtension(ext)
	pkg := extractGoPackageName(string(content))

	return CodeContext{
		FilePath:    filePath,
		FileContent: string(content),
		Language:    lang,
		PackageName: pkg,
	}, nil
}

// LoadFileWithDiff loads both the file content and git diff
func LoadFileWithDiff(filePath string, diff string) (CodeContext, error) {
	ctx, err := LoadFileContent(filePath)
	if err != nil {
		return CodeContext{}, err
	}
	ctx.Diff = diff
	return ctx, nil
}

// ExtractSurroundingContext extracts surrounding code context for a specific line range
func ExtractSurroundingContext(filePath string, startLine, endLine, contextLines int) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var lines []string
	lineNum := 1

	// Calculate range with context
	contextStart := max(1, startLine-contextLines)
	contextEnd := endLine + contextLines

	for scanner.Scan() {
		if lineNum >= contextStart && lineNum <= contextEnd {
			lines = append(lines, scanner.Text())
		}
		if lineNum > contextEnd {
			break
		}
		lineNum++
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	return strings.Join(lines, "\n"), nil
}

// NewReviewRequest creates a new ReviewRequest with common defaults
func NewReviewRequest(reviewType ReviewType, taskID, description string) ReviewRequest {
	return ReviewRequest{
		Type:            reviewType,
		TaskID:          taskID,
		TaskDescription: description,
		RequireVote:     true,
	}
}

// WithCodeFile adds file content to a ReviewRequest
func (r ReviewRequest) WithCodeFile(filePath string) (ReviewRequest, error) {
	ctx, err := LoadFileContent(filePath)
	if err != nil {
		return r, err
	}
	r.CodeContext = ctx
	return r, nil
}

// WithDiff adds a git diff to a ReviewRequest
func (r ReviewRequest) WithDiff(diff string) ReviewRequest {
	r.CodeContext.Diff = diff
	return r
}

// WithAcceptanceCriteria adds acceptance criteria to a ReviewRequest
func (r ReviewRequest) WithAcceptanceCriteria(criteria ...string) ReviewRequest {
	r.AcceptanceCriteria = criteria
	return r
}

// WithAdditionalContext adds additional context to a ReviewRequest
func (r ReviewRequest) WithAdditionalContext(context string) ReviewRequest {
	r.AdditionalContext = context
	return r
}

// WithoutVote disables the vote requirement
func (r ReviewRequest) WithoutVote() ReviewRequest {
	r.RequireVote = false
	return r
}

// getLanguageFromExtension returns the language name for a file extension
func getLanguageFromExtension(ext string) string {
	switch ext {
	case ".go":
		return "go"
	case ".js", ".jsx":
		return "javascript"
	case ".ts", ".tsx":
		return "typescript"
	case ".py":
		return "python"
	case ".java":
		return "java"
	case ".rs":
		return "rust"
	case ".c", ".h":
		return "c"
	case ".cpp", ".cc", ".cxx", ".hpp":
		return "cpp"
	case ".rb":
		return "ruby"
	case ".php":
		return "php"
	case ".swift":
		return "swift"
	case ".kt", ".kts":
		return "kotlin"
	default:
		return "text"
	}
}

// extractGoPackageName extracts the package name from Go source code
func extractGoPackageName(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "package ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return parts[1]
			}
		}
	}
	return ""
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
