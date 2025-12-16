# Agent Configuration System - Complete Index

Welcome to the Open Swarm Agent Configuration System. This directory contains comprehensive YAML configurations for coordinated multi-agent code development, testing, and review workflows.

## Quick Navigation

### For First-Time Users
1. Start with **[QUICK_REFERENCE.md](QUICK_REFERENCE.md)** - 5-minute overview
2. Read **[README.md](README.md)** - Complete guide with examples
3. Reference **[SCHEMA.md](SCHEMA.md)** - Configuration syntax and validation

### For Administrators
1. **[agents.yaml](agents.yaml)** - Master orchestration configuration
2. **[README.md](README.md)** - Extending configurations section
3. **[SCHEMA.md](SCHEMA.md)** - Configuration format validation

### For Agent Developers
1. Review **[SCHEMA.md](SCHEMA.md)** - Configuration structure
2. Check **[README.md](README.md)** - Extending Configurations section
3. Copy an existing agent config and customize

## File Directory

### Configuration Files (8 YAML files)

#### Core Implementation Agents
| File | Purpose | Focus | Model |
|------|---------|-------|-------|
| [implementation.yaml](implementation.yaml) | Production code writing | Go implementation, clean architecture | Opus 4.5 |
| [test-generator.yaml](test-generator.yaml) | Test code writing | Comprehensive testing, coverage | Sonnet 4.5 |

#### Code Review Agents (5 specialized reviewers)
| File | Purpose | Focus | Model |
|------|---------|-------|-------|
| [reviewer-architecture.yaml](reviewer-architecture.yaml) | Architecture & design | Package structure, patterns, SOLID | Opus 4.5 |
| [reviewer-functional.yaml](reviewer-functional.yaml) | Business logic | Requirements, edge cases, behavior | Sonnet 4.5 |
| [reviewer-testing.yaml](reviewer-testing.yaml) | Test quality | Coverage, patterns, determinism | Sonnet 4.5 |
| [reviewer-security.yaml](reviewer-security.yaml) | Security | Vulnerabilities, crypto, injection | Opus 4.5 |
| [reviewer-performance.yaml](reviewer-performance.yaml) | Performance | Complexity, memory, goroutines | Opus 4.5 |

#### Master Orchestration
| File | Purpose |
|------|---------|
| [agents.yaml](agents.yaml) | Workflow definitions, agent interaction, quality gates |

### Documentation Files (3 Markdown files)

| File | Purpose | Best For |
|------|---------|----------|
| [INDEX.md](INDEX.md) | This file - navigation guide | Finding what you need |
| [QUICK_REFERENCE.md](QUICK_REFERENCE.md) | Quick lookup guide | Fast answers, common patterns |
| [README.md](README.md) | Complete reference manual | Understanding agents in detail |
| [SCHEMA.md](SCHEMA.md) | Configuration specification | Creating/validating configs |

## Agent Roles at a Glance

### Implementation
- **Agent**: `implementation`
- **Role**: Write production Go code
- **Model**: Claude Opus 4.5
- **Capabilities**: Write, edit, run bash, semantic analysis
- **Temperature**: 0.2 (focused)
- **Max Tokens**: 8192

### Testing
- **Agent**: `test-generator`
- **Role**: Write comprehensive tests
- **Model**: Claude Sonnet 4.5
- **Capabilities**: Write, edit, run bash, semantic analysis
- **Temperature**: 0.3 (balanced)
- **Max Tokens**: 4096

### Review: Architecture
- **Agent**: `reviewer-architecture`
- **Role**: Evaluate design and patterns
- **Model**: Claude Opus 4.5
- **Capabilities**: Read, analyze, reference (no modifications)
- **Temperature**: 0.2 (analytical)
- **Focus**: Design patterns, package structure, SOLID principles

### Review: Functional
- **Agent**: `reviewer-functional`
- **Role**: Validate business logic
- **Model**: Claude Sonnet 4.5
- **Capabilities**: Read, analyze, reference (no modifications)
- **Temperature**: 0.2 (analytical)
- **Focus**: Requirements, edge cases, state consistency

### Review: Testing
- **Agent**: `reviewer-testing`
- **Role**: Assess test quality
- **Model**: Claude Sonnet 4.5
- **Capabilities**: Read, analyze, reference (no modifications)
- **Temperature**: 0.2 (analytical)
- **Focus**: Coverage, patterns, determinism

### Review: Security
- **Agent**: `reviewer-security`
- **Role**: Identify vulnerabilities
- **Model**: Claude Opus 4.5
- **Capabilities**: Read, analyze, reference (no modifications)
- **Temperature**: 0.1 (deterministic)
- **Focus**: Vulnerabilities, crypto, injection prevention

### Review: Performance
- **Agent**: `reviewer-performance`
- **Role**: Optimize performance
- **Model**: Claude Opus 4.5
- **Capabilities**: Read, analyze, reference (no modifications)
- **Temperature**: 0.2 (analytical)
- **Focus**: Complexity, memory, concurrency

## Typical Workflows

### Full Implementation & Review (Complete Quality Assurance)
```
1. implementation          → Write production code
2. test-generator         → Write comprehensive tests
3-7. All reviewers        → Parallel review
   ├→ Architecture review
   ├→ Functional review
   ├→ Testing review
   ├→ Security review
   └→ Performance review

Total: 7 phases, quality gates at each step
Time: Full analysis, ~2-4 hours per feature
```

### Fast Track (Urgent Fixes)
```
1. implementation          → Write code
2. test-generator         → Write minimal tests
3. reviewer-security      → Security check only
4. reviewer-functional    → Logic check only

Total: 4 phases, critical paths only
Time: 30-60 minutes
```

### Code Review Only (Existing Code)
```
All reviewers analyze existing code:
├→ reviewer-architecture
├→ reviewer-functional
├→ reviewer-testing
├→ reviewer-security
└→ reviewer-performance (optional)

Total: ~5 reviewers analyzing in parallel
Time: 1-2 hours
```

## Configuration Highlights

### Model Strategy
- **Opus 4.5**: Implementation and security/performance analysis (complex tasks)
- **Sonnet 4.5**: Testing and functional review (balanced capability)
- **Haiku 4.5**: Fast analysis and exploration (available as fallback)

### Temperature Tuning
- **0.1** (Security): Deterministic, no variance
- **0.2** (Reviews): Focused, consistent analysis
- **0.3** (Implementation): Balanced creativity and focus
- **0.4** (Testing): Creative for edge case discovery

### Tool Access Matrix
```
Tool         | Impl | Tests | Review
-------------|------|-------|--------
write        |  ✓   |  ✓    |   ✗
edit         |  ✓   |  ✓    |   ✗
read         |  ✓   |  ✓    |   ✓
bash         |  ✓   |  ✓    |   ✓
serena       |  ✓   |  ✓    |   ✓
grep/glob    |  ✓   |  ✓    |   ✓
```

## Key Features

### Specialized System Prompts
Each agent has a detailed system prompt covering:
- Role and mission statement
- Key focus areas and guidelines
- Evaluation criteria
- Expected feedback style
- Best practices for the domain

### Workflow Orchestration
Master configuration defines:
- Complete workflow sequences
- Inter-agent dependencies
- Parallel vs. sequential execution
- Quality gates and approval criteria
- Communication protocols

### Tool Configuration
Each agent has:
- Specific tools it can use
- Read/write restrictions
- Execution capabilities
- File reservation policies

### Quality Assurance
Built-in quality gates for:
- Code coverage (80%+ target)
- Security review (required, blocks merge)
- Architecture approval (required)
- Functional correctness (required)
- Performance (advisory)

## Using These Configurations

### In OpenCode
```bash
# Start a task with full implementation workflow
opencode task-start bd-123

# Run comprehensive review
opencode review src/handler.go

# Run specific reviewer
opencode review src/handler.go --reviewer security
```

### In Agent Mail
```bash
# Send to specific agent
/to @implementation
Subject: [bd-123] Implement auth handler
Body: Requirements and context...

# Run workflow
/workflow full-implementation-and-review
```

### Direct Configuration
```bash
# Load agent config
opencode --agent-config config/agents/implementation.yaml

# Validate configuration
opencode validate-config config/agents/agents.yaml
```

## Configuration Structure

### Typical Agent File (92-139 lines)
```yaml
agent:        # Name, role, description
model:        # Primary and fallback models
parameters:   # Temperature, tokens, sampling
system_prompt: # Detailed role instructions
capabilities: # Available tools and features
constraints:  # Limitations and boundaries
[specialized]: # Role-specific sections
```

### Master Configuration (275 lines)
```yaml
agents:        # All agents and their files
workflows:     # Complete workflow definitions
collaboration_rules: # How agents interact
tool_matrix:   # Tool availability
quality_gates: # Approval criteria
metadata:      # Version and tracking info
```

## Documentation Coverage

### README.md (575 lines)
- Complete agent descriptions
- Workflow explanations
- Configuration structure
- Best practices
- Troubleshooting guide
- Integration instructions

### QUICK_REFERENCE.md (369 lines)
- Agent summary table
- Workflow quick-start
- Capabilities matrix
- Common patterns
- Quick lookup table
- Troubleshooting tips

### SCHEMA.md (551 lines)
- Detailed syntax specification
- Field-by-field documentation
- Validation rules
- Configuration examples
- Best practices
- Troubleshooting guide

## Getting Started

### Step 1: Understand the System (5 minutes)
Read QUICK_REFERENCE.md for an overview.

### Step 2: Learn Individual Agents (20 minutes)
Review README.md's agent descriptions.

### Step 3: Study Configurations (15 minutes)
Look at actual .yaml files to see structure.

### Step 4: Try a Workflow (30+ minutes)
Run implementation workflow with a small task.

### Step 5: Customize (ongoing)
Modify agent configs as needed for your use case.

## Common Tasks

### To use implementation agent
→ See: [implementation.yaml](implementation.yaml), [README.md](README.md#implementationyaml)

### To use test generator
→ See: [test-generator.yaml](test-generator.yaml), [README.md](README.md#test-generatoryaml)

### To add a reviewer
→ See: [SCHEMA.md](SCHEMA.md#extending-configurations), [README.md](README.md#extending-configurations)

### To run a workflow
→ See: [QUICK_REFERENCE.md](QUICK_REFERENCE.md#quick-start-workflows), [README.md](README.md#agent-interaction-workflows)

### To validate config
→ See: [SCHEMA.md](SCHEMA.md#validation-rules)

## File Statistics

- **Total files**: 11 (8 YAML, 3 Markdown, 1 Index)
- **Total lines**: 2,509
- **Configuration YAML**: 815 lines
- **Documentation**: 1,495 lines
- **Largest file**: agents.yaml (275 lines)
- **Format**: YAML 3.1 compatible

## Version Information

- **Created**: 2025-12-13
- **Version**: 1.0.0
- **Framework**: Open Swarm
- **Target**: Go projects with multi-agent coordination

## Support & Resources

### Documentation (In This Directory)
- Complete reference: [README.md](README.md)
- Quick answers: [QUICK_REFERENCE.md](QUICK_REFERENCE.md)
- Schema reference: [SCHEMA.md](SCHEMA.md)

### Project Documentation
- Main README: `/README.md`
- Agent guidelines: `/AGENTS.md`
- Contribution guide: `/CONTRIBUTING.md`

### External Resources
- [Claude API Documentation](https://docs.anthropic.com/)
- [Effective Go](https://go.dev/doc/effective_go)
- [OpenCode Documentation](https://opencode.ai/docs/)

## Next Steps

1. **Read QUICK_REFERENCE.md** (5 min)
   - Get a quick overview of all agents
   - Understand the workflow options
   - See common patterns

2. **Review README.md** (30 min)
   - Deep dive into each agent role
   - Understand capabilities and constraints
   - Learn best practices

3. **Explore configuration files** (15 min)
   - Look at actual agent configurations
   - Understand the structure
   - See concrete examples

4. **Try a workflow** (varies)
   - Run implementation workflow with a task
   - Observe agent interactions
   - Review quality gate results

5. **Customize as needed** (ongoing)
   - Adjust temperatures or models
   - Add new specialized reviewers
   - Create domain-specific workflows

## Structure at a Glance

```
config/agents/
├── INDEX.md                      ← You are here
├── QUICK_REFERENCE.md            ← Start here (5 min)
├── README.md                     ← Complete guide (30 min)
├── SCHEMA.md                     ← Technical spec (reference)
├── agents.yaml                   ← Master configuration
├── implementation.yaml           ← Implementation agent
├── test-generator.yaml          ← Test writing agent
├── reviewer-architecture.yaml   ← Architecture reviewer
├── reviewer-functional.yaml     ← Functional reviewer
├── reviewer-testing.yaml        ← Testing reviewer
├── reviewer-security.yaml       ← Security reviewer
└── reviewer-performance.yaml    ← Performance reviewer
```

---

**Ready to dive in?** Start with [QUICK_REFERENCE.md](QUICK_REFERENCE.md) or jump to [README.md](README.md) for detailed information.
