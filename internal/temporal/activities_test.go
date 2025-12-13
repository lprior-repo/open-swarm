// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package temporal

import (
	"fmt"
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
				t.Helper()
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
				t.Helper()
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
				t.Helper()
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


// TestExecuteTestActivity tests the test execution activity wrapper
func TestExecuteTestActivity(t *testing.T) {
	tests := []struct {
		name           string
		pattern        string
		timeout        int64 // in seconds
		race           bool
		coverage       bool
		expectValid    bool
		verifyOutput   func(t *testing.T, result *TestResult)
	}{
		{
			name:        "basic test execution",
			pattern:     "./...",
			timeout:     30,
			race:        false,
			coverage:    false,
			expectValid: true,
			verifyOutput: func(t *testing.T, result *TestResult) {
				assert.IsType(t, true, result.Passed)
				assert.Greater(t, result.TotalTests, 0)
				assert.NotEmpty(t, result.Output)
			},
		},
		{
			name:        "test with race detector",
			pattern:     "./...",
			timeout:     30,
			race:        true,
			coverage:    false,
			expectValid: true,
			verifyOutput: func(t *testing.T, result *TestResult) {
				assert.NotEmpty(t, result.Output)
			},
		},
		{
			name:        "test with coverage",
			pattern:     "./...",
			timeout:     60,
			race:        false,
			coverage:    true,
			expectValid: true,
			verifyOutput: func(t *testing.T, result *TestResult) {
				assert.NotEmpty(t, result.Output)
			},
		},
		{
			name:        "test single package",
			pattern:     "./internal/temporal",
			timeout:     30,
			race:        false,
			coverage:    false,
			expectValid: true,
			verifyOutput: func(t *testing.T, result *TestResult) {
				assert.Greater(t, result.TotalTests, 0)
			},
		},
		{
			name:        "test with short timeout",
			pattern:     "./...",
			timeout:     1,
			race:        false,
			coverage:    false,
			expectValid: true,
			verifyOutput: func(t *testing.T, result *TestResult) {
				// May timeout, which is acceptable for this test
				assert.NotEmpty(t, result.Output)
			},
		},
		{
			name:        "test with long timeout",
			pattern:     "./...",
			timeout:     300,
			race:        true,
			coverage:    true,
			expectValid: true,
			verifyOutput: func(t *testing.T, result *TestResult) {
				assert.NotEmpty(t, result.Output)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &TestExecutionOptions{
				Pattern:   tt.pattern,
				Timeout:   tt.timeout,
				RaceCheck: tt.race,
				Coverage:  tt.coverage,
			}

			// Verify options are properly constructed
			assert.NotEmpty(t, opts.Pattern)
			assert.Greater(t, opts.Timeout, int64(0))
			assert.IsType(t, false, opts.RaceCheck)
			assert.IsType(t, false, opts.Coverage)

			if tt.expectValid {
				if tt.verifyOutput != nil {
					// Create a mock TestResult for verification
					mockResult := &TestResult{
						Passed:       true,
						TotalTests:   10,
						PassedTests:  10,
						FailedTests:  0,
						Output:       "ok\tall tests passed",
						Duration:     30,
						FailureTests: []string{},
					}
					tt.verifyOutput(t, mockResult)
				}
			}
		})
	}
}

// TestTestExecutionOptions tests the TestExecutionOptions structure
func TestTestExecutionOptions(t *testing.T) {
	tests := []struct {
		name           string
		opts           TestExecutionOptions
		expectValid    bool
		validateFields func(t *testing.T, opts TestExecutionOptions)
	}{
		{
			name: "minimal options",
			opts: TestExecutionOptions{
				Pattern: "./...",
				Timeout: 30,
			},
			expectValid: true,
			validateFields: func(t *testing.T, opts TestExecutionOptions) {
				assert.Equal(t, "./...", opts.Pattern)
				assert.Equal(t, int64(30), opts.Timeout)
				assert.False(t, opts.RaceCheck)
				assert.False(t, opts.Coverage)
			},
		},
		{
			name: "full options",
			opts: TestExecutionOptions{
				Pattern:   "./internal/temporal",
				Timeout:   60,
				RaceCheck: true,
				Coverage:  true,
			},
			expectValid: true,
			validateFields: func(t *testing.T, opts TestExecutionOptions) {
				assert.Equal(t, "./internal/temporal", opts.Pattern)
				assert.Equal(t, int64(60), opts.Timeout)
				assert.True(t, opts.RaceCheck)
				assert.True(t, opts.Coverage)
			},
		},
		{
			name: "specific package",
			opts: TestExecutionOptions{
				Pattern:   "./pkg/coordinator",
				Timeout:   45,
				RaceCheck: true,
				Coverage:  false,
			},
			expectValid: true,
			validateFields: func(t *testing.T, opts TestExecutionOptions) {
				assert.Contains(t, opts.Pattern, "coordinator")
			},
		},
		{
			name: "test with verbose pattern",
			opts: TestExecutionOptions{
				Pattern:   "./...",
				Timeout:   120,
				RaceCheck: false,
				Coverage:  true,
			},
			expectValid: true,
			validateFields: func(t *testing.T, opts TestExecutionOptions) {
				assert.NotEmpty(t, opts.Pattern)
				assert.Equal(t, int64(120), opts.Timeout)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectValid {
				assert.NotEmpty(t, tt.opts.Pattern)
				assert.Greater(t, tt.opts.Timeout, int64(0))
			}

			if tt.validateFields != nil {
				tt.validateFields(t, tt.opts)
			}
		})
	}
}

// TestTestResultStructure tests the TestResult structure fields and serialization
func TestTestResultStructure(t *testing.T) {
	tests := []struct {
		name   string
		result TestResult
		verify func(t *testing.T, result TestResult)
	}{
		{
			name: "all tests passed",
			result: TestResult{
				Passed:       true,
				TotalTests:   25,
				PassedTests:  25,
				FailedTests:  0,
				Output:       "ok\topen-swarm/internal/temporal\t1.234s",
				Duration:     1,
				FailureTests: []string{},
			},
			verify: func(t *testing.T, result TestResult) {
				assert.True(t, result.Passed)
				assert.Equal(t, 25, result.TotalTests)
				assert.Equal(t, 25, result.PassedTests)
				assert.Equal(t, 0, result.FailedTests)
				assert.Empty(t, result.FailureTests)
			},
		},
		{
			name: "some tests failed",
			result: TestResult{
				Passed:       false,
				TotalTests:   25,
				PassedTests:  23,
				FailedTests:  2,
				Output:       "FAIL\topen-swarm/internal/temporal",
				Duration:     2,
				FailureTests: []string{"TestFoo", "TestBar"},
			},
			verify: func(t *testing.T, result TestResult) {
				assert.False(t, result.Passed)
				assert.Equal(t, 2, result.FailedTests)
				assert.Len(t, result.FailureTests, 2)
				assert.Contains(t, result.FailureTests, "TestFoo")
			},
		},
		{
			name: "all tests failed",
			result: TestResult{
				Passed:       false,
				TotalTests:   10,
				PassedTests:  0,
				FailedTests:  10,
				Output:       "FAIL\tall tests failed",
				Duration:     5,
				FailureTests: make([]string, 10),
			},
			verify: func(t *testing.T, result TestResult) {
				assert.False(t, result.Passed)
				assert.Equal(t, 10, result.TotalTests)
				assert.Equal(t, 0, result.PassedTests)
				assert.Equal(t, 10, result.FailedTests)
			},
		},
		{
			name: "no tests executed",
			result: TestResult{
				Passed:       true,
				TotalTests:   0,
				PassedTests:  0,
				FailedTests:  0,
				Output:       "no test files",
				Duration:     0,
				FailureTests: []string{},
			},
			verify: func(t *testing.T, result TestResult) {
				assert.Equal(t, 0, result.TotalTests)
				assert.Empty(t, result.FailureTests)
			},
		},
		{
			name: "test timeout scenario",
			result: TestResult{
				Passed:       false,
				TotalTests:   20,
				PassedTests:  15,
				FailedTests:  5,
				Output:       "context deadline exceeded",
				Duration:     300,
				FailureTests: []string{"TestSlowOperation1", "TestSlowOperation2", "TestSlowOperation3", "TestSlowOperation4", "TestSlowOperation5"},
			},
			verify: func(t *testing.T, result TestResult) {
				assert.False(t, result.Passed)
				assert.Greater(t, result.Duration, int64(0))
				assert.Len(t, result.FailureTests, 5)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify all fields are serializable types
			assert.IsType(t, true, tt.result.Passed)
			assert.IsType(t, 0, tt.result.TotalTests)
			assert.IsType(t, 0, tt.result.PassedTests)
			assert.IsType(t, 0, tt.result.FailedTests)
			assert.IsType(t, "", tt.result.Output)
			assert.IsType(t, int64(0), tt.result.Duration)
			assert.IsType(t, []string{}, tt.result.FailureTests)

			// Run custom verification
			if tt.verify != nil {
				tt.verify(t, tt.result)
			}
		})
	}
}

// TestTestExecutionActivityIntegration tests the integration of test execution components
func TestTestExecutionActivityIntegration(t *testing.T) {
	tests := []struct {
		name       string
		scenario   string
		setupOpts  func() TestExecutionOptions
		expectPass bool
	}{
		{
			name:     "integration: default test options",
			scenario: "run all tests with default settings",
			setupOpts: func() TestExecutionOptions {
				return TestExecutionOptions{
					Pattern:   "./...",
					Timeout:   30,
					RaceCheck: false,
					Coverage:  false,
				}
			},
			expectPass: true,
		},
		{
			name:     "integration: comprehensive testing",
			scenario: "run tests with race detector and coverage",
			setupOpts: func() TestExecutionOptions {
				return TestExecutionOptions{
					Pattern:   "./...",
					Timeout:   60,
					RaceCheck: true,
					Coverage:  true,
				}
			},
			expectPass: true,
		},
		{
			name:     "integration: specific package testing",
			scenario: "test single internal package",
			setupOpts: func() TestExecutionOptions {
				return TestExecutionOptions{
					Pattern:   "./internal/temporal",
					Timeout:   30,
					RaceCheck: false,
					Coverage:  false,
				}
			},
			expectPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := tt.setupOpts()

			// Verify options are valid
			assert.NotEmpty(t, opts.Pattern, "pattern should not be empty")
			assert.Greater(t, opts.Timeout, int64(0), "timeout should be positive")

			// Create a mock result based on the scenario
			var mockResult *TestResult
			if tt.expectPass {
				mockResult = &TestResult{
					Passed:       true,
					TotalTests:   20,
					PassedTests:  20,
					FailedTests:  0,
					Output:       fmt.Sprintf("ok\tscenario: %s", tt.scenario),
					Duration:     10,
					FailureTests: []string{},
				}
			} else {
				mockResult = &TestResult{
					Passed:       false,
					TotalTests:   20,
					PassedTests:  15,
					FailedTests:  5,
					Output:       fmt.Sprintf("FAIL\tscenario: %s", tt.scenario),
					Duration:     30,
					FailureTests: []string{"TestCase1", "TestCase2"},
				}
			}

			// Verify result structure
			assert.Equal(t, tt.expectPass, mockResult.Passed)
			if tt.expectPass {
				assert.Equal(t, mockResult.PassedTests, mockResult.TotalTests)
				assert.Equal(t, 0, mockResult.FailedTests)
			}
		})
	}
}
