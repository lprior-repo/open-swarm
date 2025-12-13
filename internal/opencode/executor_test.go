// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package opencode

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/sst/opencode-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"open-swarm/internal/agent"
)

// MockClient is a mock implementation of agent.ClientInterface
type MockClient struct {
	mock.Mock
}

func (m *MockClient) ExecutePrompt(ctx context.Context, prompt string, opts *agent.PromptOptions) (*agent.PromptResult, error) {
	args := m.Called(ctx, prompt, opts)
	err := args.Error(1)
	if err != nil {
		err = fmt.Errorf("mock ExecutePrompt failed: %w", err)
	}
	if args.Get(0) == nil {
		return nil, err
	}
	return args.Get(0).(*agent.PromptResult), err
}

func (m *MockClient) ExecuteCommand(ctx context.Context, sessionID string, command string, args []string) (*agent.PromptResult, error) {
	callArgs := m.Called(ctx, sessionID, command, args)
	err := callArgs.Error(1)
	if err != nil {
		err = fmt.Errorf("mock ExecuteCommand failed: %w", err)
	}
	if callArgs.Get(0) == nil {
		return nil, err
	}
	return callArgs.Get(0).(*agent.PromptResult), err
}

func (m *MockClient) GetFileStatus(ctx context.Context) ([]opencode.File, error) {
	args := m.Called(ctx)
	err := args.Error(1)
	if err != nil {
		err = fmt.Errorf("mock GetFileStatus failed: %w", err)
	}
	if args.Get(0) == nil {
		return nil, err
	}
	return args.Get(0).([]opencode.File), err
}

func (m *MockClient) GetBaseURL() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockClient) GetPort() int {
	args := m.Called()
	return args.Int(0)
}

// TestExecute_Success tests successful task execution
func TestExecute_Success(t *testing.T) {
	mockClient := new(MockClient)
	executor := NewExecutor(mockClient, ExecutorConfig{MaxTurns: 10, Timeout: 5 * time.Minute})

	// Setup mock expectations
	mockClient.On("ExecutePrompt", mock.Anything, "Write a test file", mock.Anything).
		Return(&agent.PromptResult{
			SessionID: "session-123",
			MessageID: "msg-456",
			Parts: []agent.ResultPart{
				{Type: "text", Text: "I have created the test file."},
			},
		}, nil)

	mockClient.On("GetFileStatus", mock.Anything).
		Return([]opencode.File{
			{Path: "test.go"},
		}, nil)

	// Execute
	req := &ExecuteRequest{
		TaskID:      "task-001",
		Description: "Write a test file",
		Prompt:      "Write a test file",
	}

	ctx := context.Background()
	resp, err := executor.Execute(ctx, req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.Equal(t, 1, resp.Turns)
	assert.Equal(t, "session-123", resp.SessionID)
	assert.Equal(t, "I have created the test file.", resp.Output)
	assert.Len(t, resp.FilesModified, 1)
	assert.Equal(t, "test.go", resp.FilesModified[0])

	mockClient.AssertExpectations(t)
}

// TestExecute_TurnLimit tests that max turns are enforced
func TestExecute_TurnLimit(t *testing.T) {
	mockClient := new(MockClient)
	executor := NewExecutor(mockClient, ExecutorConfig{MaxTurns: 1, Timeout: 5 * time.Minute})

	mockClient.On("ExecutePrompt", mock.Anything, "test prompt", mock.Anything).
		Return(&agent.PromptResult{
			SessionID: "session-001",
			MessageID: "msg-001",
			Parts: []agent.ResultPart{
				{Type: "text", Text: "Continue?"},
			},
		}, nil)

	mockClient.On("GetFileStatus", mock.Anything).
		Return([]opencode.File{}, nil)

	req := &ExecuteRequest{
		TaskID:      "task-001",
		Description: "Multi-turn task",
		Prompt:      "test prompt",
	}

	ctx := context.Background()
	resp, err := executor.Execute(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.Equal(t, 1, resp.Turns)

	mockClient.AssertExpectations(t)
}

// TestExecute_Timeout tests that context timeout is respected
func TestExecute_Timeout(t *testing.T) {
	mockClient := new(MockClient)
	executor := NewExecutor(mockClient, ExecutorConfig{MaxTurns: 10, Timeout: 100 * time.Millisecond})

	// Mock will simulate a timeout by blocking
	mockClient.On("ExecutePrompt", mock.Anything, "slow prompt", mock.Anything).
		Run(func(args mock.Arguments) {
			time.Sleep(200 * time.Millisecond)
		}).
		Return((*agent.PromptResult)(nil), context.DeadlineExceeded)

	req := &ExecuteRequest{
		TaskID:      "task-001",
		Description: "Slow task",
		Prompt:      "slow prompt",
	}

	ctx := context.Background()
	resp, err := executor.Execute(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.False(t, resp.Success)
	assert.Contains(t, resp.ErrorMessage, "execution failed")

	mockClient.AssertExpectations(t)
}

// TestExecute_InvalidRequest tests input validation
func TestExecute_InvalidRequest(t *testing.T) {
	mockClient := new(MockClient)
	executor := NewExecutor(mockClient, ExecutorConfig{MaxTurns: 10, Timeout: 5 * time.Minute})

	testCases := []struct {
		name        string
		req         *ExecuteRequest
		expectError string
	}{
		{
			name: "Missing TaskID",
			req: &ExecuteRequest{
				TaskID:      "",
				Description: "test",
				Prompt:      "test prompt",
			},
			expectError: "TaskID is required",
		},
		{
			name: "Missing Prompt",
			req: &ExecuteRequest{
				TaskID:      "task-001",
				Description: "test",
				Prompt:      "",
			},
			expectError: "Prompt is required",
		},
		{
			name: "Prompt too long",
			req: &ExecuteRequest{
				TaskID:      "task-001",
				Description: "test",
				Prompt:      string(make([]byte, 10001)),
			},
			expectError: "exceeds maximum length",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			resp, err := executor.Execute(ctx, tc.req)

			assert.NoError(t, err)
			assert.NotNil(t, resp)
			assert.False(t, resp.Success)
			assert.Contains(t, resp.ErrorMessage, tc.expectError)
		})
	}
}

// TestExecute_FileChanges tests that modified files are tracked
func TestExecute_FileChanges(t *testing.T) {
	mockClient := new(MockClient)
	executor := NewExecutor(mockClient, ExecutorConfig{MaxTurns: 10, Timeout: 5 * time.Minute})

	mockClient.On("ExecutePrompt", mock.Anything, "create files", mock.Anything).
		Return(&agent.PromptResult{
			SessionID: "session-002",
			MessageID: "msg-002",
			Parts: []agent.ResultPart{
				{Type: "text", Text: "Created multiple files"},
			},
		}, nil)

	mockClient.On("GetFileStatus", mock.Anything).
		Return([]opencode.File{
			{Path: "main.go"},
			{Path: "main_test.go"},
			{Path: "config.yaml"},
		}, nil)

	req := &ExecuteRequest{
		TaskID:      "task-002",
		Description: "Create multiple files",
		Prompt:      "create files",
	}

	ctx := context.Background()
	resp, err := executor.Execute(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.Len(t, resp.FilesModified, 3)
	assert.Contains(t, resp.FilesModified, "main.go")
	assert.Contains(t, resp.FilesModified, "main_test.go")
	assert.Contains(t, resp.FilesModified, "config.yaml")

	mockClient.AssertExpectations(t)
}

// TestExecute_ErrorHandling tests that agent failures are handled properly
func TestExecute_ErrorHandling(t *testing.T) {
	testCases := []struct {
		name             string
		setupMock        func(*MockClient)
		expectedSuccess  bool
		expectedErrorMsg string
	}{
		{
			name: "Prompt execution fails",
			setupMock: func(m *MockClient) {
				m.On("ExecutePrompt", mock.Anything, mock.Anything, mock.Anything).
					Return((*agent.PromptResult)(nil), errors.New("connection refused"))
			},
			expectedSuccess:  false,
			expectedErrorMsg: "prompt execution failed",
		},
		{
			name: "File status retrieval fails",
			setupMock: func(m *MockClient) {
				m.On("ExecutePrompt", mock.Anything, mock.Anything, mock.Anything).
					Return(&agent.PromptResult{
						SessionID: "session-003",
						MessageID: "msg-003",
						Parts: []agent.ResultPart{
							{Type: "text", Text: "Success"},
						},
					}, nil)
				m.On("GetFileStatus", mock.Anything).
					Return(nil, errors.New("file status unavailable"))
			},
			expectedSuccess:  false,
			expectedErrorMsg: "failed to get file status",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := new(MockClient)
			executor := NewExecutor(mockClient, ExecutorConfig{MaxTurns: 10, Timeout: 5 * time.Minute})

			tc.setupMock(mockClient)

			req := &ExecuteRequest{
				TaskID:      "task-003",
				Description: "Error handling test",
				Prompt:      "test prompt",
			}

			ctx := context.Background()
			resp, err := executor.Execute(ctx, req)

			assert.NoError(t, err)
			assert.NotNil(t, resp)
			assert.Equal(t, tc.expectedSuccess, resp.Success)
			assert.Contains(t, resp.ErrorMessage, tc.expectedErrorMsg)

			mockClient.AssertExpectations(t)
		})
	}
}

// TestHeadlessExecutor_Execute_Success tests successful task execution with HeadlessExecutor
func TestHeadlessExecutor_Execute_Success(t *testing.T) {
	mockClient := new(MockClient)
	executor := NewHeadlessExecutor(mockClient, Config{
		MaxTurns: 10,
		Timeout:  5 * time.Minute,
		Provider: "claude",
	})

	// Setup mock expectations
	mockClient.On("ExecutePrompt", mock.Anything, mock.Anything, mock.Anything).
		Return(&agent.PromptResult{
			SessionID: "session-123",
			MessageID: "msg-456",
			Parts: []agent.ResultPart{
				{Type: "text", Text: "Task completed successfully"},
			},
		}, nil)

	mockClient.On("GetFileStatus", mock.Anything).
		Return([]opencode.File{
			{Path: "file1.go"},
			{Path: "file2.go"},
		}, nil)

	// Execute
	req := TaskRequest{
		Prompt:   "Write a hello world function",
		Files:    []string{"main.go"},
		MaxTurns: 5,
		Provider: "claude",
	}

	result := executor.Execute(context.Background(), req)

	// Assert
	assert.True(t, result.Success)
	assert.Equal(t, "Task completed successfully", result.Output)
	assert.Len(t, result.FilesChanged, 2)
	assert.Contains(t, result.FilesChanged, "file1.go")
	assert.Contains(t, result.FilesChanged, "file2.go")
	assert.Empty(t, result.Error)

	mockClient.AssertExpectations(t)
}

// TestHeadlessExecutor_Execute_ValidationFailure tests validation with HeadlessExecutor
func TestHeadlessExecutor_Execute_ValidationFailure(t *testing.T) {
	mockClient := new(MockClient)
	executor := NewHeadlessExecutor(mockClient, Config{})

	testCases := []struct {
		name        string
		req         TaskRequest
		expectError string
	}{
		{
			name: "Empty prompt",
			req: TaskRequest{
				Prompt: "",
			},
			expectError: "prompt is required",
		},
		{
			name: "Prompt too long",
			req: TaskRequest{
				Prompt: string(make([]byte, 50001)),
			},
			expectError: "exceeds maximum length",
		},
		{
			name: "Negative max turns",
			req: TaskRequest{
				Prompt:   "Test",
				MaxTurns: -1,
			},
			expectError: "cannot be negative",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := executor.Execute(context.Background(), tc.req)

			assert.False(t, result.Success)
			assert.Contains(t, result.Error, tc.expectError)
		})
	}
}

// TestHeadlessExecutor_Execute_PromptFailure tests prompt execution failure
func TestHeadlessExecutor_Execute_PromptFailure(t *testing.T) {
	mockClient := new(MockClient)
	executor := NewHeadlessExecutor(mockClient, Config{})

	mockClient.On("ExecutePrompt", mock.Anything, mock.Anything, mock.Anything).
		Return((*agent.PromptResult)(nil), errors.New("API error"))

	req := TaskRequest{
		Prompt: "Test prompt",
	}

	result := executor.Execute(context.Background(), req)

	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "prompt execution failed")

	mockClient.AssertExpectations(t)
}

// TestHeadlessExecutor_Execute_WithFileContext tests file context building
func TestHeadlessExecutor_Execute_WithFileContext(t *testing.T) {
	mockClient := new(MockClient)
	executor := NewHeadlessExecutor(mockClient, Config{})

	var capturedPrompt string

	mockClient.On("ExecutePrompt", mock.Anything, mock.MatchedBy(func(prompt string) bool {
		capturedPrompt = prompt
		return true
	}), mock.Anything).
		Return(&agent.PromptResult{
			SessionID: "session-123",
			MessageID: "msg-456",
			Parts: []agent.ResultPart{
				{Type: "text", Text: "Done"},
			},
		}, nil)

	mockClient.On("GetFileStatus", mock.Anything).
		Return([]opencode.File{}, nil)

	req := TaskRequest{
		Prompt: "Refactor these files",
		Files:  []string{"file1.go", "file2.go"},
	}

	result := executor.Execute(context.Background(), req)

	assert.True(t, result.Success)
	assert.Contains(t, capturedPrompt, "file1.go")
	assert.Contains(t, capturedPrompt, "file2.go")
	assert.Contains(t, capturedPrompt, "Refactor these files")

	mockClient.AssertExpectations(t)
}

// TestHeadlessExecutor_ConfigDefaults tests default configuration values
func TestHeadlessExecutor_ConfigDefaults(t *testing.T) {
	mockClient := new(MockClient)
	executor := NewHeadlessExecutor(mockClient, Config{})

	assert.Equal(t, 10, executor.config.MaxTurns)
	assert.Equal(t, 5*time.Minute, executor.config.Timeout)
	assert.Equal(t, "claude", executor.config.Provider)
}

// TestHeadlessExecutor_ModelMapping tests provider to model mapping
func TestHeadlessExecutor_ModelMapping(t *testing.T) {
	mockClient := new(MockClient)
	executor := NewHeadlessExecutor(mockClient, Config{})

	testCases := []struct {
		provider string
		expected string
	}{
		{"claude", "anthropic/claude-sonnet-4-5"},
		{"CLAUDE", "anthropic/claude-sonnet-4-5"},
		{"copilot", "github/copilot"},
		{"unknown", "anthropic/claude-sonnet-4-5"}, // default
	}

	for _, tc := range testCases {
		t.Run(tc.provider, func(t *testing.T) {
			result := executor.getModelForProvider(tc.provider)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestHeadlessExecutor_FileStatusError tests handling of file status retrieval error
func TestHeadlessExecutor_FileStatusError(t *testing.T) {
	mockClient := new(MockClient)
	executor := NewHeadlessExecutor(mockClient, Config{})

	mockClient.On("ExecutePrompt", mock.Anything, mock.Anything, mock.Anything).
		Return(&agent.PromptResult{
			SessionID: "session-123",
			MessageID: "msg-456",
			Parts: []agent.ResultPart{
				{Type: "text", Text: "Done"},
			},
		}, nil)

	mockClient.On("GetFileStatus", mock.Anything).
		Return(nil, errors.New("file status error"))

	req := TaskRequest{
		Prompt: "Test prompt",
	}

	result := executor.Execute(context.Background(), req)

	// Should still succeed but include warning in output
	assert.True(t, result.Success)
	assert.Contains(t, result.Output, "Warning")
	assert.Contains(t, result.Output, "file status error")

	mockClient.AssertExpectations(t)
}
