# Test Generation Prompt Builder

The test generation prompt builder in `internal/prompts` provides a comprehensive system for building prompts that guide test generation agents. It supports three modes of test generation with full integration of code context, lint feedback, and coverage information.

## Overview

The prompt builder helps generate structured, well-formatted prompts for test generation agents with support for:

- **Initial Test Generation** - Generate tests from fresh task descriptions
- **Test Refinement** - Refine tests based on lint feedback
- **Test Enhancement** - Improve tests based on coverage reports

## Types

### TestGenerationRequest

The main request type that encapsulates all parameters for test generation:

```go
request := &TestGenerationRequest{
    Mode:               TestModeInitial,
    TaskDescription:    "Write tests for user authentication",
    OutputPath:         "auth_test.go",
    Language:           "Go",
    TestFramework:      "testing",
    AcceptanceCriteria: &TestGenerationCriteria{...},
    CodeContext:        &TestCodeContext{...},
    LintFeedback:       &LintFeedback{...},
    CoverageReport:     &CoverageReport{...},
}
```

### TestGenerationMode

Indicates the type of test generation:

- `TestModeInitial` - Generate comprehensive tests from scratch
- `TestModeRefinement` - Refine tests based on lint feedback
- `TestModeEnhancement` - Enhance tests to improve coverage

### TestGenerationCriteria

Defines acceptance requirements with three categories:

```go
criteria := &TestGenerationCriteria{
    Required: []string{
        "Test valid input",
        "Test error handling",
    },
    Nice: []string{
        "Performance benchmarks",
    },
    Constraints: []string{
        "Must complete in < 100ms",
    },
}
```

### LintFeedback

Represents lint violations to be addressed:

```go
feedback := &LintFeedback{
    Summary:  "Multiple test naming violations",
    Errors: []string{
        "TestCreate: missing error case",
    },
    Warnings: []string{
        "Mock setup could be simplified",
    },
}
```

### CoverageReport

Contains code coverage information:

```go
report := &CoverageReport{
    TotalCoverage: 65.5,
    UncoveredFunctions: []string{
        "QueryBuilder.WithIndex",
        "Transaction.Rollback",
    },
    Report: "coverage.txt content",
}
```

### TestCodeContext

Provides source code context:

```go
ctx := &TestCodeContext{
    FilePath: "pkg/user/validation.go",
    Content:  "full source code",
    Excerpt:  "specific function code",
    Language: "go",
}
```

## Usage

### Using the Direct Builder

```go
request := &TestGenerationRequest{
    Mode:            TestModeInitial,
    TaskDescription: "Write tests for validation",
    OutputPath:      "validation_test.go",
    Language:        "Go",
    TestFramework:   "testing",
}

builder := NewTestGenerationPromptBuilder(request)
prompt := builder.Build()
// Use prompt with test generation agent
```

### Using the Fluent Builder (Recommended)

```go
prompt := NewTestGenerationBuilder().
    WithMode(TestModeInitial).
    WithTaskDescription("Write parser tests").
    WithOutputPath("parser_test.go").
    WithLanguage("Go").
    WithTestFramework("testing").
    AddRequiredCriteria("Parse valid JSON").
    AddRequiredCriteria("Reject invalid JSON").
    AddNiceCriteria("Handle large files efficiently").
    AddConstraint("Must be deterministic").
    WithAdditionalNotes("Focus on error recovery").
    BuildPrompt()
```

### With Lint Feedback (Refinement Mode)

```go
prompt := NewTestGenerationBuilder().
    WithMode(TestModeRefinement).
    WithTaskDescription("Fix lint issues in tests").
    WithOutputPath("storage_test.go").
    AddLintError("Missing test cases for error paths").
    AddLintError("Inconsistent test naming").
    AddLintWarning("Tests execute slowly").
    WithLintSummary("Found 2 errors and 1 warning").
    BuildPrompt()
```

### With Coverage Report (Enhancement Mode)

```go
report := &CoverageReport{
    TotalCoverage: 65.5,
    UncoveredFunctions: []string{
        "QueryBuilder.WithIndex",
        "Transaction.Rollback",
    },
    Report: "coverage report output",
}

prompt := NewTestGenerationBuilder().
    WithMode(TestModeEnhancement).
    WithTaskDescription("Improve code coverage for database package").
    WithOutputPath("db_test.go").
    WithCoverageReport(report).
    BuildPrompt()
```

### With Code Context

```go
prompt := NewTestGenerationBuilder().
    WithTaskDescription("Test validation functions").
    WithOutputPath("validation_test.go").
    WithCodeContextFromFile(
        "pkg/user/validation.go",
        "full source code content",
    ).
    WithCodeExcerpt("func ValidateEmail(email string) error {").
    BuildPrompt()
```

## Prompt Structure

Generated prompts follow a consistent structure:

1. **Header** - Request type, mode, language, framework
2. **Task Description** - What needs to be tested
3. **Acceptance Criteria** - Required, nice-to-have, and constraints
4. **Code Context** (optional) - Source code or relevant excerpts
5. **Lint Feedback** (optional) - Lint errors and warnings
6. **Coverage Report** (optional) - Coverage metrics and uncovered areas
7. **Output Instructions** - How to format and structure tests
8. **Generation Guidelines** - Mode-specific guidance
9. **Additional Notes** (optional) - Extra instructions
10. **Footer** - Metadata and timestamp

## Generation Modes

### TestModeInitial

Generates comprehensive test suite from scratch:

- Basic functionality (happy path)
- Error cases and edge cases
- Boundary conditions
- Integration points
- Performance considerations

### TestModeRefinement

Refines tests based on lint feedback:

- Addresses all flagged lint issues
- Improves code quality and maintainability
- Ensures tests follow project standards
- Maintains or improves coverage
- Optimizes test execution time

### TestModeEnhancement

Enhances tests to improve coverage:

- Adds tests for uncovered functions
- Covers remaining edge cases
- Improves existing test quality
- Adds integration tests where appropriate
- Tests error recovery paths

## Integration with Test Generation Agents

The prompt builder is designed to work with test generation agents via the OpenCode SDK:

```go
// Create prompt
prompt := NewTestGenerationBuilder().
    WithTaskDescription("Write tests for authentication").
    WithOutputPath("auth_test.go").
    AddRequiredCriteria("Test valid credentials").
    AddRequiredCriteria("Test invalid credentials").
    BuildPrompt()

// Send to agent
result, err := client.ExecutePrompt(ctx, prompt, &PromptOptions{
    Model:         "claude-opus-4-5",
    Agent:         "test-generator",
    TestFramework: "testing",
})
```

## File Location

All code is located in `/home/lewis/src/open-swarm/internal/prompts/`:

- `test_generation_builder.go` - Main builder implementation
- `test_generation_builder_test.go` - Comprehensive test suite
- `types.go` - Type definitions including TestGenerationRequest

## Testing

Run tests with:

```bash
go test -v ./internal/prompts/...
```

All test modes are thoroughly tested including:

- Initial mode with acceptance criteria
- Refinement mode with lint feedback
- Enhancement mode with coverage reports
- Code context integration
- Fluent builder API
- Error handling
