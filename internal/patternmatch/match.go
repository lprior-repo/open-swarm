// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

// Package patternmatch provides file pattern matching utilities.
package patternmatch

import (
	"path/filepath"
)

// Match checks if a file path matches a glob pattern.
// Also tries matching just the basename for convenience.
func Match(filePath, pattern string) (bool, error) {
	matched, err := filepath.Match(pattern, filePath)
	if err != nil {
		return false, err
	}
	if matched {
		return true, nil
	}

	// Also try matching just the basename
	basename := filepath.Base(filePath)
	return filepath.Match(pattern, basename)
}

// Overlap checks if two file patterns overlap using glob matching.
// This implements symmetric matching: either pattern can match the other.
func Overlap(pattern1, pattern2 string) bool {
	if pattern1 == pattern2 {
		return true
	}

	// Try matching in both directions (symmetric)
	match1, _ := filepath.Match(pattern1, pattern2)
	match2, _ := filepath.Match(pattern2, pattern1)

	return match1 || match2
}

// MatchAny checks if a path matches any of the given patterns.
func MatchAny(filePath string, patterns []string) bool {
	for _, pattern := range patterns {
		if matched, _ := Match(filePath, pattern); matched {
			return true
		}
	}
	return false
}
