// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package opencode

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
)

// CodeAnalyzer provides code analysis capabilities for agents.
// Uses treesitter for parsing and structural analysis.
type CodeAnalyzer interface {
	// AnalyzeFile analyzes a single source file and returns structure
	AnalyzeFile(ctx context.Context, filePath string) (*CodeAnalysis, error)

	// AnalyzeFiles analyzes multiple files and returns their relationships
	AnalyzeFiles(ctx context.Context, filePaths []string) (*ProjectAnalysis, error)

	// FindSymbols finds all symbols (functions, types, etc.) in a file
	FindSymbols(ctx context.Context, filePath string) ([]*Symbol, error)

	// FindReferences finds all uses of a symbol
	FindReferences(ctx context.Context, filePath string, symbolName string) ([]*Reference, error)

	// ValidateSyntax checks if code is syntactically valid
	ValidateSyntax(ctx context.Context, filePath string) (bool, []SyntaxError, error)

	// GetCodeComplexity analyzes code complexity (cyclomatic, etc.)
	GetCodeComplexity(ctx context.Context, filePath string) (*ComplexityMetrics, error)
}

// CodeAnalysis represents analysis of a single file
type CodeAnalysis struct {
	FilePath   string           // Path to the file
	Language   string           // Programming language
	Symbols    []*Symbol        // Top-level symbols in file
	Imports    []string         // Imported packages/modules
	IsValid    bool             // Whether code is syntactically valid
	Issues     []SyntaxError    // Any syntax issues
	Complexity ComplexityMetrics // Code complexity metrics
}

// ProjectAnalysis represents analysis of multiple files
type ProjectAnalysis struct {
	Files        []*CodeAnalysis // Analysis of each file
	Relationships []Dependency    // Dependencies between files
	MainPackages []string        // Main entry points
}

// Symbol represents a code symbol (function, type, variable, etc.)
type Symbol struct {
	Name       string        // Symbol name
	Kind       SymbolKind    // Type of symbol (function, class, etc.)
	FilePath   string        // Where it's defined
	LineNumber int           // Line number of definition
	EndLine    int           // End line of definition
	Parameters []Parameter   // For functions: parameters
	ReturnType string        // For functions: return type
}

// SymbolKind represents the type of a symbol
type SymbolKind string

const (
	SymbolFunction   SymbolKind = "function"
	SymbolType       SymbolKind = "type"
	SymbolInterface  SymbolKind = "interface"
	SymbolClass      SymbolKind = "class"
	SymbolMethod     SymbolKind = "method"
	SymbolVariable   SymbolKind = "variable"
	SymbolConstant   SymbolKind = "constant"
	SymbolPackage    SymbolKind = "package"
)

// Parameter represents a function parameter
type Parameter struct {
	Name string // Parameter name
	Type string // Parameter type
}

// Reference represents a use of a symbol
type Reference struct {
	FilePath   string // Where it's referenced
	LineNumber int    // Line number
	ColumnNum  int    // Column number
	Context    string // Code context around the reference
}

// SyntaxError represents a syntax error in code
type SyntaxError struct {
	LineNumber int    // Line with error
	Column     int    // Column with error
	Message    string // Error description
}

// Dependency represents a dependency between files
type Dependency struct {
	From string // Source file
	To   string // Target file
	Type string // Type of dependency (import, call, etc.)
}

// ComplexityMetrics represents code complexity measurements
type ComplexityMetrics struct {
	CyclomaticComplexity int    // Cyclomatic complexity
	LinesOfCode          int    // Total lines of code
	Functions            int    // Number of functions
	AverageFunctionSize  int    // Average function size
	NestedDepth          int    // Max nesting depth
	Issues               string // Any complexity issues detected
}

// DefaultCodeAnalyzer implements CodeAnalyzer using Go's ast package
type DefaultCodeAnalyzer struct {
	// Uses Go's built-in ast parser for code analysis
}

// NewCodeAnalyzer creates a new CodeAnalyzer
func NewCodeAnalyzer() CodeAnalyzer {
	return &DefaultCodeAnalyzer{}
}

// AnalyzeFile analyzes a single Go file
func (a *DefaultCodeAnalyzer) AnalyzeFile(ctx context.Context, filePath string) (*CodeAnalysis, error) {
	// Read the file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return &CodeAnalysis{
			FilePath: filePath,
			Language: "go",
			IsValid:  false,
			Issues: []SyntaxError{{
				LineNumber: 0,
				Column:     0,
				Message:    "Failed to read file: " + err.Error(),
			}},
		}, nil
	}

	// Parse the file
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, content, parser.AllErrors)

	analysis := &CodeAnalysis{
		FilePath:   filePath,
		Language:   "go",
		Symbols:    []*Symbol{},
		Imports:    []string{},
		Issues:     []SyntaxError{},
		Complexity: ComplexityMetrics{},
	}

	// If parsing failed, record syntax errors
	if err != nil {
		analysis.IsValid = false
		analysis.Issues = append(analysis.Issues, SyntaxError{
			LineNumber: 1,
			Column:     0,
			Message:    err.Error(),
		})
		return analysis, nil
	}

	analysis.IsValid = true

	// Extract imports
	for _, decl := range file.Decls {
		if gen, ok := decl.(*ast.GenDecl); ok && gen.Tok == token.IMPORT {
			for _, spec := range gen.Specs {
				if ispec, ok := spec.(*ast.ImportSpec); ok {
					path := strings.Trim(ispec.Path.Value, "\"")
					analysis.Imports = append(analysis.Imports, path)
				}
			}
		}
	}

	// Extract symbols
	for _, decl := range file.Decls {
		switch decl := decl.(type) {
		case *ast.FuncDecl:
			symbol := &Symbol{
				Name:       decl.Name.Name,
				Kind:       SymbolFunction,
				FilePath:   filePath,
				LineNumber: fset.Position(decl.Pos()).Line,
				EndLine:    fset.Position(decl.End()).Line,
				ReturnType: typeToString(decl.Type.Results),
			}
			if decl.Recv != nil {
				symbol.Kind = SymbolMethod
			}
			analysis.Symbols = append(analysis.Symbols, symbol)

		case *ast.GenDecl:
			// Handle type declarations
			if decl.Tok == token.TYPE {
				for _, spec := range decl.Specs {
					if tspec, ok := spec.(*ast.TypeSpec); ok {
						symbol := &Symbol{
							Name:       tspec.Name.Name,
							Kind:       typeToSymbolKind(tspec.Type),
							FilePath:   filePath,
							LineNumber: fset.Position(tspec.Pos()).Line,
							EndLine:    fset.Position(tspec.End()).Line,
						}
						analysis.Symbols = append(analysis.Symbols, symbol)
					}
				}
			}
			// Handle constants and variables
			if decl.Tok == token.CONST || decl.Tok == token.VAR {
				for _, spec := range decl.Specs {
					if vspec, ok := spec.(*ast.ValueSpec); ok {
						kind := SymbolVariable
						if decl.Tok == token.CONST {
							kind = SymbolConstant
						}
						for _, name := range vspec.Names {
							analysis.Symbols = append(analysis.Symbols, &Symbol{
								Name:       name.Name,
								Kind:       kind,
								FilePath:   filePath,
								LineNumber: fset.Position(vspec.Pos()).Line,
								EndLine:    fset.Position(vspec.End()).Line,
							})
						}
					}
				}
			}
		}
	}

	// Calculate complexity metrics
	analysis.Complexity = calculateComplexity(file, content)

	return analysis, nil
}

// AnalyzeFiles analyzes multiple files
func (a *DefaultCodeAnalyzer) AnalyzeFiles(ctx context.Context, filePaths []string) (*ProjectAnalysis, error) {
	analysis := &ProjectAnalysis{
		Files:         []*CodeAnalysis{},
		Relationships: []Dependency{},
		MainPackages:  []string{},
	}

	for _, filePath := range filePaths {
		fileAnalysis, err := a.AnalyzeFile(ctx, filePath)
		if err != nil {
			continue
		}
		analysis.Files = append(analysis.Files, fileAnalysis)
	}

	return analysis, nil
}

// FindSymbols finds symbols in a file
func (a *DefaultCodeAnalyzer) FindSymbols(ctx context.Context, filePath string) ([]*Symbol, error) {
	analysis, err := a.AnalyzeFile(ctx, filePath)
	if err != nil {
		return nil, err
	}
	return analysis.Symbols, nil
}

// FindReferences finds uses of a symbol (simplified implementation)
func (a *DefaultCodeAnalyzer) FindReferences(ctx context.Context, filePath string, symbolName string) ([]*Reference, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	var references []*Reference

	for lineNum, line := range lines {
		if strings.Contains(line, symbolName) {
			// Find column position
			col := strings.Index(line, symbolName)
			references = append(references, &Reference{
				FilePath:   filePath,
				LineNumber: lineNum + 1,
				ColumnNum:  col,
				Context:    strings.TrimSpace(line),
			})
		}
	}

	return references, nil
}

// ValidateSyntax validates code syntax
func (a *DefaultCodeAnalyzer) ValidateSyntax(ctx context.Context, filePath string) (bool, []SyntaxError, error) {
	analysis, err := a.AnalyzeFile(ctx, filePath)
	if err != nil {
		return false, nil, err
	}
	return analysis.IsValid, analysis.Issues, nil
}

// GetCodeComplexity calculates complexity metrics
func (a *DefaultCodeAnalyzer) GetCodeComplexity(ctx context.Context, filePath string) (*ComplexityMetrics, error) {
	analysis, err := a.AnalyzeFile(ctx, filePath)
	if err != nil {
		return nil, err
	}
	return &analysis.Complexity, nil
}

// Helper functions

// typeToString converts a field list to a string representation
func typeToString(fl *ast.FieldList) string {
	if fl == nil || len(fl.List) == 0 {
		return ""
	}
	var types []string
	for _, field := range fl.List {
		types = append(types, fieldTypeToString(field.Type))
	}
	return strings.Join(types, ", ")
}

// fieldTypeToString converts a type expression to string
func fieldTypeToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.ArrayType:
		return "[]" + fieldTypeToString(t.Elt)
	case *ast.MapType:
		return "map[" + fieldTypeToString(t.Key) + "]" + fieldTypeToString(t.Value)
	case *ast.StarExpr:
		return "*" + fieldTypeToString(t.X)
	case *ast.SelectorExpr:
		return fieldTypeToString(t.X) + "." + t.Sel.Name
	default:
		return "interface{}"
	}
}

// typeToSymbolKind determines the symbol kind from a type expression
func typeToSymbolKind(expr ast.Expr) SymbolKind {
	switch expr.(type) {
	case *ast.StructType:
		return SymbolClass
	case *ast.InterfaceType:
		return SymbolInterface
	default:
		return SymbolType
	}
}

// calculateComplexity calculates cyclomatic complexity and other metrics
func calculateComplexity(file *ast.File, content []byte) ComplexityMetrics {
	metrics := ComplexityMetrics{
		LinesOfCode:         len(strings.Split(string(content), "\n")),
		Functions:           0,
		CyclomaticComplexity: 1, // Base complexity is 1
		AverageFunctionSize:  0,
		NestedDepth:          0,
	}

	// Count functions and accumulate complexity
	totalFunctionLines := 0
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			metrics.Functions++
			fnStart := file.Pos() - fn.Pos()
			fnEnd := fn.End() - fn.Pos()
			fnLines := fnEnd - fnStart
			totalFunctionLines += int(fnLines)

			// Add cyclomatic complexity for this function
			metrics.CyclomaticComplexity += countComplexity(fn.Body)
		}
	}

	if metrics.Functions > 0 {
		metrics.AverageFunctionSize = totalFunctionLines / metrics.Functions
	}

	return metrics
}

// countComplexity counts decision points (cyclomatic complexity)
func countComplexity(node ast.Node) int {
	if node == nil {
		return 0
	}

	complexity := 0
	ast.Inspect(node, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt, *ast.CaseClause:
			complexity++
		}
		return true
	})
	return complexity
}
