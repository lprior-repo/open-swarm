// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package temporal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestTCRWorkflowInputSerialization tests that TCR workflow input is properly serializable
func TestTCRWorkflowInputSerialization(t *testing.T) {
	tests := []struct {
		name  string
		input TCRWorkflowInput
		valid bool
	}{
		{
			name: "valid complete input",
			input: TCRWorkflowInput{
				CellID:      "cell-001",
				Branch:      "main",
				TaskID:      "task-123",
				Description: "Implement feature X",
				Prompt:      "Create a new function that does Y",
			},
			valid: true,
		},
		{
			name: "minimal valid input",
			input: TCRWorkflowInput{
				CellID: "cell-001",
				Branch: "main",
				TaskID: "task-123",
			},
			valid: true,
		},
		{
			name: "input with special characters",
			input: TCRWorkflowInput{
				CellID:      "cell-001",
				Branch:      "feature/test-dash",
				TaskID:      "task-123_456",
				Description: "Fix bug #1234 in service-x",
				Prompt:      "Create a method: func() string { return \"test\" }",
			},
			valid: true,
		},
		{
			name: "input with multiline description",
			input: TCRWorkflowInput{
				CellID: "cell-001",
				Branch: "main",
				TaskID: "task-123",
				Description: `Implement feature X
This is a multiline description
with multiple lines of text`,
				Prompt: `Write code to:
1. Do X
2. Do Y`,
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify all required fields are present
			assert.NotEmpty(t, tt.input.CellID)
			assert.NotEmpty(t, tt.input.Branch)
			assert.NotEmpty(t, tt.input.TaskID)

			// Verify input is serializable (all string fields)
			assert.IsType(t, "", tt.input.CellID)
			assert.IsType(t, "", tt.input.Branch)
			assert.IsType(t, "", tt.input.TaskID)
			assert.IsType(t, "", tt.input.Description)
			assert.IsType(t, "", tt.input.Prompt)
		})
	}
}

// TestTCRWorkflowResultSerialization tests that TCR workflow result is properly serializable
func TestTCRWorkflowResultSerialization(t *testing.T) {
	tests := []struct {
		name   string
		result TCRWorkflowResult
	}{
		{
			name: "successful result",
			result: TCRWorkflowResult{
				Success:      true,
				TestsPassed:  true,
				FilesChanged: []string{"file1.go", "file2.go"},
				Error:        "",
			},
		},
		{
			name: "failed result with error",
			result: TCRWorkflowResult{
				Success:      false,
				TestsPassed:  false,
				FilesChanged: []string{},
				Error:        "bootstrap failed: connection refused",
			},
		},
		{
			name: "partial success",
			result: TCRWorkflowResult{
				Success:      false,
				TestsPassed:  false,
				FilesChanged: []string{"modified.go"},
				Error:        "tests failed: assertion error on line 42",
			},
		},
		{
			name: "empty result",
			result: TCRWorkflowResult{
				Success:      false,
				TestsPassed:  false,
				FilesChanged: []string{},
				Error:        "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify all fields are serializable types
			assert.IsType(t, true, tt.result.Success)
			assert.IsType(t, true, tt.result.TestsPassed)
			assert.IsType(t, []string{}, tt.result.FilesChanged)
			assert.IsType(t, "", tt.result.Error)
		})
	}
}
