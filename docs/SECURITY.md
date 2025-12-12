# Open Swarm Security Guide

This document outlines the security architecture and best practices for deploying and operating Open Swarm, a multi-agent coordination framework combining OpenCode, Agent Mail, Beads, and Temporal.

## Table of Contents

1. [Overview](#overview)
2. [Network Isolation](#network-isolation)
3. [Secrets Management](#secrets-management)
4. [OpenCode Server Security](#opencode-server-security)
5. [Temporal Authentication](#temporal-authentication)
6. [Worktree Isolation](#worktree-isolation)
7. [Input Validation](#input-validation)
8. [Security Best Practices](#security-best-practices)
9. [Incident Response](#incident-response)

---

## Overview

Open Swarm implements a layered security approach protecting:

- **Agent isolation** through Git worktrees
- **Network security** through controlled port allocation
- **Authentication** via environment-based secrets and Temporal credentials
- **Input validation** across all user inputs and configuration
- **Process isolation** through dedicated OpenCode server instances per agent

### Security Invariants

The system enforces critical security invariants:

- **INV-001**: Each agent runs on a unique, allocated port (8000-9000 range)
- **INV-002**: Agent server working directory must be set to the Git worktree
- **INV-003**: Supervisor waits for server healthcheck before SDK connection
- **INV-004**: SDK client configured with specific BaseURL (localhost:PORT)
- **INV-005**: Server process killed when workflow activity completes
- **INV-006**: Command execution uses SDK `client.Command.Execute()`

---

## Network Isolation

### Architecture

Open Swarm enforces strict network isolation at multiple levels:

#### 1. Port Allocation Management

Each agent runs on a unique port within the range **8000-9000**, enforcing strict isolation:

**File:** `/internal/infra/ports.go`

```go
// PortManager manages the allocation of ports in the range 8000-9000
type PortManager struct {
    mu        sync.Mutex
    minPort   int
    maxPort   int
    allocated map[int]bool
    nextPort  int
}
```

**Key Features:**
- Thread-safe port allocation with mutex protection
- Circular allocation strategy prevents port exhaustion
- Allocation tracking prevents conflicts across agents
- Maximum 1001 concurrent agents (ports 8000-9000)

**Usage:**
```go
// Allocate a unique port for each agent
port, err := portManager.Allocate()

// Release port when agent terminates
portManager.Release(port)
```

#### 2. OpenCode Server Isolation

Each agent runs a dedicated OpenCode server instance bound to localhost:

**File:** `/internal/infra/server.go`

**Key Security Controls:**

1. **localhost-only Binding**: Servers bind to `localhost:PORT` only
   ```go
   cmd := exec.CommandContext(ctx, "opencode", "serve",
       "--port", fmt.Sprintf("%d", port),
       "--hostname", "localhost",  // Restrict to localhost
   )
   ```

2. **Healthcheck Enforcement**: SDK client waits for server readiness
   ```go
   // Wait for /health endpoint before connecting SDK
   resp, err := client.Get(baseURL + "/health")
   if resp.StatusCode == 200 {
       // Server ready for SDK connection
   }
   ```

3. **Process Group Isolation**: Clean process termination
   ```go
   cmd.SysProcAttr = &syscall.SysProcAttr{
       Setpgid: true,  // Create new process group
   }
   ```

#### 3. Firewall Rules (Production)

Implement strict firewall rules for production deployment:

```bash
# Allow SSH access only
sudo ufw allow 22/tcp

# Allow application access (from load balancer only)
sudo ufw allow from 10.0.0.0/8 to any port 8080

# Allow Temporal (internal only)
sudo ufw allow from 10.0.0.0/8 to any port 7233

# Allow PostgreSQL (internal only)
sudo ufw allow from 10.0.0.0/8 to any port 5432

# Block all other incoming
sudo ufw default deny incoming
sudo ufw default allow outgoing
```

#### 4. Network Segmentation

Deploy services on separate networks:

```yaml
networks:
  open-swarm-network:
    driver: bridge
    ipam:
      config:
        - subnet: 10.20.0.0/16

services:
  postgresql:
    networks:
      - open-swarm-network

  temporal:
    networks:
      - open-swarm-network

  open-swarm:
    networks:
      - open-swarm-network
```

### Network Communication Flow

```
[Agent A] ------- localhost:8001 (OpenCode Server A)
                      |
                      v
                 SDK Client (port-specific)
                      |
              (restricted to localhost)

[Agent B] ------- localhost:8002 (OpenCode Server B)
                      |
                      v
                 SDK Client (port-specific)
```

**Cross-service communication:**

```
[OpenCode Server] ---> [Temporal Server:7233] (gRPC, TLS capable)
                    ---> [Agent Mail:8765] (HTTP)
                    ---> [PostgreSQL:5432] (encrypted connection)
```

---

## Secrets Management

### Environment Variable Strategy

Store all sensitive data in environment variables, never hardcode secrets:

**File:** `docker-compose.yml` / `.env` / systemd service files

#### 1. API Keys

```bash
# OpenCode/Anthropic API Key
ANTHROPIC_API_KEY=sk-ant-XXXXXXXXXXXXXXXXXXXXXXXX

# Alternative providers
OPENAI_API_KEY=sk-XXXXXXXXXXXXXXXXXXXXXXXX
```

**Best Practice:**
- Rotate API keys every 90 days
- Use read-only keys where possible
- Monitor API key usage in logs
- Revoke compromised keys immediately

#### 2. Database Credentials

```bash
# PostgreSQL credentials
DATABASE_URL=postgresql://user:password@host:5432/database
DB_USER=swarm_user
DB_PASSWORD=strong_random_password_minimum_32_chars
```

**Best Practice:**
- Use strong, randomly generated passwords (minimum 32 characters)
- Rotate credentials every 180 days
- Use separate credentials for development/staging/production
- Enable PostgreSQL password encryption (scram-sha-256)

#### 3. Temporal Configuration

```bash
# Temporal database credentials (in docker-compose.yml)
POSTGRES_USER=temporal
POSTGRES_PWD=temporal_password

# Temporal server configuration
TEMPORAL_CORS_ORIGINS=*  # Restrict in production
```

**Best Practice:**
- Enable mTLS for Temporal client connections in production
- Use separate Temporal namespaces per environment
- Enable Temporal audit logging

### Secrets Storage (Production)

#### Option 1: SystemD EnvironmentFile

```ini
# /etc/systemd/system/open-swarm.service
[Service]
EnvironmentFile=/etc/open-swarm/open-swarm.env
```

```bash
# /etc/open-swarm/open-swarm.env
ANTHROPIC_API_KEY=sk-ant-XXXXX
DATABASE_URL=postgresql://...
```

**Permissions:**
```bash
sudo chmod 600 /etc/open-swarm/open-swarm.env
sudo chown openswarm:openswarm /etc/open-swarm/open-swarm.env
```

#### Option 2: HashiCorp Vault (Recommended for Enterprise)

```bash
# Initialize Vault
vault operator init
vault operator unseal

# Store secrets
vault kv put secret/open-swarm \
  anthropic_api_key="sk-ant-xxxxx" \
  db_password="secure-password"

# Retrieve in application
export ANTHROPIC_API_KEY=$(vault kv get -field=anthropic_api_key secret/open-swarm)
```

**Benefits:**
- Centralized secret management
- Automatic credential rotation
- Complete audit trail of secret access
- Fine-grained access control policies

#### Option 3: Kubernetes Secrets (K8s Deployments)

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: open-swarm-secrets
type: Opaque
data:
  anthropic_api_key: <base64-encoded-key>
  db_password: <base64-encoded-password>

---
apiVersion: v1
kind: Pod
metadata:
  name: open-swarm
spec:
  containers:
  - name: open-swarm
    env:
    - name: ANTHROPIC_API_KEY
      valueFrom:
        secretKeyRef:
          name: open-swarm-secrets
          key: anthropic_api_key
```

### Secrets Rotation Schedule

| Secret Type | Rotation Frequency | Alert Before |
|-------------|-------------------|--------------|
| API Keys | 90 days | 14 days |
| Database Passwords | 180 days | 30 days |
| TLS Certificates | 365 days / <30 days to expiry | 60 days |
| Temporal Credentials | 180 days | 30 days |

### Secrets Audit Logging

Log all secret access:

```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "event_type": "secret_access",
  "secret_name": "anthropic_api_key",
  "user_id": "user-123",
  "action": "read",
  "status": "success",
  "ip_address": "192.168.1.1"
}
```

---

## OpenCode Server Security

### Server Lifecycle Management

Each agent gets a dedicated OpenCode server instance with strict lifecycle controls:

**File:** `/internal/infra/server.go`

#### 1. Server Startup (INV-002, INV-003)

```go
// BootServer starts an opencode server with security controls
func (sm *ServerManager) BootServer(ctx context.Context, worktreePath string, worktreeID string, port int) (*ServerHandle, error) {
    // 1. Set working directory to worktree (INV-002)
    cmd.Dir = worktreePath

    // 2. Create new process group for clean shutdown
    cmd.SysProcAttr = &syscall.SysProcAttr{
        Setpgid: true,
    }

    // 3. Wait for healthcheck before returning (INV-003)
    for {
        resp, err := client.Get(baseURL + "/health")
        if resp.StatusCode == 200 {
            return handle, nil
        }
    }
}
```

**Security Controls:**
- Server runs in agent-specific worktree directory
- No access to parent repository
- Health check ensures server is ready before SDK connection

#### 2. Server Shutdown (INV-005)

```go
// Shutdown gracefully stops the opencode server
func (sm *ServerManager) Shutdown(handle *ServerHandle) error {
    // Kill entire process group (not just main process)
    pgid, err := syscall.Getpgid(cmd.Process.Pid)

    // 1. Try graceful SIGTERM
    syscall.Kill(-pgid, syscall.SIGTERM)

    // 2. Force SIGKILL if graceful fails after timeout
    if timeoutExceeded {
        syscall.Kill(-pgid, syscall.SIGKILL)
    }
}
```

**Security Controls:**
- Clean process group termination
- Graceful shutdown with timeout
- Force kill if graceful shutdown exceeds 5 seconds
- Prevents zombie processes and resource leaks

#### 3. SDK Client Configuration (INV-004, INV-006)

```go
// NewClient creates OpenCode SDK client with specific BaseURL
func NewClient(baseURL string, port int) *Client {
    sdk := opencode.NewClient(
        option.WithBaseURL(baseURL),
        // No API key needed for local connections
    )
    return &Client{
        sdk:     sdk,
        baseURL: baseURL,
        port:    port,
    }
}

// ExecuteCommand ensures command execution uses SDK
func (c *Client) ExecuteCommand(ctx context.Context, sessionID string, command string, args []string) (*PromptResult, error) {
    cmdParams := opencode.SessionCommandParams{
        Command:   opencode.F(command),
        Arguments: opencode.F(argsStr),
    }

    // Must use SDK client (no shell command execution)
    message, err := c.sdk.Session.Command(ctx, sessionID, cmdParams)
}
```

**Security Controls:**
- Port-specific connections prevent cross-agent access
- No API key needed (localhost-only)
- All commands routed through SDK (no direct shell access)

#### 4. Health Monitoring

Continuous health monitoring ensures server availability:

```go
// IsHealthy checks if the server is still responsive
func (sm *ServerManager) IsHealthy(handle *ServerHandle) bool {
    client := &http.Client{Timeout: 2 * time.Second}
    resp, err := client.Get(handle.BaseURL + "/health")
    return resp.StatusCode == 200
}
```

**Best Practice:**
- Monitor every 30 seconds
- Alert on 3 consecutive failures
- Auto-restart failed servers
- Log all health check failures

---

## Temporal Authentication

### Temporal Server Security

Temporal provides workflow orchestration with built-in security features:

**File:** `docker-compose.yml`

#### 1. Database Authentication

```yaml
temporal:
    environment:
      - POSTGRES_USER=temporal
      - POSTGRES_PWD=temporal_password
      - POSTGRES_SEEDS=postgresql
```

**Security Controls:**
- PostgreSQL authentication required
- Encrypted connection to database
- Separate database credentials per environment

#### 2. Namespace Isolation

Create separate Temporal namespaces for each environment:

```bash
# Development namespace
tctl --address temporal:7233 namespace create --namespace dev

# Staging namespace
tctl --address temporal:7233 namespace create --namespace staging

# Production namespace
tctl --address temporal:7233 namespace create --namespace prod
```

#### 3. mTLS Configuration (Production)

Enable mutual TLS for Temporal client connections:

```bash
# Generate certificates
cfssl gencert -initca ca.json | cfssljson -bare ca
cfssl gencert -config=ca-config.json -profile=server server.json | cfssljson -bare server
cfssl gencert -config=ca-config.json -profile=client client.json | cfssljson -bare client

# Configure Temporal with mTLS
temporal-server \
  --config temporal.yml \
  --service temporal:7233 \
  --tls-cert server.pem \
  --tls-key server-key.pem \
  --tls-client-ca ca.pem
```

#### 4. Worker Authentication

Temporal workers authenticate with server:

```go
// Connect to Temporal with mTLS
tlsConfig := &tls.Config{
    Certificates: []tls.Certificate{cert},
    ClientCAs:    caCert,
}

client, err := client.Dial(client.Options{
    HostPort: "localhost:7233",
    ConnectionOptions: grpc.WithTransportCredentials(
        credentials.NewTLS(tlsConfig),
    ),
})
```

#### 5. Audit Logging

Enable Temporal audit logging:

```yaml
# temporal.yml
logging:
  level: info
  format: json

audit:
  enabled: true
  fileSize: 100  # MB
  maxBackups: 10
  outputPath: /var/log/temporal/audit.log
```

**Logged Events:**
- Workflow execution started/completed
- Activity execution and results
- Signal handling
- Worker registration/deregistration
- API access attempts

### Temporal Best Practices

1. **Encrypt data at rest**: Enable PostgreSQL encryption
   ```bash
   # Enable in postgresql.conf
   ssl = on
   password_encryption = scram-sha-256
   ```

2. **Enable TLS in transit**: Use mTLS for all connections
   ```go
   // Enforce TLS 1.2+
   tlsConfig.MinVersion = tls.VersionTLS12
   ```

3. **Network isolation**: Only allow internal connections
   ```bash
   # Firewall rule
   sudo ufw allow from 10.0.0.0/8 to any port 7233
   ```

4. **Credentials rotation**: Rotate database credentials every 180 days

5. **Monitoring**: Alert on failed authentications
   ```yaml
   - alert: TemporalAuthFailure
     expr: rate(temporal_auth_failures[5m]) > 0
     for: 1m
   ```

---

## Worktree Isolation

### Git Worktree Architecture

Git worktrees provide filesystem-level isolation between agents:

**File:** `/internal/infra/worktree.go`

#### 1. Worktree Creation

Each agent gets a dedicated, isolated worktree:

```go
// CreateWorktree creates a new Git worktree for agent isolation
func (wm *WorktreeManager) CreateWorktree(id string, branch string) (*WorktreeInfo, error) {
    worktreePath := filepath.Join(wm.baseDir, id)

    // Prevent overwrite
    if _, err := os.Stat(worktreePath); err == nil {
        return nil, fmt.Errorf("worktree %s already exists", id)
    }

    // Create worktree
    cmd := exec.Command("git", "worktree", "add", worktreePath, branch)
}
```

**Security Controls:**
- Prevents worktree reuse between agents
- Isolated filesystem prevents cross-contamination
- Independent branch checkout per agent

#### 2. Worktree Path Structure

```
/tmp/open-swarm-worktrees/
├── agent-001-abc123/      # Agent A's worktree
│   ├── .git
│   ├── src/
│   └── ...
├── agent-002-def456/      # Agent B's worktree
│   ├── .git
│   ├── src/
│   └── ...
└── agent-003-ghi789/      # Agent C's worktree
    ├── .git
    ├── src/
    └── ...
```

#### 3. File Access Isolation

```
OpenCode Server A -> Worktree A (only)
OpenCode Server B -> Worktree B (only)
OpenCode Server C -> Worktree C (only)
```

**Prevented:**
- Agent A cannot read/write Agent B's files
- Filesystem permissions enforce isolation
- Git worktree locking prevents concurrent modifications

#### 4. Worktree Cleanup

```go
// RemoveWorktree removes a Git worktree
func (wm *WorktreeManager) RemoveWorktree(id string) error {
    cmd := exec.Command("git", "worktree", "remove", worktreePath, "--force")
}

// CleanupAll removes all worktrees
func (wm *WorktreeManager) CleanupAll() error {
    for _, wt := range worktrees {
        wm.RemoveWorktree(wt.ID)
    }
    return wm.PruneWorktrees()
}
```

**Best Practice:**
- Remove worktrees after agent completion
- Run `git worktree prune` periodically
- Monitor disk usage in worktree directory

#### 5. Permissions and Ownership

Set restrictive permissions on worktree directories:

```bash
# Worktree base directory
sudo chown openswarm:openswarm /tmp/open-swarm-worktrees
sudo chmod 700 /tmp/open-swarm-worktrees

# Individual worktree
sudo chmod 700 /tmp/open-swarm-worktrees/agent-001-abc123
```

### Worktree Isolation Guarantees

| Isolation Level | Guarantee | Implementation |
|-----------------|-----------|-----------------|
| Filesystem | Agent A cannot read/write Agent B files | Git worktrees + Unix permissions |
| Process | Agent A process cannot access Agent B data | Dedicated OpenCode server per agent |
| Network | Agent A cannot connect via Agent B's port | Port isolation + localhost binding |
| Git | Agent A cannot modify Agent B's branches | Independent worktree checkouts |

---

## Input Validation

### Configuration Validation

Validate all configuration before use:

**File:** `/internal/config/config.go`

```go
// Validate validates the configuration
func (c *Config) Validate() error {
    if c.Project.Name == "" {
        return fmt.Errorf("project name is required")
    }

    if c.Project.WorkingDirectory == "" {
        return fmt.Errorf("working directory is required")
    }

    if c.Coordination.Agent.Program == "" {
        return fmt.Errorf("agent program is required")
    }

    if c.Coordination.Agent.Model == "" {
        return fmt.Errorf("agent model is required")
    }

    return nil
}
```

**Best Practice:**
- Validate configuration at startup
- Use whitelist approach (only allow known values)
- Reject invalid configuration immediately

### Port Range Validation

```go
// PortManager validates port allocation
func (pm *PortManager) Allocate() (int, error) {
    pm.mu.Lock()
    defer pm.mu.Unlock()

    if port < pm.minPort || port > pm.maxPort {
        return 0, fmt.Errorf("port %d outside valid range %d-%d",
            port, pm.minPort, pm.maxPort)
    }
}
```

**Constraints:**
- Port range: 8000-9000 (1001 max agents)
- Prevent allocation outside range
- Prevent duplicate allocations

### API Input Validation

Validate all user inputs to OpenCode server:

```go
// ExecuteCommand validates command parameters
func (c *Client) ExecuteCommand(ctx context.Context, sessionID string, command string, args []string) (*PromptResult, error) {
    if sessionID == "" {
        return nil, fmt.Errorf("session ID is required")
    }

    if command == "" {
        return nil, fmt.Errorf("command is required")
    }

    // Validate command is in allowed list
    allowedCommands := map[string]bool{
        "reserve": true,
        "release": true,
        "sync": true,
    }

    if !allowedCommands[command] {
        return nil, fmt.Errorf("unknown command: %s", command)
    }

    // Sanitize arguments
    for _, arg := range args {
        if strings.Contains(arg, ";") || strings.Contains(arg, "|") {
            return nil, fmt.Errorf("invalid characters in arguments")
        }
    }
}
```

**Validation Rules:**
- Whitelist allowed commands
- Block shell metacharacters (`;`, `|`, `&`, `>`, `<`)
- Validate session IDs are UUIDs or known formats
- Limit argument length (max 4096 chars)

### Configuration File Validation

```go
// Load validates configuration file
func Load() (*Config, error) {
    // Check file exists
    if _, err := os.Stat(configPath); os.IsNotExist(err) {
        return nil, fmt.Errorf("configuration file not found")
    }

    // Check file permissions (must be 0600 or 0640)
    fi, err := os.Stat(configPath)
    if mode := fi.Mode(); mode&0077 != 0 {
        return nil, fmt.Errorf("insecure config file permissions: %o", mode)
    }

    // Parse with strict YAML
    var cfg Config
    if err := yaml.Unmarshal(data, &cfg); err != nil {
        return nil, fmt.Errorf("invalid config: %w", err)
    }

    // Validate all required fields
    return cfg, cfg.Validate()
}
```

**Best Practice:**
- Require secure file permissions (0600)
- Validate after parsing
- Reject unknown fields
- Type-check all values

### Agent Mail Input Validation

Validate all inputs to Agent Mail MCP server:

```go
// Validate message parameters
func (s *MessageService) Send(to []string, subject string, body string) error {
    // Validate recipients
    if len(to) == 0 {
        return fmt.Errorf("at least one recipient required")
    }

    for _, agent := range to {
        if !isValidAgentName(agent) {
            return fmt.Errorf("invalid recipient: %s", agent)
        }
    }

    // Validate subject
    if len(subject) == 0 || len(subject) > 256 {
        return fmt.Errorf("subject must be 1-256 characters")
    }

    // Validate body
    if len(body) == 0 || len(body) > 1_000_000 {
        return fmt.Errorf("body must be 1-1M characters")
    }

    return nil
}
```

**Message Validation Rules:**
- Recipient count: 1-100
- Subject length: 1-256 chars
- Body length: 1-1MB
- Valid agent names (alphanumeric + underscore)

### File Reservation Validation

Validate file reservation patterns:

```go
// ValidateReservationPattern validates glob pattern
func ValidateReservationPattern(pattern string) error {
    // Reject absolute paths
    if strings.HasPrefix(pattern, "/") {
        return fmt.Errorf("absolute paths not allowed")
    }

    // Reject parent directory traversal
    if strings.Contains(pattern, "..") {
        return fmt.Errorf("parent directory traversal not allowed")
    }

    // Reject patterns too broad
    if pattern == "**" || pattern == "*" {
        return fmt.Errorf("pattern too broad: %s", pattern)
    }

    // Validate glob syntax
    if _, err := filepath.Match(pattern, "test.txt"); err != nil {
        return fmt.Errorf("invalid glob pattern: %w", err)
    }

    return nil
}
```

**Pattern Validation Rules:**
- No absolute paths
- No parent directory traversal (`..`)
- No overly broad patterns (`*`, `**`)
- Valid glob syntax

### Temporal Activity Input Validation

```go
// ValidateActivityInput validates activity parameters
func ValidateActivityInput(ctx context.Context, taskID string, description string) error {
    // Task ID must be non-empty
    if taskID == "" {
        return fmt.Errorf("task ID is required")
    }

    // Task ID must match expected format
    if !regexp.MustCompile(`^open-swarm-[a-z0-9]+\.[0-9]+\.[0-9]+$`).MatchString(taskID) {
        return fmt.Errorf("invalid task ID format: %s", taskID)
    }

    // Description must be under 10KB
    if len(description) > 10_000 {
        return fmt.Errorf("description too long: %d > 10000", len(description))
    }

    return nil
}
```

---

## Security Best Practices

### 1. Principle of Least Privilege

#### Agent Permissions

Grant minimal required permissions:

```json
{
  "tools": {
    "write": true,
    "edit": true,
    "read": true,
    "bash": false,
    "agent-mail_*": true
  },
  "permission": {
    "bash": {
      "*": "ask",
      "git *": "allow",
      "go *": "allow"
    },
    "agent-mail_send_message": "ask"
  }
}
```

**Best Practices:**
- Use "ask" for sensitive operations (interactive approval)
- Use "allow" only for safe operations
- Never use "allow" for bash commands
- Require explicit confirmation for messaging

#### Service Permissions

Use systemd security hardening:

```ini
[Service]
# Restrict system access
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=yes
ReadWritePaths=/opt/open-swarm/logs /opt/open-swarm/data

# Resource limits
LimitNOFILE=65536
LimitNPROC=32768
MemoryMax=2G
CPUQuota=200%

# Process isolation
PrivateDevices=yes
ProtectClock=yes
RestrictSUIDSGID=yes
```

### 2. Defense in Depth

Implement multiple layers of security:

```
Layer 1: Network Isolation (port allocation, localhost binding)
  ↓
Layer 2: Filesystem Isolation (worktrees, permissions)
  ↓
Layer 3: Process Isolation (dedicated servers per agent)
  ↓
Layer 4: Input Validation (whitelist, sanitization)
  ↓
Layer 5: Authentication (environment secrets, mTLS)
  ↓
Layer 6: Audit Logging (all actions logged)
  ↓
Layer 7: Incident Response (monitoring, alerts)
```

### 3. Secure Defaults

All security-sensitive values have secure defaults:

| Setting | Default | Rationale |
|---------|---------|-----------|
| Port Range | 8000-9000 | Limited, non-privileged range |
| Binding | localhost | No network access by default |
| Process Group | Separate | Clean isolation per agent |
| Permissions | 0700 | Restrictive (owner only) |
| Health Check Timeout | 10 seconds | Fast failure detection |
| Message Auth | Ask | Explicit approval required |
| File Reservation | Exclusive | Prevent concurrent edits |

### 4. Audit Logging

Log all security-relevant events:

```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "event_type": "server_startup",
  "agent_id": "agent-001-abc123",
  "port": 8001,
  "status": "started",
  "duration_ms": 2500
}
```

**Logged Events:**
- Agent registration/deregistration
- Server startup/shutdown
- File reservations (request/release)
- Message sends (recipients, size)
- Authentication attempts (success/failure)
- Configuration changes
- Secrets access
- Permission denials

**Retention:** Keep logs for at least 90 days

### 5. Secret Rotation

Implement automated secret rotation:

```bash
#!/bin/bash
# rotate-secrets.sh

# Rotate API keys every 90 days
if [ $(($(date +%s) - $(stat -c %Y /etc/open-swarm/anthropic_api_key))) -gt 7776000 ]; then
    echo "Rotating ANTHROPIC_API_KEY..."
    # Generate new key
    NEW_KEY=$(generate_api_key)
    # Update secret storage
    vault kv put secret/open-swarm anthropic_api_key="$NEW_KEY"
    # Notify ops team
    send_alert "API key rotated"
fi
```

**Schedule:**
- API keys: Every 90 days
- Database passwords: Every 180 days
- TLS certificates: Every 365 days (auto-renewal at 30 days to expiry)

### 6. Monitoring and Alerting

Set up comprehensive monitoring:

```yaml
# Prometheus alert rules
- alert: HighErrorRate
  expr: rate(open_swarm_workflows_total{status="failed"}[5m]) > 0.05
  for: 5m
  labels:
    severity: warning

- alert: AuthenticationFailure
  expr: rate(temporal_auth_failures[5m]) > 0
  for: 1m
  labels:
    severity: critical

- alert: UnauthorizedPortAccess
  expr: rate(unauthorized_port_access[5m]) > 0
  for: 1m
  labels:
    severity: critical

- alert: SecretAccessAnomaly
  expr: rate(secret_access[5m]) > 10
  for: 5m
  labels:
    severity: warning
```

### 7. Incident Response Plan

#### Detection

Implement detection for:
- Failed authentication attempts
- Unauthorized file access
- Network anomalies
- Process anomalies
- Configuration changes
- Secret exposure

#### Response Steps

1. **Detect**: Monitoring alerts on security event
2. **Isolate**: Kill affected agent/process
3. **Investigate**: Review audit logs
4. **Contain**: Block user/agent if compromised
5. **Eradicate**: Rotate credentials
6. **Recover**: Restore from backup if needed
7. **Review**: Document incident and improve

#### Incident Log Template

```json
{
  "incident_id": "INC-2024-001",
  "timestamp": "2024-01-15T10:30:00Z",
  "severity": "high",
  "type": "authentication_failure",
  "description": "Multiple failed authentication attempts",
  "affected_agent": "agent-001",
  "status": "investigated",
  "resolution": "Agent API key rotated",
  "timeline": [
    {
      "time": "2024-01-15T10:30:00Z",
      "action": "Alert triggered"
    },
    {
      "time": "2024-01-15T10:31:00Z",
      "action": "Agent isolated"
    }
  ]
}
```

### 8. Secure Development Practices

#### Code Review

- Mandatory code review for security changes
- Use "reviewer" agent for security checks
- Enable gosec linting

```bash
# Run security linter
golangci-lint run --enable gosec

# Check vulnerable dependencies
govulncheck ./...
nancy sleuth  # Scan dependencies
```

#### Dependency Management

```bash
# Keep dependencies up to date
go get -u ./...

# Check for vulnerabilities
go list -json -m all | nancy sleuth

# Vendor dependencies
go mod vendor
```

#### Configuration Management

- Use ConfigMaps for non-sensitive config
- Use Secrets for sensitive config
- Validate configuration at startup
- Version control config (excluding secrets)

### 9. Production Hardening Checklist

Before deploying to production:

- [ ] Enable TLS for all external connections
- [ ] Configure firewall rules
- [ ] Set up secrets management (Vault)
- [ ] Enable audit logging
- [ ] Configure monitoring and alerting
- [ ] Test incident response procedure
- [ ] Set up log aggregation
- [ ] Configure database encryption
- [ ] Enable backup and disaster recovery
- [ ] Review and document security policies
- [ ] Train team on security procedures
- [ ] Conduct security audit
- [ ] Get security approval

---

## Temporal-Specific Security Recommendations

### 1. Workflow Execution Isolation

- Run workflows in separate namespaces by tenant
- Implement activity context validation
- Use encrypted data transfer between activities

### 2. Activity Input/Output Filtering

- Validate all activity inputs
- Encrypt sensitive data in activity results
- Log activity execution with sanitized arguments

### 3. Worker Authentication

- Authenticate workers with mTLS
- Use separate worker identity per agent
- Audit worker registration/deregistration

### 4. Temporal Web UI Security

```bash
# Restrict access to UI
sudo ufw allow from 10.0.0.0/8 to any port 8233

# Or disable in production if not needed
TEMPORAL_DISABLE_UI=true
```

---

## Compliance Considerations

### GDPR Compliance

- Implement data retention policies
- Provide user consent management
- Support data export and deletion
- Maintain audit trails for data access

### SOC 2 Type II

- Change management procedures
- Access control logging
- Backup and disaster recovery
- Security incident response
- Regular security assessments

### PCI DSS (if handling payment data)

- Encrypt data in transit (TLS 1.2+)
- Encrypt data at rest
- Implement access controls
- Maintain audit logs
- Regular security testing

---

## References

- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [CIS Benchmarks](https://www.cisecurity.org/benchmark)
- [NIST Cybersecurity Framework](https://www.nist.gov/cyberframework)
- [Go Security Best Practices](https://golang.org/doc/effective_go)
- [Temporal Security](https://docs.temporal.io/security)
- [PostgreSQL Security](https://www.postgresql.org/docs/current/sql-syntax.html)

---

## Support and Reporting

### Reporting Security Issues

Do not file public issues for security vulnerabilities. Instead:

1. Email security details to: `security@example.com`
2. Include reproduction steps
3. Avoid disclosing details on public channels
4. Allow 90 days for remediation before public disclosure

### Security Contacts

- **Security Lead**: [email]
- **Infrastructure Team**: [email]
- **On-Call Ops**: [email]

---

**Last Updated:** 2024-01-15
**Version:** 1.0
**Next Review:** 2024-07-15
