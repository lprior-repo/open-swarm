// Package gates implements anti-cheating verification gates for agent execution.
// These gates ensure AI agents cannot lie about working code, skip hard work,
// or ignore requirements through immutable verification checks.
package gates

import (
	"context"
	"fmt"
)

// GateType identifies which verification gate failed.
type GateType string

const (
	// GateRequirements ensures agent proves understanding of the task.
	GateRequirements GateType = "requirements_verification"

	// GateTestImmutability ensures tests cannot be modified or disabled.
	GateTestImmutability GateType = "test_immutability"

	// GateEmpiricalHonesty ensures agent cannot claim success with failing tests.
	GateEmpiricalHonesty GateType = "empirical_honesty"

	// GateHardWork ensures stubbed code fails tests, no shortcuts.
	GateHardWork GateType = "hard_work_enforcement"

	// GateDriftDetection ensures agent stays aligned with original requirement.
	GateDriftDetection GateType = "requirement_drift_detection"
)

// GateError represents a failure to pass a verification gate.
type GateError struct {
	Gate      GateType
	TaskID    string
	Message   string
	Details   string
	Timestamp int64
}

// Error implements the error interface.
func (e *GateError) Error() string {
	return fmt.Sprintf("[%s] Task %s: %s", e.Gate, e.TaskID, e.Message)
}

// Unwrap allows error wrapping.
func (e *GateError) Unwrap() error {
	return nil
}

// TestResult contains the outcome of test execution.
type TestResult struct {
	Total    int      // Total tests executed.
	Passed   int      // Tests that passed.
	Failed   int      // Tests that failed.
	Output   string   // Raw test output (stdout + stderr).
	Failures []string // Individual failure messages.
	ExitCode int      // Process exit code.
}

// IsPassing returns true if all tests passed.
func (r *TestResult) IsPassing() bool {
	return r.Failed == 0 && r.Total > 0
}

const percentageMultiplier = 100

// PassRate returns the percentage of tests passing.
func (r *TestResult) PassRate() float64 {
	if r.Total == 0 {
		return 0
	}
	return float64(r.Passed) / float64(r.Total) * percentageMultiplier
}

// Requirement represents a task requirement from a Beads task.
type Requirement struct {
	TaskID      string   // Beads task ID
	Title       string   // Task title
	Description string   // Full task description
	Acceptance  string   // Acceptance criteria
	Scenarios   []string // Expected test scenarios
	EdgeCases   []string // Known edge cases to test
}

// Gate is the interface for all verification gates.
type Gate interface {
	// Check verifies the gate condition. Returns nil if passed, *GateError if failed.
	Check(ctx context.Context) error

	// Type returns which gate this is.
	Type() GateType

	// Name returns human-readable name.
	Name() string
}

// GateChain manages sequential execution of gates.
type GateChain struct {
	gates []Gate
}

// NewGateChain creates a new gate chain.
func NewGateChain(gates ...Gate) *GateChain {
	return &GateChain{gates: gates}
}

// Execute runs all gates in sequence. Returns first failure, or nil if all pass.
func (gc *GateChain) Execute(ctx context.Context) error {
	for _, g := range gc.gates {
		if err := g.Check(ctx); err != nil {
			return err
		}
	}
	return nil
}

// ExecuteParallel runs all gates in parallel (useful for independent checks).
// Returns all errors that occurred.
func (gc *GateChain) ExecuteParallel(ctx context.Context) []error {
	errChan := make(chan error, len(gc.gates))

	for _, g := range gc.gates {
		go func(gate Gate) {
			errChan <- gate.Check(ctx)
		}(g)
	}

	var errs []error
	for i := 0; i < len(gc.gates); i++ {
		if err := <-errChan; err != nil {
			errs = append(errs, err)
		}
	}

	return errs
}

// GateBuilder is a fluent builder for constructing verification workflows.
type GateBuilder struct {
	gates []Gate
}

// NewGateBuilder creates a new gate builder.
func NewGateBuilder() *GateBuilder {
	return &GateBuilder{}
}

// Add adds a gate to the builder.
func (gb *GateBuilder) Add(g Gate) *GateBuilder {
	gb.gates = append(gb.gates, g)
	return gb
}

// Build returns a GateChain ready to execute.
func (gb *GateBuilder) Build() *GateChain {
	return NewGateChain(gb.gates...)
}
