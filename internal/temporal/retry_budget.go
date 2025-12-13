// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package temporal

// Default maximum number of retries allowed per gate
const DefaultMaxRetries = 2

// RetryBudget tracks retry attempts for different gate types in Enhanced TCR workflow
// Supports max 2 retries for test-gen and impl gates as per workflow specification
type RetryBudget struct {
	// TestGenRetries tracks retry attempts for test generation gate
	TestGenRetries int

	// ImplRetries tracks retry attempts for implementation gate
	ImplRetries int

	// MaxRetries is the maximum number of retries allowed per gate (default: 2)
	MaxRetries int
}

// GateType identifies different gates in the Enhanced TCR workflow
type GateType string

const (
	// GateTestGen represents the test generation gate
	GateTestGen GateType = "test_gen"
	// GateImpl represents the implementation gate
	GateImpl GateType = "impl"
)

// NewRetryBudget creates a new RetryBudget with default max retries (2)
func NewRetryBudget() *RetryBudget {
	return &RetryBudget{
		TestGenRetries: 0,
		ImplRetries:    0,
		MaxRetries:     DefaultMaxRetries,
	}
}

// NewRetryBudgetWithMax creates a new RetryBudget with custom max retries
func NewRetryBudgetWithMax(maxRetries int) *RetryBudget {
	return &RetryBudget{
		TestGenRetries: 0,
		ImplRetries:    0,
		MaxRetries:     maxRetries,
	}
}

// CanRetry checks if retries are still available for the given gate type
// Returns true if current retry count is less than max retries
func (rb *RetryBudget) CanRetry(gate GateType) bool {
	switch gate {
	case GateTestGen:
		return rb.TestGenRetries < rb.MaxRetries
	case GateImpl:
		return rb.ImplRetries < rb.MaxRetries
	default:
		return false
	}
}

// IncrementRetry increments the retry counter for the given gate type
// Returns the new retry count, or -1 if gate type is invalid
func (rb *RetryBudget) IncrementRetry(gate GateType) int {
	switch gate {
	case GateTestGen:
		rb.TestGenRetries++
		return rb.TestGenRetries
	case GateImpl:
		rb.ImplRetries++
		return rb.ImplRetries
	default:
		return -1
	}
}

// GetRetryCount returns the current retry count for the given gate type
func (rb *RetryBudget) GetRetryCount(gate GateType) int {
	switch gate {
	case GateTestGen:
		return rb.TestGenRetries
	case GateImpl:
		return rb.ImplRetries
	default:
		return -1
	}
}

// ResetRetryCounters resets all retry counters to zero
func (rb *RetryBudget) ResetRetryCounters() {
	rb.TestGenRetries = 0
	rb.ImplRetries = 0
}

// ResetGateRetry resets the retry counter for a specific gate type
func (rb *RetryBudget) ResetGateRetry(gate GateType) {
	switch gate {
	case GateTestGen:
		rb.TestGenRetries = 0
	case GateImpl:
		rb.ImplRetries = 0
	}
}

// RemainingRetries returns the number of retries remaining for the given gate type
func (rb *RetryBudget) RemainingRetries(gate GateType) int {
	switch gate {
	case GateTestGen:
		return rb.MaxRetries - rb.TestGenRetries
	case GateImpl:
		return rb.MaxRetries - rb.ImplRetries
	default:
		return 0
	}
}

// IsExhausted checks if the retry budget is exhausted for the given gate type
func (rb *RetryBudget) IsExhausted(gate GateType) bool {
	return !rb.CanRetry(gate)
}

// TotalRetries returns the total number of retries attempted across all gates
func (rb *RetryBudget) TotalRetries() int {
	return rb.TestGenRetries + rb.ImplRetries
}
