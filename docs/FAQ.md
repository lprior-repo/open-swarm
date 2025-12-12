# Open Swarm Frequently Asked Questions

**Last Updated:** 2025-12-12
**Version:** 1.0

## Table of Contents

1. [TCR vs DAG Workflows](#tcr-vs-dag-workflows)
2. [Scaling Limits](#scaling-limits)
3. [OpenCode vs Other Agents](#opencode-vs-other-agents)
4. [Temporal Advantages](#temporal-advantages)
5. [Cost Considerations](#cost-considerations)
6. [Failure Recovery](#failure-recovery)

---

## TCR vs DAG Workflows

### When should I use TCR (Test-Commit-Revert)?

**Use TCR when you need:**

- **Single, isolated task execution** - One feature, bug fix, or feature chunk
- **Atomic operations** - All-or-nothing: either the code works (tests pass) or it's reverted
- **Simple dependency management** - No complex task ordering needed
- **Fast, independent work** - Tasks that don't depend on other agents' work
- **Parallel task execution** - Multiple agents working on unrelated features simultaneously

**TCR is ideal for:**
- Implementing individual features with TDD
- Bug fixes that must maintain test coverage
- Code generation tasks
- Refactoring with regression testing
- Feature branches that merge to main

**Example use cases:**
```
Feature A: Add JWT authentication (Agent 1)
Feature B: Add caching layer (Agent 2)
Feature C: Add monitoring (Agent 3)
→ All run in parallel TCR cells without blocking each other
```

### When should I use DAG (Directed Acyclic Graph)?

**Use DAG when you need:**

- **Multiple interdependent tasks** - Tasks that must run in a specific order
- **Complex build pipelines** - Build → test → lint → deploy stages
- **Shared dependencies** - Multiple tasks depending on common prerequisite
- **CI/CD workflows** - Multi-stage deployment pipelines
- **Parallel execution with dependencies** - Run independent subtasks simultaneously within constraints
- **Human intervention recovery** - Pause, fix issues, and retry from the FixApplied signal

**DAG is ideal for:**
- Build pipelines (compile → test → deploy)
- Microservices orchestration
- Database migration chains
- Multi-stage testing (unit → integration → performance)
- Workflows requiring human review between stages

**Example use cases:**
```
Schema Design (step 1)
    ↓
Database Migration (step 2)
    ↓
Service Tests (step 3, with parallel: unit-test, integration-test, lint)
    ↓
Deployment (step 4)
```

### Quick Decision Tree

```
Does your task have dependencies on other tasks?
├─ NO → Use TCR
│  "I'm building a single feature, bug fix, or isolated change"
│
└─ YES → Do you need to run dependent tasks in sequence?
   ├─ YES → Use DAG
   │  "My workflow: compile → test → deploy"
   │  "Three services need to build sequentially"
   │
   └─ NO → Use TCR
      "My tasks are independent, can run in parallel"
```

### Can I mix TCR and DAG?

**Yes, in several patterns:**

1. **DAG orchestrating TCR workflows** - DAG coordinates multiple TCR cells
   ```
   DAG Stage 1: Launch TCR for Feature A
   DAG Stage 2: Wait for Feature A completion, then launch TCR for Feature B
   DAG Stage 3: Deploy both features
   ```

2. **TCR with manual DAG-like sequencing**
   ```
   Agent 1: /task-start feature-auth (TCR)
   Agent 1: /coordinate agent-2 "Auth done, you can start"
   Agent 2: Waits for message, then /task-start feature-api (TCR)
   ```

3. **DAG calling multiple OpenCode instances**
   ```
   DAG Task 1: Spin up OpenCode server on port 8000 (TCR cell A)
   DAG Task 2: Spin up OpenCode server on port 8001 (TCR cell B)
   DAG Task 3: Run integration tests when both complete
   ```

---

## Scaling Limits

### How many agents can work on the same codebase simultaneously?

**Practical limits:**

| Factor | Limit | Notes |
|--------|-------|-------|
| **Concurrent TCR cells** | 50 | Limited by available ports (8000-9000 = 1000 ports) |
| **Concurrent DAG workflows** | Unlimited | Limited by Temporal server capacity |
| **File reservations** | No hard limit | Managed by advisory locking |
| **OpenCode servers** | 50 concurrent | ~200MB RAM per server |
| **Agent Mail users** | 1000+ | Git-backed, scales linearly |

**Realistic numbers:**
- **Small team** (2-5 agents): No bottlenecks
- **Medium team** (6-20 agents): Monitor port availability and Temporal server resources
- **Large team** (20+ agents): Consider horizontally scaling Temporal workers

### What are the bottlenecks?

1. **Port availability (8000-9000)**
   - 1000 ports available for TCR cells
   - Each TCR cell needs one port
   - With 50 concurrent cells, you'll hit port limits
   - **Solution:** Configure different port ranges or use dynamic allocation

2. **Temporal server capacity**
   - Default: 1 Temporal server (localhost:7233)
   - Can handle hundreds of workflows
   - **Solution:** Run Temporal cluster for production (scales horizontally)

3. **OpenCode server resources**
   - ~200MB RAM per running server
   - 4GB system RAM → ~20 concurrent servers
   - **Solution:** Add system memory or stagger task execution

4. **Git repository size**
   - Large repos slow worktree creation
   - Each TCR cell creates a new worktree
   - **Solution:** Use shallow clones or split large monorepos

5. **LLM throughput**
   - Token generation rate limits tasks
   - If using external LLM (Claude API), rate limits apply
   - **Solution:** Increase request concurrency or run local LLMs

### Can I run more than 50 concurrent TCR cells?

**Technically:** Yes, but not recommended without changes

**Limitations:**
- Port range (8000-9000) only allows 1000 ports
- Each cell needs exclusive port
- 50 cells (current default) = half of available ports
- Going beyond 50 requires:
  1. Changing port range configuration
  2. Or waiting for cells to complete before launching new ones

**Workaround:**
```bash
# Configure different port ranges for different worker nodes
# Worker 1: ports 8000-9000
# Worker 2: ports 9000-10000
# Worker 3: ports 10000-11000
```

### How do I monitor scalability?

**Check resource usage:**
```bash
# Monitor active OpenCode processes
ps aux | grep "opencode serve" | wc -l

# Check port usage
netstat -tlnp | grep -E "800[0-9]|900[0-9]" | wc -l

# Monitor Temporal workflows
temporal workflow list

# Check system resources
top  # Look for memory, CPU usage
df -h  # Disk space
```

**Red flags:**
- More processes than expected (zombie servers)
- More ports in use than workflows running
- Memory usage > 80% of system RAM
- Temporal server unresponsive

---

## OpenCode vs Other Agents

### What makes OpenCode different?

| Feature | OpenCode | Claude Code | Other LLMs |
|---------|----------|-------------|-----------|
| **Provider agnostic** | Yes (any LLM) | Claude only | Varies |
| **MCP support** | Native | Via Claude API | Limited |
| **File operations** | Full filesystem | Full filesystem | Varies |
| **Terminal access** | Yes | Yes | Varies |
| **Session management** | Yes | Yes | Limited |
| **Cost** | Pay-per-request | Pay-per-request | Varies |
| **Integration** | SST ecosystem | Anthropic | Limited |

### Why OpenCode for Open Swarm?

**OpenCode is chosen because:**

1. **Provider flexibility** - Use any LLM (Claude, GPT, local models)
2. **MCP ecosystem integration** - Works with Agent Mail, Serena, custom tools
3. **Workspace isolation** - Each cell runs independently
4. **Git-aware** - Understands Git worktrees, commits, branches
5. **Performance** - Lightweight, fast startup (~2-3 seconds)
6. **Cost-effective** - Pay only for tokens used

### Can I use Claude Code instead of OpenCode?

**Technically:** Not directly, but you can:

1. **Use Claude Code as the LLM backend for OpenCode**
   - OpenCode can route to Claude's API
   - Configure in `opencode.json`
   - Same benefits as using Claude Code directly

2. **Manual Claude Code workflow**
   - Use Claude Code for code analysis
   - Manually coordinate with Agent Mail
   - Less automated, more manual work

3. **Hybrid approach**
   - OpenCode for TCR cell execution
   - Claude Code for code review and planning
   - Agent Mail for coordination

**Recommendation:** Use OpenCode (with Claude as backend) rather than replacing it entirely.

### Should I use local LLMs with OpenCode?

**Advantages:**
- No API costs
- Private code (stays local)
- Faster iteration (no network latency)
- No rate limits

**Disadvantages:**
- Requires powerful hardware (8GB+ RAM, fast CPU)
- Lower code quality (local models are weaker)
- Harder to debug (fewer monitoring options)
- Maintenance burden

**Recommended setup:**
- **Development:** Local LLM (Llama 2, Mistral)
- **Production:** Claude API (via OpenCode)
- **Hybrid:** Local for drafts, Claude for reviews

---

## Temporal Advantages

### Why use Temporal instead of simpler orchestration?

**Temporal provides:**

| Feature | Temporal | Simple Queue | Kubernetes | Shell Scripts |
|---------|----------|--------------|-----------|---------------|
| **Fault tolerance** | Yes (durable execution) | Manual retry | Limited | No |
| **State persistence** | Yes (automatic) | Manual | Complex | No |
| **Retries** | Yes (exponential backoff) | Manual | Limited | No |
| **Timeouts** | Yes (configurable) | Manual | Limited | Manual |
| **Monitoring/History** | Web UI + detailed logs | Logs only | Metrics | None |
| **Parallel execution** | Yes (coordinated) | With extra code | Yes | Shell coordination |
| **Human signals** | Yes (pause/resume) | Manual webhooks | Limited | None |
| **Cost** | Self-hosted or cloud | Minimal | High infra | Minimal |

### What problems does Temporal solve?

1. **Process crashes** - If OpenCode server dies, Temporal retries automatically
2. **Network failures** - Temporal handles transient failures gracefully
3. **Long-running tasks** - Can continue for hours/days with heartbeats
4. **Human intervention** - DAG workflows can wait for "FixApplied" signal
5. **Observability** - Complete execution history visible in Web UI
6. **Scalability** - Distributes work across multiple workers

### Example: Why Temporal matters

**Without Temporal:**
```bash
# Shell script (fragile)
opencode serve --port 8000 &
PID=$!
wait $PID
# If opencode crashes before completing, task is lost
# How do you retry? How do you know it failed?
# What if network drops mid-execution?
```

**With Temporal:**
```go
// Temporal workflow (robust)
err := workflow.ExecuteActivity(ctx, BootstrapCell, input)
if err != nil {
  // Temporal automatically retries 3 times
  // Maintains heartbeat every 30 seconds
  // Records complete execution history
  // Visible in Web UI for debugging
  return err
}
```

### Can I use Open Swarm without Temporal?

**Possible alternatives:**

1. **Shell scripts** - Simple but fragile
2. **Kubernetes CronJobs** - Good for scheduled work, poor for coordination
3. **Airflow/Prefect** - Complex, overkill for small teams
4. **Custom queue system** - Reinvent the wheel

**Verdict:** Temporal is the right tool for the job. It handles the hard problems that other tools don't.

---

## Cost Considerations

### What are the costs of running Open Swarm?

**Breakdown by component:**

| Component | Cost | Notes |
|-----------|------|-------|
| **Temporal Server** | $0 (self-hosted) | Docker Compose setup included |
| **PostgreSQL** | $0 (self-hosted) | Included in Docker Compose |
| **Agent Mail** | $0 (self-hosted) | Python MCP server, no external deps |
| **OpenCode** | $0 (tool) | Free to use |
| **Beads** | $0 (self-hosted) | Git-backed, no external services |
| **Serena** | $0 (tool) | LSP server, runs locally |
| **LLM API** | $$ (usage-based) | Claude, GPT, etc. - pay per token |
| **Infrastructure** | $ (hardware) | Server/VM to run everything on |

### How much does it cost to run a single TCR workflow?

**Example with Claude:**

```
Task: Implement a small feature
- Prompt: 500 tokens → $0.0015 (input)
- Generation: 2000 tokens → $0.006 (output)
- Per-execution cost: ~$0.0075

Daily (100 workflows): ~$0.75/day
Monthly: ~$22.50/month
```

**Cost factors:**
1. **Prompt size** - More context = more input tokens
2. **Code complexity** - Complex features need more output tokens
3. **LLM provider** - Claude 3.5 Sonnet vs GPT-4 vs local model
4. **Testing overhead** - Running tests consumes tokens (hidden cost)

### How to optimize LLM costs?

1. **Use cheaper models for drafts**
   ```json
   {
     "models": {
       "draft": "claude-3-haiku",        // Cheaper, faster
       "review": "claude-3-5-sonnet"     // More capable, for reviews
     }
   }
   ```

2. **Batch tasks**
   - Run multiple TCR cells simultaneously
   - Better resource utilization
   - Better per-token economics at scale

3. **Cache prompts**
   - Reuse prompts for similar tasks
   - Agent Mail provides message threading
   - Reduce duplicate token usage

4. **Use local LLMs for drafts**
   - Llama 2 (Meta) - Free
   - Mistral (Mistral AI) - Free
   - Run locally, zero API costs

5. **Optimize prompts**
   - Clear, specific prompts = fewer retries
   - Poor prompts = failed workflows = wasted tokens
   - Good prompt engineering saves money

### Example cost scenarios

**Scenario 1: Small team (5 agents, 50 tasks/month)**
```
Tokens per task: 2,500 (500 input + 2000 output)
Total tokens: 125,000
Cost (Claude): ~$0.38/day = $11.50/month
Infrastructure: $10/month (small VM)
Total: ~$22/month
```

**Scenario 2: Medium team (20 agents, 200 tasks/month)**
```
Tokens per task: 3,000 (more complex work)
Total tokens: 600,000
Cost (Claude): ~$1.80/day = $54/month
Infrastructure: $50/month (medium VM)
Total: ~$104/month
```

**Scenario 3: Large team (50 agents, 500 tasks/month)**
```
Tokens per task: 3,500 (complex coordination)
Total tokens: 1,750,000
Cost (Claude): ~$5.25/day = $158/month
Infrastructure: $200/month (large VM/cloud)
Total: ~$358/month (plus Temporal cloud if used)
```

### Should I use cloud Temporal or self-hosted?

**Self-hosted (Docker Compose):**
- **Cost:** ~$10-100/month (VM costs)
- **Setup:** 5 minutes
- **Operations:** Minimal (Docker handles it)
- **Best for:** Small to medium teams

**Temporal Cloud:**
- **Cost:** $0.50/workflow execution (approx)
- **Setup:** minutes (managed service)
- **Operations:** None (Temporal handles it)
- **Best for:** Large teams, high availability requirements

**Break-even analysis:**
```
Temporal Cloud: $0.50 per workflow
Daily (100 workflows): $50/day
Monthly: ~$1,500/month

Self-hosted: $50/month
Can run 3,000 workflows/month cost-free

Cloud wins when: >300 workflows/month (at $0.50/workflow)
Self-hosted wins when: <300 workflows/month
```

---

## Failure Recovery

### How does Open Swarm recover from failures?

**Recovery depends on failure type:**

| Failure Type | Detection | Recovery | Time |
|--------------|-----------|----------|------|
| **OpenCode crash** | Temporal heartbeat timeout (30s) | Automatic retry (3 attempts) | 1-2 minutes |
| **Network timeout** | Activity timeout (10 min) | Automatic retry based on policy | 1 minute |
| **Test failure** | Test runner returns error | TCR: revert; DAG: human signal | Immediate |
| **Git conflict** | Worktree error | Manual resolution required | Hours |
| **Temporal server down** | Connection refused | Hold in queue, retry when up | Variable |
| **Agent Mail disconnected** | Message send fails | Retry on reconnection | 1-2 minutes |
| **Beads sync fails** | Git push error | Retry on next sync | Hours |

### TCR Workflow Recovery

**On test failure:**
```
1. Tests run → FAIL
2. Changes automatically reverted (git reset --hard)
3. Cell cleaned up
4. Developer investigates
5. Developer adjusts prompt or code
6. Re-runs workflow
```

**Key point:** TCR is "atomic" - either all changes are committed or none are.

**Recovery steps:**
```bash
# 1. Check what failed
temporal workflow show --workflow-id reactor-TASK-001

# 2. Improve your prompt with lessons learned
# (Add more specific instructions)

# 3. Resubmit
reactor-client --workflow tcr --task TASK-001 --prompt "..."

# 4. Monitor retry
temporal workflow show --workflow-id reactor-TASK-001 --follow
```

### DAG Workflow Recovery

**On task failure:**
```
1. Task fails
2. DAG pauses and waits for "FixApplied" signal
3. Developer fixes the issue
4. Developer sends signal: client.SignalWorkflow(..., "FixApplied", "...")
5. DAG retries from the beginning
```

**Recovery steps:**
```bash
# 1. Check which task failed
temporal workflow show --workflow-id build-pipeline-001

# 2. Developer fixes the issue
# (e.g., fixes compilation error, updates dependencies)

# 3. Send signal to retry
temporal workflow signal \
  --workflow-id build-pipeline-001 \
  --name FixApplied \
  --input '{"message": "Fixed compilation error"}'

# 4. DAG retries (starts from beginning)
temporal workflow show --workflow-id build-pipeline-001 --follow
```

### What if a cell crashes mid-execution?

**Sequence of events:**

```
1. OpenCode server crashes (or network dies)
2. Temporal heartbeat timeout (30 seconds with no update)
3. Automatic retry:
   - Activity retries up to 3 times (if policy configured)
   - 1s → 2s → 4s → ... exponential backoff
4. After 3 retries, activity fails
5. Workflow handles failure (commit, revert, signal, etc.)
```

**From user perspective:**
```bash
# Check status
temporal workflow describe --workflow-id reactor-TASK-001

# Output might show:
# - "Retrying activity BootstrapCell (attempt 2 of 3)"
# - "Activity failed after 3 retries"
# - "Workflow waiting for manual intervention"

# If retryable, Temporal handles it automatically
# If not, workflow enters failure state
```

### Can I recover a failed TCR workflow?

**No direct recovery (by design):**

TCR workflows are designed to be atomic:
- Either all changes are committed (tests passed)
- Or all changes are reverted (tests failed)

There's no middle ground.

**If TCR fails, you must:**
1. Investigate why tests failed
2. Improve the prompt or code
3. Submit a new TCR workflow

**Example:**
```bash
# First attempt failed
reactor-client --workflow tcr --task TASK-001 \
  --prompt "Implement pagination"

# Failed: off-by-one error in calculation

# Second attempt with better instructions
reactor-client --workflow tcr --task TASK-001 \
  --prompt "Implement pagination.
  - Page numbers start at 1 (not 0)
  - pageSize default: 20, max: 100
  - Include 3 test cases:
    * page=1 (first page)
    * page=2 (second page)
    * page=0 (should fail or return page 1)"
```

### Can I recover a failed DAG workflow?

**Yes, via signal:**

DAG workflows wait for "FixApplied" signal when any task fails.

**Recovery flow:**
```
DAG runs → Task X fails → DAG pauses
    ↓ (Developer fixes issue)
Signal "FixApplied" sent
    ↓
DAG retries (from beginning)
    ↓
All tasks pass → Success
```

**Why retry from beginning?**

Because dependencies might have changed. Example:
```
Task 1: Compile (fails)
Task 2: Test (skipped, depends on Task 1)

Developer fixes compilation error → must recompile
Then Task 2 can run with new binary
```

### What if Temporal server crashes?

**Graceful degradation:**

1. **Workflows running in memory**: In-progress work may be lost
2. **Workflow history**: Persisted to PostgreSQL (always safe)
3. **Upon restart**: Workflow resumes from last checkpoint

**From user perspective:**
```bash
# Temporal server crashes
# Your running workflows pause

# Someone restarts Temporal
docker-compose up -d temporal

# Workflows resume automatically
# Where did they leave off? Temporal knows!
temporal workflow describe --workflow-id reactor-TASK-001

# Shows:
# - Current status
# - What's been completed
# - What will run next
# - Complete history of everything that happened
```

**Best practices:**
1. Run Temporal in production-grade setup (not just Docker)
2. Use Temporal Cloud for high availability
3. Back up PostgreSQL regularly
4. Monitor Temporal server health

### How do I debug a failed workflow?

**Comprehensive debugging workflow:**

```bash
# 1. Get basic status
temporal workflow describe --workflow-id reactor-TASK-001

# 2. View complete execution history
temporal workflow show --workflow-id reactor-TASK-001

# 3. Examine specific activity
temporal activity describe --workflow-id reactor-TASK-001 \
  --activity-id ExecuteTask

# 4. View activity logs
temporal activity logs --workflow-id reactor-TASK-001 \
  --activity-id ExecuteTask

# 5. Check Temporal Web UI
# Navigate to: http://localhost:8233
# Search for your workflow
# Click through activities for detailed info

# 6. View application logs
tail -f ~/.temporal/worker.log
tail -f ~/.opencode/logs

# 7. Check OpenCode output
# Open Temporal Web UI → Workflow Details
# Find ExecuteTask activity
# Check stdout/stderr captured from agent
```

### Failure scenarios and recovery

**Scenario 1: "Could not connect to OpenCode server"**
```
Cause: Port conflict or OpenCode slow to start
Recovery:
1. Check: lsof -i :8000-9000
2. Kill zombie processes: pkill -f "opencode serve"
3. Retry workflow
```

**Scenario 2: "Test suite failed"**
```
TCR: Workflow automatically reverts, developer retries
DAG: Developer fixes issue, sends FixApplied signal

Check logs:
temporal workflow show --workflow-id <id> | grep output
```

**Scenario 3: "Git worktree conflicts"**
```
Cause: Concurrent modification or cleanup failure
Recovery:
1. Check: git worktree list
2. Clean up: git worktree remove ./worktrees/* --force
3. Verify: git worktree list (should be empty)
4. Retry workflow
```

**Scenario 4: "Temporal server disconnected"**
```
Cause: Network or Temporal service down
Recovery:
1. Check: temporal server health
2. If down: docker-compose up -d
3. Monitor: temporal workflow list (verify server responds)
4. Workflow resumes automatically from checkpoint
```

**Scenario 5: "Port allocation failed - all ports in use"**
```
Cause: Too many concurrent TCR cells
Recovery:
1. Check: netstat -tlnp | grep -E "800[0-9]|900[0-9]" | wc -l
2. Kill: lsof -i :8000-9000 | awk 'NR>1 {print $2}' | xargs kill -9
3. Or wait for some cells to complete
4. Retry workflow
```

### Prevention: Best practices

1. **Regular monitoring**
   ```bash
   # Daily checks
   temporal workflow list
   ps aux | grep opencode
   docker stats
   ```

2. **Resource limits**
   - Monitor disk space (Git storage)
   - Monitor memory (OpenCode servers)
   - Monitor port usage (1000 port limit)

3. **Proper cleanup**
   ```bash
   # After session ends
   /session-end
   bd sync
   git worktree prune
   ```

4. **Retry policies**
   - Set reasonable timeouts (not too aggressive)
   - Max retries: 3 (usually good)
   - Backoff: exponential (prevents server overload)

5. **Testing**
   - Test prompts locally before scaling
   - Run first attempt with monitoring on
   - Adjust based on real performance

---

## More Resources

- [TCR-WORKFLOW.md](./TCR-WORKFLOW.md) - Detailed TCR pattern documentation
- [DAG-WORKFLOW.md](./DAG-WORKFLOW.md) - Detailed DAG pattern documentation
- [TROUBLESHOOTING.md](./TROUBLESHOOTING.md) - Specific error solutions
- [MONITORING.md](./MONITORING.md) - Monitoring and observability guide
- [DEPLOYMENT.md](./DEPLOYMENT.md) - Production deployment guide
- [README.md](../README.md) - Project overview

## Support

For issues or questions:

1. **Check existing docs** - Start with TROUBLESHOOTING.md
2. **Review Temporal logs** - `temporal workflow show --workflow-id <id>`
3. **Check system health** - `docker ps`, `ps aux`, `lsof -i`
4. **File issue in Beads** - `bd create "Issue description"`
5. **Review AGENTS.md** - Multi-agent coordination patterns
