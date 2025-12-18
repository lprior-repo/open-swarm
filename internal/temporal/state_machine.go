package temporal

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/log"
)

// StateTransition represents a transition rule in the state machine.
// It defines what state to move to based on current state and gate result.
type StateTransition struct {
	FromState   WorkflowState
	GatePassed  bool
	ToState     WorkflowState
	Description string
}

// StateMachine manages workflow state transitions with retry semantics.
type StateMachine struct {
	logger              log.Logger
	currentState        WorkflowState
	maxRetries          int
	maxFixAttempts      int
	currentRetryCount   int
	currentFixAttempt   int
	stateRetryCount     map[WorkflowState]int
	stateFixAttempts    map[WorkflowState]int
	isRegenerationMode  bool
}

// NewStateMachine creates a new state machine, initializing at gate 0 (Bootstrap).
func NewStateMachine(logger log.Logger, maxRetries, maxFixAttempts int) *StateMachine {
	return &StateMachine{
		logger:             logger,
		currentState:       StateBootstrap, // Initialize at gate 0
		maxRetries:         maxRetries,
		maxFixAttempts:     maxFixAttempts,
		currentRetryCount:  0,
		currentFixAttempt:  0,
		stateRetryCount:    make(map[WorkflowState]int),
		stateFixAttempts:   make(map[WorkflowState]int),
		isRegenerationMode: false,
	}
}

// CurrentState returns the current workflow state.
func (sm *StateMachine) CurrentState() WorkflowState {
	return sm.currentState
}

// IsTerminal returns true if the state machine has reached a terminal state.
func (sm *StateMachine) IsTerminal() bool {
	return sm.currentState == StateComplete || sm.currentState == StateFailed
}

// TransitionResult holds information about a state transition.
type TransitionResult struct {
	NextState          WorkflowState
	ShouldRetry        bool
	ShouldRegenerate   bool
	TerminalState      bool
	Reason             string
}

// Transition moves the state machine to the next state based on gate result.
// - On gate success: advance to next state
// - On gate failure: evaluate retry policy (stay in state or move back for retry)
// - On max retries: move to failed state
func (sm *StateMachine) Transition(gatePassed bool, gateResult *GateResult) TransitionResult {
	result := TransitionResult{
		NextState:     sm.currentState,
		ShouldRetry:   false,
		TerminalState: false,
		Reason:        "no transition",
	}

	// Define state transitions: what to do on success or failure
	transitions := sm.defineTransitions()

	// Find the appropriate transition
	for _, transition := range transitions {
		if transition.FromState == sm.currentState && transition.GatePassed == gatePassed {
			sm.logger.Info("State Transition",
				"from", sm.currentState,
				"to", transition.ToState,
				"gatePassed", gatePassed,
				"reason", transition.Description)

			result.NextState = transition.ToState
			result.Reason = transition.Description

			if gatePassed {
				// Success: move to next state and reset retry counters
				sm.currentState = transition.ToState
				sm.stateRetryCount[transition.ToState] = 0
				sm.stateFixAttempts[transition.ToState] = 0
				sm.currentFixAttempt = 0

				// If we reached terminal state, mark it
				if sm.IsTerminal() {
					result.TerminalState = true
				}
			} else {
				// Failure: check if this is a terminal failure or a retryable failure
				if transition.ToState == StateFailed {
					// Fatal failure - move directly to failed state
					sm.currentState = StateFailed
					result.TerminalState = true
					sm.logger.Warn("Fatal gate failure - moving to failed state",
						"fromState", transition.FromState,
						"reason", transition.Description)
				} else if transition.ToState == sm.currentState {
					// Retryable failure - same state means retry/regenerate
					result = sm.handleFailure(transition, gateResult)
				} else {
					// Move to specified state (usually for going back to earlier gates)
					sm.currentState = transition.ToState
					sm.stateRetryCount[transition.ToState] = 0
					sm.stateFixAttempts[transition.ToState] = 0
					sm.currentFixAttempt = 0
				}
			}
			break
		}
	}

	return result
}

// handleFailure determines retry strategy after a gate failure.
// - Returns to previous gate for retry (up to max retries)
// - On max retries: moves to failed state
func (sm *StateMachine) handleFailure(transition StateTransition, gateResult *GateResult) TransitionResult {
	result := TransitionResult{
		NextState:         sm.currentState,
		ShouldRetry:       false,
		ShouldRegenerate:  false,
		TerminalState:     false,
		Reason:            "stay in current state for retry",
	}

	// Increment fix attempt counter
	sm.stateFixAttempts[sm.currentState]++
	sm.currentFixAttempt = sm.stateFixAttempts[sm.currentState]

	// Check if we should continue with targeted fixes
	if sm.currentFixAttempt < sm.maxFixAttempts {
		sm.logger.Info("Gate failed - attempting targeted fix",
			"state", sm.currentState,
			"fixAttempt", sm.currentFixAttempt,
			"maxFixAttempts", sm.maxFixAttempts)
		result.ShouldRetry = true
		return result
	}

	// Max fix attempts exhausted - check if we can regenerate
	sm.logger.Info("Max fix attempts reached - checking if regeneration allowed",
		"state", sm.currentState,
		"fixAttemptsUsed", sm.currentFixAttempt,
		"currentRetryCount", sm.currentRetryCount,
		"maxRetries", sm.maxRetries)

	// Check if we can still perform a regeneration
	if sm.currentRetryCount < sm.maxRetries {
		// Increment regeneration count and regenerate
		sm.currentRetryCount++
		sm.stateRetryCount[sm.currentState]++

		// Reset fix attempts for next regeneration cycle
		sm.stateFixAttempts[sm.currentState] = 0
		sm.currentFixAttempt = 0

		// Regenerate from last stable gate
		result.ShouldRegenerate = true
		result.NextState = sm.getRegenerationStartPoint()
		sm.logger.Info("Starting full regeneration cycle",
			"currentState", sm.currentState,
			"regenerationStartPoint", result.NextState,
			"retryCount", sm.currentRetryCount)
		return result
	}

	// Max retries exceeded - move to failed state
	sm.logger.Warn("Max retries exceeded - moving to failed state",
		"state", sm.currentState,
		"retriesUsed", sm.currentRetryCount)
	sm.currentState = StateFailed
	result.NextState = StateFailed
	result.TerminalState = true
	result.Reason = fmt.Sprintf("max retries exceeded (%d attempts)", sm.currentRetryCount)

	return result
}

// getRegenerationStartPoint determines where to start regeneration after a failure.
// For GenTest, LintTest, VerifyRED: restart the generation phase
// For GenImpl: restart from GenImpl
// For VerifyGREEN, MultiReview: restart from GenImpl (full regeneration)
func (sm *StateMachine) getRegenerationStartPoint() WorkflowState {
	switch sm.currentState {
	case StateGenTest, StateLintTest, StateVerifyRED:
		// Test generation phase failure - restart from GenTest
		return StateGenTest
	case StateGenImpl:
		// Implementation generation failure - restart GenImpl
		return StateGenImpl
	case StateVerifyGREEN, StateMultiReview:
		// Implementation validation failure - restart from GenImpl
		return StateGenImpl
	default:
		// Default to current state
		return sm.currentState
	}
}

// defineTransitions defines all valid state transitions.
// This is the core of the DAG navigation logic.
func (sm *StateMachine) defineTransitions() []StateTransition {
	return []StateTransition{
		// Bootstrap state (gate 0)
		{StateBootstrap, true, StateGenTest, "bootstrap successful, move to test generation"},
		{StateBootstrap, false, StateFailed, "bootstrap failed - cannot continue"},

		// Test Generation Phase (gates 1-3)
		{StateGenTest, true, StateLintTest, "tests generated successfully"},
		{StateGenTest, false, StateGenTest, "test generation failed, retry"},

		{StateLintTest, true, StateVerifyRED, "tests linted successfully"},
		{StateLintTest, false, StateGenTest, "lint failed, regenerate tests"},

		{StateVerifyRED, true, StateGenImpl, "tests verified to fail, move to implementation"},
		{StateVerifyRED, false, StateGenTest, "tests did not fail as expected, regenerate"},

		// Implementation Phase (gates 4-6)
		{StateGenImpl, true, StateVerifyGREEN, "implementation generated successfully"},
		{StateGenImpl, false, StateGenImpl, "implementation generation failed, retry"},

		{StateVerifyGREEN, true, StateMultiReview, "tests pass, move to review"},
		{StateVerifyGREEN, false, StateVerifyGREEN, "tests failed, retry fix"},

		{StateMultiReview, true, StateCommit, "review approved, move to commit"},
		{StateMultiReview, false, StateMultiReview, "review feedback, retry fix"},

		// Commit state (gate 7)
		{StateCommit, true, StateComplete, "changes committed successfully"},
		{StateCommit, false, StateFailed, "commit failed - cannot recover"},

		// Terminal states
		{StateComplete, false, StateComplete, "workflow already complete"},
		{StateFailed, false, StateFailed, "workflow already failed"},
	}
}

// RetryInfo returns diagnostic information about current retry state.
type RetryInfo struct {
	CurrentState       WorkflowState
	CurrentRetry       int
	MaxRetries         int
	CurrentFix         int
	MaxFixAttempts     int
	StateRetries       map[WorkflowState]int
	StateFixAttempts   map[WorkflowState]int
	IsRegenerationMode bool
}

// GetRetryInfo returns current retry and state information.
func (sm *StateMachine) GetRetryInfo() RetryInfo {
	return RetryInfo{
		CurrentState:       sm.currentState,
		CurrentRetry:       sm.currentRetryCount,
		MaxRetries:         sm.maxRetries,
		CurrentFix:         sm.currentFixAttempt,
		MaxFixAttempts:     sm.maxFixAttempts,
		StateRetries:       sm.stateRetryCount,
		StateFixAttempts:   sm.stateFixAttempts,
		IsRegenerationMode: sm.isRegenerationMode,
	}
}

// RecordState logs the state machine state at a point in time.
type StateSnapshot struct {
	Timestamp      time.Time
	State          WorkflowState
	RetryCount     int
	FixAttempt     int
	GateResult     *GateResult
	Transition     TransitionResult
}

// LogStateChange logs a state machine transition for debugging.
func (sm *StateMachine) LogStateChange(snapshot StateSnapshot) {
	sm.logger.Info("State Machine Transition",
		"timestamp", snapshot.Timestamp,
		"fromState", snapshot.State,
		"toState", snapshot.Transition.NextState,
		"retryCount", snapshot.RetryCount,
		"fixAttempt", snapshot.FixAttempt,
		"shouldRetry", snapshot.Transition.ShouldRetry,
		"terminal", snapshot.Transition.TerminalState,
		"reason", snapshot.Transition.Reason)
}

// StateMetrics provides metrics about state machine execution.
type StateMetrics struct {
	TotalStates      int
	StatesVisited    []WorkflowState
	TotalRetries     int
	TotalFixAttempts int
	StateVisitCounts map[WorkflowState]int
	AverageRetryTime time.Duration
}

// GetMetrics returns execution metrics from the state machine.
func (sm *StateMachine) GetMetrics() StateMetrics {
	visitCounts := make(map[WorkflowState]int)
	visited := []WorkflowState{}

	for state := range sm.stateRetryCount {
		visitCounts[state]++
		visited = append(visited, state)
	}

	return StateMetrics{
		TotalStates:      len(visited),
		StatesVisited:    visited,
		TotalRetries:     sm.currentRetryCount,
		TotalFixAttempts: sm.currentFixAttempt,
		StateVisitCounts: visitCounts,
	}
}

// CanTransitionTo checks if a transition to a given state is valid from current state.
func (sm *StateMachine) CanTransitionTo(targetState WorkflowState) bool {
	transitions := sm.defineTransitions()
	for _, transition := range transitions {
		if transition.FromState == sm.currentState && transition.ToState == targetState {
			return true
		}
	}
	return false
}

// Reset resets the state machine to initial state (for testing).
func (sm *StateMachine) Reset() {
	sm.currentState = StateBootstrap
	sm.currentRetryCount = 0
	sm.currentFixAttempt = 0
	sm.stateRetryCount = make(map[WorkflowState]int)
	sm.stateFixAttempts = make(map[WorkflowState]int)
	sm.isRegenerationMode = false
}
