# Agent Configuration Schema

This document describes the YAML schema for agent configuration files.

## Top-Level Structure

```yaml
agent:              # Agent identity and metadata
model:              # AI model configuration
parameters:         # Model parameters (temperature, tokens, etc.)
system_prompt:      # Role-specific instructions
capabilities:       # Available tools and features
constraints:        # Limitations and boundaries
[role-specific]:    # Additional sections for specific agent types
```

## Section Specifications

### `agent` Section

Defines agent identity and basic metadata.

```yaml
agent:
  name: string              # Unique identifier (kebab-case)
  description: string       # Human-readable description
  role: string              # Agent role (implementer|tester|reviewer)
  review_type?: string      # For reviewers: type of review (architecture|functional|testing|security|performance)
```

**Requirements**:
- `name`: Must be unique within the project
- `name`: Use kebab-case (lowercase with hyphens)
- `description`: Should be concise but descriptive
- `role`: Must be one of defined roles

**Examples**:
```yaml
agent:
  name: test-generator
  description: Test writing specialist for Go projects
  role: tester

agent:
  name: reviewer-security
  description: Security review specialist for Go projects
  role: reviewer
  review_type: security
```

### `model` Section

Specifies AI models to use.

```yaml
model:
  primary: string           # Primary model (full model identifier)
  fallback: string          # Fallback model if primary unavailable
```

**Model Options**:
- `anthropic/claude-opus-4-5` - Most capable, best for complex tasks
- `anthropic/claude-sonnet-4-5` - Balanced capability and speed
- `anthropic/claude-haiku-4-5` - Fastest, for simple tasks

**Guidelines**:
- Use Opus for complex analysis and critical tasks
- Use Sonnet for balanced general tasks
- Use Haiku for fast, focused analysis
- Always provide a fallback

**Examples**:
```yaml
model:
  primary: anthropic/claude-opus-4-5
  fallback: anthropic/claude-sonnet-4-5

model:
  primary: anthropic/claude-sonnet-4-5
  fallback: anthropic/claude-haiku-4-5
```

### `parameters` Section

Model hyperparameters controlling output behavior.

```yaml
parameters:
  temperature: float        # 0.0-1.0 (determinism vs creativity)
  max_tokens: integer       # 1024-8192 (max output length)
  top_p?: float            # 0.0-1.0 (nucleus sampling)
  top_k?: integer          # Positive integer (top-k sampling)
```

**Parameter Ranges**:

| Parameter | Min | Max | Default | Notes |
|-----------|-----|-----|---------|-------|
| temperature | 0.0 | 1.0 | 0.2 | Lower = deterministic, Higher = creative |
| max_tokens | 1024 | 8192 | 4096 | Must match task complexity |
| top_p | 0.0 | 1.0 | 0.9 | Nucleus sampling parameter |
| top_k | 1 | 100 | 40 | Top-k sampling parameter |

**Temperature Guidelines**:
- **0.0-0.1**: Deterministic, analytical (validation, security)
- **0.2-0.3**: Focused, consistent (review, implementation)
- **0.4-0.5**: Creative, balanced (testing, documentation)
- **0.6+**: Very creative (not recommended for code)

**Token Guidelines**:
- **1024-2048**: Simple analysis, quick tasks
- **2048-4096**: Standard implementation, review
- **4096-8192**: Complex tasks, long outputs

**Examples**:
```yaml
parameters:
  temperature: 0.1
  max_tokens: 4096
  top_p: 0.9
  top_k: 40

parameters:
  temperature: 0.3
  max_tokens: 8192
  top_p: 0.9
```

### `system_prompt` Section

Role-specific instructions for the agent.

```yaml
system_prompt: |
  Multi-line string describing agent behavior,
  goals, guidelines, and constraints.
```

**Requirements**:
- Use YAML multiline string format (`|`)
- Clear, concise instructions
- Include specific guidelines for the role
- Reference frameworks or patterns when applicable
- Define success criteria

**Structure Guidelines**:
1. Agent title and mission statement
2. Key focus areas or review scope
3. Specific guidelines and best practices
4. Evaluation criteria
5. Feedback style expectations

**Examples**:
See individual agent YAML files for complete examples.

### `capabilities` Section

Defines tools and features the agent can use.

```yaml
capabilities:
  tools:
    write?: boolean        # Can create new files
    edit?: boolean         # Can modify existing files
    read: boolean          # Can read files
    bash?: boolean         # Can execute bash commands
    glob?: boolean         # Can match file patterns
    grep?: boolean         # Can search file contents
    serena_find_symbol?: boolean
    serena_find_referencing_symbols?: boolean
    serena_get_symbols_overview?: boolean
    serena_replace_symbol_body?: boolean
    serena_insert_after_symbol?: boolean
    serena_rename_symbol?: boolean

  features?: string[]      # Domain-specific capabilities
```

**Tool Categories**:

**File Operations**:
- `read`: Read file contents
- `write`: Create new files
- `edit`: Modify existing files

**Code Search**:
- `glob`: Match files by pattern
- `grep`: Search file contents
- `serena_*`: Semantic code analysis

**Command Execution**:
- `bash`: Run shell commands

**Feature List** (examples):
```yaml
features:
  - table_driven_tests
  - mocking_interfaces
  - design_pattern_analysis
  - coverage_analysis
  - vulnerability_detection
```

**Examples**:
```yaml
# Implementation agent
capabilities:
  tools:
    write: true
    edit: true
    read: true
    bash: true
    glob: true
    serena_find_symbol: true
    serena_replace_symbol_body: true

  features:
    - clean_architecture
    - dependency_injection
    - interface_driven

# Review agent
capabilities:
  tools:
    read: true
    bash: true
    grep: true
    glob: true
    serena_find_symbol: true
    serena_find_referencing_symbols: true

  features:
    - vulnerability_detection
    - complexity_analysis
```

### `constraints` Section

Defines agent limitations and boundaries.

```yaml
constraints:
  max_steps: integer                # Maximum steps before stopping
  file_reservation_ttl_seconds?: integer  # Time to hold file locks
  can_modify_code?: boolean         # Can modify source files
  can_modify_tests?: boolean        # Can modify test files
  read_only?: boolean               # Read-only mode
  requires_test_validation?: boolean # Tests must validate
```

**Standard Ranges**:

| Constraint | Implementation | Reviewer | Tester |
|-----------|-------|----------|--------|
| max_steps | 40-50 | 30-40 | 40-50 |
| file_reservation_ttl_seconds | 1800-3600 | N/A | 1800 |
| can_modify_code | true | false | false |
| can_modify_tests | false | false | true |
| read_only | false | true | false |

**Examples**:
```yaml
# Implementation
constraints:
  max_steps: 50
  file_reservation_ttl_seconds: 3600
  can_modify_code: true
  requires_test_validation: true

# Reviewer
constraints:
  max_steps: 40
  can_modify_code: false
  read_only: true
```

### Role-Specific Sections

Additional sections based on agent role.

#### Implementation Agent

```yaml
code_style:
  formatter: string          # Code formatter (gofmt)
  linter: string             # Linter (golangci-lint)
  line_length: integer       # Max line length

architectural_patterns:
  - string                   # Supported patterns

testing_requirements:
  validate_with_tests: boolean
  minimum_coverage: float    # 0.0-1.0
  integration_tests: boolean
```

#### Test Generator Agent

```yaml
test_patterns:
  - pattern: string          # File pattern (*_test.go)
    framework: string        # Test framework
    assertion_library: string

output_preferences:
  verbose_assertions: boolean
  include_comments: boolean
  test_documentation: boolean
```

#### Reviewer Agents

```yaml
focus_areas:
  - string                   # What to focus on

output_preferences:
  include_diagrams?: boolean
  explain_patterns?: boolean
  suggest_refactoring?: boolean
```

#### Security Reviewer

```yaml
security_standards:
  - string                   # Standards to follow

threat_model:
  - string                   # Threat types to consider
```

#### Performance Reviewer

```yaml
performance_metrics:
  - string                   # Metrics to analyze

optimization_priorities:
  - string                   # What to optimize first
```

## Master Configuration (`agents.yaml`)

The master configuration file orchestrates all agents.

```yaml
agents:
  {agent-name}:
    file: string             # Config file path
    description: string      # Description
    primary_responsibility: string
    interaction_sequence:
      - action: string
      - depends_on: [agent-names]

workflows:
  {workflow-name}:
    name: string
    description: string
    steps:
      - step: integer
        agent: string
        task: string
        required: boolean
        depends_on: [integers]
        output: string

collaboration_rules:
  communication:
    protocol: string         # agent-mail
    threads_by_review_type: boolean

  feedback_handling:
    critical_blocks_merge: boolean
    security_blocks_merge: boolean

tool_matrix:
  {agent-name}:
    - string                 # Available tools

quality_gates:
  {stage}:
    - string                 # Quality criteria

metadata:
  version: string
  last_updated: string
  created_for: string
```

## Validation Rules

### Required Fields
- `agent.name`
- `agent.description`
- `agent.role`
- `model.primary`
- `model.fallback`
- `parameters.temperature`
- `parameters.max_tokens`
- `system_prompt`
- `capabilities.tools`
- `constraints.max_steps`

### Conditional Requirements
- `agent.review_type` required if `role: reviewer`
- `file_reservation_ttl_seconds` required for write-capable agents
- `can_modify_code` must be `false` if `read_only: true`

### Value Constraints
- `temperature`: 0.0 ≤ value ≤ 1.0
- `max_tokens`: 1024 ≤ value ≤ 8192
- `max_steps`: 20 ≤ value ≤ 100
- `file_reservation_ttl_seconds`: 60 ≤ value ≤ 86400
- `name`: Must match regex `^[a-z][a-z0-9-]*[a-z0-9]$`

## YAML Syntax Notes

### Multiline Strings
```yaml
# Preserve newlines (use | or |+)
system_prompt: |
  Line 1
  Line 2
  Line 3

# Without trailing newlines (use |-)
system_prompt: |-
  Line 1
  Line 2
```

### Lists
```yaml
# Inline list
features: [feature1, feature2, feature3]

# Block list
features:
  - feature1
  - feature2
  - feature3
```

### Nested Objects
```yaml
capabilities:
  tools:
    read: true
    write: true
    bash: true
```

## Configuration Examples

### Minimal Configuration
```yaml
agent:
  name: my-reviewer
  description: Custom reviewer for specific domain
  role: reviewer
  review_type: custom

model:
  primary: anthropic/claude-sonnet-4-5
  fallback: anthropic/claude-haiku-4-5

parameters:
  temperature: 0.2
  max_tokens: 4096

system_prompt: |
  You are a custom reviewer. Focus on: area1, area2, area3.

capabilities:
  tools:
    read: true
    grep: true
    glob: true

constraints:
  max_steps: 40
  read_only: true
```

### Complete Configuration
See individual agent files in `config/agents/` directory.

## Best Practices

1. **Keep it DRY**: Use master config for shared settings
2. **Clear Prompts**: Make system prompts specific and actionable
3. **Appropriate Models**: Match model capability to task complexity
4. **Conservative Defaults**: Use lower temperatures for critical tasks
5. **Document Changes**: Update README when modifying configs
6. **Test Configs**: Try new configurations with sample tasks first
7. **Version Control**: Include metadata for tracking changes

## Common Configuration Patterns

### Pattern: Simple Read-Only Reviewer
```yaml
model:
  primary: anthropic/claude-sonnet-4-5
  fallback: anthropic/claude-haiku-4-5

parameters:
  temperature: 0.2
  max_tokens: 4096

constraints:
  max_steps: 40
  read_only: true
```

### Pattern: Complex Implementation Agent
```yaml
model:
  primary: anthropic/claude-opus-4-5
  fallback: anthropic/claude-sonnet-4-5

parameters:
  temperature: 0.2
  max_tokens: 8192

constraints:
  max_steps: 50
  file_reservation_ttl_seconds: 3600
  can_modify_code: true
  requires_test_validation: true
```

## Troubleshooting Configuration

| Issue | Solution |
|-------|----------|
| Invalid YAML syntax | Use YAML validator, check indentation |
| Agent not found | Check `name` field, verify file in agents/ |
| Model not available | Check OpenCode config for available models |
| Agent stops early | Increase `max_steps` |
| Inaccurate output | Adjust `temperature` or model selection |

## References

- [YAML Specification](https://yaml.org/spec/)
- [OpenCode Documentation](https://opencode.ai/docs/)
- [Claude API Guide](https://docs.anthropic.com/)
- Agent Configuration Files: `config/agents/*.yaml`
- Master Configuration: `config/agents/agents.yaml`
