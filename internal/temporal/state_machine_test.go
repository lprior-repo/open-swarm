package temporal

import (
	"testing"
	"time"
)

// mockLogger implements a simple logger for testing.
type mockLogger struct{}

func (m *mockLogger) Debug(msg string, keyvals ...interface{})   {}
func (m *mockLogger) Info(msg string, keyvals ...interface{})    {}
func (m *mockLogger) Warn(msg string, keyvals ...interface{})    {}
func (m *mockLogger) Error(msg string, keyvals ...interface{})   {}

// TestNewStateMachine tests state machine initialization.
func TestNewStateMachine(t *testing.T) {
	logger := &mockLogger{}
	sm := NewStateMachine(logger, 2, 5)

	if sm.CurrentState() != StateBootstrap {
		t.Errorf("expected initial state StateBootstrap, got %s", sm.CurrentState())
	}

	if sm.IsTerminal() {
		t.Error("initial state should not be terminal")
	}
}

// TestSuccessfulTransitions tests forward transitions on gate success.
func TestSuccessfulTransitions(t *testing.T) {
	logger := &mockLogger{}
	sm := NewStateMachine(logger, 2, 5)

	testCases := []struct {
		currentState WorkflowState
		nextState    WorkflowState
		description  string
	}{
		{StateBootstrap, StateGenTest, "bootstrap to gentest"},
		{StateGenTest, StateLintTest, "gentest to linttest"},
		{StateLintTest, StateVerifyRED, "linttest to verifyred"},
		{StateVerifyRED, StateGenImpl, "verifyred to genimpl"},
		{StateGenImpl, StateVerifyGREEN, "genimpl to verifygreen"},
		{StateVerifyGREEN, StateMultiReview, "verifygreen to multireview"},
		{StateMultiReview, StateCommit, "multireview to commit"},
		{StateCommit, StateComplete, "commit to complete"},
	}

	for _, tc := range testCases {
		sm.Reset()
		sm.currentState = tc.currentState

		// Manually transition to the expected state for testing
		result := sm.Transition(true, &GateResult{
			GateName: string(tc.currentState),
			Passed:   true,
		})

		if result.NextState != tc.nextState {
			t.Errorf("transition %s: expected %s, got %s", tc.description, tc.nextState, result.NextState)
		}

		if result.TerminalState && tc.nextState != StateComplete && tc.nextState != StateFailed {
			t.Errorf("transition %s: unexpected terminal state", tc.description)
		}
	}
}

// TestRetryOnFailure tests that gate failures stay in current state.
func TestRetryOnFailure(t *testing.T) {
	logger := &mockLogger{}
	sm := NewStateMachine(logger, 2, 5)

	testCases := []WorkflowState{
		StateGenTest,
		StateGenImpl,
		StateVerifyGREEN,
		StateMultiReview,
	}

	for _, state := range testCases {
		sm.Reset()
		sm.currentState = state

		result := sm.Transition(false, &GateResult{
			GateName: string(state),
			Passed:   false,
			Error:    "test error",
		})

		// Should stay in same state (retry)
		if result.NextState != state {
			t.Errorf("state %s failure: expected to stay in %s, got %s", state, state, result.NextState)
		}

		// Should signal retry needed
		if !result.ShouldRetry {
			t.Errorf("state %s: expected ShouldRetry=true", state)
		}

		// Should still be in original state
		if sm.CurrentState() != state {
			t.Errorf("state %s: current state should remain %s, got %s", state, state, sm.CurrentState())
		}
	}
}

// TestMaxFixAttemptsExceeded tests transition to regeneration after max fixes.
func TestMaxFixAttemptsExceeded(t *testing.T) {
	logger := &mockLogger{}
	sm := NewStateMachine(logger, 2, 3) // 3 max fix attempts

	sm.currentState = StateVerifyGREEN

	// Simulate multiple failures
	for i := 0; i < 3; i++ {
		result := sm.Transition(false, &GateResult{
			GateName: "VerifyGREEN",
			Passed:   false,
			Error:    "tests failed",
		})

		// First 2 should retry, third should regenerate
		if i < 2 {
			if !result.ShouldRetry {
				t.Errorf("attempt %d: expected retry", i+1)
			}
		} else {
			if !result.ShouldRegenerate {
				t.Errorf("attempt %d: expected regenerate", i+1)
			}
			if result.NextState != StateGenImpl {
				t.Errorf("expected regeneration start at GenImpl, got %s", result.NextState)
			}
		}
	}
}

// TestMaxRetriesExceeded tests failure after all retries exhausted.
func TestMaxRetriesExceeded(t *testing.T) {
	logger := &mockLogger{}
	sm := NewStateMachine(logger, 1, 2) // 1 retry, 2 fix attempts

	sm.currentState = StateVerifyGREEN

	// First failure: should retry
	result := sm.Transition(false, &GateResult{GateName: "VerifyGREEN", Passed: false})
	if !result.ShouldRetry {
		t.Error("first failure: expected retry")
	}

	// Second failure: should trigger regeneration (max fix attempts reached)
	result = sm.Transition(false, &GateResult{GateName: "VerifyGREEN", Passed: false})
	if !result.ShouldRegenerate {
		t.Errorf("second failure: expected regenerate, got retry=%v regenerate=%v",
			result.ShouldRetry, result.ShouldRegenerate)
	}

	// Third cycle: Now we're in the regeneration (which resets us to StateVerifyGREEN again)
	// The next two failures should again try to regenerate, but max retries exceeded
	// Reset fix attempts for new regeneration cycle (simulating the actual workflow)
	sm.stateFixAttempts[StateVerifyGREEN] = 0
	sm.currentFixAttempt = 0

	// First attempt of second regeneration cycle
	result = sm.Transition(false, &GateResult{GateName: "VerifyGREEN", Passed: false})
	if !result.ShouldRetry {
		t.Error("third failure: expected retry in new cycle")
	}

	// Second attempt of second regeneration cycle - max retries exceeded
	result = sm.Transition(false, &GateResult{GateName: "VerifyGREEN", Passed: false})
	// Should trigger regeneration again, but max retries already exceeded
	if result.NextState != StateFailed {
		t.Errorf("fourth failure: expected StateFailed, got %s", result.NextState)
	}
	if !result.TerminalState {
		t.Error("fourth failure: expected terminal state")
	}
}

// TestTerminalStates tests that terminal states don't transition.
func TestTerminalStates(t *testing.T) {
	logger := &mockLogger{}
	sm := NewStateMachine(logger, 2, 5)

	terminalStates := []WorkflowState{StateComplete, StateFailed}

	for _, terminal := range terminalStates {
		sm.Reset()
		sm.currentState = terminal

		if !sm.IsTerminal() {
			t.Errorf("state %s should be terminal", terminal)
		}

		// Attempt transition from terminal state (should stay)
		result := sm.Transition(true, &GateResult{
			GateName: string(terminal),
			Passed:   true,
		})

		if result.NextState != terminal {
			t.Errorf("terminal state %s should not transition, got %s", terminal, result.NextState)
		}
	}
}

// TestBootstrapFailureIsFatal tests bootstrap failure leads to failed state.
func TestBootstrapFailureIsFatal(t *testing.T) {
	logger := &mockLogger{}
	sm := NewStateMachine(logger, 2, 5)

	// Bootstrap should be in StateBootstrap initially
	if sm.CurrentState() != StateBootstrap {
		t.Fatalf("expected initial state StateBootstrap, got %s", sm.CurrentState())
	}

	// Bootstrap failure should directly go to failed state (no retry allowed)
	result := sm.Transition(false, &GateResult{
		GateName: "Bootstrap",
		Passed:   false,
		Error:    "bootstrap failed",
	})

	// Bootstrap has no retry - failure is fatal
	if result.NextState != StateFailed {
		t.Errorf("bootstrap failure should lead to StateFailed, got %s", result.NextState)
	}

	if !result.TerminalState {
		t.Error("bootstrap failure should be terminal")
	}
}

// TestCommitFailureIsFatal tests commit failure leads to failed state.
func TestCommitFailureIsFatal(t *testing.T) {
	logger := &mockLogger{}
	sm := NewStateMachine(logger, 2, 5)

	sm.currentState = StateCommit

	// Commit failure should directly go to failed state (no retry allowed)
	result := sm.Transition(false, &GateResult{
		GateName: "Commit",
		Passed:   false,
		Error:    "commit failed",
	})

	if result.NextState != StateFailed {
		t.Errorf("commit failure should lead to StateFailed, got %s", result.NextState)
	}

	if !result.TerminalState {
		t.Error("commit failure should be terminal")
	}
}

// TestRegenerationStartPoints tests correct regeneration points.
func TestRegenerationStartPoints(t *testing.T) {
	logger := &mockLogger{}
	sm := NewStateMachine(logger, 2, 5)

	testCases := []struct {
		failureState WorkflowState
		restartAt    WorkflowState
		description  string
	}{
		{StateGenTest, StateGenTest, "test gen failure restarts at test gen"},
		{StateLintTest, StateGenTest, "lint failure restarts at test gen"},
		{StateVerifyRED, StateGenTest, "verify red failure restarts at test gen"},
		{StateGenImpl, StateGenImpl, "impl gen failure restarts at impl gen"},
		{StateVerifyGREEN, StateGenImpl, "verify green failure restarts at impl gen"},
		{StateMultiReview, StateGenImpl, "review failure restarts at impl gen"},
	}

	for _, tc := range testCases {
		sm.Reset()
		sm.currentState = tc.failureState

		// Simulate failures until we reach regeneration decision
		var result TransitionResult
		for i := 0; i < 10; i++ {
			result = sm.Transition(false, &GateResult{
				GateName: string(tc.failureState),
				Passed:   false,
			})
			// Stop once we get regeneration signal
			if result.ShouldRegenerate {
				break
			}
		}

		// The returned NextState should be the regeneration start point
		if result.NextState != tc.restartAt {
			t.Errorf("%s: expected restart at %s, got %s", tc.description, tc.restartAt, result.NextState)
		}

		// Also verify getRegenerationStartPoint() returns the right value
		// when in the current failure state
		restartPoint := sm.getRegenerationStartPoint()
		if restartPoint != tc.restartAt {
			t.Errorf("%s (direct check): expected restart at %s, got %s", tc.description, tc.restartAt, restartPoint)
		}
	}
}

// TestRetryInfo tests retry information retrieval.
func TestRetryInfo(t *testing.T) {
	logger := &mockLogger{}
	sm := NewStateMachine(logger, 2, 5)

	sm.currentRetryCount = 1
	sm.currentFixAttempt = 3
	sm.currentState = StateVerifyGREEN

	info := sm.GetRetryInfo()

	if info.CurrentState != StateVerifyGREEN {
		t.Errorf("expected state %s, got %s", StateVerifyGREEN, info.CurrentState)
	}

	if info.CurrentRetry != 1 {
		t.Errorf("expected retry count 1, got %d", info.CurrentRetry)
	}

	if info.CurrentFix != 3 {
		t.Errorf("expected fix attempt 3, got %d", info.CurrentFix)
	}
}

// TestCanTransitionTo tests valid transition checks.
func TestCanTransitionTo(t *testing.T) {
	logger := &mockLogger{}
	sm := NewStateMachine(logger, 2, 5)

	testCases := []struct {
		from     WorkflowState
		to       WorkflowState
		expected bool
	}{
		{StateBootstrap, StateGenTest, true},
		{StateBootstrap, StateGenImpl, false},
		{StateGenTest, StateLintTest, true},
		{StateGenTest, StateVerifyRED, false},
		{StateCommit, StateComplete, true},
		{StateComplete, StateGenTest, false},
	}

	for _, tc := range testCases {
		sm.Reset()
		sm.currentState = tc.from

		can := sm.CanTransitionTo(tc.to)
		if can != tc.expected {
			t.Errorf("from %s to %s: expected %v, got %v", tc.from, tc.to, tc.expected, can)
		}
	}
}

// TestStateSnapshot tests snapshot creation and logging.
func TestStateSnapshot(t *testing.T) {
	logger := &mockLogger{}
	sm := NewStateMachine(logger, 2, 5)

	snapshot := StateSnapshot{
		Timestamp: time.Now(),
		State:     StateGenTest,
		RetryCount: 1,
		FixAttempt: 2,
		GateResult: &GateResult{
			GateName: "GenTest",
			Passed:   false,
		},
		Transition: TransitionResult{
			NextState:    StateGenTest,
			ShouldRetry:  true,
			TerminalState: false,
			Reason:        "retry",
		},
	}

	// Should not panic
	sm.LogStateChange(snapshot)
}

// TestStateMetrics tests metrics retrieval.
func TestStateMetrics(t *testing.T) {
	logger := &mockLogger{}
	sm := NewStateMachine(logger, 2, 5)

	// Simulate some transitions
	sm.currentState = StateGenTest
	sm.stateRetryCount[StateGenTest] = 1
	sm.stateRetryCount[StateLintTest] = 1

	metrics := sm.GetMetrics()

	if metrics.TotalStates != 2 {
		t.Errorf("expected 2 states visited, got %d", metrics.TotalStates)
	}

	if metrics.TotalRetries != 0 {
		t.Errorf("expected 0 total retries, got %d", metrics.TotalRetries)
	}
}

// TestReset tests state machine reset.
func TestReset(t *testing.T) {
	logger := &mockLogger{}
	sm := NewStateMachine(logger, 2, 5)

	// Modify state machine
	sm.currentState = StateVerifyGREEN
	sm.currentRetryCount = 5
	sm.currentFixAttempt = 3

	// Reset
	sm.Reset()

	if sm.CurrentState() != StateBootstrap {
		t.Errorf("after reset, expected StateBootstrap, got %s", sm.CurrentState())
	}

	if sm.currentRetryCount != 0 {
		t.Errorf("after reset, expected 0 retries, got %d", sm.currentRetryCount)
	}

	if sm.currentFixAttempt != 0 {
		t.Errorf("after reset, expected 0 fix attempts, got %d", sm.currentFixAttempt)
	}
}

// TestCompleteSuccessPath tests a complete successful workflow.
func TestCompleteSuccessPath(t *testing.T) {
	logger := &mockLogger{}
	sm := NewStateMachine(logger, 2, 5)

	successPath := []WorkflowState{
		StateBootstrap,
		StateGenTest,
		StateLintTest,
		StateVerifyRED,
		StateGenImpl,
		StateVerifyGREEN,
		StateMultiReview,
		StateCommit,
		StateComplete,
	}

	for i, expectedState := range successPath {
		if sm.CurrentState() != expectedState {
			t.Errorf("step %d: expected state %s, got %s", i, expectedState, sm.CurrentState())
		}

		if i < len(successPath)-1 {
			result := sm.Transition(true, &GateResult{
				GateName: string(expectedState),
				Passed:   true,
			})

			if result.TerminalState && expectedState != StateCommit {
				t.Errorf("step %d: unexpected terminal state", i)
			}
		}
	}

	if !sm.IsTerminal() {
		t.Error("final state should be terminal")
	}

	if sm.CurrentState() != StateComplete {
		t.Errorf("expected final state StateComplete, got %s", sm.CurrentState())
	}
}

// TestRetryLoopGate2ToGate1 tests that Gate 2 (LintTest) failure returns to Gate 1 (GenTest).
func TestRetryLoopGate2ToGate1(t *testing.T) {
	logger := &mockLogger{}
	sm := NewStateMachine(logger, 2, 5)

	// Start from StateBootstrap and advance to StateLintTest (Gate 2)
	sm.Transition(true, &GateResult{GateName: "Bootstrap", Passed: true})        // → StateGenTest
	sm.Transition(true, &GateResult{GateName: "GenTest", Passed: true})           // → StateLintTest

	if sm.CurrentState() != StateLintTest {
		t.Fatalf("expected to reach StateLintTest, got %s", sm.CurrentState())
	}

	// Simulate Gate 2 failure
	result := sm.Transition(false, &GateResult{
		GateName: "LintTest",
		Passed:   false,
		Error:    "lint check failed",
	})

	// Should go back to Gate 1 (StateGenTest)
	if result.NextState != StateGenTest {
		t.Errorf("Gate 2 failure: expected to go back to Gate 1 (StateGenTest), got %s", result.NextState)
	}

	if sm.CurrentState() != StateGenTest {
		t.Errorf("Current state should be StateGenTest, got %s", sm.CurrentState())
	}
}

// TestRetryLoopGate3ToGate1 tests that Gate 3 (VerifyRED) failure returns to Gate 1 (GenTest).
func TestRetryLoopGate3ToGate1(t *testing.T) {
	logger := &mockLogger{}
	sm := NewStateMachine(logger, 2, 5)

	// Advance to StateVerifyRED (Gate 3)
	sm.Transition(true, &GateResult{GateName: "Bootstrap", Passed: true})
	sm.Transition(true, &GateResult{GateName: "GenTest", Passed: true})
	sm.Transition(true, &GateResult{GateName: "LintTest", Passed: true})

	if sm.CurrentState() != StateVerifyRED {
		t.Fatalf("expected to reach StateVerifyRED, got %s", sm.CurrentState())
	}

	// Simulate Gate 3 failure
	result := sm.Transition(false, &GateResult{
		GateName: "VerifyRED",
		Passed:   false,
		Error:    "tests did not fail as expected",
	})

	// Should go back to Gate 1 (StateGenTest)
	if result.NextState != StateGenTest {
		t.Errorf("Gate 3 failure: expected to go back to Gate 1 (StateGenTest), got %s", result.NextState)
	}

	if sm.CurrentState() != StateGenTest {
		t.Errorf("Current state should be StateGenTest, got %s", sm.CurrentState())
	}
}

// TestRetryLoopGate5BackToGate4 tests retry loop for Gate 5 (VerifyGREEN) with regeneration to Gate 4 (GenImpl).
func TestRetryLoopGate5BackToGate4(t *testing.T) {
	logger := &mockLogger{}
	sm := NewStateMachine(logger, 2, 2) // 2 max retries, 2 max fix attempts

	// Advance to StateVerifyGREEN (Gate 5)
	sm.Transition(true, &GateResult{GateName: "Bootstrap", Passed: true})
	sm.Transition(true, &GateResult{GateName: "GenTest", Passed: true})
	sm.Transition(true, &GateResult{GateName: "LintTest", Passed: true})
	sm.Transition(true, &GateResult{GateName: "VerifyRED", Passed: true})
	sm.Transition(true, &GateResult{GateName: "GenImpl", Passed: true})

	if sm.CurrentState() != StateVerifyGREEN {
		t.Fatalf("expected to reach StateVerifyGREEN, got %s", sm.CurrentState())
	}

	// First failure - should retry in same state
	result := sm.Transition(false, &GateResult{
		GateName: "VerifyGREEN",
		Passed:   false,
		Error:    "tests failed",
	})

	if !result.ShouldRetry {
		t.Error("first failure: expected ShouldRetry=true")
	}

	// Second failure - should trigger regeneration back to Gate 4 (GenImpl)
	result = sm.Transition(false, &GateResult{
		GateName: "VerifyGREEN",
		Passed:   false,
		Error:    "tests failed",
	})

	if !result.ShouldRegenerate {
		t.Error("second failure: expected ShouldRegenerate=true")
	}

	if result.NextState != StateGenImpl {
		t.Errorf("expected regeneration to StateGenImpl (Gate 4), got %s", result.NextState)
	}
}

// TestRetryLoopGate6BackToGate4 tests retry loop for Gate 6 (MultiReview) with regeneration to Gate 4 (GenImpl).
func TestRetryLoopGate6BackToGate4(t *testing.T) {
	logger := &mockLogger{}
	sm := NewStateMachine(logger, 2, 2) // 2 max retries, 2 max fix attempts

	// Advance to StateMultiReview (Gate 6)
	sm.Transition(true, &GateResult{GateName: "Bootstrap", Passed: true})
	sm.Transition(true, &GateResult{GateName: "GenTest", Passed: true})
	sm.Transition(true, &GateResult{GateName: "LintTest", Passed: true})
	sm.Transition(true, &GateResult{GateName: "VerifyRED", Passed: true})
	sm.Transition(true, &GateResult{GateName: "GenImpl", Passed: true})
	sm.Transition(true, &GateResult{GateName: "VerifyGREEN", Passed: true})

	if sm.CurrentState() != StateMultiReview {
		t.Fatalf("expected to reach StateMultiReview, got %s", sm.CurrentState())
	}

	// First failure - should retry in same state
	result := sm.Transition(false, &GateResult{
		GateName: "MultiReview",
		Passed:   false,
		Error:    "review feedback",
	})

	if !result.ShouldRetry {
		t.Error("first failure: expected ShouldRetry=true")
	}

	// Second failure - should trigger regeneration back to Gate 4 (GenImpl)
	result = sm.Transition(false, &GateResult{
		GateName: "MultiReview",
		Passed:   false,
		Error:    "review feedback",
	})

	if !result.ShouldRegenerate {
		t.Error("second failure: expected ShouldRegenerate=true")
	}

	if result.NextState != StateGenImpl {
		t.Errorf("expected regeneration to StateGenImpl (Gate 4), got %s", result.NextState)
	}
}

// TestMaxRetryLimitEnforcement tests that max retry limits are enforced.
func TestMaxRetryLimitEnforcement(t *testing.T) {
	logger := &mockLogger{}
	sm := NewStateMachine(logger, 1, 1) // Only 1 retry and 1 fix attempt allowed

	// Advance to StateVerifyGREEN
	sm.Transition(true, &GateResult{GateName: "Bootstrap", Passed: true})
	sm.Transition(true, &GateResult{GateName: "GenTest", Passed: true})
	sm.Transition(true, &GateResult{GateName: "LintTest", Passed: true})
	sm.Transition(true, &GateResult{GateName: "VerifyRED", Passed: true})
	sm.Transition(true, &GateResult{GateName: "GenImpl", Passed: true})

	// First failure - should trigger regeneration immediately (max fix attempts = 1)
	result := sm.Transition(false, &GateResult{
		GateName: "VerifyGREEN",
		Passed:   false,
		Error:    "tests failed",
	})

	if !result.ShouldRegenerate {
		t.Error("first failure: expected regeneration after 1 fix attempt")
	}

	if result.NextState != StateGenImpl {
		t.Errorf("expected regeneration to StateGenImpl, got %s", result.NextState)
	}

	// Current retry count should be 1 after first regeneration
	if sm.currentRetryCount != 1 {
		t.Errorf("expected retry count 1, got %d", sm.currentRetryCount)
	}

	// Simulate regeneration cycle - reset counters as the workflow would
	sm.stateFixAttempts[StateVerifyGREEN] = 0
	sm.currentFixAttempt = 0

	// Second cycle, first failure - max retries are exhausted (1 retry allowed, already used 1)
	// The state machine should now go directly to failed state
	result = sm.Transition(false, &GateResult{
		GateName: "VerifyGREEN",
		Passed:   false,
		Error:    "tests failed",
	})

	// After 1 retry exhausted and max retries = 1, the next failure should fail
	if result.NextState != StateFailed {
		t.Errorf("max retries exceeded: expected StateFailed, got %s", result.NextState)
	}

	if !result.TerminalState {
		t.Error("max retries exceeded: expected terminal state")
	}
}
