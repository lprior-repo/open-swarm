package temporal

import (
	"fmt"
	"regexp"
	"strings"

	opencode "github.com/sst/opencode-sdk-go"

	"open-swarm/internal/patternmatch"
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
	extracted = removeDuplicates(extracted)
	result.ExtractedPaths = extracted

	// Build map of actual file paths for quick lookup
	actualPathMap := p.buildActualPathMap(actualFiles)

	// Handle case with no extracted paths
	if len(extracted) == 0 {
		p.handleNoExtractedPaths(result, actualFiles)
		return result
	}

	// Validate extracted paths against actual files
	p.validateExtractedPaths(result, extracted, actualPathMap)

	// Find and record unexpected files
	p.findUnexpectedFiles(result, actualPathMap)

	result.Valid = len(result.ValidatedPaths) > 0 || len(result.UnexpectedPaths) > 0
	return result
}

// buildActualPathMap creates a map for fast lookup of actual file paths
func (p *OutputParser) buildActualPathMap(actualFiles []opencode.File) map[string]bool {
	pathMap := make(map[string]bool)
	for _, file := range actualFiles {
		if file.Path != "" {
			pathMap[file.Path] = true
		}
	}
	return pathMap
}

// handleNoExtractedPaths handles the case when no paths are extracted from output
func (p *OutputParser) handleNoExtractedPaths(result *FileParseResult, actualFiles []opencode.File) {
	if len(actualFiles) > 0 {
		result.Warnings = append(result.Warnings, "No explicit file paths found in output, falling back to actual modified files")
		for _, file := range actualFiles {
			if file.Path != "" {
				result.ValidatedPaths = append(result.ValidatedPaths, file.Path)
			}
		}
	}
	result.Valid = len(result.ValidatedPaths) > 0
}

// validateExtractedPaths validates extracted paths against actual files
func (p *OutputParser) validateExtractedPaths(result *FileParseResult, extracted []string, actualPathMap map[string]bool) {
	for _, extractedPath := range extracted {
		// Try exact match first
		if actualPathMap[extractedPath] {
			result.ValidatedPaths = append(result.ValidatedPaths, extractedPath)
			continue
		}

		// Try suffix matching (e.g., "foo.go" matches "pkg/foo.go")
		matched := p.trySuffixMatch(result, extractedPath, actualPathMap)
		if !matched {
			result.MissingPaths = append(result.MissingPaths, extractedPath)
		}
	}
}

// trySuffixMatch attempts to match a path using suffix matching
func (p *OutputParser) trySuffixMatch(result *FileParseResult, extractedPath string, actualPathMap map[string]bool) bool {
	for actualPath := range actualPathMap {
		if strings.HasSuffix(actualPath, extractedPath) {
			result.ValidatedPaths = append(result.ValidatedPaths, actualPath)
			if actualPath != extractedPath {
				result.Warnings = append(result.Warnings, fmt.Sprintf("Matched '%s' to '%s' via suffix", extractedPath, actualPath))
			}
			return true
		}
	}
	return false
}

// findUnexpectedFiles finds modified files not mentioned in output
func (p *OutputParser) findUnexpectedFiles(result *FileParseResult, actualPathMap map[string]bool) {
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
}

// extractPaths extracts file paths from raw text output
func (p *OutputParser) extractPaths(rawOutput string) []string {
	var paths []string
	lines := strings.Split(rawOutput, "\n")

	// Compile regexes once
	filePrefixRegex := regexp.MustCompile(`(?i)FILE:\s*([^\s]+)`)
	modifiedPrefixRegex := regexp.MustCompile(`(?i)(Modified|Created|Updated|Changed):\s*([^\s]+)`)
	standalonePrefixRegex := regexp.MustCompile(`([a-zA-Z0-9_\-./]+\.(tsx|jsx|go|ts|js|py|java|cpp|c|h|rs|rb|php|cs|swift))`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		p.extractFromLine(&paths, line, filePrefixRegex, modifiedPrefixRegex, standalonePrefixRegex)
	}

	return paths
}

// extractFromLine tries to extract paths from a single line using various patterns
func (p *OutputParser) extractFromLine(paths *[]string, line string, fileRegex, modifiedRegex, standaloneRegex *regexp.Regexp) {
	// Try FILE: prefix
	if matches := fileRegex.FindStringSubmatch(line); len(matches) > 1 {
		if path := strings.TrimSpace(matches[1]); path != "" {
			*paths = append(*paths, path)
		}
		return
	}

	// Try Modified/Created prefix
	if matches := modifiedRegex.FindStringSubmatch(line); len(matches) > 2 {
		if path := strings.TrimSpace(matches[2]); path != "" {
			*paths = append(*paths, path)
		}
		return
	}

	// Try standalone file paths (be conservative to avoid false positives)
	p.extractStandalonePaths(paths, line, standaloneRegex)
}

// extractStandalonePaths extracts standalone file paths from a line
func (p *OutputParser) extractStandalonePaths(paths *[]string, line string, regex *regexp.Regexp) {
	matches := regex.FindAllStringSubmatch(line, -1)
	if len(matches) == 0 {
		return
	}

	for _, match := range matches {
		if len(match) > 1 {
			path := strings.TrimSpace(match[1])
			// Only add if it looks like a real path (contains / or is just a filename)
			if path != "" && (strings.Contains(path, "/") || !strings.Contains(path, " ")) {
				*paths = append(*paths, path)
			}
		}
	}
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
	return patternmatch.Match(filePath, pattern)
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
