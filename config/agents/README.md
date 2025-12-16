# Agent Configuration Guide

This directory contains YAML configuration files for different agent roles in the Open Swarm multi-agent coordination framework.

## Overview

Each agent has a specialized role and configuration defining:
- **Model & Performance**: AI model selection, temperature, token limits
- **System Prompts**: Specialized instructions for the agent's domain
- **Capabilities**: Tools and features the agent can use
- **Constraints**: Limitations and scope boundaries
- **Workflow Integration**: How agents interact in coordinated processes

## Agent Roles

### `implementation.yaml` - Implementation Agent

**Role**: Production code implementation specialist

**Specialization**: Writing production-ready Go code

**Model**: Claude Opus 4.5 (primary), Claude Sonnet 4.5 (fallback)

**Key Features**:
- Clean Architecture pattern (Handler → Service → Repository)
- Dependency injection and interface-driven design
- Comprehensive error handling
- Concurrency-safe implementations
- Performance-aware coding

**Capabilities**:
- Write and edit source files
- Use all Serena semantic tools for code navigation and editing
- Run bash for testing and validation
- Full code modification authority

**Constraints**:
- Cannot modify tests
- Requires test validation before completion
- 3600-second file reservation TTL

**Use When**:
- Implementing new features
- Writing business logic
- Refactoring existing code
- Building new services or packages

---

### `test-generator.yaml` - Test Generator Agent

**Role**: Comprehensive test writing specialist

**Specialization**: Writing Go tests with high coverage

**Model**: Claude Sonnet 4.5 (primary), Claude Haiku 4.5 (fallback)

**Key Features**:
- Table-driven test patterns
- Mock interface design
- Edge case and error path testing
- Benchmark test generation
- Integration test design

**Capabilities**:
- Write and edit test files
- Use Serena for code understanding
- Run bash for test execution and validation
- Full test modification authority

**Constraints**:
- Cannot modify production code
- Aim for 80%+ code coverage
- 1800-second file reservation TTL
- Temperature 0.3 (creative but focused)

**Use When**:
- Writing unit tests for new code
- Improving test coverage
- Creating integration tests
- Designing test fixtures and mocks

---

### `reviewer-architecture.yaml` - Architecture Reviewer

**Role**: Architecture and design pattern specialist

**Specialization**: Evaluating architectural soundness and design patterns

**Model**: Claude Opus 4.5 (primary), Claude Sonnet 4.5 (fallback)

**Key Focus Areas**:
- Clean Architecture compliance
- Design pattern correctness
- Separation of concerns
- Dependency management
- Package organization
- Interface design
- Scalability and extensibility

**Capabilities**:
- Read-only code analysis
- Use Serena for deep code understanding
- Grep and glob for pattern matching
- Cannot modify code

**Constraints**:
- Read-only mode (no modifications)
- Maximum 40 steps
- Lower temperature (0.2) for consistency

**Feedback Includes**:
- Design pattern analysis
- Architectural recommendations
- Technical debt identification
- Refactoring suggestions

**Use When**:
- Reviewing major feature implementation
- Assessing package structure changes
- Evaluating interface designs
- Planning large refactoring efforts

---

### `reviewer-functional.yaml` - Functional Reviewer

**Role**: Business logic and requirement compliance specialist

**Specialization**: Validating functional correctness

**Model**: Claude Sonnet 4.5 (primary), Claude Haiku 4.5 (fallback)

**Key Focus Areas**:
- Business logic correctness
- Requirement compliance
- Edge case handling
- Error handling appropriateness
- State consistency
- Integration correctness
- User behavior validation

**Capabilities**:
- Read-only code analysis
- Can run bash for validation
- Use Serena for code understanding
- Cannot modify code

**Constraints**:
- Read-only mode
- Lower temperature (0.2) for accuracy
- Maximum 40 steps

**Feedback Includes**:
- Functional correctness assessment
- Edge case scenarios
- Missing functionality identification
- Integration point validation
- Test case suggestions

**Use When**:
- Validating feature implementation against requirements
- Testing user-facing behavior
- Checking state management
- Verifying integration with other components

---

### `reviewer-testing.yaml` - Testing Reviewer

**Role**: Test quality and coverage specialist

**Specialization**: Ensuring comprehensive and maintainable tests

**Model**: Claude Sonnet 4.5 (primary), Claude Haiku 4.5 (fallback)

**Key Focus Areas**:
- Test coverage analysis (target: 80%+)
- Edge case and boundary testing
- Error path coverage
- Mock and stub effectiveness
- Test independence and determinism
- Table-driven test patterns
- Test naming and documentation

**Capabilities**:
- Read-only code analysis
- Can run bash for coverage analysis
- Grep for test pattern matching
- Use Serena for understanding test structure
- Cannot modify code

**Constraints**:
- Read-only mode
- Coverage targets: 80% unit, 60% integration, 95% critical paths
- Maximum 40 steps

**Feedback Includes**:
- Coverage gap identification
- Additional test suggestions
- Flakiness detection
- Pattern recommendations
- Metrics and measurements

**Use When**:
- Reviewing new test code
- Assessing test coverage
- Identifying untested code paths
- Improving test maintainability

---

### `reviewer-security.yaml` - Security Reviewer

**Role**: Security vulnerability and best practice specialist

**Specialization**: Identifying security risks and vulnerabilities

**Model**: Claude Opus 4.5 (primary), Claude Sonnet 4.5 (fallback)

**Key Focus Areas**:
- Input validation and sanitization
- Authentication and authorization
- Cryptographic correctness
- Injection attack prevention
- Sensitive data protection
- Secure configuration
- Dependency vulnerabilities
- Information leakage

**Capabilities**:
- Read-only code analysis
- Can run bash for dependency checks
- Comprehensive code searching
- Use Serena for detailed analysis
- Cannot modify code

**Constraints**:
- Read-only mode
- Lowest temperature (0.1) for accuracy
- Maximum 50 steps
- Can block code merge

**Severity Levels**:
- **CRITICAL**: Immediate exploitation possible
- **HIGH**: Exploitable under specific conditions
- **MEDIUM**: Potential vulnerability
- **LOW**: Best practice violation

**Use When**:
- Reviewing any code changes (required for all changes)
- Handling sensitive data or credentials
- Implementing authentication/authorization
- Using cryptographic operations
- Processing untrusted input

---

### `reviewer-performance.yaml` - Performance Reviewer

**Role**: Performance and efficiency specialist

**Specialization**: Identifying performance optimization opportunities

**Model**: Claude Opus 4.5 (primary), Claude Sonnet 4.5 (fallback)

**Key Focus Areas**:
- Algorithm complexity analysis
- Memory allocation efficiency
- Goroutine management
- Synchronization overhead
- Caching opportunities
- I/O optimization
- Hot path identification
- Scalability assessment

**Capabilities**:
- Read-only code analysis
- Can run bash for benchmarking
- Grep for performance pattern matching
- Use Serena for code structure understanding
- Cannot modify code

**Constraints**:
- Read-only mode
- Temperature 0.2 for analytical accuracy
- Maximum 40 steps
- Advisory feedback (doesn't block merge)

**Impact Classification**:
- **CRITICAL**: >50% degradation
- **HIGH**: 10-50% degradation
- **MEDIUM**: 5-10% degradation
- **LOW**: <5% degradation

**Feedback Includes**:
- Complexity analysis
- Benchmark suggestions
- Memory optimization opportunities
- Concurrency improvements
- Performance impact estimates

**Use When**:
- Reviewing critical path implementations
- Optimizing performance-sensitive code
- Analyzing scalability concerns
- Planning performance improvements

---

## Agent Interaction Workflows

### Full Implementation and Review Workflow

Complete workflow for implementing a feature with comprehensive review:

```
1. Implementation Agent
   ↓ (writes production code)

2. Test Generator
   ↓ (writes tests covering code)

3-7. Review Agents (parallel)
   ├→ Architecture Reviewer
   ├→ Functional Reviewer
   ├→ Testing Reviewer
   ├→ Security Reviewer
   └→ Performance Reviewer (optional)

   (Provide feedback to implementation team)
```

### Fast Implementation Workflow

Expedited workflow for urgent fixes:

```
1. Implementation Agent
   ↓
2. Test Generator (minimal coverage)
   ↓
3. Security Reviewer (critical issues only)
   ↓
4. Functional Reviewer
```

### Review-Only Workflow

Review existing code without modifications:

```
All review agents analyze existing code:
├→ Architecture Reviewer
├→ Functional Reviewer
├→ Testing Reviewer
├→ Security Reviewer
└→ Performance Reviewer (optional)
```

## Configuration Structure

Each agent configuration file follows this structure:

```yaml
agent:
  name: unique-identifier
  description: human-readable description
  role: role-type

model:
  primary: model-for-primary-tasks
  fallback: model-for-fallback

parameters:
  temperature: 0.0-1.0  # Lower = more deterministic, Higher = more creative
  max_tokens: 1024-8192  # Max output length
  top_p: 0.9            # Nucleus sampling
  top_k: 40             # Top-k sampling

system_prompt: |
  Detailed instructions for agent behavior and goals

capabilities:
  tools:
    # Which tools the agent can use
  features:
    # What specialized features are enabled

constraints:
  max_steps: number
  file_reservation_ttl_seconds: seconds
  # Role-specific constraints

# Role-specific sections (varies by agent type)
```

## Model Selection Guide

| Model | Best For | Reasoning |
|-------|----------|-----------|
| Claude Opus 4.5 | Complex implementation, security, performance | Most capable, best for nuanced analysis |
| Claude Sonnet 4.5 | General implementation, review, testing | Good balance of capability and speed |
| Claude Haiku 4.5 | Quick analysis, fast exploration | Fast, sufficient for focused tasks |

## Temperature Tuning

| Temperature | Behavior | Use For |
|-------------|----------|---------|
| 0.1 | Very deterministic, consistent | Security review, validation |
| 0.2 | Focused analysis | Functional review, architecture |
| 0.3 | Balanced | Test generation, implementation |
| 0.4 | Creative | Documentation |

## Tool Capabilities Matrix

| Tool | Implementation | Test-Gen | Review | Notes |
|------|---------------|----------|--------|-------|
| write | ✅ | ✅ | ❌ | Create new files |
| edit | ✅ | ✅ | ❌ | Modify existing files |
| read | ✅ | ✅ | ✅ | Read file contents |
| bash | ✅ | ✅ | ✅ | Execute commands |
| serena_* | ✅ | ✅ | ✅ | Code navigation |
| grep | ✅ | ✅ | ✅ | Pattern search |
| glob | ✅ | ✅ | ✅ | File pattern matching |

## Quality Gates

### For Implementation
- Test coverage ≥ 80%
- Zero lint errors
- Architecture approved
- Security review passed

### For Testing
- Coverage ≥ 80%
- Edge cases covered
- Tests are deterministic
- No flaky tests

### For Code Review
- Security review passed (required)
- Functional review passed (required)
- Architecture review passed (required)
- Performance reviewed (optional)

## File Reservation TTLs

| Agent | TTL | Reasoning |
|-------|-----|-----------|
| implementation | 3600s (1 hour) | Longer work sessions |
| test-generator | 1800s (30 min) | Quick test writing |
| reviewers | N/A | Read-only, no reservations |

## Best Practices

### 1. Sequential Workflow
Always follow workflows in order. Don't start reviews before implementation is complete.

### 2. Clear Communication
Each agent should document what it did and pass results to the next agent.

### 3. Context Preservation
Use Agent Mail threads to maintain context across agent interactions.

### 4. Feedback Prioritization
- **Blocks merge**: Security, Functional, Architecture issues
- **Advisory**: Performance improvements

### 5. Temperature Control
Use lower temperatures for analysis, higher for creative tasks.

### 6. Tool Discipline
Agents should only use tools they're configured for.

## Extending Configurations

To add a new agent:

1. Create `reviewer-{type}.yaml` (if a reviewer)
2. Define role, model, parameters
3. Write focused system prompt
4. List specific capabilities
5. Set appropriate constraints
6. Update `agents.yaml` with workflow integration

Example template:

```yaml
agent:
  name: reviewer-custom
  description: Custom reviewer for specific domain
  role: reviewer
  review_type: custom

model:
  primary: anthropic/claude-opus-4-5
  fallback: anthropic/claude-sonnet-4-5

parameters:
  temperature: 0.2
  max_tokens: 4096

system_prompt: |
  Specific instructions for your domain...

capabilities:
  tools:
    read: true
    bash: true
    grep: true
    glob: true
    serena_find_symbol: true
    serena_find_referencing_symbols: true

constraints:
  max_steps: 40
  can_modify_code: false
  read_only: true

focus_areas:
  - area1
  - area2
```

## Integration with OpenCode

These configurations are designed to work with OpenCode's agent system. Reference in `opencode.json`:

```json
{
  "agent": {
    "your-agent-name": {
      "description": "...",
      "model": "anthropic/model",
      "temperature": 0.2,
      "max_steps": 40,
      "prompt": "System prompt text..."
    }
  }
}
```

Or load from YAML:

```bash
# Future enhancement: load agent config from YAML files
opencode --agent-config config/agents/implementation.yaml
```

## Troubleshooting

### Agent Not Completing Tasks
- Check max_steps constraint (may be too low)
- Verify temperature setting (too high = unfocused)
- Check file reservations aren't blocking

### Low Code Quality
- Review system prompt clarity
- Check capabilities match needs
- Adjust temperature (lower for consistency)

### Slow Processing
- Consider switching to faster model (Haiku)
- Reduce max_tokens if acceptable
- Split large tasks into smaller ones

## References

- [Agent Mail MCP](https://github.com/Dicklesworthstone/mcp_agent_mail) - Agent communication
- [Serena MCP](https://oraios.github.io/serena/) - Semantic code analysis
- [Beads MCP](https://github.com/steveyegge/beads) - Task management
- [Claude API Documentation](https://docs.anthropic.com/) - Model details
- [Effective Go](https://go.dev/doc/effective_go) - Go best practices
