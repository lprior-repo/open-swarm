// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package planner

// ParsedTask represents a task parsed from user input
type ParsedTask struct {
	Title       string
	Description string
	Priority    int
	DependsOn   []int // Indices of tasks this depends on
}
