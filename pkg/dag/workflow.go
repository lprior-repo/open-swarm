package dag

import (
	"go.temporal.io/sdk/workflow"
)

// TddDagWorkflow implements the Test-Driven Development loop.
// It executes the DAG and waits for human signals upon failure,
// preserving state between retries to support checkpointing.
func TddDagWorkflow(ctx workflow.Context, input WorkflowInput) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting TDD DAG Workflow", "workflowID", input.WorkflowID)

	engine := NewEngine()
	attempt := 1
	var state *State // This variable will hold the state across retries.

	for {
		logger.Info("TDD Cycle Start", "attempt", attempt)

		// 1. Execute the DAG, passing in the previous state.
		// The state will be nil on the first attempt.
		var err error
		state, err = engine.Run(ctx, input.Tasks, state)

		if err == nil {
			logger.Info("TDD Cycle Succeeded!", "attempts", attempt)
			return nil
		}

		// 2. Failure Handling
		logger.Error("TDD Cycle Failed", "attempt", attempt, "error", err)

		// 3. Wait for 'FixApplied' signal
		logger.Info("Waiting for 'FixApplied' signal to retry...")
		var signalVal string
		workflow.GetSignalChannel(ctx, "FixApplied").Receive(ctx, &signalVal)

		logger.Info("Received FixApplied signal, will restart from the beginning.", "message", signalVal)
		attempt++
		// Reset state to nil to force a fresh run after a code fix has been applied.
		state = nil
	}
}
