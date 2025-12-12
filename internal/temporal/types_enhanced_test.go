// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package temporal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnhancedTCRInput_Fields(t *testing.T) {
	input := EnhancedTCRInput{
		CellID:             "test-cell-1",
		Branch:             "main",
		TaskID:             "task-123",
		Description:        "Add user validation",
		AcceptanceCriteria: "Must validate email format",
		ReviewersCount:     3,
	}

	assert.Equal(t, "test-cell-1", input.CellID)
	assert.Equal(t, "main", input.Branch)
	assert.Equal(t, "task-123", input.TaskID)
	assert.Equal(t, "Add user validation", input.Description)
	assert.Equal(t, "Must validate email format", input.AcceptanceCriteria)
	assert.Equal(t, 3, input.ReviewersCount)
}
