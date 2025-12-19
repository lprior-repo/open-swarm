// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package opencode

import (
	"context"
	"testing"
)

func TestNewCodeFinder(t *testing.T) {
	finder := NewCodeFinder()
	if finder == nil {
		t.Fatal("NewCodeFinder returned nil")
	}

	_, ok := finder.(*DefaultCodeFinder)
	if !ok {
		t.Errorf("expected *DefaultCodeFinder, got %T", finder)
	}
}

func TestFindFiles_EmptyPattern(t *testing.T) {
	ctx := context.Background()
	finder := NewCodeFinder()

	_, err := finder.FindFiles(ctx, "")
	if err == nil {
		t.Errorf("expected error for empty pattern")
	}
}

func TestFindFiles_ValidPattern(t *testing.T) {
	ctx := context.Background()
	finder := NewCodeFinder()

	matches, err := finder.FindFiles(ctx, "*.go")
	if err != nil {
		t.Fatalf("FindFiles failed: %v", err)
	}

	// Current implementation returns empty slice (placeholder)
	if len(matches) != 0 {
		t.Errorf("expected empty matches from placeholder implementation, got %d", len(matches))
	}
}

func TestFindFiles_GlobPatterns(t *testing.T) {
	ctx := context.Background()
	finder := NewCodeFinder()

	patterns := []string{
		"*.go",
		"**/test/**",
		"src/**/*.tsx",
		"**/*_test.go",
	}

	for _, pattern := range patterns {
		_, err := finder.FindFiles(ctx, pattern)
		if err != nil {
			t.Errorf("FindFiles with pattern %s failed: %v", pattern, err)
		}
	}
}

func TestFindSymbols_EmptyQuery(t *testing.T) {
	ctx := context.Background()
	finder := NewCodeFinder()

	_, err := finder.FindSymbols(ctx, "")
	if err == nil {
		t.Errorf("expected error for empty query")
	}
}

func TestFindSymbols_ValidQuery(t *testing.T) {
	ctx := context.Background()
	finder := NewCodeFinder()

	matches, err := finder.FindSymbols(ctx, "Handler")
	if err != nil {
		t.Fatalf("FindSymbols failed: %v", err)
	}

	// Current implementation returns empty slice (placeholder)
	if len(matches) != 0 {
		t.Errorf("expected empty matches from placeholder implementation, got %d", len(matches))
	}
}

func TestFindSymbols_VariousQueries(t *testing.T) {
	ctx := context.Background()
	finder := NewCodeFinder()

	queries := []string{
		"Handler",
		"MyClass",
		"setupServer",
		"ErrorHandler",
		"processRequest",
	}

	for _, query := range queries {
		_, err := finder.FindSymbols(ctx, query)
		if err != nil {
			t.Errorf("FindSymbols with query %s failed: %v", query, err)
		}
	}
}

func TestFindText_EmptyPattern(t *testing.T) {
	ctx := context.Background()
	finder := NewCodeFinder()

	_, err := finder.FindText(ctx, "")
	if err == nil {
		t.Errorf("expected error for empty pattern")
	}
}

func TestFindText_ValidPattern(t *testing.T) {
	ctx := context.Background()
	finder := NewCodeFinder()

	matches, err := finder.FindText(ctx, "return.*Errorf")
	if err != nil {
		t.Fatalf("FindText failed: %v", err)
	}

	// Current implementation returns empty slice (placeholder)
	if len(matches) != 0 {
		t.Errorf("expected empty matches from placeholder implementation, got %d", len(matches))
	}
}

func TestFindText_RegexPatterns(t *testing.T) {
	ctx := context.Background()
	finder := NewCodeFinder()

	patterns := []string{
		"return.*Errorf",
		"async function.*",
		"@deprecated",
		"TODO.*",
		"\\berror\\b",
	}

	for _, pattern := range patterns {
		_, err := finder.FindText(ctx, pattern)
		if err != nil {
			t.Errorf("FindText with pattern %s failed: %v", pattern, err)
		}
	}
}

func TestCodeFinderInterface_FileMatch(t *testing.T) {
	match := FileMatch{
		Path: "src/main.go",
		Size: 1024,
	}

	if match.Path != "src/main.go" {
		t.Errorf("expected Path src/main.go, got %s", match.Path)
	}

	if match.Size != 1024 {
		t.Errorf("expected Size 1024, got %d", match.Size)
	}
}

func TestCodeFinderInterface_SymbolMatch(t *testing.T) {
	match := SymbolMatch{
		Name:       "ProcessRequest",
		Type:       "function",
		File:       "src/handler.go",
		Line:       42,
		Definition: "func ProcessRequest(req *Request) error",
	}

	if match.Name != "ProcessRequest" {
		t.Errorf("expected Name ProcessRequest, got %s", match.Name)
	}

	if match.Type != "function" {
		t.Errorf("expected Type function, got %s", match.Type)
	}

	if match.Line != 42 {
		t.Errorf("expected Line 42, got %d", match.Line)
	}
}

func TestCodeFinderInterface_TextMatch(t *testing.T) {
	match := TextMatch{
		File:    "src/handler.go",
		Line:    42,
		Column:  5,
		Text:    "\treturn fmt.Errorf(\"error: %v\", err)",
		Snippet: "if err != nil {\n\treturn fmt.Errorf(\"error: %v\", err)\n}",
	}

	if match.File != "src/handler.go" {
		t.Errorf("expected File src/handler.go, got %s", match.File)
	}

	if match.Line != 42 {
		t.Errorf("expected Line 42, got %d", match.Line)
	}

	if match.Column != 5 {
		t.Errorf("expected Column 5, got %d", match.Column)
	}
}

func TestDefaultCodeFinder_Implementation(t *testing.T) {
	ctx := context.Background()
	finder := &DefaultCodeFinder{}

	// Verify it implements CodeFinder interface
	var _ CodeFinder = finder

	// Test all methods complete without panic
	_, _ = finder.FindFiles(ctx, "*.go")
	_, _ = finder.FindSymbols(ctx, "Handler")
	_, _ = finder.FindText(ctx, "error")
}

func TestFindFilesMultipleCalls(t *testing.T) {
	ctx := context.Background()
	finder := NewCodeFinder()

	// Multiple calls should be safe
	for i := 0; i < 3; i++ {
		_, err := finder.FindFiles(ctx, "*.go")
		if err != nil {
			t.Errorf("call %d failed: %v", i+1, err)
		}
	}
}

func TestFindSymbolsMultipleCalls(t *testing.T) {
	ctx := context.Background()
	finder := NewCodeFinder()

	queries := []string{"Handler", "Processor", "Manager"}
	for _, query := range queries {
		_, err := finder.FindSymbols(ctx, query)
		if err != nil {
			t.Errorf("FindSymbols(%s) failed: %v", query, err)
		}
	}
}

func TestFindTextMultipleCalls(t *testing.T) {
	ctx := context.Background()
	finder := NewCodeFinder()

	patterns := []string{"error", "return", "defer"}
	for _, pattern := range patterns {
		_, err := finder.FindText(ctx, pattern)
		if err != nil {
			t.Errorf("FindText(%s) failed: %v", pattern, err)
		}
	}
}

func TestContextHandling(t *testing.T) {
	finder := NewCodeFinder()

	// Test with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Current implementation should handle gracefully
	_, err := finder.FindFiles(ctx, "*.go")
	if err != nil && err == context.Canceled {
		// This is acceptable - SDK would propagate context cancellation
	}
}

func TestFileMatchFields(t *testing.T) {
	tests := []struct {
		name string
		path string
		size int64
	}{
		{"simple file", "main.go", 100},
		{"nested file", "src/internal/handler.go", 5000},
		{"large file", "vendor/lib/package.go", 1000000},
		{"zero size", "empty.go", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match := FileMatch{
				Path: tt.path,
				Size: tt.size,
			}

			if match.Path != tt.path {
				t.Errorf("expected Path %s, got %s", tt.path, match.Path)
			}

			if match.Size != tt.size {
				t.Errorf("expected Size %d, got %d", tt.size, match.Size)
			}
		})
	}
}

func TestSymbolMatchFields(t *testing.T) {
	tests := []struct {
		name string
		sType string
		line int
	}{
		{"function", "function", 10},
		{"method", "method", 42},
		{"class", "class", 5},
		{"variable", "variable", 100},
		{"constant", "constant", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match := SymbolMatch{
				Name:       "TestSymbol",
				Type:       tt.sType,
				File:       "test.go",
				Line:       tt.line,
				Definition: "test definition",
			}

			if match.Type != tt.sType {
				t.Errorf("expected Type %s, got %s", tt.sType, match.Type)
			}

			if match.Line != tt.line {
				t.Errorf("expected Line %d, got %d", tt.line, match.Line)
			}
		})
	}
}

func TestTextMatchFields(t *testing.T) {
	tests := []struct {
		name   string
		line   int
		column int
		text   string
	}{
		{"first line", 1, 0, "package main"},
		{"mid file", 42, 5, "\treturn err"},
		{"end of file", 999, 10, "// EOF"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match := TextMatch{
				File:    "test.go",
				Line:    tt.line,
				Column:  tt.column,
				Text:    tt.text,
				Snippet: tt.text,
			}

			if match.Line != tt.line {
				t.Errorf("expected Line %d, got %d", tt.line, match.Line)
			}

			if match.Column != tt.column {
				t.Errorf("expected Column %d, got %d", tt.column, match.Column)
			}

			if match.Text != tt.text {
				t.Errorf("expected Text %s, got %s", tt.text, match.Text)
			}
		})
	}
}
