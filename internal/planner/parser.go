// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package planner

import (
	"regexp"
	"strconv"
	"strings"
)

// PlanParser parses user plans into structured tasks
type PlanParser struct {
	numberedPattern *regexp.Regexp
	bulletPattern   *regexp.Regexp
	headerPattern   *regexp.Regexp
}

// NewPlanParser creates a new plan parser
func NewPlanParser() *PlanParser {
	return &PlanParser{
		numberedPattern: regexp.MustCompile(`^(\d+)\.\s+(.+?)(\s*\[P(\d+)\])?\s*(\(depends on:\s*(\d+(?:,\s*\d+)*)\))?$`),
		bulletPattern:   regexp.MustCompile(`^[-*]\s+(.+?)(\s*\[P(\d+)\])?\s*(\(depends on:\s*(\d+(?:,\s*\d+)*)\))?$`),
		headerPattern:   regexp.MustCompile(`^##\s+Task\s+(\d+):\s+(.+)$`),
	}
}

// Parse extracts tasks from a user plan text
func (p *PlanParser) Parse(input string) ([]ParsedTask, error) {
	if strings.TrimSpace(input) == "" {
		return []ParsedTask{}, nil
	}

	lines := strings.Split(input, "\n")
	var tasks []ParsedTask
	var currentTask *ParsedTask
	taskMap := make(map[int]int) // Maps task number to index

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") && !p.headerPattern.MatchString(line) {
			continue
		}

		// Check for task header
		if matches := p.headerPattern.FindStringSubmatch(line); matches != nil {
			if currentTask != nil {
				tasks = append(tasks, *currentTask)
			}
			taskNum, _ := strconv.Atoi(matches[1])
			currentTask = &ParsedTask{
				Title:    matches[2],
				Priority: 1,
			}
			taskMap[taskNum] = len(tasks)
			continue
		}

		// Check for numbered list
		if matches := p.numberedPattern.FindStringSubmatch(line); matches != nil {
			if currentTask != nil {
				tasks = append(tasks, *currentTask)
			}

			taskNum, _ := strconv.Atoi(matches[1])
			title := strings.TrimSpace(matches[2])
			priority := 1
			if matches[4] != "" {
				if pri, err := strconv.Atoi(matches[4]); err == nil {
					priority = pri
				}
			}

			var deps []int
			if matches[6] != "" {
				deps = parseDependencies(matches[6], taskMap)
			}

			currentTask = &ParsedTask{
				Title:     title,
				Priority:  priority,
				DependsOn: deps,
			}
			taskMap[taskNum] = len(tasks)
			continue
		}

		// Check for bullet list
		if matches := p.bulletPattern.FindStringSubmatch(line); matches != nil {
			if currentTask != nil {
				tasks = append(tasks, *currentTask)
			}

			title := strings.TrimSpace(matches[1])
			priority := 1
			if matches[3] != "" {
				if pri, err := strconv.Atoi(matches[3]); err == nil {
					priority = pri
				}
			}

			var deps []int
			if matches[5] != "" {
				deps = parseDependencies(matches[5], taskMap)
			}

			currentTask = &ParsedTask{
				Title:     title,
				Priority:  priority,
				DependsOn: deps,
			}
			continue
		}

		// Check for description
		if currentTask != nil && strings.HasPrefix(line, "Description:") {
			currentTask.Description = strings.TrimSpace(strings.TrimPrefix(line, "Description:"))
		}
	}

	if currentTask != nil {
		tasks = append(tasks, *currentTask)
	}

	return tasks, nil
}

func parseDependencies(depStr string, taskMap map[int]int) []int {
	parts := strings.Split(depStr, ",")
	var deps []int

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if num, err := strconv.Atoi(part); err == nil {
			if idx, ok := taskMap[num]; ok {
				deps = append(deps, idx)
			}
		}
	}

	return deps
}
