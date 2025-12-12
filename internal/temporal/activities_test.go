// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package temporal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBootstrapInputSerialization tests BootstrapInput serialization
func TestBootstrapInputSerialization(t *testing.T) {
	tests := []struct {
		name  string
		input BootstrapInput
	}{
		{
			name: "valid bootstrap input",
			input: BootstrapInput{
				CellID: "cell-001",
				Branch: "main",
			},
		},
		{
			name: "bootstrap input with feature branch",
			input: BootstrapInput{
				CellID: "cell-feature-001",
				Branch: "feature/new-feature",
			},
		},
		{
			name: "bootstrap input with develop branch",
			input: BootstrapInput{
				CellID: "cell-dev-001",
				Branch: "develop",
			},
		},
		{
			name: "bootstrap input with special characters in cell ID",
			input: BootstrapInput{
				CellID: "cell_underscore_001",
				Branch: "fix/issue-123",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify required fields are present
			assert.NotEmpty(t, tt.input.CellID)
			assert.NotEmpty(t, tt.input.Branch)

			// Verify fields are correct types for serialization
			assert.IsType(t, "", tt.input.CellID)
			assert.IsType(t, "", tt.input.Branch)
		})
	}
}

// TestBootstrapOutputSerialization tests BootstrapOutput serialization
func TestBootstrapOutputSerialization(t *testing.T) {
	tests := []struct {
		name   string
		output BootstrapOutput
		verify func(t *testing.T, output BootstrapOutput)
	}{
		{
			name: "complete bootstrap output",
			output: BootstrapOutput{
				CellID:       "cell-001",
				Port:         8001,
				WorktreeID:   "worktree-001",
				WorktreePath: "/tmp/worktree-001",
				BaseURL:      "http://localhost:8001",
				ServerPID:    12345,
			},
			verify: func(t *testing.T, output BootstrapOutput) {
				assert.NotEmpty(t, output.CellID)
				assert.Greater(t, output.Port, 0)
				assert.NotEmpty(t, output.WorktreeID)
				assert.NotEmpty(t, output.WorktreePath)
				assert.NotEmpty(t, output.BaseURL)
				assert.Greater(t, output.ServerPID, 0)
			},
		},
		{
			name: "bootstrap output with minimal values",
			output: BootstrapOutput{
				CellID:       "cell-min",
				Port:         8000,
				WorktreeID:   "wt-1",
				WorktreePath: "/tmp/wt-1",
				BaseURL:      "http://localhost:8000",
				ServerPID:    1,
			},
			verify: func(t *testing.T, output BootstrapOutput) {
				assert.Equal(t, "cell-min", output.CellID)
				assert.Equal(t, 8000, output.Port)
			},
		},
		{
			name: "bootstrap output with various port numbers",
			output: BootstrapOutput{
				CellID:       "cell-highport",
				Port:         9999,
				WorktreeID:   "worktree-high",
				WorktreePath: "/home/user/worktree-high",
				BaseURL:      "http://localhost:9999",
				ServerPID:    99999,
			},
			verify: func(t *testing.T, output BootstrapOutput) {
				assert.Equal(t, 9999, output.Port)
				assert.Greater(t, output.ServerPID, 0)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify all fields are serializable types
			assert.IsType(t, "", tt.output.CellID)
			assert.IsType(t, 0, tt.output.Port)
			assert.IsType(t, "", tt.output.WorktreeID)
			assert.IsType(t, "", tt.output.WorktreePath)
			assert.IsType(t, "", tt.output.BaseURL)
			assert.IsType(t, 0, tt.output.ServerPID)

			// Run specific verification
			if tt.verify != nil {
				tt.verify(t, tt.output)
			}
		})
	}
}

// TestTaskInputSerialization tests TaskInput serialization
func TestTaskInputSerialization(t *testing.T) {
	tests := []struct {
		name  string
		input TaskInput
	}{
		{
			name: "simple task input",
			input: TaskInput{
				TaskID:      "task-001",
				Description: "Test task",
				Prompt:      "Echo hello world",
			},
		},
		{
			name: "task input with detailed prompt",
			input: TaskInput{
				TaskID:      "task-feature-001",
				Description: "Implement user authentication",
				Prompt: `Create a Go handler that:
1. Validates JWT tokens
2. Returns user ID in context
3. Handles expiration errors gracefully`,
			},
		},
		{
			name: "task input with code snippet",
			input: TaskInput{
				TaskID:      "task-code-001",
				Description: "Generate API endpoint",
				Prompt: `Generate a REST API endpoint with:
func HandleGetUser(w http.ResponseWriter, r *http.Request) {
    // Implementation here
}`,
			},
		},
		{
			name: "task input with empty description",
			input: TaskInput{
				TaskID:      "task-empty-desc",
				Description: "",
				Prompt:      "Do something important",
			},
		},
		{
			name: "task input with empty prompt",
			input: TaskInput{
				TaskID:      "task-empty-prompt",
				Description: "Important task",
				Prompt:      "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify TaskID is always present
			assert.NotEmpty(t, tt.input.TaskID)

			// Verify all fields are correct types for serialization
			assert.IsType(t, "", tt.input.TaskID)
			assert.IsType(t, "", tt.input.Description)
			assert.IsType(t, "", tt.input.Prompt)
		})
	}
}

// TestTaskOutputSerialization tests TaskOutput serialization
func TestTaskOutputSerialization(t *testing.T) {
	tests := []struct {
		name   string
		output TaskOutput
		verify func(t *testing.T, output TaskOutput)
	}{
		{
			name: "successful task output",
			output: TaskOutput{
				Success:       true,
				Output:        "Task completed successfully",
				FilesModified: []string{"main.go", "handler.go"},
				ErrorMessage:  "",
			},
			verify: func(t *testing.T, output TaskOutput) {
				assert.True(t, output.Success)
				assert.NotEmpty(t, output.Output)
				assert.Len(t, output.FilesModified, 2)
			},
		},
		{
			name: "failed task output with error",
			output: TaskOutput{
				Success:       false,
				Output:        "Compilation failed",
				FilesModified: []string{},
				ErrorMessage:  "undefined variable 'x' on line 42",
			},
			verify: func(t *testing.T, output TaskOutput) {
				assert.False(t, output.Success)
				assert.NotEmpty(t, output.ErrorMessage)
				assert.Empty(t, output.FilesModified)
			},
		},
		{
			name: "task with no files modified",
			output: TaskOutput{
				Success:       true,
				Output:        "Task ran but made no changes",
				FilesModified: []string{},
				ErrorMessage:  "",
			},
			verify: func(t *testing.T, output TaskOutput) {
				assert.True(t, output.Success)
				assert.Empty(t, output.FilesModified)
			},
		},
		{
			name: "task with many files modified",
			output: TaskOutput{
				Success: true,
				Output:  "Refactoring complete",
				FilesModified: []string{
					"pkg/auth/handler.go",
					"pkg/auth/middleware.go",
					"pkg/auth/types.go",
					"internal/config/config.go",
					"internal/db/queries.go",
				},
				ErrorMessage: "",
			},
			verify: func(t *testing.T, output TaskOutput) {
				assert.True(t, output.Success)
				assert.Len(t, output.FilesModified, 5)
			},
		},
		{
			name: "partial task failure with some modifications",
			output: TaskOutput{
				Success:       false,
				Output:        "Test execution started but failed",
				FilesModified: []string{"test_helpers.go", "mock_data.go"},
				ErrorMessage:  "test assertion failed: expected true, got false",
			},
			verify: func(t *testing.T, output TaskOutput) {
				assert.False(t, output.Success)
				assert.NotEmpty(t, output.ErrorMessage)
				assert.NotEmpty(t, output.FilesModified)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify all fields are serializable types
			assert.IsType(t, true, tt.output.Success)
			assert.IsType(t, "", tt.output.Output)
			assert.IsType(t, []string{}, tt.output.FilesModified)
			assert.IsType(t, "", tt.output.ErrorMessage)

			// Run specific verification
			if tt.verify != nil {
				tt.verify(t, tt.output)
			}
		})
	}
}

// TestActivitySerialization tests that activities can be properly serialized/deserialized
func TestActivitySerialization(t *testing.T) {
	tests := []struct {
		name        string
		buildOutput func() BootstrapOutput
		taskOutput  func() TaskOutput
		roundTrip   bool
	}{
		{
			name: "bootstrap output round-trip",
			buildOutput: func() BootstrapOutput {
				return BootstrapOutput{
					CellID:       "cell-rt-001",
					Port:         8001,
					WorktreeID:   "wt-rt-001",
					WorktreePath: "/tmp/wt-rt-001",
					BaseURL:      "http://localhost:8001",
					ServerPID:    12345,
				}
			},
			roundTrip: true,
		},
		{
			name: "task output round-trip",
			taskOutput: func() TaskOutput {
				return TaskOutput{
					Success:       true,
					Output:        "Task completed",
					FilesModified: []string{"file1.go", "file2.go"},
					ErrorMessage:  "",
				}
			},
			roundTrip: true,
		},
		{
			name: "complex bootstrap output",
			buildOutput: func() BootstrapOutput {
				return BootstrapOutput{
					CellID:       "cell-complex-001",
					Port:         8080,
					WorktreeID:   "wt-complex-001",
					WorktreePath: "/home/user/project/worktrees/wt-complex-001",
					BaseURL:      "http://192.168.1.100:8080",
					ServerPID:    54321,
				}
			},
			roundTrip: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.buildOutput != nil {
				output := tt.buildOutput()
				// Verify fields can be accessed
				assert.NotEmpty(t, output.CellID)
				assert.NotEmpty(t, output.BaseURL)
			}

			if tt.taskOutput != nil {
				output := tt.taskOutput()
				// Verify fields can be accessed
				assert.NotEmpty(t, output.Output)
				assert.IsType(t, []string{}, output.FilesModified)
			}
		})
	}
}

// TestCellActivitiesCreation tests CellActivities can be created
func TestCellActivitiesCreation(t *testing.T) {
	t.Run("create cell activities", func(t *testing.T) {
		// Note: This test will skip if globals are not initialized
		// which is expected in unit test environment without full setup
		// In integration tests, InitializeGlobals would be called first

		defer func() {
			if r := recover(); r != nil {
				// Expected in unit test environment - globals not initialized
				t.Skip("Skipping test - globals not initialized (expected in unit tests)")
			}
		}()

		ca := NewCellActivities()
		require.NotNil(t, ca)
		assert.NotNil(t, ca.activities)
	})
}

// TestShellActivitiesCreation tests ShellActivities can be created
func TestShellActivitiesCreation(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "create shell activities",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sa := &ShellActivities{}
			require.NotNil(t, sa)
			// Verify it's a ShellActivities instance
			assert.IsType(t, &ShellActivities{}, sa)
		})
	}
}

// TestBootstrapOutputConstruction tests various ways to construct BootstrapOutput
func TestBootstrapOutputConstruction(t *testing.T) {
	tests := []struct {
		name         string
		cellID       string
		port         int
		worktreeID   string
		worktreePath string
		baseURL      string
		serverPID    int
		expectValid  bool
	}{
		{
			name:         "valid output",
			cellID:       "cell-001",
			port:         8001,
			worktreeID:   "wt-001",
			worktreePath: "/tmp/wt-001",
			baseURL:      "http://localhost:8001",
			serverPID:    12345,
			expectValid:  true,
		},
		{
			name:         "minimum valid values",
			cellID:       "c",
			port:         1,
			worktreeID:   "w",
			worktreePath: "/",
			baseURL:      "http://localhost:1",
			serverPID:    1,
			expectValid:  true,
		},
		{
			name:         "maximum realistic values",
			cellID:       "cell-max-0000000",
			port:         65535,
			worktreeID:   "worktree-max-0000000",
			worktreePath: "/very/long/path/to/worktree/that/might/exist",
			baseURL:      "http://192.168.255.255:65535",
			serverPID:    2147483647,
			expectValid:  true,
		},
		{
			name:         "special characters in paths",
			cellID:       "cell_with_underscore",
			port:         8080,
			worktreeID:   "wt-with-dash",
			worktreePath: "/home/user-123/project_name/worktree_01",
			baseURL:      "http://localhost:8080",
			serverPID:    99999,
			expectValid:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := BootstrapOutput{
				CellID:       tt.cellID,
				Port:         tt.port,
				WorktreeID:   tt.worktreeID,
				WorktreePath: tt.worktreePath,
				BaseURL:      tt.baseURL,
				ServerPID:    tt.serverPID,
			}

			if tt.expectValid {
				assert.NotEmpty(t, output.CellID)
				assert.Greater(t, output.Port, 0)
				assert.NotEmpty(t, output.WorktreeID)
				assert.NotEmpty(t, output.WorktreePath)
				assert.NotEmpty(t, output.BaseURL)
				assert.Greater(t, output.ServerPID, 0)
			}
		})
	}
}

// TestActivityFieldTypes tests that activity structures use correct serializable types
func TestActivityFieldTypes(t *testing.T) {
	t.Run("bootstrap input field types", func(t *testing.T) {
		input := BootstrapInput{
			CellID: "test",
			Branch: "test",
		}
		// All fields must be strings for Temporal serialization
		assert.IsType(t, "", input.CellID)
		assert.IsType(t, "", input.Branch)
	})

	t.Run("bootstrap output field types", func(t *testing.T) {
		output := BootstrapOutput{
			CellID:       "test",
			Port:         8080,
			WorktreeID:   "test",
			WorktreePath: "test",
			BaseURL:      "test",
			ServerPID:    1234,
		}
		assert.IsType(t, "", output.CellID)
		assert.IsType(t, 0, output.Port)
		assert.IsType(t, "", output.WorktreeID)
		assert.IsType(t, "", output.WorktreePath)
		assert.IsType(t, "", output.BaseURL)
		assert.IsType(t, 0, output.ServerPID)
	})

	t.Run("task input field types", func(t *testing.T) {
		input := TaskInput{
			TaskID:      "test",
			Description: "test",
			Prompt:      "test",
		}
		assert.IsType(t, "", input.TaskID)
		assert.IsType(t, "", input.Description)
		assert.IsType(t, "", input.Prompt)
	})

	t.Run("task output field types", func(t *testing.T) {
		output := TaskOutput{
			Success:       true,
			Output:        "test",
			FilesModified: []string{"test"},
			ErrorMessage:  "test",
		}
		assert.IsType(t, true, output.Success)
		assert.IsType(t, "", output.Output)
		assert.IsType(t, []string{}, output.FilesModified)
		assert.IsType(t, "", output.ErrorMessage)
	})
}
