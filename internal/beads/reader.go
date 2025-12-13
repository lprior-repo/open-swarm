// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package beads

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"open-swarm/internal/temporal"
)

// Task represents a Beads task from .beads/issues.jsonl
type Task struct {
	ID                 string `json:"id"`
	Title              string `json:"title"`
	Description        string `json:"description"`
	AcceptanceCriteria string `json:"acceptance_criteria"`
	Dependencies       []struct {
		IssueID   string `json:"issue_id"`
		DependsOn string `json:"depends_on_id"`
		Type      string `json:"type"` // "parent-child", "blocks"
	} `json:"dependencies"`
}

// GetTask retrieves a task from Beads issues.jsonl
func GetTask(taskID string) (*Task, error) {
	jsonlPath := getJSONLPath()
	// #nosec G304 - jsonlPath is from getJSONLPath which returns hardcoded .beads/issues.jsonl
	file, err := os.Open(jsonlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", jsonlPath, err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			err = fmt.Errorf("failed to close file: %w", closeErr)
		}
	}()

	decoder := json.NewDecoder(file)
	for decoder.More() {
		var task Task
		if err := decoder.Decode(&task); err != nil {
			continue
		}
		if task.ID == taskID {
			return &task, nil
		}
	}
	return nil, fmt.Errorf("task %s not found", taskID)
}

// GetDependencies returns task IDs that this task depends on (blocks relationship)
func GetDependencies(taskID string) ([]string, error) {
	task, err := GetTask(taskID)
	if err != nil {
		return nil, err
	}

	var deps []string
	for _, dep := range task.Dependencies {
		if dep.Type == "blocks" && dep.IssueID == taskID {
			deps = append(deps, dep.DependsOn)
		}
	}
	return deps, nil
}

// BuildDAG converts a Beads task and dependencies into a Temporal DAG workflow input
func BuildDAG(taskID string) (*temporal.DAGWorkflowInput, error) {
	_, err := GetTask(taskID)
	if err != nil {
		return nil, err
	}

	input := &temporal.DAGWorkflowInput{
		WorkflowID: taskID,
		Tasks:      []temporal.Task{},
	}

	visited := make(map[string]bool)
	taskMap := make(map[string]temporal.Task)

	if err := buildDAGRecursive(taskID, visited, taskMap); err != nil {
		return nil, err
	}

	for _, t := range taskMap {
		input.Tasks = append(input.Tasks, t)
	}
	return input, nil
}

// buildDAGRecursive recursively builds the DAG by traversing dependencies
func buildDAGRecursive(taskID string, visited map[string]bool, taskMap map[string]temporal.Task) error {
	if visited[taskID] {
		return nil
	}
	visited[taskID] = true

	task, err := GetTask(taskID)
	if err != nil {
		return err
	}

	// Use description as prompt, fallback to title
	prompt := task.Description
	if prompt == "" {
		prompt = task.Title
	}

	deps, err := GetDependencies(taskID)
	if err != nil {
		return err
	}

	taskMap[task.ID] = temporal.Task{
		Name:    task.ID,
		Command: prompt,
		Deps:    deps,
	}

	// Recursively process dependencies
	for _, depID := range deps {
		if err := buildDAGRecursive(depID, visited, taskMap); err != nil {
			return err
		}
	}

	return nil
}

// ReadTaskDAG reads a task and dependencies from Beads, returning a DAG workflow input
func ReadTaskDAG(taskID string) (*temporal.DAGWorkflowInput, error) {
	return BuildDAG(taskID)
}

// getJSONLPath finds the .beads/issues.jsonl file
func getJSONLPath() string {
	if _, err := os.Stat(".beads/issues.jsonl"); err == nil {
		return ".beads/issues.jsonl"
	}

	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	if output, err := cmd.Output(); err == nil {
		gitRoot := strings.TrimSpace(string(output))
		return filepath.Join(gitRoot, ".beads/issues.jsonl")
	}

	return ".beads/issues.jsonl"
}
