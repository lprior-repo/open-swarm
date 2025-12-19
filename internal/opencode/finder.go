// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package opencode

import (
	"context"
	"fmt"
)

// FileMatch represents a file found by the finder.
type FileMatch struct {
	Path string // File path relative to project root
	Size int64  // File size in bytes
}

// SymbolMatch represents a code symbol found by the finder.
type SymbolMatch struct {
	Name       string // Symbol name (function, type, variable, etc.)
	Type       string // Symbol type (function, class, method, variable, etc.)
	File       string // File path where symbol is defined
	Line       int    // Line number in file
	Definition string // Symbol definition/signature
}

// TextMatch represents a text pattern found in files.
type TextMatch struct {
	File    string // File path
	Line    int    // Line number
	Column  int    // Column number
	Text    string // Line content
	Snippet string // Context around match (3 lines)
}

// CodeFinder provides smart code search capabilities using OpenCode SDK.
// It wraps the Find service to help agents navigate and understand codebases
// without manually reading files.
type CodeFinder interface {
	// FindFiles searches for files matching a glob pattern.
	// Example: "*.go", "**/test/**", "src/**/*.tsx"
	FindFiles(ctx context.Context, pattern string) ([]FileMatch, error)

	// FindSymbols searches for code symbols by name or pattern.
	// Finds functions, types, methods, variables, constants, etc.
	// Example: "Handler", "MyClass", "setupServer"
	FindSymbols(ctx context.Context, query string) ([]SymbolMatch, error)

	// FindText searches for text patterns using regex.
	// Useful for finding error handling patterns, specific keywords, etc.
	// Example: "return.*Errorf", "async function.*", "@deprecated"
	FindText(ctx context.Context, pattern string) ([]TextMatch, error)
}

// DefaultCodeFinder implements CodeFinder interface.
// Note: This is a placeholder implementation. A real implementation would
// integrate with the OpenCode SDK's Find service.
type DefaultCodeFinder struct {
	// In a real implementation, this would hold the OpenCode client
	// client *opencode.Client
}

// NewCodeFinder creates a new code finder instance.
func NewCodeFinder() CodeFinder {
	return &DefaultCodeFinder{}
}

// FindFiles searches for files matching the given pattern.
func (f *DefaultCodeFinder) FindFiles(ctx context.Context, pattern string) ([]FileMatch, error) {
	if pattern == "" {
		return nil, fmt.Errorf("pattern cannot be empty")
	}

	// TODO: Integrate with OpenCode SDK Find.Files()
	// Example integration (when SDK available):
	/*
	results, err := f.client.Find.Files(ctx, opencode.FindFilesParams{
		Query: opencode.F(pattern),
	})
	if err != nil {
		return nil, err
	}

	matches := make([]FileMatch, 0, len(*results))
	for _, r := range *results {
		matches = append(matches, FileMatch{
			Path: r.Path,
			Size: r.Size,
		})
	}
	return matches, nil
	*/

	return []FileMatch{}, nil
}

// FindSymbols searches for code symbols matching the given query.
func (f *DefaultCodeFinder) FindSymbols(ctx context.Context, query string) ([]SymbolMatch, error) {
	if query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}

	// TODO: Integrate with OpenCode SDK Find.Symbols()
	// Example integration (when SDK available):
	/*
	results, err := f.client.Find.Symbols(ctx, opencode.FindSymbolsParams{
		Query: opencode.F(query),
	})
	if err != nil {
		return nil, err
	}

	matches := make([]SymbolMatch, 0, len(*results))
	for _, r := range *results {
		matches = append(matches, SymbolMatch{
			Name:       r.Name,
			Type:       r.Type,
			File:       r.File,
			Line:       r.Line,
			Definition: r.Definition,
		})
	}
	return matches, nil
	*/

	return []SymbolMatch{}, nil
}

// FindText searches for text patterns using regex.
func (f *DefaultCodeFinder) FindText(ctx context.Context, pattern string) ([]TextMatch, error) {
	if pattern == "" {
		return nil, fmt.Errorf("pattern cannot be empty")
	}

	// TODO: Integrate with OpenCode SDK Find.Text()
	// Example integration (when SDK available):
	/*
	results, err := f.client.Find.Text(ctx, opencode.FindTextParams{
		Pattern: opencode.F(pattern),
	})
	if err != nil {
		return nil, err
	}

	matches := make([]TextMatch, 0, len(*results))
	for _, r := range *results {
		matches = append(matches, TextMatch{
			File:    r.File,
			Line:    r.Line,
			Column:  r.Column,
			Text:    r.Text,
			Snippet: r.Snippet,
		})
	}
	return matches, nil
	*/

	return []TextMatch{}, nil
}

// Usage examples for agents:
//
// Finding all handler functions:
//   handlers, err := finder.FindSymbols(ctx, "Handler")
//   // Agent gets [PostHandler, GetHandler, DeleteHandler, etc.]
//
// Finding all error handling patterns:
//   errors, err := finder.FindText(ctx, "return.*Errorf")
//   // Agent sees all error handling in codebase
//
// Finding test files:
//   tests, err := finder.FindFiles(ctx, "**/*_test.go")
//   // Agent knows which files have tests
//
// Finding middleware functions:
//   middleware, err := finder.FindSymbols(ctx, "Middleware")
//   // Agent discovers middleware pattern in codebase
