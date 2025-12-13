// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package temporal

import "testing"

func TestNewRetryBudget(t *testing.T) {
	rb := NewRetryBudget()

	if rb.MaxRetries != 2 {
		t.Errorf("Expected MaxRetries = 2, got %d", rb.MaxRetries)
	}

	if rb.TestGenRetries != 0 {
		t.Errorf("Expected TestGenRetries = 0, got %d", rb.TestGenRetries)
	}

	if rb.ImplRetries != 0 {
		t.Errorf("Expected ImplRetries = 0, got %d", rb.ImplRetries)
	}
}

func TestNewRetryBudgetWithMax(t *testing.T) {
	rb := NewRetryBudgetWithMax(5)

	if rb.MaxRetries != 5 {
		t.Errorf("Expected MaxRetries = 5, got %d", rb.MaxRetries)
	}

	if rb.TestGenRetries != 0 {
		t.Errorf("Expected TestGenRetries = 0, got %d", rb.TestGenRetries)
	}

	if rb.ImplRetries != 0 {
		t.Errorf("Expected ImplRetries = 0, got %d", rb.ImplRetries)
	}
}

func TestCanRetry_InitialState(t *testing.T) {
	rb := NewRetryBudget()

	if !rb.CanRetry(GateTestGen) {
		t.Error("Expected CanRetry(GateTestGen) = true initially")
	}

	if !rb.CanRetry(GateImpl) {
		t.Error("Expected CanRetry(GateImpl) = true initially")
	}
}

func TestCanRetry_AfterMaxRetries(t *testing.T) {
	rb := NewRetryBudget()

	// Exhaust TestGen retries
	rb.IncrementRetry(GateTestGen)
	rb.IncrementRetry(GateTestGen)

	if rb.CanRetry(GateTestGen) {
		t.Error("Expected CanRetry(GateTestGen) = false after 2 retries")
	}

	// Impl should still have retries
	if !rb.CanRetry(GateImpl) {
		t.Error("Expected CanRetry(GateImpl) = true when only TestGen is exhausted")
	}
}

func TestCanRetry_InvalidGate(t *testing.T) {
	rb := NewRetryBudget()

	invalidGate := GateType("invalid")
	if rb.CanRetry(invalidGate) {
		t.Error("Expected CanRetry(invalid) = false")
	}
}

func TestIncrementRetry_TestGen(t *testing.T) {
	rb := NewRetryBudget()

	count := rb.IncrementRetry(GateTestGen)
	if count != 1 {
		t.Errorf("Expected first increment to return 1, got %d", count)
	}

	count = rb.IncrementRetry(GateTestGen)
	if count != 2 {
		t.Errorf("Expected second increment to return 2, got %d", count)
	}

	if rb.TestGenRetries != 2 {
		t.Errorf("Expected TestGenRetries = 2, got %d", rb.TestGenRetries)
	}
}

func TestIncrementRetry_Impl(t *testing.T) {
	rb := NewRetryBudget()

	count := rb.IncrementRetry(GateImpl)
	if count != 1 {
		t.Errorf("Expected first increment to return 1, got %d", count)
	}

	count = rb.IncrementRetry(GateImpl)
	if count != 2 {
		t.Errorf("Expected second increment to return 2, got %d", count)
	}

	if rb.ImplRetries != 2 {
		t.Errorf("Expected ImplRetries = 2, got %d", rb.ImplRetries)
	}
}

func TestIncrementRetry_InvalidGate(t *testing.T) {
	rb := NewRetryBudget()

	invalidGate := GateType("invalid")
	count := rb.IncrementRetry(invalidGate)

	if count != -1 {
		t.Errorf("Expected -1 for invalid gate, got %d", count)
	}
}

func TestGetRetryCount(t *testing.T) {
	rb := NewRetryBudget()

	rb.IncrementRetry(GateTestGen)
	rb.IncrementRetry(GateTestGen)
	rb.IncrementRetry(GateImpl)

	testGenCount := rb.GetRetryCount(GateTestGen)
	if testGenCount != 2 {
		t.Errorf("Expected TestGen count = 2, got %d", testGenCount)
	}

	implCount := rb.GetRetryCount(GateImpl)
	if implCount != 1 {
		t.Errorf("Expected Impl count = 1, got %d", implCount)
	}
}

func TestGetRetryCount_InvalidGate(t *testing.T) {
	rb := NewRetryBudget()

	invalidGate := GateType("invalid")
	count := rb.GetRetryCount(invalidGate)

	if count != -1 {
		t.Errorf("Expected -1 for invalid gate, got %d", count)
	}
}

func TestResetRetryCounters(t *testing.T) {
	rb := NewRetryBudget()

	// Add some retries
	rb.IncrementRetry(GateTestGen)
	rb.IncrementRetry(GateTestGen)
	rb.IncrementRetry(GateImpl)

	// Reset
	rb.ResetRetryCounters()

	if rb.TestGenRetries != 0 {
		t.Errorf("Expected TestGenRetries = 0 after reset, got %d", rb.TestGenRetries)
	}

	if rb.ImplRetries != 0 {
		t.Errorf("Expected ImplRetries = 0 after reset, got %d", rb.ImplRetries)
	}

	// Should be able to retry again
	if !rb.CanRetry(GateTestGen) {
		t.Error("Expected CanRetry(GateTestGen) = true after reset")
	}

	if !rb.CanRetry(GateImpl) {
		t.Error("Expected CanRetry(GateImpl) = true after reset")
	}
}

func TestResetGateRetry(t *testing.T) {
	rb := NewRetryBudget()

	// Add retries to both gates
	rb.IncrementRetry(GateTestGen)
	rb.IncrementRetry(GateTestGen)
	rb.IncrementRetry(GateImpl)
	rb.IncrementRetry(GateImpl)

	// Reset only TestGen
	rb.ResetGateRetry(GateTestGen)

	if rb.TestGenRetries != 0 {
		t.Errorf("Expected TestGenRetries = 0 after reset, got %d", rb.TestGenRetries)
	}

	if rb.ImplRetries != 2 {
		t.Errorf("Expected ImplRetries = 2 (unchanged), got %d", rb.ImplRetries)
	}
}

func TestRemainingRetries(t *testing.T) {
	rb := NewRetryBudget()

	// Initially should have 2 remaining
	remaining := rb.RemainingRetries(GateTestGen)
	if remaining != 2 {
		t.Errorf("Expected 2 remaining retries initially, got %d", remaining)
	}

	// After one retry
	rb.IncrementRetry(GateTestGen)
	remaining = rb.RemainingRetries(GateTestGen)
	if remaining != 1 {
		t.Errorf("Expected 1 remaining retry, got %d", remaining)
	}

	// After two retries
	rb.IncrementRetry(GateTestGen)
	remaining = rb.RemainingRetries(GateTestGen)
	if remaining != 0 {
		t.Errorf("Expected 0 remaining retries, got %d", remaining)
	}
}

func TestRemainingRetries_InvalidGate(t *testing.T) {
	rb := NewRetryBudget()

	invalidGate := GateType("invalid")
	remaining := rb.RemainingRetries(invalidGate)

	if remaining != 0 {
		t.Errorf("Expected 0 for invalid gate, got %d", remaining)
	}
}

func TestIsExhausted(t *testing.T) {
	rb := NewRetryBudget()

	// Initially not exhausted
	if rb.IsExhausted(GateTestGen) {
		t.Error("Expected IsExhausted = false initially")
	}

	// Exhaust retries
	rb.IncrementRetry(GateTestGen)
	rb.IncrementRetry(GateTestGen)

	if !rb.IsExhausted(GateTestGen) {
		t.Error("Expected IsExhausted = true after max retries")
	}

	// Other gate should not be exhausted
	if rb.IsExhausted(GateImpl) {
		t.Error("Expected IsExhausted(GateImpl) = false when only TestGen is exhausted")
	}
}

func TestTotalRetries(t *testing.T) {
	rb := NewRetryBudget()

	// Initially zero
	total := rb.TotalRetries()
	if total != 0 {
		t.Errorf("Expected total = 0 initially, got %d", total)
	}

	// Add retries to both gates
	rb.IncrementRetry(GateTestGen)
	rb.IncrementRetry(GateTestGen)
	rb.IncrementRetry(GateImpl)

	total = rb.TotalRetries()
	if total != 3 {
		t.Errorf("Expected total = 3, got %d", total)
	}
}

func TestRetryWorkflow_Scenario(t *testing.T) {
	rb := NewRetryBudget()

	// Test TestGen gate exhaustion
	testGenExhaustionScenario(t, rb)

	// Test Impl gate availability
	testImplGateAvailability(t, rb)

	// Test total retry count
	if rb.TotalRetries() != 3 {
		t.Errorf("Expected total retries = 3, got %d", rb.TotalRetries())
	}

	// Test reset functionality
	testResetFunctionality(t, rb)
}

// testGenExhaustionScenario tests exhausting TestGen retries
func testGenExhaustionScenario(t *testing.T, rb *RetryBudget) {
	// First attempt at test gen - fails
	if !rb.CanRetry(GateTestGen) {
		t.Fatal("Should be able to attempt TestGen")
	}

	// Retry 1
	rb.IncrementRetry(GateTestGen)
	if rb.RemainingRetries(GateTestGen) != 1 {
		t.Errorf("Expected 1 retry remaining, got %d", rb.RemainingRetries(GateTestGen))
	}

	// Retry 2
	rb.IncrementRetry(GateTestGen)
	if rb.RemainingRetries(GateTestGen) != 0 {
		t.Errorf("Expected 0 retries remaining, got %d", rb.RemainingRetries(GateTestGen))
	}

	// Cannot retry anymore
	if rb.CanRetry(GateTestGen) {
		t.Error("Should not be able to retry after exhausting budget")
	}
}

// testImplGateAvailability tests that Impl gate remains available
func testImplGateAvailability(t *testing.T, rb *RetryBudget) {
	// Impl gate should still be available
	if !rb.CanRetry(GateImpl) {
		t.Error("Impl gate should still be available")
	}

	// Try impl gate
	rb.IncrementRetry(GateImpl)
	if rb.GetRetryCount(GateImpl) != 1 {
		t.Errorf("Expected Impl retry count = 1, got %d", rb.GetRetryCount(GateImpl))
	}
}

// testResetFunctionality tests reset behavior
func testResetFunctionality(t *testing.T, rb *RetryBudget) {
	// Reset and start fresh
	rb.ResetRetryCounters()
	if rb.TotalRetries() != 0 {
		t.Errorf("Expected total retries = 0 after reset, got %d", rb.TotalRetries())
	}

	if !rb.CanRetry(GateTestGen) || !rb.CanRetry(GateImpl) {
		t.Error("Should be able to retry both gates after reset")
	}
}

func TestCustomMaxRetries(t *testing.T) {
	rb := NewRetryBudgetWithMax(5)

	// Should allow 5 retries
	for i := 0; i < 5; i++ {
		if !rb.CanRetry(GateTestGen) {
			t.Fatalf("Should be able to retry at attempt %d", i)
		}
		rb.IncrementRetry(GateTestGen)
	}

	// 6th attempt should fail
	if rb.CanRetry(GateTestGen) {
		t.Error("Should not be able to retry after 5 attempts")
	}

	if rb.GetRetryCount(GateTestGen) != 5 {
		t.Errorf("Expected retry count = 5, got %d", rb.GetRetryCount(GateTestGen))
	}
}
