package gates

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"time"
)

// TestImmutabilityGate ensures tests cannot be modified or disabled.
// Tests are locked read-only and executed in process isolation.
type TestImmutabilityGate struct {
	taskID          string
	testFilePath    string
	originalHash    string // SHA256 hash of test file at start
	testBinary      string // Path to compiled test binary
	timestamp       int64
	checkInterval   time.Duration
	checksPerformed int
}

// NewTestImmutabilityGate creates a new test immutability gate.
func NewTestImmutabilityGate(taskID string, testFilePath string) *TestImmutabilityGate {
	return &TestImmutabilityGate{
		taskID:        taskID,
		testFilePath:  testFilePath,
		timestamp:     time.Now().Unix(),
		checkInterval: 100 * time.Millisecond, // Check every 100ms during execution
	}
}

// SetTestBinary sets the path to the pre-compiled test binary.
func (tig *TestImmutabilityGate) SetTestBinary(binaryPath string) {
	tig.testBinary = binaryPath
}

// Type returns the gate type.
func (tig *TestImmutabilityGate) Type() GateType {
	return GateTestImmutability
}

// Name returns the human-readable name.
func (tig *TestImmutabilityGate) Name() string {
	return "Test Immutability Lock"
}

// Check verifies that tests are locked read-only and cannot be modified.
func (tig *TestImmutabilityGate) Check(_ context.Context) error {
	// Step 1: Lock test file to read-only
	if err := tig.lockTestFile(); err != nil {
		return &GateError{
			Gate:      tig.Type(),
			TaskID:    tig.taskID,
			Message:   "failed to lock test file",
			Details:   err.Error(),
			Timestamp: time.Now().Unix(),
		}
	}

	// Step 2: Record hash of test file
	hash, err := tig.hashTestFile()
	if err != nil {
		return &GateError{
			Gate:      tig.Type(),
			TaskID:    tig.taskID,
			Message:   "failed to hash test file",
			Details:   err.Error(),
			Timestamp: time.Now().Unix(),
		}
	}
	tig.originalHash = hash

	// Step 3: Use process isolation (test runs in separate process, not in agent process)
	if tig.testBinary == "" {
		return &GateError{
			Gate:      tig.Type(),
			TaskID:    tig.taskID,
			Message:   "test binary not set",
			Details:   "Test binary path must be set before execution. Tests must run in isolated process.",
			Timestamp: time.Now().Unix(),
		}
	}

	// Step 4: Verify test binary exists and is executable
	fileInfo, err := os.Stat(tig.testBinary)
	if err != nil {
		return &GateError{
			Gate:      tig.Type(),
			TaskID:    tig.taskID,
			Message:   "test binary not found or not executable",
			Details:   fmt.Sprintf("Path: %s, Error: %v", tig.testBinary, err),
			Timestamp: time.Now().Unix(),
		}
	}

	// Check executable bit on unix
	if fileInfo.Mode()&0o111 == 0 {
		return &GateError{
			Gate:      tig.Type(),
			TaskID:    tig.taskID,
			Message:   "test binary not executable",
			Details:   fmt.Sprintf("Path: %s does not have executable permissions", tig.testBinary),
			Timestamp: time.Now().Unix(),
		}
	}

	// Step 5: Continuous verification (monitor during execution)
	// This would be called periodically by the orchestrator
	if err := tig.verifyTestFileIntegrity(); err != nil {
		return err
	}

	return nil
}

// lockTestFile sets the test file to read-only to prevent modifications.
func (tig *TestImmutabilityGate) lockTestFile() error {
	// Set file to read-only (0o444 = r--r--r--)
	if err := os.Chmod(tig.testFilePath, 0o444); err != nil { //nolint:gosec
		return fmt.Errorf("failed to lock test file: %w", err)
	}
	return nil
}

// hashTestFile returns the SHA256 hash of the test file.
func (tig *TestImmutabilityGate) hashTestFile() (string, error) {
	content, err := os.ReadFile(tig.testFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read test file: %w", err)
	}

	hash := sha256.Sum256(content)
	return fmt.Sprintf("%x", hash), nil
}

// verifyTestFileIntegrity checks that the test file hasn't been modified.
func (tig *TestImmutabilityGate) verifyTestFileIntegrity() error {
	if tig.originalHash == "" {
		return nil // No baseline to compare against
	}

	currentHash, err := tig.hashTestFile()
	if err != nil {
		return &GateError{
			Gate:      tig.Type(),
			TaskID:    tig.taskID,
			Message:   "failed to verify test file integrity",
			Details:   err.Error(),
			Timestamp: time.Now().Unix(),
		}
	}

	if currentHash != tig.originalHash {
		return &GateError{
			Gate:      tig.Type(),
			TaskID:    tig.taskID,
			Message:   "test file has been modified",
			Details:   fmt.Sprintf("Original hash: %s, Current hash: %s. Tests are immutable and cannot be changed.", tig.originalHash, currentHash),
			Timestamp: time.Now().Unix(),
		}
	}

	tig.checksPerformed++
	return nil
}

// UnlockTestFile removes read-only protection (for cleanup after execution).
func (tig *TestImmutabilityGate) UnlockTestFile() error {
	// Restore write permissions for cleanup
	if err := os.Chmod(tig.testFilePath, 0o644); err != nil { //nolint:gosec
		return fmt.Errorf("failed to unlock test file: %w", err)
	}
	return nil
}
