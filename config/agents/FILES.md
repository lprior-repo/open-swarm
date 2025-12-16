# Agent Configuration Files - Complete Listing

## Overview

This directory (`config/agents/`) contains 12 files organized into two categories: Configuration files and Documentation files.

**Total Size**: ~96 KB | **Total Lines**: ~2,509 | **Created**: 2025-12-13

---

## Configuration Files (8 YAML files)

### Core Agent Configurations

#### 1. implementation.yaml
**Size**: 2.5 KB (92 lines)
**Type**: Implementation Agent Configuration

Configures the production code implementation specialist agent.

```yaml
Role: Write production Go code
Model: Claude Opus 4.5 (primary), Claude Sonnet 4.5 (fallback)
Temperature: 0.2
Max Tokens: 8192
Max Steps: 50
```

**Key Sections**:
- Agent identity and role
- Model selection strategy
- Code style and architecture patterns
- Testing requirements
- Capabilities: write, edit, read, bash, serena_all
- Constraints: file reservation TTL 3600s, requires test validation

**Use Cases**:
- Implementing new features
- Writing business logic
- Refactoring code
- Building services

---

#### 2. test-generator.yaml
**Size**: 1.9 KB (74 lines)
**Type**: Test Writing Agent Configuration

Configures the comprehensive test writing specialist agent.

```yaml
Role: Write comprehensive Go tests
Model: Claude Sonnet 4.5 (primary), Claude Haiku 4.5 (fallback)
Temperature: 0.3
Max Tokens: 4096
Max Steps: 50
```

**Key Sections**:
- Agent identity and role
- Model selection strategy
- Test patterns and frameworks
- Coverage targets (80%+)
- Capabilities: write, edit, read, bash, serena_all
- Constraints: file reservation TTL 1800s, cannot modify source

**Use Cases**:
- Writing unit tests
- Writing integration tests
- Improving test coverage
- Creating test fixtures

---

### Review Agent Configurations

#### 3. reviewer-architecture.yaml
**Size**: 2.6 KB (92 lines)
**Type**: Architecture Reviewer Configuration

Configures the architecture and design pattern review specialist.

```yaml
Role: Review architecture and design patterns
Model: Claude Opus 4.5 (primary), Claude Sonnet 4.5 (fallback)
Temperature: 0.2
Max Tokens: 6144
Max Steps: 40
```

**Key Sections**:
- Agent identity and review type
- Model selection strategy
- Focus areas: packages, interfaces, SOLID, design patterns
- Architectural patterns to validate
- Capabilities: read-only, semantic analysis
- Constraints: no file modifications

**Review Focus**:
- Design pattern correctness
- Package organization
- SOLID principles compliance
- Interface design
- Dependency management
- Scalability assessment

---

#### 4. reviewer-functional.yaml
**Size**: 2.6 KB (99 lines)
**Type**: Functional Reviewer Configuration

Configures the business logic and requirement validation specialist.

```yaml
Role: Review functional correctness
Model: Claude Sonnet 4.5 (primary), Claude Haiku 4.5 (fallback)
Temperature: 0.2
Max Tokens: 4096
Max Steps: 40
```

**Key Sections**:
- Agent identity and review type
- Model selection strategy
- Focus areas: business logic, requirements, edge cases
- Testing perspective for validation
- Capabilities: read-only, semantic analysis
- Constraints: no file modifications

**Review Focus**:
- Business logic correctness
- Requirement compliance
- Edge case handling
- Error path validation
- State consistency
- Integration verification

---

#### 5. reviewer-testing.yaml
**Size**: 2.8 KB (108 lines)
**Type**: Testing Reviewer Configuration

Configures the test quality and coverage analysis specialist.

```yaml
Role: Review test quality and coverage
Model: Claude Sonnet 4.5 (primary), Claude Haiku 4.5 (fallback)
Temperature: 0.2
Max Tokens: 4096
Max Steps: 40
```

**Key Sections**:
- Agent identity and review type
- Model selection strategy
- Coverage targets: 80% unit, 60% integration, 95% critical
- Focus areas: coverage, patterns, determinism
- Capabilities: read-only, semantic analysis
- Constraints: no file modifications

**Review Focus**:
- Test coverage analysis
- Coverage gap identification
- Test pattern validation
- Mock effectiveness
- Test independence
- Flakiness detection

---

#### 6. reviewer-security.yaml
**Size**: 3.6 KB (135 lines)
**Type**: Security Reviewer Configuration

Configures the security vulnerability and best practice specialist.

```yaml
Role: Review security vulnerabilities
Model: Claude Opus 4.5 (primary), Claude Sonnet 4.5 (fallback)
Temperature: 0.1 (deterministic)
Max Tokens: 6144
Max Steps: 50
```

**Key Sections**:
- Agent identity and review type
- Model selection strategy (Opus for rigor)
- Security standards: OWASP Top 10, CWE
- Threat model specification
- Capabilities: read-only, semantic analysis
- Constraints: no file modifications

**Security Focus**:
- Input validation
- Authentication/authorization
- Cryptographic operations
- Injection attack prevention
- Sensitive data protection
- Dependency vulnerabilities
- Secret management

**Severity Classification**:
- CRITICAL: Immediate exploitation possible
- HIGH: Exploitable under specific conditions
- MEDIUM: Potential vulnerability
- LOW: Best practice violation

---

#### 7. reviewer-performance.yaml
**Size**: 3.7 KB (139 lines)
**Type**: Performance Reviewer Configuration

Configures the performance optimization and efficiency specialist.

```yaml
Role: Review performance and efficiency
Model: Claude Opus 4.5 (primary), Claude Sonnet 4.5 (fallback)
Temperature: 0.2
Max Tokens: 5120
Max Steps: 40
```

**Key Sections**:
- Agent identity and review type
- Model selection strategy
- Performance metrics to analyze
- Optimization priorities
- Capabilities: read-only, semantic analysis
- Constraints: no file modifications

**Performance Focus**:
- Algorithm complexity
- Memory efficiency
- Goroutine management
- Lock contention
- Caching opportunities
- I/O optimization
- Scalability assessment

**Impact Classification**:
- CRITICAL: >50% degradation
- HIGH: 10-50% degradation
- MEDIUM: 5-10% degradation
- LOW: <5% degradation

---

### Master Configuration

#### 8. agents.yaml
**Size**: 7.2 KB (275 lines)
**Type**: Master Orchestration Configuration

Master configuration file that ties all agents together and defines workflows.

**Key Sections**:

- **agents**: Definitions for all 7 agent roles
  - File paths to individual configurations
  - Primary responsibilities
  - Interaction sequences
  - Dependencies

- **workflows**: Three complete workflow definitions
  - `full_implementation_and_review`: 7 steps, comprehensive QA
  - `fast_implementation_workflow`: 4 steps, urgent fixes
  - `review_only_workflow`: 5 parallel reviewers

- **collaboration_rules**:
  - Communication protocol (Agent Mail)
  - Feedback handling priorities
  - Conflict resolution strategy

- **tool_matrix**: Tool availability by agent

- **quality_gates**:
  - Implementation gate: coverage, lint, approvals
  - Testing gate: coverage, edge cases, determinism
  - Code review gate: security, functional, architecture (blocks), performance (advisory)

- **metadata**: Version, created date, project info

**Use Case**: Orchestrating multi-agent workflows and defining quality standards.

---

## Documentation Files (4 Markdown files)

### Quick Start Documentation

#### 1. INDEX.md
**Size**: 13 KB (complete navigation guide)
**Type**: Navigation and Overview

Your starting point for the agent configuration system.

**Contents**:
- Quick navigation for different user types
- File directory with descriptions
- Agent roles summary table
- Typical workflows overview
- Configuration highlights
- Getting started steps
- File statistics

**Best For**: Finding what you need and understanding the big picture.

**Reading Time**: 5-10 minutes

---

#### 2. QUICK_REFERENCE.md
**Size**: 9.4 KB (quick lookup guide)
**Type**: Quick Reference Card

Fast lookup guide for common tasks and agent information.

**Contents**:
- File structure overview
- Agent summary table
- Quick start workflows
- Agent capabilities at a glance
- Key constraints table
- When to use each agent
- Configuration keys reference
- System prompts summary
- Common patterns
- Integration examples
- Troubleshooting matrix

**Best For**: Quick answers, finding specific information, common patterns.

**Reading Time**: 5 minutes for overview, reference as needed

---

### Comprehensive Documentation

#### 3. README.md
**Size**: 15 KB (comprehensive reference manual)
**Type**: Complete Reference

Full documentation for all aspects of the agent configuration system.

**Contents**:

**Agent Descriptions** (detailed):
- Agent role and specialization
- Model selection rationale
- System prompt explanation
- Capabilities breakdown
- Constraints and limitations
- Feedback format
- Use cases and when to use

**Workflows Section**:
- Full implementation and review workflow
- Fast implementation workflow
- Review-only workflow

**Configuration Guide**:
- Configuration structure overview
- Each section explained in detail
- Best practices
- Extending configurations
- Integration with OpenCode

**Best Practices**:
- Sequential workflow principles
- Clear communication guidelines
- Context preservation
- Feedback prioritization
- Temperature control
- Tool discipline

**Troubleshooting**:
- Common issues and solutions
- Configuration adjustments
- Performance tuning

**Best For**: Deep understanding, detailed reference, best practices.

**Reading Time**: 30-40 minutes for full read, reference as needed

---

#### 4. SCHEMA.md
**Size**: 13 KB (technical specification)
**Type**: Configuration Specification

Technical documentation of YAML configuration structure and validation.

**Contents**:

**Schema Specification**:
- Top-level structure
- Each section with detailed specifications
- Required fields and requirements
- Value constraints and ranges
- Field-by-field documentation

**Parameter Ranges**:
- Temperature guidelines (0.0-1.0)
- Token ranges (1024-8192)
- Max steps (20-100)

**Validation Rules**:
- Required fields checklist
- Conditional requirements
- Value constraint validation
- Naming conventions

**YAML Syntax Notes**:
- Multiline string formatting
- List syntax
- Nested object structure

**Configuration Examples**:
- Minimal configuration
- Complete configuration template
- Pattern examples

**Best Practices**:
- DRY principles
- Clear prompts
- Appropriate model selection
- Conservative defaults
- Documentation and versioning

**Troubleshooting**:
- Common configuration issues
- Solutions and fixes

**Best For**: Creating new configurations, validation, technical reference.

**Reading Time**: 20-30 minutes for understanding, reference as needed

---

### System Documentation

#### 5. FILES.md
**Size**: This file (complete file listing)
**Type**: File Documentation

Comprehensive listing and description of all files in the configuration system.

**Contents**:
- Overview of all files
- Individual file descriptions
- Size and line counts
- Key sections and contents
- Use cases for each file
- Reading time estimates
- Cross-references

**Best For**: Understanding what each file contains and finding specific files.

**Reading Time**: 10-15 minutes

---

## File Organization

### Directory Structure

```
config/agents/
├── FILES.md                     ← You are here
├── INDEX.md                     ← Start here
├── README.md                    ← Comprehensive guide
├── QUICK_REFERENCE.md           ← Quick lookup
├── SCHEMA.md                    ← Technical spec
│
├── agents.yaml                  ← Master config
├── implementation.yaml          ← Implementation agent
├── test-generator.yaml         ← Test writing agent
├── reviewer-architecture.yaml  ← Architecture reviewer
├── reviewer-functional.yaml    ← Functional reviewer
├── reviewer-testing.yaml       ← Testing reviewer
├── reviewer-security.yaml      ← Security reviewer
└── reviewer-performance.yaml   ← Performance reviewer
```

### File Dependencies

```
agents.yaml
├── implementation.yaml
├── test-generator.yaml
├── reviewer-architecture.yaml
├── reviewer-functional.yaml
├── reviewer-testing.yaml
├── reviewer-security.yaml
└── reviewer-performance.yaml

Documentation:
├── INDEX.md (entry point)
├── QUICK_REFERENCE.md (quick answers)
├── README.md (detailed guide)
├── SCHEMA.md (technical spec)
└── FILES.md (file listing)
```

---

## Reading Paths

### For First-Time Users (25 minutes)
1. INDEX.md (5 min) - Overview and navigation
2. QUICK_REFERENCE.md (5 min) - Quick summary
3. Individual agent YAML files (10 min) - See real configs
4. README.md sections (5 min) - Deep dive as needed

### For Administrators (45 minutes)
1. QUICK_REFERENCE.md (5 min) - Overview
2. agents.yaml (10 min) - Understand workflows
3. README.md (20 min) - Best practices and integration
4. SCHEMA.md (10 min) - Validation and extension

### For Developers (1 hour)
1. QUICK_REFERENCE.md (5 min) - Quick overview
2. Relevant agent YAML file (10 min) - Understand role
3. SCHEMA.md (20 min) - Configuration structure
4. README.md (20 min) - Best practices
5. agents.yaml (5 min) - Workflow integration

### For Configuration Extension (30 minutes)
1. SCHEMA.md (15 min) - Configuration template
2. agents.yaml (10 min) - Understand workflows
3. Similar agent YAML (5 min) - Use as template

---

## File Access Patterns

### By Task

**"I want to understand the system"**
→ Start with INDEX.md, then QUICK_REFERENCE.md

**"I need to create a new agent"**
→ Read SCHEMA.md, reference similar agent YAML, update agents.yaml

**"I want to run a workflow"**
→ Check QUICK_REFERENCE.md workflows, see agents.yaml orchestration

**"I need to modify an agent's behavior"**
→ Edit the agent's YAML file, reference SCHEMA.md for syntax

**"I want detailed information about a specific agent"**
→ Read README.md agent section, reference YAML file

**"I need to troubleshoot an issue"**
→ Check README.md or QUICK_REFERENCE.md troubleshooting sections

---

## Content Summary

### Configuration Coverage
- 2 Implementation agents (writing code/tests)
- 5 Review agents (architecture, functional, testing, security, performance)
- 1 Master configuration (workflows and orchestration)
- Complete system for end-to-end development workflows

### Documentation Coverage
- Quick reference (5 minutes)
- Comprehensive guide (30 minutes)
- Technical specification (20 minutes)
- File documentation (this file)

### Workflow Support
- Full implementation and review (7 steps, 2-4 hours)
- Fast track for urgent fixes (4 steps, 30-60 minutes)
- Code review only (5 steps, 1-2 hours)

### Quality Standards
- Test coverage target: 80%+
- Security review: Required (blocks merge)
- Functional review: Required (blocks merge)
- Architecture review: Required (blocks merge)
- Performance review: Advisory

---

## Integration Points

### With OpenCode
- Configuration files loadable by OpenCode
- Agents available as subagents
- Workflows can be triggered via commands

### With Agent Mail
- Inter-agent communication via messages
- Thread-based coordination
- File reservation management

### With Serena
- Semantic code analysis for all agents
- Symbol navigation and analysis
- Code modification with semantic awareness

### With Beads
- Task management integration
- Status tracking
- Issue correlation

### With Project Files
- AGENTS.md (project guidelines)
- CONTRIBUTING.md (contribution guide)
- opencode.json (agent system config)

---

## Statistics

| Metric | Value |
|--------|-------|
| Total Files | 12 |
| Configuration Files (YAML) | 8 |
| Documentation Files (MD) | 4 |
| Total Size | ~96 KB |
| Total Lines | ~2,509 |
| Configuration Lines | ~815 |
| Documentation Lines | ~1,694 |
| Agents Configured | 7 |
| Workflows Defined | 3 |
| Review Types | 5 |
| Quality Gates | 12+ |

---

## Version Information

- **System Version**: 1.0.0
- **Created**: 2025-12-13
- **Framework**: Open Swarm
- **Target Language**: Go
- **Integration**: OpenCode, Agent Mail, Serena, Beads

---

## Next Steps

1. **Start with** → INDEX.md (5 min navigation guide)
2. **Quick overview** → QUICK_REFERENCE.md (5 min summary)
3. **Deep dive** → README.md (comprehensive reference)
4. **Technical details** → SCHEMA.md (specification)
5. **Try it out** → Run a workflow with your code

---

## Questions?

Refer to the appropriate file:
- **"What agents are available?"** → QUICK_REFERENCE.md
- **"How do I use this?"** → README.md
- **"What's the configuration syntax?"** → SCHEMA.md
- **"Where do I find X?"** → INDEX.md
- **"What does file Y contain?"** → FILES.md (this file)

