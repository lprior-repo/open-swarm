# Agent Activity Wrapper for Temporal Workflows

## File Created
- **Location**: `internal/temporal/activities_agent.go`
- **Purpose**: Thin activity wrapper for agent invocation in Temporal workflows

## Key Components

### 1. AgentActivities Struct
Main activity handler for OpenCode/Claude agent calls with:
- Agent invocation via OpenCode SDK
- Error handling and timeout management
- Result parsing and structured output
- File modification tracking

### 2. Core Activities

#### InvokeAgent(ctx, input) → AgentResult
- Primary activity for agent invocation
- Invokes agent with configurable model/agent selection
- Streams output with heartbeat support
- Long timeout support (default 5 minutes)
- Auto-detects file modifications
- Handles errors gracefully

**Input**: AgentInvokeInput
- Bootstrap: Cell information
- Prompt: User prompt
- Agent: Agent name (build, plan, general)
- Model: Model ID (e.g., claude-sonnet-4-5)
- SessionID: Optional session reuse
- TimeoutSeconds: Custom timeout
- StreamOutput: Enable streaming

**Output**: AgentResult
- Success: Operation success flag
- SessionID: Session ID used
- MessageID: Response message ID
- Output: Complete agent response
- ToolResults: Tool execution results
- FilesModified: List of modified files
- Duration: Execution time
- Error: Error details if failed
- PartialOutput: Streaming chunks if enabled

#### StreamedInvokeAgent(ctx, input) → AgentResult
- Agent invocation with streaming output
- Real-time progress tracking
- Heartbeat management
- Suitable for long-running operations
- Chunks output for visibility

### 3. Error Handling

#### ClassifyError(err, duration, timeout) → ErrorCause
Categorizes errors for workflow retry logic:
- **Timeout**: Operation exceeded time limit
- **Network**: Connection/network issues
- **InvalidInput**: Invalid parameters
- **AgentError**: Agent returned error
- **Unknown**: Unclassified error

### 4. Result Parsing

#### AgentResultParser
Parses and validates agent output:
- ExtractCodeBlocks(): Extract code from markdown blocks
- ExtractStructuredData(): Find JSON/structured sections
- ValidateResult(): Check result completeness

### 5. Activity Options Helper
ActivityOptions() → Recommended Temporal settings:
- StartToCloseTimeout: 5 minutes
- HeartbeatTimeout: 30 seconds
- RetryPolicy: 3 attempts with exponential backoff
- HeartbeatInterval: 30 seconds

## Design Patterns

### 1. Thin Wrapper Pattern
- Wraps OpenCode client calls
- Delegates to existing CellActivities
- Serializable input/output
- Temporal-compatible result types

### 2. Heartbeat Management
- Records heartbeat immediately on start
- Records heartbeat after response
- Updates during streaming
- Prevents timeout during long operations

### 3. Error Context
- Wraps errors with context
- Preserves original error
- Provides error cause classification
- Includes timing information

### 4. File Tracking
- Auto-detects modified files
- Captures via cell.Client.GetFileStatus()
- Returns list of modified paths
- Handles failures gracefully

### 5. Streaming Support
- Optional output streaming
- Configurable chunk size
- Progress callbacks
- Maintains heartbeat during streaming

## Integration with Existing Code

### With CellActivities
- Reuses cell reconstruction logic
- Leverages bootstrap output format
- Uses same client infrastructure

### With Agent Package
- Uses agent.PromptOptions
- Delegates to agent.Client
- Respects existing SDK patterns

### With EnhancedActivities
- Compatible with gate workflows
- Works with file locking
- Integrates with result types

### With Workflows
- Used in TCR workflow phases
- Compatible with DAG workflows
- Supports signal-driven workflows

## Usage Example in Workflow

```go
// In a Temporal workflow
agentActivities := NewAgentActivities()

// Invoke with default options
result, err := workflow.ExecuteActivity(ctx, agentActivities.InvokeAgent, &AgentInvokeInput{
    Bootstrap: bootstrap,
    Prompt: "Implement feature X",
    Agent: "build",
    Model: "anthropic/claude-sonnet-4-5",
    Title: "Feature Implementation",
    TimeoutSeconds: 600, // 10 minutes
})

// Check result
if result.Success {
    files := result.FilesModified
    output := result.Output
} else {
    errorCause := ClassifyError(result.Error)
    // Handle based on cause
}
```

## Activity Options for Workflows

```go
// When registering with Temporal worker:
w.RegisterActivity(NewAgentActivities().InvokeAgent)
w.RegisterActivity(NewAgentActivities().StreamedInvokeAgent)

// When executing in workflow:
options := ActivityOptions()
ctx = workflow.WithActivityOptions(ctx, &temporal.ActivityOptions{
    StartToCloseTimeout: 5 * time.Minute,
    HeartbeatTimeout: 30 * time.Second,
    RetryPolicy: &temporal.RetryPolicy{
        InitialInterval: time.Second,
        BackoffCoefficient: 2.0,
        MaximumInterval: time.Minute,
        MaximumAttempts: 3,
    },
})
```

## Error Handling in Workflows

```go
result, err := workflow.ExecuteActivity(ctx, InvokeAgent, input)
if err != nil {
    cause := ClassifyError(err, duration, input.TimeoutSeconds)
    
    switch cause {
    case ErrorCauseTimeout:
        // Increase timeout and retry
    case ErrorCauseNetwork:
        // Wait and retry
    case ErrorCauseInvalidInput:
        // Fail immediately, don't retry
    case ErrorCauseAgentError:
        // Log and try different approach
    }
}
```

## Testing Considerations

1. **Mock OpenCode Server**: Use mock client for testing
2. **Streaming Simulation**: Test with ChunkSize parameter
3. **Error Injection**: Test each error cause
4. **Timeout Handling**: Test timeout scenarios
5. **File Tracking**: Verify file modification detection

## Future Enhancements

1. **Parallel Invocation**: Multiple agents simultaneously
2. **Context Injection**: Pass file content to agent
3. **Tool Handling**: Better tool result parsing
4. **Metrics**: Token usage tracking
5. **Logging**: Enhanced structured logging
