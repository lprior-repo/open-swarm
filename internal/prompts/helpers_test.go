package prompts

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetLanguageFromExtension(t *testing.T) {
	tests := []struct {
		ext  string
		want string
	}{
		{".go", "go"},
		{".js", "javascript"},
		{".jsx", "javascript"},
		{".ts", "typescript"},
		{".tsx", "typescript"},
		{".py", "python"},
		{".java", "java"},
		{".rs", "rust"},
		{".c", "c"},
		{".h", "c"},
		{".cpp", "cpp"},
		{".rb", "ruby"},
		{".php", "php"},
		{".swift", "swift"},
		{".kt", "kotlin"},
		{".unknown", "text"},
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			got := getLanguageFromExtension(tt.ext)
			if got != tt.want {
				t.Errorf("getLanguageFromExtension(%s) = %s, want %s", tt.ext, got, tt.want)
			}
		})
	}
}

func TestExtractGoPackageName(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "simple package",
			content: "package main\n\nfunc main() {}",
			want:    "main",
		},
		{
			name:    "package with comments",
			content: "// Package prompts\npackage prompts\n\ntype Builder struct{}",
			want:    "prompts",
		},
		{
			name:    "package with extra whitespace",
			content: "  package   test  \n\nfunc Test() {}",
			want:    "test",
		},
		{
			name:    "no package declaration",
			content: "func main() {}",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractGoPackageName(tt.content)
			if got != tt.want {
				t.Errorf("extractGoPackageName() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestLoadFileContent(t *testing.T) {
	// Create a temporary file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	content := "package test\n\nfunc TestFunc() {}"

	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	ctx, err := LoadFileContent(testFile)
	if err != nil {
		t.Fatalf("LoadFileContent failed: %v", err)
	}

	if ctx.FilePath != testFile {
		t.Errorf("FilePath = %s, want %s", ctx.FilePath, testFile)
	}
	if ctx.FileContent != content {
		t.Errorf("FileContent = %s, want %s", ctx.FileContent, content)
	}
	if ctx.Language != "go" {
		t.Errorf("Language = %s, want go", ctx.Language)
	}
	if ctx.PackageName != "test" {
		t.Errorf("PackageName = %s, want test", ctx.PackageName)
	}
}

func TestLoadFileContent_NonExistent(t *testing.T) {
	_, err := LoadFileContent("/non/existent/file.go")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestLoadFileWithDiff(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	content := "package test\n\nfunc TestFunc() {}"
	diff := "+func NewFunc() {}"

	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	ctx, err := LoadFileWithDiff(testFile, diff)
	if err != nil {
		t.Fatalf("LoadFileWithDiff failed: %v", err)
	}

	if ctx.Diff != diff {
		t.Errorf("Diff = %s, want %s", ctx.Diff, diff)
	}
	if ctx.FileContent != content {
		t.Errorf("FileContent should also be set")
	}
}

func TestExtractSurroundingContext(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	content := `package test

func Function1() {
	// line 4
	// line 5
	code()
	// line 7
	// line 8
}

func Function2() {
}`

	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	t.Run("extract with context", func(t *testing.T) {
		// Extract lines 5-6 with 1 line of context
		surrounding, err := ExtractSurroundingContext(testFile, 5, 6, 1)
		if err != nil {
			t.Fatalf("ExtractSurroundingContext failed: %v", err)
		}

		// Should include lines 4-7
		expected := "	// line 4\n	// line 5\n	code()\n	// line 7"
		if surrounding != expected {
			t.Errorf("Got:\n%s\n\nWant:\n%s", surrounding, expected)
		}
	})

	t.Run("extract at start", func(t *testing.T) {
		// Extract line 1 with context
		surrounding, err := ExtractSurroundingContext(testFile, 1, 1, 2)
		if err != nil {
			t.Fatalf("ExtractSurroundingContext failed: %v", err)
		}

		if surrounding == "" {
			t.Error("Expected non-empty result")
		}
	})
}

func TestNewReviewRequest(t *testing.T) {
	request := NewReviewRequest(ReviewTypeArchitecture, "TASK-001", "Test description")

	if request.Type != ReviewTypeArchitecture {
		t.Errorf("Type = %s, want %s", request.Type, ReviewTypeArchitecture)
	}
	if request.TaskID != "TASK-001" {
		t.Error("TaskID not set correctly")
	}
	if request.TaskDescription != "Test description" {
		t.Error("TaskDescription not set correctly")
	}
	if !request.RequireVote {
		t.Error("RequireVote should default to true")
	}
}

func TestReviewRequestFluentMethods(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	err := os.WriteFile(testFile, []byte("package test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	t.Run("WithCodeFile", func(t *testing.T) {
		request := NewReviewRequest(ReviewTypeFunctional, "TASK-001", "Test")
		request, err = request.WithCodeFile(testFile)
		if err != nil {
			t.Fatalf("WithCodeFile failed: %v", err)
		}

		if request.CodeContext.FilePath != testFile {
			t.Error("FilePath not set")
		}
		if request.CodeContext.FileContent == "" {
			t.Error("FileContent not loaded")
		}
	})

	t.Run("WithCodeFile_NonExistent", func(t *testing.T) {
		request := NewReviewRequest(ReviewTypeFunctional, "TASK-001", "Test")
		_, err := request.WithCodeFile("/non/existent/file.go")
		if err == nil {
			t.Error("Expected error for non-existent file")
		}
	})

	t.Run("WithDiff", func(t *testing.T) {
		request := NewReviewRequest(ReviewTypeFunctional, "TASK-001", "Test")
		request = request.WithDiff("+new line")

		if request.CodeContext.Diff != "+new line" {
			t.Error("Diff not set correctly")
		}
	})

	t.Run("WithAcceptanceCriteria", func(t *testing.T) {
		request := NewReviewRequest(ReviewTypeFunctional, "TASK-001", "Test")
		request = request.WithAcceptanceCriteria("criterion 1", "criterion 2")

		if len(request.AcceptanceCriteria) != 2 {
			t.Errorf("Expected 2 criteria, got %d", len(request.AcceptanceCriteria))
		}
	})

	t.Run("WithAdditionalContext", func(t *testing.T) {
		request := NewReviewRequest(ReviewTypeFunctional, "TASK-001", "Test")
		request = request.WithAdditionalContext("extra info")

		if request.AdditionalContext != "extra info" {
			t.Error("AdditionalContext not set correctly")
		}
	})

	t.Run("WithoutVote", func(t *testing.T) {
		request := NewReviewRequest(ReviewTypeFunctional, "TASK-001", "Test")
		request = request.WithoutVote()

		if request.RequireVote {
			t.Error("RequireVote should be false")
		}
	})

	t.Run("Chaining", func(t *testing.T) {
		request := NewReviewRequest(ReviewTypeArchitecture, "TASK-001", "Test").
			WithDiff("+code").
			WithAcceptanceCriteria("clean").
			WithAdditionalContext("info").
			WithoutVote()

		if request.CodeContext.Diff != "+code" {
			t.Error("Diff not set in chain")
		}
		if len(request.AcceptanceCriteria) != 1 {
			t.Error("Criteria not set in chain")
		}
		if request.AdditionalContext != "info" {
			t.Error("Context not set in chain")
		}
		if request.RequireVote {
			t.Error("Vote not disabled in chain")
		}
	})
}

func TestMax(t *testing.T) {
	tests := []struct {
		a, b, want int
	}{
		{1, 2, 2},
		{2, 1, 2},
		{5, 5, 5},
		{-1, 0, 0},
		{-5, -3, -3},
	}

	for _, tt := range tests {
		got := max(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("max(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}
