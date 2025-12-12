// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package temporal

// EnhancedTCRInput defines input for the Enhanced 6-Gate TCR workflow
type EnhancedTCRInput struct {
	CellID             string
	Branch             string
	TaskID             string
	Description        string
	AcceptanceCriteria string
	ReviewersCount     int
}
