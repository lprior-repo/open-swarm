# Single Agent Demo

This demo shows the complete end-to-end flow for running a single OpenCode agent in Open Swarm.

## Flow Overview

```
┌─────────────────────────────────────────────────────────────┐
│                   Single Agent Demo Flow                     │
└─────────────────────────────────────────────────────────────┘

1. Setup Infrastructure
   ├─ Create PortManager (8000-8100 range)
   └─ Create ServerManager

2. Allocate Port
   ├─ Request unique port from PortManager
   └─ Track allocation for cleanup

3. Boot OpenCode Server
   ├─ Start `opencode serve --port X --hostname localhost`
   ├─ Set working directory to project root
   ├─ Wait for health check (/health endpoint)
   └─ Return ServerHandle with PID, BaseURL

4. Create SDK Client
   ├─ Initialize opencode-sdk-go client
   ├─ Configure with server BaseURL
   └─ Return Client wrapper

5. Execute Task
   ├─ Create session (or reuse existing)
   ├─ Send prompt to agent
   ├─ Agent processes using available tools
   └─ Return PromptResult with session/message IDs

6. Verify Results
   ├─ Use SDK ReadFile() to check output
   ├─ Use SDK GetFileStatus() to list files
   └─ Validate expected outcomes

7. Cleanup
   ├─ Shutdown server (SIGTERM → SIGKILL)
   ├─ Release port
   └─ Exit
```

## Architecture

### Components

**PortManager** (`internal/infra/ports.go`)
- Manages port allocation in range 8000-9000
- Thread-safe allocation/release
- Prevents port conflicts between agents
- **Invariant**: Each agent gets unique port

**ServerManager** (`internal/infra/server.go`)
- Boots `opencode serve` processes
- Monitors health via `/health` endpoint
- Handles graceful shutdown with SIGTERM/SIGKILL
- **Invariants**:
  - Working directory set to Git worktree
  - Health check passes before SDK connection
  - Process killed when activity completes

**Client** (`internal/agent/client.go`)
- Wraps opencode-sdk-go for high-level operations
- Provides ExecutePrompt(), ExecuteCommand(), ReadFile()
- **Invariants**:
  - Configured with specific BaseURL (localhost:PORT)
  - All commands use SDK client

### Data Flow

```
User Request
    ↓
main.go
    ↓
┌───────────────┐
│ PortManager   │  → Allocate(8000)
└───────────────┘
    ↓
┌───────────────┐
│ ServerManager │  → BootServer(cwd, "demo-agent", 8000)
└───────────────┘  → Wait for health check
    ↓
┌───────────────┐
│ SDK Client    │  → NewClient("http://localhost:8000", 8000)
└───────────────┘
    ↓
┌───────────────┐
│ Execute Task  │  → ExecutePrompt(ctx, prompt, opts)
└───────────────┘
    ↓
    OpenCode Agent
      ├─ Parse prompt
      ├─ Plan actions
      ├─ Use tools (Write, Bash, etc.)
      └─ Return result
    ↓
┌───────────────┐
│ Verify        │  → ReadFile(ctx, "hello.txt")
└───────────────┘  → GetFileStatus(ctx)
    ↓
Cleanup & Exit
```

## Running the Demo

### Prerequisites

```bash
# Install OpenCode CLI
curl -fsSL https://opencode.ai/install | bash

# Verify installation
opencode --version
```

### Build and Run

```bash
# Build
make build
# or
go build -o bin/single-agent-demo ./cmd/single-agent-demo

# Run
./bin/single-agent-demo
```

### Expected Output

```
🚀 Single OpenCode Agent Demo
================================
Working directory: /home/user/open-swarm

📦 Step 1: Setting up infrastructure...
   ✅ Port range: 8000-8100 (101 ports available)
   ✅ Server manager ready (health timeout: 10s)

🔌 Step 2: Allocating port...
   ✅ Allocated port: 8000
   📊 Ports in use: 1, Available: 100

🖥️  Step 3: Booting OpenCode server on port 8000...
   ✅ Server running at http://localhost:8000 (PID: 12345)
   ⏱️  Boot time: 2.3s
   ✅ Health check passed

🔗 Step 4: Creating SDK client...
   ✅ Client connected to http://localhost:8000

🎯 Step 5: Executing task...
   Prompt: Create a simple hello.txt file with message...

📊 Task Results:
   Session ID: ses_abc123
   Message ID: msg_xyz789
   Duration: 5.2s
   Response parts: 2

✅ Step 6: Verifying results...
   ✅ File verified via SDK: hello.txt
   📄 Content: Hello from OpenCode agent!

🏥 Final health check...
   ✅ Server still healthy

✨ Demo completed successfully!
Total execution time: 8.1s
   Server will shutdown automatically...

🛑 Shutting down server...
   ✅ Server shutdown complete
```

## Testing

### Unit Tests

```bash
# Test infrastructure components
go test ./internal/infra/... -v

# Test agent client
go test ./internal/agent/... -v
```

### Integration Tests

```bash
# Run E2E test (requires opencode installed)
go test ./test/... -tags=integration -v

# Run specific test
go test ./test/... -tags=integration -run TestSingleAgentE2E -v
```

## Troubleshooting

### Server Fails to Boot

**Symptom**: `Failed to boot server: context deadline exceeded`

**Causes**:
- OpenCode CLI not installed
- Port already in use
- Working directory doesn't exist

**Solutions**:
```bash
# Check opencode installed
which opencode

# Check port availability
lsof -i :8000

# Verify working directory
pwd
ls -la
```

### Health Check Fails

**Symptom**: `Server health check failed after boot`

**Causes**:
- Server crashed during startup
- Network issues (firewall)
- Insufficient permissions

**Solutions**:
```bash
# Check server logs
journalctl -u opencode

# Test health endpoint manually
curl http://localhost:8000/health

# Check permissions
ls -la $(which opencode)
```

### Task Execution Fails

**Symptom**: `Task execution failed: connection refused`

**Causes**:
- Server died after boot
- SDK timeout
- Invalid prompt

**Solutions**:
```bash
# Verify server is running
ps aux | grep opencode

# Check server health
curl http://localhost:8000/health

# Increase timeout in main.go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
```

### File Verification Fails

**Symptom**: `Could not read hello.txt via SDK`

**Causes**:
- Agent used different filename
- File not tracked by SDK yet
- Timing issue (agent still processing)

**Solutions**:
- Check GetFileStatus() output for actual files created
- Increase sleep duration before verification
- Check agent's response parts for clues

## Configuration

### Port Range

Edit `main.go`:
```go
portMgr := infra.NewPortManager(8000, 8100)  // Change range
```

### Health Check Timeout

Edit `internal/infra/server.go`:
```go
serverMgr := infra.NewServerManager()
serverMgr.SetHealthTimeout(20 * time.Second)  // Increase timeout
```

### Task Timeout

Edit `main.go`:
```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
```

## Next Steps

After understanding the single-agent flow:

1. **Multi-Agent Coordination**: See `docs/ARCHITECTURE.md`
2. **Temporal Workflows**: See `docs/TCR-WORKFLOW.md`
3. **DAG Workflows**: See `docs/DAG-WORKFLOW.md`
4. **Merge Queue**: See `internal/mergequeue/README.md`

## References

- OpenCode SDK: https://github.com/sst/opencode-sdk-go
- OpenCode Docs: https://opencode.ai/docs
- Architecture: `docs/ARCHITECTURE.md`
- Contributing: `CONTRIBUTING.md`
