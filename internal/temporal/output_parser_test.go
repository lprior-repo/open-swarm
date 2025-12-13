package temporal

import (
	"strings"
	"testing"

	opencode "github.com/sst/opencode-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOutputParser_ParseFilePaths_FilePrefixFormat(t *testing.T) {
	parser := NewOutputParser()

	output := `I've created the following files:
FILE: internal/foo/bar.go
FILE: internal/foo/baz.go

All done!`

	actualFiles := []opencode.File{
		{Path: "internal/foo/bar.go"},
		{Path: "internal/foo/baz.go"},
	}

	result := parser.ParseFilePaths(output, actualFiles)

	assert.True(t, result.Valid)
	assert.ElementsMatch(t, []string{"internal/foo/bar.go", "internal/foo/baz.go"}, result.ExtractedPaths)
	assert.ElementsMatch(t, []string{"internal/foo/bar.go", "internal/foo/baz.go"}, result.ValidatedPaths)
	assert.Empty(t, result.MissingPaths)
	assert.Empty(t, result.UnexpectedPaths)
}

func TestOutputParser_ParseFilePaths_ModifiedPrefixFormat(t *testing.T) {
	parser := NewOutputParser()

	output := `Changes made:
Modified: pkg/handler.go
Created: pkg/handler_test.go
Updated: pkg/types.go`

	actualFiles := []opencode.File{
		{Path: "pkg/handler.go"},
		{Path: "pkg/handler_test.go"},
		{Path: "pkg/types.go"},
	}

	result := parser.ParseFilePaths(output, actualFiles)

	assert.True(t, result.Valid)
	assert.ElementsMatch(t, []string{"pkg/handler.go", "pkg/handler_test.go", "pkg/types.go"}, result.ExtractedPaths)
	assert.ElementsMatch(t, []string{"pkg/handler.go", "pkg/handler_test.go", "pkg/types.go"}, result.ValidatedPaths)
	assert.Empty(t, result.MissingPaths)
}

func TestOutputParser_ParseFilePaths_StandaloneFormat(t *testing.T) {
	parser := NewOutputParser()

	output := `I've updated the following:
- internal/service/auth.go
- internal/service/auth_test.go`

	actualFiles := []opencode.File{
		{Path: "internal/service/auth.go"},
		{Path: "internal/service/auth_test.go"},
	}

	result := parser.ParseFilePaths(output, actualFiles)

	assert.True(t, result.Valid)
	assert.Contains(t, result.ExtractedPaths, "internal/service/auth.go")
	assert.Contains(t, result.ExtractedPaths, "internal/service/auth_test.go")
	assert.Len(t, result.ValidatedPaths, 2)
}

func TestOutputParser_ParseFilePaths_SuffixMatching(t *testing.T) {
	parser := NewOutputParser()

	output := `FILE: foo.go
FILE: bar_test.go`

	actualFiles := []opencode.File{
		{Path: "internal/pkg/foo.go"},
		{Path: "internal/pkg/bar_test.go"},
	}

	result := parser.ParseFilePaths(output, actualFiles)

	assert.True(t, result.Valid)
	assert.ElementsMatch(t, []string{"foo.go", "bar_test.go"}, result.ExtractedPaths)
	assert.ElementsMatch(t, []string{"internal/pkg/foo.go", "internal/pkg/bar_test.go"}, result.ValidatedPaths)
	assert.Empty(t, result.MissingPaths)
	assert.Len(t, result.Warnings, 2) // Suffix match warnings
}

func TestOutputParser_ParseFilePaths_MixedFormats(t *testing.T) {
	parser := NewOutputParser()

	output := `I made the following changes:
FILE: cmd/main.go
Modified: internal/config.go
Also updated internal/server.go`

	actualFiles := []opencode.File{
		{Path: "cmd/main.go"},
		{Path: "internal/config.go"},
		{Path: "internal/server.go"},
	}

	result := parser.ParseFilePaths(output, actualFiles)

	assert.True(t, result.Valid)
	assert.Len(t, result.ExtractedPaths, 3)
	assert.Len(t, result.ValidatedPaths, 3)
	assert.Empty(t, result.MissingPaths)
}

func TestOutputParser_ParseFilePaths_MissingPaths(t *testing.T) {
	parser := NewOutputParser()

	output := `FILE: internal/foo.go
FILE: internal/bar.go
FILE: internal/baz.go`

	actualFiles := []opencode.File{
		{Path: "internal/foo.go"},
		// bar.go and baz.go don't actually exist
	}

	result := parser.ParseFilePaths(output, actualFiles)

	assert.True(t, result.Valid) // Still valid because we have one validated path
	assert.ElementsMatch(t, []string{"internal/foo.go", "internal/bar.go", "internal/baz.go"}, result.ExtractedPaths)
	assert.ElementsMatch(t, []string{"internal/foo.go"}, result.ValidatedPaths)
	assert.ElementsMatch(t, []string{"internal/bar.go", "internal/baz.go"}, result.MissingPaths)
}

func TestOutputParser_ParseFilePaths_UnexpectedPaths(t *testing.T) {
	parser := NewOutputParser()

	output := `FILE: internal/foo.go`

	actualFiles := []opencode.File{
		{Path: "internal/foo.go"},
		{Path: "internal/bar.go"}, // Not mentioned in output
		{Path: "internal/baz.go"}, // Not mentioned in output
	}

	result := parser.ParseFilePaths(output, actualFiles)

	assert.True(t, result.Valid)
	assert.ElementsMatch(t, []string{"internal/foo.go"}, result.ExtractedPaths)
	assert.ElementsMatch(t, []string{"internal/foo.go"}, result.ValidatedPaths)
	assert.Empty(t, result.MissingPaths)
	assert.ElementsMatch(t, []string{"internal/bar.go", "internal/baz.go"}, result.UnexpectedPaths)
	assert.Contains(t, result.Warnings[0], "modified but not mentioned")
}

func TestOutputParser_ParseFilePaths_NoExplicitPaths_FallbackToActual(t *testing.T) {
	parser := NewOutputParser()

	output := `I've completed the task successfully.
The implementation is working as expected.`

	actualFiles := []opencode.File{
		{Path: "internal/service.go"},
		{Path: "internal/service_test.go"},
	}

	result := parser.ParseFilePaths(output, actualFiles)

	assert.True(t, result.Valid)
	assert.Empty(t, result.ExtractedPaths)
	assert.ElementsMatch(t, []string{"internal/service.go", "internal/service_test.go"}, result.ValidatedPaths)
	assert.Len(t, result.Warnings, 1)
	assert.Contains(t, result.Warnings[0], "falling back to actual modified files")
}

func TestOutputParser_ParseFilePaths_NoPathsNoFiles(t *testing.T) {
	parser := NewOutputParser()

	output := `I've analyzed the code but made no changes.`

	actualFiles := []opencode.File{}

	result := parser.ParseFilePaths(output, actualFiles)

	assert.False(t, result.Valid)
	assert.Empty(t, result.ExtractedPaths)
	assert.Empty(t, result.ValidatedPaths)
}

func TestOutputParser_ParseFilePaths_DuplicatePaths(t *testing.T) {
	parser := NewOutputParser()

	output := `FILE: internal/foo.go
FILE: internal/foo.go
Modified: internal/foo.go`

	actualFiles := []opencode.File{
		{Path: "internal/foo.go"},
	}

	result := parser.ParseFilePaths(output, actualFiles)

	assert.True(t, result.Valid)
	// Should deduplicate
	assert.ElementsMatch(t, []string{"internal/foo.go"}, result.ExtractedPaths)
	assert.ElementsMatch(t, []string{"internal/foo.go"}, result.ValidatedPaths)
}

func TestOutputParser_ParseFilePaths_CaseInsensitivePrefixes(t *testing.T) {
	parser := NewOutputParser()

	output := `file: internal/foo.go
MODIFIED: internal/bar.go`

	actualFiles := []opencode.File{
		{Path: "internal/foo.go"},
		{Path: "internal/bar.go"},
	}

	result := parser.ParseFilePaths(output, actualFiles)

	assert.True(t, result.Valid)
	assert.Len(t, result.ValidatedPaths, 2)
}

func TestOutputParser_ParseFilePaths_MultipleExtensions(t *testing.T) {
	parser := NewOutputParser()

	output := `Updated files:
- main.go
- handler.ts
- component.tsx
- script.js
- styles.css (skipped, not tracked)
- utils.py`

	actualFiles := []opencode.File{
		{Path: "cmd/main.go"},
		{Path: "frontend/handler.ts"},
		{Path: "frontend/component.tsx"},
		{Path: "public/script.js"},
		{Path: "backend/utils.py"},
	}

	result := parser.ParseFilePaths(output, actualFiles)

	assert.True(t, result.Valid)
	// Should extract .go, .ts, .tsx, .js, .py but not .css
	for _, path := range result.ExtractedPaths {
		if strings.HasSuffix(path, ".css") {
			t.Errorf("Should not extract .css files, but got: %s", path)
		}
	}

	// Check that all expected files were extracted (by basename suffix matching)
	hasGo := false
	hasTs := false
	hasTsx := false
	hasJs := false
	hasPy := false

	for _, path := range result.ExtractedPaths {
		if strings.HasSuffix(path, "main.go") {
			hasGo = true
		}
		if strings.HasSuffix(path, "handler.ts") {
			hasTs = true
		}
		if strings.HasSuffix(path, "component.tsx") {
			hasTsx = true
		}
		if strings.HasSuffix(path, "script.js") {
			hasJs = true
		}
		if strings.HasSuffix(path, "utils.py") {
			hasPy = true
		}
	}

	assert.True(t, hasGo, "Should extract main.go")
	assert.True(t, hasTs, "Should extract handler.ts")
	assert.True(t, hasTsx, "Should extract component.tsx")
	assert.True(t, hasJs, "Should extract script.js")
	assert.True(t, hasPy, "Should extract utils.py")
}

func TestOutputParser_GetAllModifiedPaths(t *testing.T) {
	parser := NewOutputParser()

	result := &FileParseResult{
		ValidatedPaths:  []string{"internal/foo.go", "internal/bar.go"},
		UnexpectedPaths: []string{"internal/baz.go"},
	}

	allPaths := parser.GetAllModifiedPaths(result)

	assert.ElementsMatch(t, []string{"internal/foo.go", "internal/bar.go", "internal/baz.go"}, allPaths)
}

func TestOutputParser_GetAllModifiedPaths_WithDuplicates(t *testing.T) {
	parser := NewOutputParser()

	result := &FileParseResult{
		ValidatedPaths:  []string{"internal/foo.go", "internal/bar.go"},
		UnexpectedPaths: []string{"internal/foo.go", "internal/baz.go"}, // foo.go appears in both
	}

	allPaths := parser.GetAllModifiedPaths(result)

	// Should deduplicate
	assert.ElementsMatch(t, []string{"internal/foo.go", "internal/bar.go", "internal/baz.go"}, allPaths)
	assert.Len(t, allPaths, 3)
}

func TestOutputParser_MatchPattern_ExactMatch(t *testing.T) {
	parser := NewOutputParser()

	matched, err := parser.MatchPattern("internal/foo.go", "internal/foo.go")
	require.NoError(t, err)
	assert.True(t, matched)
}

func TestOutputParser_MatchPattern_GlobMatch(t *testing.T) {
	parser := NewOutputParser()

	matched, err := parser.MatchPattern("internal/foo.go", "internal/*.go")
	require.NoError(t, err)
	assert.True(t, matched)
}

func TestOutputParser_MatchPattern_BasenameMatch(t *testing.T) {
	parser := NewOutputParser()

	matched, err := parser.MatchPattern("internal/pkg/foo.go", "foo.go")
	require.NoError(t, err)
	assert.True(t, matched)
}

func TestOutputParser_MatchPattern_NoMatch(t *testing.T) {
	parser := NewOutputParser()

	matched, err := parser.MatchPattern("internal/foo.go", "*.py")
	require.NoError(t, err)
	assert.False(t, matched)
}

func TestRemoveDuplicates(t *testing.T) {
	input := []string{"a", "b", "c", "a", "b", "d"}
	expected := []string{"a", "b", "c", "d"}

	result := removeDuplicates(input)

	assert.Equal(t, expected, result)
}

func TestRemoveDuplicates_PreservesOrder(t *testing.T) {
	input := []string{"z", "a", "m", "a", "z"}
	expected := []string{"z", "a", "m"}

	result := removeDuplicates(input)

	assert.Equal(t, expected, result)
}
