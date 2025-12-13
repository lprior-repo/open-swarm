// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package beads

import (
	"testing"
)

func TestGetTask(t *testing.T) {
	// Test reading a known task from the .beads/issues.jsonl
	task, err := GetTask("open-swarm-0lx")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}

	if task == nil {
		t.Fatal("GetTask returned nil task")
	}

	if task.ID != "open-swarm-0lx" {
		t.Errorf("expected task ID 'open-swarm-0lx', got %q", task.ID)
	}

	if task.Title != "Build conflict analyzer using Agent Mail reservations" {
		t.Errorf("unexpected title: %s", task.Title)
	}
}

func TestGetTaskNotFound(t *testing.T) {
	_, err := GetTask("nonexistent-task-id")
	if err == nil {
		t.Fatal("GetTask should return error for nonexistent task")
	}
}

func TestGetDependencies(t *testing.T) {
	// Test a task with dependencies
	deps, err := GetDependencies("open-swarm-64o.7")
	if err != nil {
		t.Fatalf("GetDependencies failed: %v", err)
	}

	// open-swarm-64o.7 should depend on open-swarm-64o and open-swarm-64o.6
	expectedDeps := map[string]bool{
		"open-swarm-64o":   true,
		"open-swarm-64o.6": true,
	}

	if len(deps) != len(expectedDeps) {
		t.Errorf("expected %d dependencies, got %d: %v", len(expectedDeps), len(deps), deps)
	}

	for _, dep := range deps {
		if !expectedDeps[dep] {
			t.Errorf("unexpected dependency: %s", dep)
		}
	}
}

func TestBuildDAG(t *testing.T) {
	// Test building a DAG from a single task
	dag, err := BuildDAG("open-swarm-04p")
	if err != nil {
		t.Fatalf("BuildDAG failed: %v", err)
	}

	if dag == nil {
		t.Fatal("BuildDAG returned nil")
	}

	if dag.WorkflowID != "open-swarm-04p" {
		t.Errorf("expected workflow ID 'open-swarm-04p', got %q", dag.WorkflowID)
	}

	if len(dag.Tasks) == 0 {
		t.Fatal("BuildDAG returned empty task list")
	}
}

func TestReadTaskDAG(t *testing.T) {
	// Test the public ReadTaskDAG function
	dag, err := ReadTaskDAG("open-swarm-04p")
	if err != nil {
		t.Fatalf("ReadTaskDAG failed: %v", err)
	}

	if dag == nil {
		t.Fatal("ReadTaskDAG returned nil")
	}

	if dag.WorkflowID != "open-swarm-04p" {
		t.Errorf("expected workflow ID 'open-swarm-04p', got %q", dag.WorkflowID)
	}
}
