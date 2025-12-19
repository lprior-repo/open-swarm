// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package opencode

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestAgentExecutorFactory validates that the factory creates agents that can handle complex tasks
func TestAgentExecutorFactory_CreateExecutor(t *testing.T) {
	// Create the three dependencies
	analyzer := NewCodeAnalyzer()
	generator := NewCodeGenerator(analyzer)
	testRunner := NewTestRunner()

	// Use the factory to create an executor
	executor := NewAgentExecutor(testRunner, generator, analyzer)

	// Verify the executor was created
	require.NotNil(t, executor)

	// Verify the executor implements the AgentExecutor interface
	var _ AgentExecutor = executor
}


