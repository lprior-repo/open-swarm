package temporal

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	opencode "github.com/sst/opencode-sdk-go"
)

// FileParseResult represents the outcome of parsing agent output for file paths
type FileParseResult struct {
	// ExtractedPaths are the file paths extracted from the agent output
	ExtractedPaths []string

	// ValidatedPaths are the extracted paths that match actual modified files
	ValidatedPaths []string

	// MissingPaths are extracted paths that don't match any modified files
	MissingPaths []string

	// UnexpectedPaths are modified files that weren't mentioned in the output
	UnexpectedPaths []string

	// Valid indicates if the parse was successful (has extracted paths)
	Valid bool

	// Warnings contains non-critical issues found during parsing
	Warnings []string
}

// OutputParser provides utilities for parsing agent output
type OutputParser struct{}

// NewOutputParser creates a new OutputParser instance
func NewOutputParser() *OutputParser {
	return &OutputParser{}
}

// ParseFilePaths extracts file paths from agent output text
// Handles multiple formats:
// - "FILE: path/to/file.go"
// - "Modified: path/to/file.go"
// - Standalone file paths with common extensions
// - Suffix matching against actual files
func (p *OutputParser) ParseFilePaths(rawOutput string, actualFiles []opencode.File) *FileParseResult {
	result := &FileParseResult{
		ExtractedPaths:  []string{},
		ValidatedPaths:  []string{},
		MissingPaths:    []string{},
		UnexpectedPaths: []string{},
		Warnings:        []string{},
		Valid:           false,
	}

	// Extract file paths from output
	extracted := p.extractPaths(rawOutput)

	// Remove duplicates
	extracted = removeDuplicates(extracted)
	result.ExtractedPaths = extracted

	// Build map of actual file paths for quick lookup
	actualPathMap := make(map[string]bool)
	for _, file := range actualFiles {
		if file.Path != "" {
			actualPathMap[file.Path] = true
		}
	}

	// If no paths extracted but we have actual files, try suffix matching
	if len(extracted) == 0 && len(actualFiles) > 0 {
		result.Warnings = append(result.Warnings, "No explicit file paths found in output, falling back to actual modified files")
		for _, file := range actualFiles {
			if file.Path != "" {
				result.ValidatedPaths = append(result.ValidatedPaths, file.Path)
			}
		}
		result.Valid = len(result.ValidatedPaths) > 0
		return result
	}

	// Validate extracted paths against actual files
	for _, extractedPath := range extracted {
		matched := false

		// Try exact match first
		if actualPathMap[extractedPath] {
			result.ValidatedPaths = append(result.ValidatedPaths, extractedPath)
			matched = true
			continue
		}

		// Try suffix matching (e.g., "foo.go" matches "pkg/foo.go")
		for actualPath := range actualPathMap {
			if strings.HasSuffix(actualPath, extractedPath) {
				result.ValidatedPaths = append(result.ValidatedPaths, actualPath)
				matched = true
				if actualPath != extractedPath {
					result.Warnings = append(result.Warnings, fmt.Sprintf("Matched '%s' to '%s' via suffix", extractedPath, actualPath))
				}
				break
			}
		}

		if !matched {
			result.MissingPaths = append(result.MissingPaths, extractedPath)
		}
	}

	// Find unexpected files (modified but not mentioned)
	validatedMap := make(map[string]bool)
	for _, path := range result.ValidatedPaths {
		validatedMap[path] = true
	}

	for actualPath := range actualPathMap {
		if !validatedMap[actualPath] {
			result.UnexpectedPaths = append(result.UnexpectedPaths, actualPath)
		}
	}

	if len(result.UnexpectedPaths) > 0 {
		result.Warnings = append(result.Warnings, fmt.Sprintf("%d file(s) modified but not mentioned in output", len(result.UnexpectedPaths)))
	}

	result.Valid = len(result.ValidatedPaths) > 0 || len(result.UnexpectedPaths) > 0

	return result
}

// extractPaths extracts file paths from raw text output
func (p *OutputParser) extractPaths(rawOutput string) []string {
	var paths []string

	lines := strings.Split(rawOutput, "\n")

	// Pattern 1: "FILE: path/to/file.go"
	filePrefixRegex := regexp.MustCompile(`(?i)FILE:\s*([^\s]+)`)

	// Pattern 2: "Modified: path/to/file.go" or "Created: path/to/file.go"
	modifiedPrefixRegex := regexp.MustCompile(`(?i)(Modified|Created|Updated|Changed):\s*([^\s]+)`)

	// Pattern 3: Standalone paths with common extensions
	// Note: Longer extensions must come first (tsx before ts, jsx before js)
	standalonePrefixRegex := regexp.MustCompile(`([a-zA-Z0-9_\-./]+\.(tsx|jsx|go|ts|js|py|java|cpp|c|h|rs|rb|php|cs|swift))`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Try FILE: prefix
		if matches := filePrefixRegex.FindStringSubmatch(line); len(matches) > 1 {
			path := strings.TrimSpace(matches[1])
			if path != "" {
				paths = append(paths, path)
			}
			continue
		}

		// Try Modified/Created prefix
		if matches := modifiedPrefixRegex.FindStringSubmatch(line); len(matches) > 2 {
			path := strings.TrimSpace(matches[2])
			if path != "" {
				paths = append(paths, path)
			}
			continue
		}

		// Try standalone file paths (be conservative to avoid false positives)
		if matches := standalonePrefixRegex.FindAllStringSubmatch(line, -1); len(matches) > 0 {
			for _, match := range matches {
				if len(match) > 1 {
					path := strings.TrimSpace(match[1])
					// Only add if it looks like a real path (contains / or is just a filename)
					if path != "" && (strings.Contains(path, "/") || !strings.Contains(path, " ")) {
						paths = append(paths, path)
					}
				}
			}
		}
	}

	return paths
}

// GetAllModifiedPaths returns all paths that should be tracked (validated + unexpected)
// This is useful when you want to track all actual changes regardless of what the agent reported
func (p *OutputParser) GetAllModifiedPaths(result *FileParseResult) []string {
	allPaths := make([]string, 0, len(result.ValidatedPaths)+len(result.UnexpectedPaths))
	allPaths = append(allPaths, result.ValidatedPaths...)
	allPaths = append(allPaths, result.UnexpectedPaths...)
	return removeDuplicates(allPaths)
}

// MatchPattern checks if a file path matches a glob pattern
// Useful for file locking and validation
func (p *OutputParser) MatchPattern(filePath, pattern string) (bool, error) {
	matched, err := filepath.Match(pattern, filePath)
	if err != nil {
		return false, fmt.Errorf("failed to match pattern %q against %q: %w", pattern, filePath, err)
	}
	if matched {
		return true, nil
	}

	// Also try matching just the basename
	basename := filepath.Base(filePath)
	matched, err = filepath.Match(pattern, basename)
	if err != nil {
		return false, fmt.Errorf("failed to match pattern %q against basename %q: %w", pattern, basename, err)
	}

	return matched, nil
}

// removeDuplicates removes duplicate strings while preserving order
func removeDuplicates(items []string) []string {
	seen := make(map[string]bool)
	result := []string{}

	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}
