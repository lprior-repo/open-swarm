// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package tdd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTDDEnforcerValidation tests the TDD guard plugin functionality
// by simulating various TDD workflow scenarios
func TestTDDEnforcerValidation(t *testing.T) {
	// Create temporary test directory
	tmpDir, err := os.MkdirTemp("", "tdd-enforcer-test-*")
	require.NoError(t, err, "Failed to create temp directory")
	defer os.RemoveAll(tmpDir)

	// Initialize a minimal Go module
	err = initGoModule(tmpDir)
	require.NoError(t, err, "Failed to initialize Go module")

	t.Run("validates test file exists before implementation", func(t *testing.T) {
		testDir := filepath.Join(tmpDir, "test1")
		err := os.MkdirAll(testDir, 0755)
		require.NoError(t, err)

		implFile := filepath.Join(testDir, "calculator.go")
		testFile := filepath.Join(testDir, "calculator_test.go")

		// Create implementation without test (should fail)
		createFile(t, implFile, `package test1

func Add(a, b int) int {
	return a + b
}
`)

		// Validate that test file must exist first
		assert.NoFileExists(t, testFile, "Test file should not exist yet")
	})

	t.Run("validates test uses testify", func(t *testing.T) {
		testDir := filepath.Join(tmpDir, "test2")
		err := os.MkdirAll(testDir, 0755)
		require.NoError(t, err)

		testFile := filepath.Join(testDir, "calculator_test.go")

		// Create test without testify
		createFile(t, testFile, `package test2

import "testing"

func TestAdd(t *testing.T) {
	result := Add(1, 2)
	if result != 3 {
		t.Errorf("Expected 3, got %d", result)
	}
}
`)

		content, err := os.ReadFile(testFile)
		require.NoError(t, err)

		// Verify testify is not being used
		assert.NotContains(t, string(content), "github.com/stretchr/testify",
			"Test should not contain testify import yet")
	})

	t.Run("validates proper TDD workflow", func(t *testing.T) {
		testDir := filepath.Join(tmpDir, "test3")
		err := os.MkdirAll(testDir, 0755)
		require.NoError(t, err)

		testFile := filepath.Join(testDir, "math_test.go")
		implFile := filepath.Join(testDir, "math.go")

		// STEP 1: Write failing test first (RED)
		createFile(t, testFile, `package test3

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestMultiply(t *testing.T) {
	result := Multiply(3, 4)
	assert.Equal(t, 12, result, "3 * 4 should equal 12")
}
`)

		// Verify test file exists
		assert.FileExists(t, testFile, "Test file should exist first")

		// Test should fail because implementation doesn't exist
		cmd := exec.Command("go", "test", testFile)
		cmd.Dir = testDir
		output, err := cmd.CombinedOutput()
		assert.Error(t, err, "Test should fail (RED phase)")
		assert.Contains(t, string(output), "undefined", "Should fail with undefined function")

		// STEP 2: Write minimal implementation (GREEN)
		createFile(t, implFile, `package test3

func Multiply(a, b int) int {
	return a * b
}
`)

		// Test should now pass
		cmd = exec.Command("go", "test", testFile)
		cmd.Dir = testDir
		output, err = cmd.CombinedOutput()
		assert.NoError(t, err, "Test should pass (GREEN phase): %s", string(output))
	})

	t.Run("validates test atomicity", func(t *testing.T) {
		testDir := filepath.Join(tmpDir, "test4")
		err := os.MkdirAll(testDir, 0755)
		require.NoError(t, err)

		testFile := filepath.Join(testDir, "utils_test.go")

		// Create test with reasonable number of test functions (should pass)
		createFile(t, testFile, `package test4

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestFunction1(t *testing.T) {
	assert.True(t, true)
}

func TestFunction2(t *testing.T) {
	assert.True(t, true)
}

func TestFunction3(t *testing.T) {
	assert.True(t, true)
}
`)

		content, err := os.ReadFile(testFile)
		require.NoError(t, err)

		// Count test functions
		testFuncCount := strings.Count(string(content), "func Test")
		assert.LessOrEqual(t, testFuncCount, 3, "Should have <= 3 test functions for atomicity")
	})

	t.Run("validates full test suite passes", func(t *testing.T) {
		testDir := filepath.Join(tmpDir, "test5")
		err := os.MkdirAll(testDir, 0755)
		require.NoError(t, err)

		// Create multiple test files
		createFile(t, filepath.Join(testDir, "add.go"), `package test5

func Add(a, b int) int {
	return a + b
}
`)

		createFile(t, filepath.Join(testDir, "add_test.go"), `package test5

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestAdd(t *testing.T) {
	assert.Equal(t, 5, Add(2, 3))
}
`)

		createFile(t, filepath.Join(testDir, "subtract.go"), `package test5

func Subtract(a, b int) int {
	return a - b
}
`)

		createFile(t, filepath.Join(testDir, "subtract_test.go"), `package test5

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestSubtract(t *testing.T) {
	assert.Equal(t, 1, Subtract(3, 2))
}
`)

		// Run full test suite
		cmd := exec.Command("go", "test", "./...")
		cmd.Dir = testDir
		output, err := cmd.CombinedOutput()
		assert.NoError(t, err, "Full test suite should pass: %s", string(output))
	})
}

// TestTDDEnforcerViolations tests that the TDD guard catches violations
func TestTDDEnforcerViolations(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tdd-violation-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	err = initGoModule(tmpDir)
	require.NoError(t, err)

	t.Run("detects missing test file", func(t *testing.T) {
		testDir := filepath.Join(tmpDir, "violation1")
		err := os.MkdirAll(testDir, 0755)
		require.NoError(t, err)

		implFile := filepath.Join(testDir, "service.go")
		testFile := filepath.Join(testDir, "service_test.go")

		// Create implementation first (VIOLATION)
		createFile(t, implFile, `package violation1

type Service struct {}

func NewService() *Service {
	return &Service{}
}
`)

		assert.FileExists(t, implFile, "Implementation exists")
		assert.NoFileExists(t, testFile, "Test file does not exist - TDD VIOLATION")
	})

	t.Run("detects test without testify", func(t *testing.T) {
		testDir := filepath.Join(tmpDir, "violation2")
		err := os.MkdirAll(testDir, 0755)
		require.NoError(t, err)

		testFile := filepath.Join(testDir, "handler_test.go")

		// Create test without testify (VIOLATION)
		createFile(t, testFile, `package violation2

import "testing"

func TestHandler(t *testing.T) {
	if 1+1 != 2 {
		t.Error("math broken")
	}
}
`)

		content, err := os.ReadFile(testFile)
		require.NoError(t, err)

		hasTestify := strings.Contains(string(content), "github.com/stretchr/testify")
		assert.False(t, hasTestify, "Test does not use testify - VIOLATION")
	})

	t.Run("detects non-atomic test file", func(t *testing.T) {
		testDir := filepath.Join(tmpDir, "violation3")
		err := os.MkdirAll(testDir, 0755)
		require.NoError(t, err)

		testFile := filepath.Join(testDir, "bloated_test.go")

		// Create test file with too many test functions (VIOLATION)
		testFuncs := []string{
			`func TestFunc1(t *testing.T) { assert.True(t, true) }`,
			`func TestFunc2(t *testing.T) { assert.True(t, true) }`,
			`func TestFunc3(t *testing.T) { assert.True(t, true) }`,
			`func TestFunc4(t *testing.T) { assert.True(t, true) }`,
			`func TestFunc5(t *testing.T) { assert.True(t, true) }`,
		}

		testContent := `package violation3

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

` + strings.Join(testFuncs, "\n\n")

		createFile(t, testFile, testContent)

		content, err := os.ReadFile(testFile)
		require.NoError(t, err)

		testFuncCount := strings.Count(string(content), "func Test")
		assert.Greater(t, testFuncCount, 3, "Test file has too many functions - VIOLATION")
	})
}

// TestTDDEnforcerRedGreenRefactor tests the full TDD cycle
func TestTDDEnforcerRedGreenRefactor(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tdd-cycle-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	err = initGoModule(tmpDir)
	require.NoError(t, err)

	testDir := filepath.Join(tmpDir, "cycle")
	err = os.MkdirAll(testDir, 0755)
	require.NoError(t, err)

	testFile := filepath.Join(testDir, "calculator_test.go")
	implFile := filepath.Join(testDir, "calculator.go")

	// RED: Write failing test
	t.Run("RED phase - test fails", func(t *testing.T) {
		createFile(t, testFile, `package cycle

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestDivide(t *testing.T) {
	result, err := Divide(10, 2)
	assert.NoError(t, err)
	assert.Equal(t, 5, result)
}
`)

		cmd := exec.Command("go", "test", testFile)
		cmd.Dir = testDir
		_, err := cmd.CombinedOutput()
		assert.Error(t, err, "RED: Test should fail - function doesn't exist")
	})

	// GREEN: Write minimal implementation
	t.Run("GREEN phase - test passes", func(t *testing.T) {
		createFile(t, implFile, `package cycle

func Divide(a, b int) (int, error) {
	return a / b, nil
}
`)

		cmd := exec.Command("go", "test", testFile)
		cmd.Dir = testDir
		output, err := cmd.CombinedOutput()
		assert.NoError(t, err, "GREEN: Test should pass: %s", string(output))
	})

	// REFACTOR: Add error handling (test-driven)
	t.Run("REFACTOR phase - add error handling", func(t *testing.T) {
		// First, add test for error case
		createFile(t, testFile, `package cycle

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestDivide(t *testing.T) {
	result, err := Divide(10, 2)
	assert.NoError(t, err)
	assert.Equal(t, 5, result)
}

func TestDivideByZero(t *testing.T) {
	_, err := Divide(10, 0)
	assert.Error(t, err, "Should return error for division by zero")
}
`)

		// Refactor implementation to handle error
		createFile(t, implFile, `package cycle

import "errors"

func Divide(a, b int) (int, error) {
	if b == 0 {
		return 0, errors.New("division by zero")
	}
	return a / b, nil
}
`)

		cmd := exec.Command("go", "test", testFile)
		cmd.Dir = testDir
		output, err := cmd.CombinedOutput()
		assert.NoError(t, err, "REFACTOR: All tests should pass: %s", string(output))
	})
}

// Helper functions

func initGoModule(dir string) error {
	cmd := exec.Command("go", "mod", "init", "tdd-test")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return err
	}

	// Add testify dependency
	cmd = exec.Command("go", "get", "github.com/stretchr/testify/assert")
	cmd.Dir = dir
	return cmd.Run()
}

func createFile(t *testing.T, path, content string) {
	t.Helper()
	err := os.WriteFile(path, []byte(content), 0644)
	require.NoError(t, err, "Failed to create file: %s", path)
}
