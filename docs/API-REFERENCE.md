# Open Swarm API Reference

## Overview

Open Swarm is a multi-agent coordination framework for orchestrating parallel task execution using isolated agent cells, Git worktrees, and OpenCode SDK clients. This document provides comprehensive documentation of all public APIs, including workflow input/output types, activity signatures, configuration options, and CLI flags.

## Table of Contents

1. [Configuration API](#configuration-api)
2. [Coordinator API](#coordinator-api)
3. [Agent API](#agent-api)
4. [Workflow Activities](#workflow-activities)
5. [Temporal Workflows](#temporal-workflows)
6. [Infrastructure API](#infrastructure-api)
7. [CLI Commands](#cli-commands)

---

## Configuration API

### Package: `open-swarm/internal/config`

The Configuration API manages all settings for Open Swarm projects, including project metadata, model preferences, MCP servers, and coordination behavior.

#### Config

```go
type Config struct {
    Project      ProjectConfig      // Project-level settings
    Model        ModelConfig        // Model preferences
    MCPServers   MCPServersConfig   // MCP server connections
    Behavior     BehaviorConfig     // Agent behavior settings
    Coordination CoordinationConfig // Coordination settings
    Build        BuildConfig        // Build command configuration
}
```

Main configuration container. Load using `config.Load()` which reads from `.claude/opencode.yaml`.

**Example YAML structure:**
```yaml
project:
  name: "my-project"
  description: "Project description"
  working_directory: "/path/to/project"

model:
  default: "claude-sonnet-4-5"
  agents:
    build: "claude-sonnet-4-5"
    plan: "claude-opus-4-1"

mcpServers:
  agent-mail:
    command: "mcp-agent-mail"
    description: "Agent Mail coordination server"
    enabled: true
    autostart: true
    priority: 1

behavior:
  auto_coordinate: true
  check_reservations: true
  auto_register: true
  preserve_threads: true
  use_todos: true

coordination:
  agent:
    program: "claude-code"
    model: "claude-sonnet-4-5"
  messages:
    auto_ack: true
    check_interval: 5
    importance_threshold: "normal"
  reservations:
    default_ttl: 3600
    auto_renew: true
    renew_threshold: 600
  threads:
    auto_create: true
    subject_prefix: "[open-swarm]"

build:
  commands:
    test: "go test ./..."
    build: "go build"
    lint: "golangci-lint run"
    fmt: "gofmt -w"
  slots:
    test: "test-slot"
    build: "build-slot"
```

#### ProjectConfig

```go
type ProjectConfig struct {
    Name             string // Project name (required)
    Description      string // Project description
    WorkingDirectory string // Working directory path (required)
}
```

Holds project-level metadata. The `WorkingDirectory` must be an absolute path to the project root.

#### ModelConfig

```go
type ModelConfig struct {
    Default string            // Default model for all agents
    Agents  map[string]string // Per-agent model overrides
}
```

Specifies which LLM models to use. The `Default` applies to all agents unless overridden in the `Agents` map.

#### MCPServerConfig

```go
type MCPServerConfig struct {
    Command     string // Command to start the server
    Description string // Server description
    Enabled     bool   // Whether the server is enabled
    AutoStart   bool   // Whether to start automatically
    Priority    int    // Server priority (higher = more important)
}
```

Configuration for a single MCP (Model Context Protocol) server. Used for extending OpenCode with additional capabilities.

#### BehaviorConfig

```go
type BehaviorConfig struct {
    AutoCoordinate    bool // Automatically coordinate agents
    CheckReservations bool // Check file reservations before operations
    AutoRegister      bool // Automatically register new agents
    PreserveThreads   bool // Keep message threads active
    UseTodos          bool // Use TodoWrite for task tracking
}
```

Controls agent behavior and coordination preferences.

#### CoordinationConfig

```go
type CoordinationConfig struct {
    Agent        AgentConfig        // Agent identity settings
    Messages     MessagesConfig     // Message handling
    Reservations ReservationsConfig // File reservation settings
    Threads      ThreadsConfig      // Thread behavior
}
```

Central coordination configuration container.

#### AgentConfig

```go
type AgentConfig struct {
    Program string // Agent program (e.g., "claude-code")
    Model   string // LLM model for the agent
}
```

Specifies agent identity and execution environment.

#### MessagesConfig

```go
type MessagesConfig struct {
    AutoAck             bool   // Automatically acknowledge received messages
    CheckInterval       int    // Message check interval in seconds
    ImportanceThreshold string // Minimum message importance to process
}
```

Controls message handling behavior.

#### ReservationsConfig

```go
type ReservationsConfig struct {
    DefaultTTL     int  // Default TTL in seconds for file reservations
    AutoRenew      bool // Automatically renew expiring reservations
    RenewThreshold int  // Seconds before expiry to trigger renewal
}
```

Manages file reservation behavior.

#### ThreadsConfig

```go
type ThreadsConfig struct {
    AutoCreate    bool   // Automatically create threads for new topics
    SubjectPrefix string // Prefix for new thread subjects
}
```

Controls thread creation and management.

#### BuildConfig

```go
type BuildConfig struct {
    Commands BuildCommands // Build commands
    Slots    BuildSlots    // Build slot names
}
```

Specifies build operations and resource slots.

#### BuildCommands

```go
type BuildCommands struct {
    Test  string // Test command (e.g., "go test ./...")
    Build string // Build command (e.g., "go build")
    Lint  string // Lint command (e.g., "golangci-lint run")
    Fmt   string // Format command (e.g., "gofmt -w")
}
```

Define all build-related commands.

#### BuildSlots

```go
type BuildSlots struct {
    Test  string // Test slot name
    Build string // Build slot name
}
```

Define build slot names for resource coordination.

#### Load() Function

```go
func Load() (*Config, error)
```

Loads configuration from `.claude/opencode.yaml` in the current working directory.

**Returns:**
- `*Config`: Parsed configuration
- `error`: Error if file not found, unreadable, or invalid YAML

**Errors:**
- File not found at `.claude/opencode.yaml`
- YAML parsing errors
- Invalid configuration format

**Example:**
```go
cfg, err := config.Load()
if err != nil {
    log.Fatalf("Failed to load configuration: %v", err)
}
```

#### Validate() Method

```go
func (c *Config) Validate() error
```

Validates the configuration for required fields.

**Validation Rules:**
- `Project.Name` is required
- `Project.WorkingDirectory` is required
- `Coordination.Agent.Program` is required
- `Coordination.Agent.Model` is required

**Returns:**
- `error`: Validation error, or nil if valid

**Example:**
```go
if err := cfg.Validate(); err != nil {
    log.Fatalf("Invalid configuration: %v", err)
}
```

---

## Coordinator API

### Package: `open-swarm/pkg/coordinator`

The Coordinator API manages multi-agent coordination for a project, tracking agents, synchronizing state, and monitoring project status.

#### Coordinator

```go
type Coordinator struct {
    // Private fields
    config       *config.Config
    projectKey   string
    agentManager *agent.Manager
}
```

Main coordinator for the project. Manages agent registration, status tracking, and synchronization with Agent Mail.

#### New() Function

```go
func New(cfg *config.Config) (*Coordinator, error)
```

Creates a new Coordinator instance.

**Parameters:**
- `cfg *config.Config`: Project configuration (required)

**Returns:**
- `*Coordinator`: New coordinator instance
- `error`: Error if config is nil

**Example:**
```go
cfg, _ := config.Load()
coord, err := coordinator.New(cfg)
if err != nil {
    log.Fatalf("Failed to create coordinator: %v", err)
}
```

#### Status

```go
type Status struct {
    ActiveAgents        int       // Number of active agents
    UnreadMessages      int       // Number of unread messages
    ActiveReservations  int       // Number of active file reservations
    ActiveThreads       int       // Number of active message threads
    LastSync            time.Time // Timestamp of last sync
    MCPServersConnected bool      // Whether MCP servers are connected
}
```

Current coordination status snapshot.

#### GetStatus() Method

```go
func (c *Coordinator) GetStatus() *Status
```

Returns the current coordination status.

**Returns:**
- `*Status`: Current status snapshot

**Example:**
```go
status := coord.GetStatus()
fmt.Printf("Active agents: %d\n", status.ActiveAgents)
fmt.Printf("Unread messages: %d\n", status.UnreadMessages)
```

#### ListAgents() Method

```go
func (c *Coordinator) ListAgents() []agent.Agent
```

Returns all active agents in the project.

**Returns:**
- `[]agent.Agent`: Slice of all registered agents

**Example:**
```go
agents := coord.ListAgents()
for _, agent := range agents {
    fmt.Printf("%s (%s, %s)\n", agent.Name, agent.Program, agent.Model)
}
```

#### RegisterAgent() Method

```go
func (c *Coordinator) RegisterAgent(name, program, model, taskDesc string) error
```

Registers a new agent in the project.

**Parameters:**
- `name string`: Agent name (e.g., "BlueLake")
- `program string`: Program name (e.g., "claude-code")
- `model string`: LLM model (e.g., "claude-sonnet-4-5")
- `taskDesc string`: Current task description

**Returns:**
- `error`: Error if registration fails

**Example:**
```go
err := coord.RegisterAgent(
    "BlueLake",
    "claude-code",
    "claude-sonnet-4-5",
    "Implementing API endpoints",
)
if err != nil {
    log.Fatalf("Registration failed: %v", err)
}
```

#### Sync() Method

```go
func (c *Coordinator) Sync() error
```

Synchronizes coordinator state with Agent Mail. Includes:
1. Ensure project is registered
2. Fetch recent messages
3. Update agent list
4. Check file reservations
5. Sync with beads and serena

**Returns:**
- `error`: Error if sync fails

**Example:**
```go
if err := coord.Sync(); err != nil {
    log.Fatalf("Sync failed: %v", err)
}
```

#### GetProjectKey() Method

```go
func (c *Coordinator) GetProjectKey() string
```

Returns the project key used with Agent Mail.

**Returns:**
- `string`: Project key (working directory path)

**Example:**
```go
projectKey := coord.GetProjectKey()
```

---

## Agent API

### Package: `open-swarm/pkg/agent`

The Agent API manages agent registration, discovery, and basic agent information tracking.

#### Agent

```go
type Agent struct {
    Name            string // Unique agent name
    Program         string // Program name (e.g., "claude-code")
    Model           string // LLM model (e.g., "claude-sonnet-4-5")
    TaskDescription string // Current task description
    LastActive      string // RFC3339 timestamp of last activity
    ProjectKey      string // Project this agent belongs to
}
```

Represents an agent working on the project.

#### Manager

```go
type Manager struct {
    // Private fields
    projectKey string
    agents     map[string]Agent
    mu         sync.RWMutex
}
```

Manages agent registry. Thread-safe.

#### NewManager() Function

```go
func NewManager(projectKey string) *Manager
```

Creates a new agent manager.

**Parameters:**
- `projectKey string`: Project identifier

**Returns:**
- `*Manager`: New manager instance

**Example:**
```go
manager := agent.NewManager("/path/to/project")
```

#### Register() Method

```go
func (m *Manager) Register(a Agent) error
```

Registers a new agent.

**Parameters:**
- `a Agent`: Agent to register (Name is required)

**Returns:**
- `error`: Error if Name is empty

**Example:**
```go
err := manager.Register(agent.Agent{
    Name:            "GreenCastle",
    Program:         "claude-code",
    Model:           "claude-sonnet-4-5",
    TaskDescription: "Implementing database schema",
    LastActive:      time.Now().Format(time.RFC3339),
})
```

#### Get() Method

```go
func (m *Manager) Get(name string) (Agent, bool)
```

Retrieves an agent by name.

**Parameters:**
- `name string`: Agent name

**Returns:**
- `Agent`: Agent data
- `bool`: True if found, false otherwise

**Example:**
```go
if agent, found := manager.Get("GreenCastle"); found {
    fmt.Printf("Agent: %s\n", agent.Name)
}
```

#### List() Method

```go
func (m *Manager) List() []Agent
```

Returns all registered agents.

**Returns:**
- `[]Agent`: Slice of all agents

**Example:**
```go
agents := manager.List()
for _, a := range agents {
    fmt.Printf("%s: %s\n", a.Name, a.TaskDescription)
}
```

#### CountActive() Method

```go
func (m *Manager) CountActive() int
```

Returns the number of active agents.

**Returns:**
- `int`: Count of registered agents

#### Remove() Method

```go
func (m *Manager) Remove(name string) error
```

Removes an agent from the registry.

**Parameters:**
- `name string`: Agent name

**Returns:**
- `error`: Error if agent not found

#### Update() Method

```go
func (m *Manager) Update(a Agent) error
```

Updates an agent's information.

**Parameters:**
- `a Agent`: Updated agent data (Name is required)

**Returns:**
- `error`: Error if agent not found

---

## Workflow Activities

### Package: `open-swarm/internal/workflow`

Workflow activities provide the core operations for agent cell lifecycle and task execution.

#### Activities

```go
type Activities struct {
    // Private fields
    portManager      *infra.PortManager
    serverManager    *infra.ServerManager
    worktreeManager  *infra.WorktreeManager
}
```

Container for all workflow activities.

#### NewActivities() Function

```go
func NewActivities(
    portMgr *infra.PortManager,
    serverMgr *infra.ServerManager,
    worktreeMgr *infra.WorktreeManager,
) *Activities
```

Creates a new Activities instance.

**Parameters:**
- `portMgr`: Port manager for allocating ports
- `serverMgr`: Server manager for lifecycle
- `worktreeMgr`: Worktree manager for isolation

**Example:**
```go
activities := workflow.NewActivities(
    infra.NewPortManager(8000, 9000),
    infra.NewServerManager(),
    infra.NewWorktreeManager(repoDir, baseDir),
)
```

#### CellBootstrap

```go
type CellBootstrap struct {
    CellID       string                  // Unique cell identifier
    Port         int                     // Allocated port
    WorktreeID   string                  // Git worktree ID
    WorktreePath string                  // Path to worktree
    ServerHandle *infra.ServerHandle     // Running server instance
    Client       *agent.Client           // OpenCode SDK client
}
```

Represents a bootstrapped agent cell with all resources allocated.

#### BootstrapCell() Activity

```go
func (a *Activities) BootstrapCell(
    ctx context.Context,
    cellID string,
    branch string,
) (*CellBootstrap, error)
```

Creates a complete isolated cell for agent execution. This activity:
1. Allocates a unique port (INV-001)
2. Creates an isolated Git worktree
3. Boots an OpenCode server (INV-002, INV-003)
4. Sets up SDK client (INV-004)

**Parameters:**
- `ctx context.Context`: Context for cancellation
- `cellID string`: Unique identifier for the cell
- `branch string`: Git branch to checkout in worktree

**Returns:**
- `*CellBootstrap`: Bootstrapped cell with resources
- `error`: Error if any step fails

**Enforced Invariants:**
- INV-001: Each agent runs OpenCode serve on a unique port
- INV-002: Server working directory must be set to the Git worktree
- INV-003: Supervisor must wait for server healthcheck (200 OK) before SDK connection
- INV-004: SDK client must be configured with specific BaseURL (localhost:PORT)

**Example:**
```go
cell, err := activities.BootstrapCell(ctx, "primary", "main")
if err != nil {
    log.Fatalf("Bootstrap failed: %v", err)
}
fmt.Printf("Cell running on port %d\n", cell.Port)
fmt.Printf("Worktree at %s\n", cell.WorktreePath)
```

#### TeardownCell() Activity

```go
func (a *Activities) TeardownCell(
    ctx context.Context,
    cell *CellBootstrap,
) error
```

Destroys a cell and releases all resources. Steps:
1. Shut down OpenCode server (INV-005)
2. Remove Git worktree
3. Release port back to pool

**Parameters:**
- `ctx context.Context`: Context for cancellation
- `cell *CellBootstrap`: Cell to tear down

**Returns:**
- `error`: Error if any cleanup fails (collects all errors)

**Enforced Invariants:**
- INV-005: Server process must be killed when workflow activity completes

**Example:**
```go
defer func() {
    if err := activities.TeardownCell(ctx, cell); err != nil {
        log.Printf("Warning: teardown errors: %v", err)
    }
}()
```

#### ExecuteTask() Activity

```go
func (a *Activities) ExecuteTask(
    ctx context.Context,
    cell *CellBootstrap,
    task *agent.TaskContext,
) (*agent.ExecutionResult, error)
```

Executes a task within a cell. Steps:
1. Verify server health
2. Execute prompt via SDK (INV-006)
3. Get file modifications
4. Return execution result

**Parameters:**
- `ctx context.Context`: Context for cancellation
- `cell *CellBootstrap`: Target cell for execution
- `task *agent.TaskContext`: Task to execute

**Returns:**
- `*agent.ExecutionResult`: Task execution result
- `error`: Error if execution fails

**Enforced Invariants:**
- INV-006: Command execution must use SDK client.ExecutePrompt()

**Example:**
```go
result, err := activities.ExecuteTask(ctx, cell, &agent.TaskContext{
    TaskID:  "task-001",
    Prompt:  "Implement the user authentication endpoint",
    Files:   []string{"internal/auth/auth.go"},
})
if err != nil {
    log.Fatalf("Task failed: %v", err)
}
fmt.Printf("Files modified: %v\n", result.FilesModified)
```

#### RunTests() Activity

```go
func (a *Activities) RunTests(
    ctx context.Context,
    cell *CellBootstrap,
) (bool, error)
```

Executes tests in the cell using the configured test command.

**Parameters:**
- `ctx context.Context`: Context for cancellation
- `cell *CellBootstrap`: Target cell

**Returns:**
- `bool`: True if tests passed, false otherwise
- `error`: Error if execution fails

**Example:**
```go
passed, err := activities.RunTests(ctx, cell)
if err != nil {
    log.Fatalf("Test execution failed: %v", err)
}
if !passed {
    fmt.Println("Tests failed")
}
```

#### CommitChanges() Activity

```go
func (a *Activities) CommitChanges(
    ctx context.Context,
    cell *CellBootstrap,
    message string,
) error
```

Commits all changes in the worktree.

**Parameters:**
- `ctx context.Context`: Context for cancellation
- `cell *CellBootstrap`: Target cell
- `message string`: Commit message

**Returns:**
- `error`: Error if git operations fail

**Example:**
```go
err := activities.CommitChanges(ctx, cell, "Add authentication endpoints")
if err != nil {
    log.Fatalf("Commit failed: %v", err)
}
```

#### RevertChanges() Activity

```go
func (a *Activities) RevertChanges(
    ctx context.Context,
    cell *CellBootstrap,
) error
```

Reverts all changes in the worktree to HEAD.

**Parameters:**
- `ctx context.Context`: Context for cancellation
- `cell *CellBootstrap`: Target cell

**Returns:**
- `error`: Error if revert fails

**Example:**
```go
if err := activities.RevertChanges(ctx, cell); err != nil {
    log.Fatalf("Revert failed: %v", err)
}
```

---

## Agent Client API

### Package: `open-swarm/internal/agent`

The Agent Client API provides high-level wrappers around the OpenCode SDK for session management, prompt execution, and file operations.

#### Client

```go
type Client struct {
    // Private fields
    sdk     *opencode.Client
    baseURL string
    port    int
}
```

Wraps the OpenCode SDK client with reactor-specific functionality.

#### NewClient() Function

```go
func NewClient(baseURL string, port int) *Client
```

Creates a new OpenCode SDK client configured for a specific server instance.

**Parameters:**
- `baseURL string`: Server base URL (e.g., "http://localhost:8000")
- `port int`: Server port number

**Returns:**
- `*Client`: New client instance

**Enforced Invariants:**
- INV-004: SDK client must be configured with specific BaseURL (localhost:PORT)

**Example:**
```go
client := agent.NewClient("http://localhost:8000", 8000)
```

#### PromptOptions

```go
type PromptOptions struct {
    SessionID    string   // Session ID to use (if empty, creates new)
    Title        string   // Title for new session
    Model        string   // Model to use (e.g., "anthropic/claude-sonnet-4-5")
    Agent        string   // Agent to use (e.g., "build", "plan")
    NoReply      bool     // Context injection without AI response
    SystemPrompt string   // Override system prompt
    Tools        []string // Tools to enable
}
```

Configures how a prompt is executed.

#### PromptResult

```go
type PromptResult struct {
    SessionID string       // Session ID
    MessageID string       // Message ID
    Parts     []ResultPart // Response parts
}
```

Contains the result of a prompt execution.

#### ResultPart

```go
type ResultPart struct {
    Type       string      // "text" or "tool"
    Text       string      // Text content
    ToolName   string      // Tool name if type == "tool"
    ToolResult interface{} // Tool execution result
}
```

Represents a part of the response.

#### ExecutePrompt() Method

```go
func (c *Client) ExecutePrompt(
    ctx context.Context,
    prompt string,
    opts *PromptOptions,
) (*PromptResult, error)
```

Sends a prompt to the OpenCode server and returns the response.

**Parameters:**
- `ctx context.Context`: Context for cancellation
- `prompt string`: The prompt to send
- `opts *PromptOptions`: Execution options (can be nil for defaults)

**Returns:**
- `*PromptResult`: Response from the server
- `error`: Error if execution fails

**Example:**
```go
result, err := client.ExecutePrompt(ctx, "Implement user login", &agent.PromptOptions{
    Title: "Auth implementation",
    Agent: "build",
})
if err != nil {
    log.Fatalf("Prompt failed: %v", err)
}
fmt.Printf("Response: %s\n", result.GetText())
```

#### ExecuteCommand() Method

```go
func (c *Client) ExecuteCommand(
    ctx context.Context,
    sessionID string,
    command string,
    args []string,
) (*PromptResult, error)
```

Executes a slash command on the OpenCode server.

**Parameters:**
- `ctx context.Context`: Context for cancellation
- `sessionID string`: Target session ID
- `command string`: Command name (e.g., "shell")
- `args []string`: Command arguments

**Returns:**
- `*PromptResult`: Command result
- `error`: Error if execution fails

**Enforced Invariants:**
- INV-006: Command execution must use SDK

**Example:**
```go
result, err := client.ExecuteCommand(ctx, sessionID, "shell", []string{"go", "test", "./..."})
```

#### ListSessions() Method

```go
func (c *Client) ListSessions(ctx context.Context) ([]opencode.Session, error)
```

Returns all sessions on the server.

**Returns:**
- `[]opencode.Session`: Slice of sessions
- `error`: Error if operation fails

#### GetSession() Method

```go
func (c *Client) GetSession(ctx context.Context, sessionID string) (*opencode.Session, error)
```

Retrieves a specific session.

**Parameters:**
- `sessionID string`: Target session ID

**Returns:**
- `*opencode.Session`: Session data
- `error`: Error if not found

#### DeleteSession() Method

```go
func (c *Client) DeleteSession(ctx context.Context, sessionID string) error
```

Deletes a session.

**Parameters:**
- `sessionID string`: Session to delete

**Returns:**
- `error`: Error if deletion fails

#### AbortSession() Method

```go
func (c *Client) AbortSession(ctx context.Context, sessionID string) error
```

Aborts a running session.

**Parameters:**
- `sessionID string`: Session to abort

**Returns:**
- `error`: Error if abort fails

#### GetFileStatus() Method

```go
func (c *Client) GetFileStatus(ctx context.Context) ([]opencode.File, error)
```

Retrieves the status of tracked files.

**Returns:**
- `[]opencode.File`: File status information
- `error`: Error if operation fails

#### ReadFile() Method

```go
func (c *Client) ReadFile(ctx context.Context, path string) (string, error)
```

Reads the content of a file.

**Parameters:**
- `path string`: File path

**Returns:**
- `string`: File content
- `error`: Error if read fails

#### TaskContext

```go
type TaskContext struct {
    TaskID      string   // Unique task identifier
    Description string   // Task description
    Files       []string // Related file paths
    Prompt      string   // Task prompt
}
```

Represents the context for a task execution.

#### ExecutionResult

```go
type ExecutionResult struct {
    Success       bool     // Whether task succeeded
    Output        string   // Execution output
    FilesModified []string // Modified file paths
    TestsPassed   bool     // Whether tests passed
    ErrorMessage  string   // Error message if failed
    SessionID     string   // Associated session ID
}
```

Result of a task execution.

---

## Temporal Workflows

### Package: `open-swarm/internal/temporal`

Temporal workflows orchestrate complex multi-step operations using Temporal's workflow engine.

#### TCRWorkflowInput

```go
type TCRWorkflowInput struct {
    CellID      string // Unique cell identifier
    Branch      string // Git branch to use
    TaskID      string // Task identifier
    Description string // Task description
    Prompt      string // Task prompt
}
```

Input parameters for the Test-Commit-Revert workflow.

#### TCRWorkflowResult

```go
type TCRWorkflowResult struct {
    Success      bool     // Whether workflow succeeded
    TestsPassed  bool     // Whether tests passed
    FilesChanged []string // Files changed by the task
    Error        string   // Error message if failed
}
```

Result of the Test-Commit-Revert workflow.

#### TCRWorkflow() Function

```go
func TCRWorkflow(
    ctx workflow.Context,
    input TCRWorkflowInput,
) (*TCRWorkflowResult, error)
```

Implements the Test-Commit-Revert pattern:
1. Bootstrap isolated cell
2. Execute task
3. Run tests
4. Commit changes if tests pass, otherwise revert
5. Tear down cell

**Parameters:**
- `ctx workflow.Context`: Temporal workflow context
- `input TCRWorkflowInput`: Workflow input parameters

**Returns:**
- `*TCRWorkflowResult`: Workflow result
- `error`: Error if workflow fails (recoverable)

**Activity Configuration:**
- StartToCloseTimeout: 10 minutes
- HeartbeatTimeout: 30 seconds
- RetryPolicy: MaximumAttempts: 1 (no retries for non-idempotent operations)

**Guarantees:**
- Teardown always executes (saga pattern)
- Uses disconnected context for cleanup operations
- Uses 3 retries for cleanup operations (idempotent)

**Example:**
```go
client, _ := temporal.NewClient(client.Options{})
defer client.Close()

we, _ := client.ExecuteWorkflow(context.Background(),
    client.StartWorkflowOptions{
        ID:        "tcr-task-001",
        TaskQueue: "reactor-task-queue",
    },
    temporal.TCRWorkflow,
    temporal.TCRWorkflowInput{
        CellID:      "primary",
        Branch:      "main",
        TaskID:      "task-001",
        Description: "Implement auth endpoints",
        Prompt:      "Create user login and signup endpoints",
    },
)

var result temporal.TCRWorkflowResult
_ = we.Get(context.Background(), &result)

if result.Success {
    fmt.Printf("Files changed: %v\n", result.FilesChanged)
}
```

---

## Infrastructure API

### Package: `open-swarm/internal/infra`

Infrastructure APIs manage low-level resources: ports, servers, and worktrees.

#### PortManager

```go
type PortManager struct {
    // Private fields
    mu        sync.Mutex
    minPort   int
    maxPort   int
    allocated map[int]bool
    nextPort  int
}
```

Manages allocation of ports in the range specified at creation.

**Enforced Invariants:**
- INV-001: Each agent runs OpenCode serve on a unique port

#### NewPortManager() Function

```go
func NewPortManager(minPort, maxPort int) *PortManager
```

Creates a new port manager with the specified range.

**Parameters:**
- `minPort int`: Minimum port (inclusive)
- `maxPort int`: Maximum port (inclusive)

**Returns:**
- `*PortManager`: New manager instance

**Example:**
```go
portMgr := infra.NewPortManager(8000, 9000) // 1001 ports available
```

#### Allocate() Method

```go
func (pm *PortManager) Allocate() (int, error)
```

Reserves the next available port.

**Returns:**
- `int`: Allocated port number
- `error`: Error if no ports available

**Example:**
```go
port, err := portMgr.Allocate()
if err != nil {
    log.Fatalf("Port allocation failed: %v", err)
}
fmt.Printf("Allocated port: %d\n", port)
```

#### Release() Method

```go
func (pm *PortManager) Release(port int) error
```

Frees a previously allocated port.

**Parameters:**
- `port int`: Port to release

**Returns:**
- `error`: Error if port was not allocated

**Example:**
```go
if err := portMgr.Release(8000); err != nil {
    log.Printf("Warning: %v", err)
}
```

#### AllocatedCount() Method

```go
func (pm *PortManager) AllocatedCount() int
```

Returns the number of currently allocated ports.

#### AvailableCount() Method

```go
func (pm *PortManager) AvailableCount() int
```

Returns the number of available ports.

#### IsAllocated() Method

```go
func (pm *PortManager) IsAllocated(port int) bool
```

Checks if a specific port is currently allocated.

#### ServerHandle

```go
type ServerHandle struct {
    Port       int              // Server port
    WorktreeID string           // Associated worktree
    WorkDir    string           // Working directory
    Cmd        *exec.Cmd        // Process command
    BaseURL    string           // Server base URL
    PID        int              // Process ID
}
```

Represents a running OpenCode server instance.

**Enforced Invariants:**
- INV-002: Working directory must be set to Git worktree
- INV-003: Must wait for healthcheck before SDK connection

#### ServerManager

```go
type ServerManager struct {
    // Private fields
    opencodeCommand string
    healthTimeout   time.Duration
    healthInterval  time.Duration
}
```

Handles the lifecycle of OpenCode serve processes.

#### NewServerManager() Function

```go
func NewServerManager() *ServerManager
```

Creates a new server manager.

**Default Configuration:**
- HealthTimeout: 10 seconds
- HealthInterval: 200 milliseconds

**Returns:**
- `*ServerManager`: New manager instance

#### BootServer() Method

```go
func (sm *ServerManager) BootServer(
    ctx context.Context,
    worktreePath string,
    worktreeID string,
    port int,
) (*ServerHandle, error)
```

Starts an OpenCode server on the specified port and working directory.

**Parameters:**
- `ctx context.Context`: Context for cancellation
- `worktreePath string`: Path to Git worktree
- `worktreeID string`: Worktree identifier
- `port int`: Port to listen on

**Returns:**
- `*ServerHandle`: Running server instance
- `error`: Error if boot fails

**Enforced Invariants:**
- INV-002: Agent server working directory must be set to the Git worktree
- INV-003: Supervisor must wait for server healthcheck (200 OK) before SDK connection

**Health Check:**
- Polls `/health` endpoint every 200ms
- Timeout: 10 seconds
- Success: 200 OK response

**Example:**
```go
handle, err := serverMgr.BootServer(ctx, "/path/to/worktree", "wt-001", 8000)
if err != nil {
    log.Fatalf("Server boot failed: %v", err)
}
fmt.Printf("Server ready at %s\n", handle.BaseURL)
```

#### Shutdown() Method

```go
func (sm *ServerManager) Shutdown(handle *ServerHandle) error
```

Gracefully stops the OpenCode server.

**Parameters:**
- `handle *ServerHandle`: Server instance to shut down

**Returns:**
- `error`: Error if shutdown fails

**Enforced Invariants:**
- INV-005: Server process must be killed when activity completes

**Shutdown Sequence:**
1. Send SIGTERM to process group
2. Wait up to 5 seconds
3. Force kill with SIGKILL if timeout

**Example:**
```go
if err := serverMgr.Shutdown(handle); err != nil {
    log.Printf("Warning: %v", err)
}
```

#### IsHealthy() Method

```go
func (sm *ServerManager) IsHealthy(handle *ServerHandle) bool
```

Checks if the server is still responsive.

**Returns:**
- `bool`: True if server responds with 200 OK

#### SetOpencodeCommand() Method

```go
func (sm *ServerManager) SetOpencodeCommand(cmd string)
```

Overrides the OpenCode command (useful for testing).

#### SetHealthTimeout() Method

```go
func (sm *ServerManager) SetHealthTimeout(timeout time.Duration)
```

Sets the health check timeout duration.

#### WorktreeManager

```go
type WorktreeManager struct {
    // Private fields
    baseDir string // Base directory for worktrees
    repoDir string // Repository directory
}
```

Manages Git worktrees for agent isolation.

#### WorktreeInfo

```go
type WorktreeInfo struct {
    ID   string // Worktree ID
    Path string // Full path to worktree
}
```

Information about a worktree.

#### NewWorktreeManager() Function

```go
func NewWorktreeManager(repoDir, baseDir string) *WorktreeManager
```

Creates a new worktree manager.

**Parameters:**
- `repoDir string`: Path to Git repository
- `baseDir string`: Base directory for creating worktrees

**Returns:**
- `*WorktreeManager`: New manager instance

**Example:**
```go
wtMgr := infra.NewWorktreeManager(".", "./worktrees")
```

#### CreateWorktree() Method

```go
func (wm *WorktreeManager) CreateWorktree(
    id string,
    branch string,
) (*WorktreeInfo, error)
```

Creates a new Git worktree for agent isolation.

**Parameters:**
- `id string`: Unique worktree identifier
- `branch string`: Git branch to checkout

**Returns:**
- `*WorktreeInfo`: Worktree information
- `error`: Error if creation fails

**Example:**
```go
wt, err := wtMgr.CreateWorktree("cell-001", "main")
if err != nil {
    log.Fatalf("Worktree creation failed: %v", err)
}
fmt.Printf("Worktree at %s\n", wt.Path)
```

#### RemoveWorktree() Method

```go
func (wm *WorktreeManager) RemoveWorktree(id string) error
```

Removes a Git worktree.

**Parameters:**
- `id string`: Worktree identifier

**Returns:**
- `error`: Error if removal fails

#### ListWorktrees() Method

```go
func (wm *WorktreeManager) ListWorktrees() ([]*WorktreeInfo, error)
```

Lists all worktrees in the repository.

**Returns:**
- `[]*WorktreeInfo`: Slice of worktree information
- `error`: Error if listing fails

#### CleanupAll() Method

```go
func (wm *WorktreeManager) CleanupAll() error
```

Removes all worktrees in the base directory and prunes stale references.

**Returns:**
- `error`: Error if cleanup fails

**Example:**
```go
if err := wtMgr.CleanupAll(); err != nil {
    log.Printf("Warning: %v", err)
}
```

#### PruneWorktrees() Method

```go
func (wm *WorktreeManager) PruneWorktrees() error
```

Removes worktree administrative information for missing worktrees.

**Returns:**
- `error`: Error if pruning fails

---

## CLI Commands

### open-swarm CLI

Main command-line interface for project coordination.

#### Installation

```bash
go install open-swarm/cmd/open-swarm@latest
```

#### Usage

```
Usage: open-swarm <command>

Commands:
  status    Show project coordination status
  agents    List all active agents
  sync      Synchronize with coordination state
  version   Show version information
  help      Show this help message
```

#### open-swarm status

```bash
open-swarm status
```

Shows current coordination status including active agents, unread messages, file reservations, and message threads.

**Output:**
```
📊 Project Status
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

🤖 Agents: 3 active
📬 Messages: 5 unread
📁 File Reservations: 2 active
🧵 Threads: 1 active

✓ Status check complete
```

#### open-swarm agents

```bash
open-swarm agents
```

Lists all active agents with their status and task descriptions.

**Output:**
```
🤖 Active Agents
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

GreenCastle (claude-code, claude-sonnet-4-5)
  Task: Implementing API endpoints
  Last active: 2025-12-12T15:30:45Z

BlueLake (claude-code, claude-sonnet-4-5)
  Task: Writing unit tests
  Last active: 2025-12-12T15:28:20Z

Total: 2 agents
```

#### open-swarm sync

```bash
open-swarm sync
```

Synchronizes coordinator state with Agent Mail. Includes:
1. Project registration
2. Agent list update
3. Message queue check
4. File reservation sync

**Output:**
```
🔄 Synchronizing with coordination state...
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

✓ Project registered in Agent Mail
✓ Agent list synchronized
✓ Message queue checked
✓ File reservations updated

✓ Synchronization complete
```

#### open-swarm version

```bash
open-swarm version
```

Shows version information.

**Output:**
```
Open Swarm version 0.1.0
```

#### open-swarm help

```bash
open-swarm help
```

Shows help message.

### reactor CLI

Command-line interface for executing workflows.

#### Installation

```bash
go install open-swarm/cmd/reactor@latest
```

#### Usage

```
Usage: reactor [flags]

Flags:
  -repo string
        Git repository directory (default ".")
  -worktrees string
        Base directory for worktrees (default "./worktrees")
  -branch string
        Branch to use for worktrees (default "main")
  -max-agents int
        Maximum concurrent agents (default 50)
  -task string
        Task ID to execute (required)
  -desc string
        Task description
  -prompt string
        Task prompt (required)
  -parallel
        Run tasks in parallel mode (not yet implemented)
```

#### Example: Execute a Task

```bash
reactor \
  -repo /path/to/project \
  -worktrees ./worktrees \
  -branch main \
  -task "auth-impl-001" \
  -desc "Implement user authentication" \
  -prompt "Create user login and signup endpoints with JWT tokens"
```

**Output:**
```
🚀 Reactor-SDK v6.0.0 - Enterprise Agent Orchestrator
📊 Configuration:
   Repository: /path/to/project
   Worktree Base: ./worktrees
   Branch: main
   Max Agents: 50
   Port Range: 8000-9000

🔧 Initializing infrastructure...
🧹 Cleaning up existing worktrees...
⚙️  Initializing workflow engine...
🎯 Starting task execution...
📦 Bootstrapping agent cell...
✅ Cell bootstrapped on port 8000
📁 Worktree: ./worktrees/cell-primary-1702396245

⚙️  Executing task...
✅ Task completed successfully
📝 Output:
[Task output here...]

📁 Modified files:
   - internal/auth/auth.go
   - internal/auth/jwt.go

🧪 Running tests...
✅ Tests passed

💾 Committing changes...
✅ Changes committed

✅ Reactor execution complete
```

### reactor-client CLI

Command-line client for submitting Temporal workflows.

#### Installation

```bash
go install open-swarm/cmd/reactor-client@latest
```

#### Usage

```
Usage: reactor-client [flags]

Flags:
  -workflow string
        Workflow type: tcr or dag (default "tcr")
  -task string
        Task ID (required)
  -prompt string
        Task prompt (required)
  -desc string
        Task description
  -branch string
        Git branch (default "main")
```

#### Example: Submit TCR Workflow

```bash
reactor-client \
  -workflow tcr \
  -task "feature-001" \
  -desc "Implement user dashboard" \
  -prompt "Create a dashboard showing user statistics" \
  -branch develop
```

**Output:**
```
✅ Workflow started
   ID: reactor-feature-001
   RunID: 12345678-1234-1234-1234-123456789012
   Web UI: http://localhost:8233/namespaces/default/workflows/reactor-feature-001

⏳ Waiting for workflow to complete...
✅ Workflow succeeded!
   Tests: PASSED
   Files changed: [internal/dashboard/dashboard.go internal/dashboard/views.go]
```

#### Temporal Web UI

Monitor workflows at: http://localhost:8233/namespaces/default/workflows/

---

## Error Handling

All APIs use consistent error handling patterns:

### Configuration Errors

```go
cfg, err := config.Load()
if err != nil {
    // Possible errors:
    // - "configuration file not found: .claude/opencode.yaml"
    // - "failed to parse config: [YAML error]"
    // - "[validation errors from Validate()]"
    log.Fatalf("Failed to load configuration: %v", err)
}
```

### Infrastructure Errors

```go
// Port exhaustion
port, err := portMgr.Allocate()
if err != nil {
    // "no available ports in range 8000-9000 (all 1001 ports allocated)"
}

// Server boot timeout
handle, err := serverMgr.BootServer(ctx, path, id, port)
if err != nil {
    // "opencode server on port 8000 failed to become ready within 10s"
}

// Worktree operations
wt, err := wtMgr.CreateWorktree(id, branch)
if err != nil {
    // "failed to create worktree: [git error]"
}
```

### Workflow Errors

```go
var result temporal.TCRWorkflowResult
err := we.Get(context.Background(), &result)
if err != nil {
    // Workflow execution error (rare)
    log.Fatalf("Workflow failed: %v", err)
}

if !result.Success {
    // Task execution failed
    log.Printf("Task failed: %s", result.Error)
}
```

---

## Configuration Examples

### Minimal Configuration

```yaml
project:
  name: "my-project"
  working_directory: "/path/to/project"

coordination:
  agent:
    program: "claude-code"
    model: "claude-sonnet-4-5"
```

### Full Configuration

See the "Full Configuration" section in the Configuration API section above.

### Environment-Specific Configuration

Create separate configuration files and load them explicitly:

```go
// For production
cfg, _ := yaml.Unmarshal(prodConfigBytes, &config.Config{})

// For testing
cfg, _ := yaml.Unmarshal(testConfigBytes, &config.Config{})
```

---

## Best Practices

### 1. Resource Cleanup

Always defer cleanup of resources:

```go
cell, _ := activities.BootstrapCell(ctx, "primary", "main")
defer func() {
    if err := activities.TeardownCell(ctx, cell); err != nil {
        log.Printf("Warning: teardown error: %v", err)
    }
}()
```

### 2. Context Usage

Always pass context for cancellation and timeouts:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()

result, err := activities.ExecuteTask(ctx, cell, task)
```

### 3. Error Aggregation

Collect all errors during cleanup:

```go
var errs []error

if err := serverMgr.Shutdown(handle); err != nil {
    errs = append(errs, err)
}

if err := wtMgr.RemoveWorktree(id); err != nil {
    errs = append(errs, err)
}

if len(errs) > 0 {
    return fmt.Errorf("cleanup errors: %v", errs)
}
```

### 4. Workflow Compensation

Use saga pattern for complex workflows:

```go
// Ensure cleanup even if workflow fails
defer func() {
    disconnCtx, _ := workflow.NewDisconnectedContext(ctx)
    _ = workflow.ExecuteActivity(disconnCtx, cleanup).Get(disconnCtx, nil)
}()
```

### 5. Port Management

Use PortManager to avoid port conflicts:

```go
portMgr := infra.NewPortManager(8000, 9000)
defer func() {
    for _, port := range allocatedPorts {
        _ = portMgr.Release(port)
    }
}()
```

---

## Thread Safety

The following types are thread-safe:

- `agent.Manager` (uses sync.RWMutex)
- `infra.PortManager` (uses sync.Mutex)

The following types are NOT thread-safe:

- `config.Config` (immutable after loading, safe to share)
- `infra.ServerManager` (stateless, safe to share)
- `infra.WorktreeManager` (assumes sequential Git operations)

---

## Version Information

- **Open Swarm Version:** 0.1.0
- **Go Version:** 1.18+
- **Temporal Go SDK:** Latest
- **OpenCode SDK:** Latest

---

## Related Documentation

- [TCR Workflow Details](TCR-WORKFLOW.md)
- [DAG Workflow Details](DAG-WORKFLOW.md)
- [Deployment Guide](DEPLOYMENT.md)
- [Troubleshooting Guide](TROUBLESHOOTING.md)
- [Monitoring Guide](MONITORING.md)
