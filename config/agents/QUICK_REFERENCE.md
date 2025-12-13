# Agent Configuration Quick Reference

## File Structure

```
config/agents/
├── README.md                          # Full documentation
├── QUICK_REFERENCE.md                 # This file
├── agents.yaml                        # Master configuration & workflows
├── implementation.yaml                # Implementation agent
├── test-generator.yaml               # Test generation agent
├── reviewer-architecture.yaml        # Architecture reviewer
├── reviewer-functional.yaml          # Functional correctness reviewer
├── reviewer-testing.yaml             # Test quality reviewer
├── reviewer-security.yaml            # Security vulnerability reviewer
└── reviewer-performance.yaml         # Performance optimization reviewer
```

## Agent Summary

| Agent | Model | Purpose | Modifies | Temperature |
|-------|-------|---------|----------|-------------|
| **implementation** | Opus 4.5 | Write production code | Source files | 0.2 |
| **test-generator** | Sonnet 4.5 | Write tests | Test files | 0.3 |
| **reviewer-architecture** | Opus 4.5 | Review design/patterns | None | 0.2 |
| **reviewer-functional** | Sonnet 4.5 | Review logic/requirements | None | 0.2 |
| **reviewer-testing** | Sonnet 4.5 | Review test quality | None | 0.2 |
| **reviewer-security** | Opus 4.5 | Review security | None | 0.1 |
| **reviewer-performance** | Opus 4.5 | Review performance | None | 0.2 |

## Quick Start Workflows

### Option 1: Full Implementation & Review
```
1. implementation          → Writes code
2. test-generator         → Writes tests
3. All reviewers (parallel) → Review everything
```
**Use for**: Major features, critical code, public APIs

### Option 2: Fast Track (Urgent Fixes)
```
1. implementation          → Writes code
2. test-generator         → Writes minimal tests
3. reviewer-security      → Security check
4. reviewer-functional    → Logic check
```
**Use for**: Hotfixes, bug fixes, critical issues

### Option 3: Code Review Only
```
All reviewers read existing code
├→ Architecture
├→ Functional
├→ Testing
├→ Security
└→ Performance (optional)
```
**Use for**: PR review, code audit, refactor validation

## Agent Capabilities at a Glance

### Can Write/Edit
- **implementation**: Source code (*.go)
- **test-generator**: Test code (*_test.go)
- **Reviewers**: Nothing (read-only)

### Can Read
- **All agents**: All code files

### Can Run Tools
- **implementation**: bash, serena, grep, glob
- **test-generator**: bash, serena, grep, glob
- **Reviewers**: bash (for validation), serena, grep, glob

### Can Use Serena (Code Analysis)
- **All agents**: Full semantic analysis tools

## Key Constraints

| Agent | Max Steps | TTL | Max Tokens |
|-------|-----------|-----|-----------|
| implementation | 50 | 3600s | 8192 |
| test-generator | 50 | 1800s | 4096 |
| reviewer-* | 40 | N/A | 4096-6144 |

## When to Use Each Agent

### Implementation Agent
- Writing new features
- Fixing bugs
- Refactoring code
- Implementing requirements
**Output**: Production-ready Go code

### Test Generator Agent
- Writing unit tests
- Writing integration tests
- Improving coverage
- Creating test fixtures
**Output**: Test code with >80% coverage

### Architecture Reviewer
- Evaluating design patterns
- Assessing package structure
- Reviewing interface design
- Planning refactoring
**Output**: Architectural feedback & recommendations

### Functional Reviewer
- Validating requirements
- Testing business logic
- Checking edge cases
- Verifying integration
**Output**: Functional correctness assessment

### Testing Reviewer
- Assessing test quality
- Finding coverage gaps
- Improving test patterns
- Validating test strategy
**Output**: Test quality feedback

### Security Reviewer
- Finding vulnerabilities
- Validating crypto usage
- Checking input sanitization
- Reviewing authentication
**Output**: Security assessment (can block merge)

### Performance Reviewer
- Optimizing algorithms
- Reducing allocations
- Analyzing goroutines
- Improving cache efficiency
**Output**: Performance recommendations (advisory)

## Configuration Keys

### Model Selection
```yaml
model:
  primary: anthropic/claude-opus-4-5      # Best, slower
  fallback: anthropic/claude-sonnet-4-5   # Good, faster
```

### Temperature Guidelines
```
0.1  → Very deterministic (security, validation)
0.2  → Focused, analytical (review, architecture)
0.3  → Balanced (implementation, testing)
0.4  → Creative (documentation)
```

### Capabilities
```yaml
capabilities:
  tools:
    write: true/false
    edit: true/false
    read: true/false
    bash: true/false
    serena_*: true/false
```

### Constraints
```yaml
constraints:
  max_steps: 30-50
  file_reservation_ttl_seconds: 1800-3600
  can_modify_code: true/false
  read_only: true/false
```

## System Prompts Summary

### implementation.yaml
Focus: Clean, production-ready Go code
- Clean Architecture (Handler → Service → Repository)
- SOLID principles
- Idiomatic Go patterns
- Comprehensive error handling
- Performance considerations

### test-generator.yaml
Focus: Comprehensive test coverage
- Table-driven tests
- Edge case testing
- Mock interfaces
- 80%+ coverage target
- Deterministic tests

### reviewer-architecture.yaml
Focus: Design and patterns
- Design pattern analysis
- Package organization
- Dependency management
- Interface design
- Scalability assessment

### reviewer-functional.yaml
Focus: Business logic correctness
- Requirement compliance
- Edge case handling
- Error path validation
- State consistency
- Integration verification

### reviewer-testing.yaml
Focus: Test quality and coverage
- Coverage analysis
- Test patterns
- Mock usage
- Test independence
- Flakiness detection

### reviewer-security.yaml
Focus: Security vulnerabilities
- Input validation
- Authentication/authorization
- Cryptographic operations
- Injection prevention
- Data protection

### reviewer-performance.yaml
Focus: Performance optimization
- Complexity analysis
- Memory efficiency
- Goroutine management
- Lock contention
- Cache optimization

## Configuration Metadata

**Version**: 1.0.0
**Created**: 2025-12-13
**Framework**: Open Swarm
**Language**: Go
**Integration**: OpenCode, Agent Mail, Serena, Beads

## Editing Agent Configurations

### To Modify an Agent
1. Edit the corresponding YAML file
2. Update model, temperature, or constraints as needed
3. Modify system prompt for behavioral changes
4. Update capabilities for new tools
5. Test with a small task first

### To Add a New Agent
1. Create `reviewer-{type}.yaml`
2. Follow the template structure
3. Define specific role and responsibilities
4. Add to `agents.yaml` workflows
5. Document in README.md

### To Change a Workflow
1. Edit `agents.yaml` under `workflows:`
2. Reorder steps or modify dependencies
3. Update interaction sequences
4. Document in README

## Integration Examples

### In OpenCode Commands
```bash
# Run implementation workflow
opencode task-start my-feature

# Run full review
opencode review src/pkg/handler.go

# Run security review only
opencode review src/pkg/handler.go --reviewer security
```

### In Agent Mail Messages
```
To: @implementation
Subject: [bd-123] Implement auth handler
Body:
  Create authentication handler with JWT support.
  Should integrate with existing user service.
  Review required from security and architecture reviewers.

Thread: bd-123
```

### File Reservations
```bash
# Reserve files before modification
/reserve internal/api/**/*.go

# Release when complete
/release
```

## Common Patterns

### Pattern: Feature Implementation
```
1. Create Beads task
2. Reserve files: implementation, test-generator agents
3. Run implementation workflow
4. Gather feedback from reviewers
5. Address critical issues (security, functional)
6. Optional: address advisory feedback (performance)
7. Merge when approved
```

### Pattern: Hotfix
```
1. Create urgent Beads task
2. Run fast implementation workflow
3. Security review only for critical check
4. Merge immediately if approved
```

### Pattern: Code Audit
```
1. Select code to review
2. Run review-only workflow
3. Gather feedback from all reviewers
4. Create improvement issues
5. Prioritize by severity
```

## Troubleshooting

| Issue | Solution |
|-------|----------|
| Agent stops mid-task | Increase `max_steps` |
| Low quality output | Reduce temperature or increase tokens |
| Slow processing | Switch to faster model (Haiku) |
| Agent not understanding context | Review system prompt clarity |
| Too creative/unfocused | Lower temperature (min 0.1) |
| Too rigid/robotic | Raise temperature (max 0.4) |

## File Paths

- Configuration directory: `config/agents/`
- Master config: `config/agents/agents.yaml`
- Individual agents: `config/agents/{agent-name}.yaml`
- Documentation: `config/agents/README.md`
- This guide: `config/agents/QUICK_REFERENCE.md`

## Model Availability

- **Claude Opus 4.5**: Most capable, recommended for complex tasks
- **Claude Sonnet 4.5**: Good balance, recommended for most tasks
- **Claude Haiku 4.5**: Fastest, recommended for quick analysis

Check OpenCode configuration for actual available models.

## Resources

- Full documentation: `config/agents/README.md`
- Agent Mail docs: MCP server configuration
- Serena docs: Semantic code analysis
- OpenCode docs: Agent and command configuration
- Project guidelines: `AGENTS.md`

## Support

For questions or issues:
1. Check `README.md` detailed documentation
2. Review `AGENTS.md` project guidelines
3. Check `opencode.json` for agent definitions
4. Consult OpenCode documentation
