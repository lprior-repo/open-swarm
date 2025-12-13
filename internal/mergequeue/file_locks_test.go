// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package mergequeue

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"open-swarm/internal/filelock"
)

func TestFileLockCoordinator_AcquireLocksForChange(t *testing.T) {
	registry := filelock.NewMemoryRegistry()
	flc := NewFileLockCoordinator(registry)
	ctx := context.Background()

	// Create a test change
	change := &ChangeRequest{
		ID:            "agent-1",
		FilesModified: []string{"file1.go", "file2.go"},
	}

	// Acquire locks
	err := flc.AcquireLocksForChange(ctx, change)
	if err != nil {
		t.Fatalf("Failed to acquire locks: %v", err)
	}

	// Verify locks were acquired
	for _, path := range change.FilesModified {
		absPath, _ := filepath.Abs(path)
		locks := registry.Check(absPath)
		if len(locks) != 1 {
			t.Errorf("Expected 1 lock on %s, got %d", absPath, len(locks))
		}
		if len(locks) > 0 && locks[0].Holder != change.ID {
			t.Errorf("Expected lock holder to be %s, got %s", change.ID, locks[0].Holder)
		}
	}
}

func TestFileLockCoordinator_AcquireLocksForChange_Conflict(t *testing.T) {
	registry := filelock.NewMemoryRegistry()
	flc := NewFileLockCoordinator(registry)
	ctx := context.Background()

	// First change acquires locks
	change1 := &ChangeRequest{
		ID:            "agent-1",
		FilesModified: []string{"file1.go"},
	}
	err := flc.AcquireLocksForChange(ctx, change1)
	if err != nil {
		t.Fatalf("Failed to acquire locks for change1: %v", err)
	}

	// Second change tries to acquire lock on same file - should conflict
	change2 := &ChangeRequest{
		ID:            "agent-2",
		FilesModified: []string{"file1.go"},
	}
	err = flc.AcquireLocksForChange(ctx, change2)
	if err == nil {
		t.Fatal("Expected conflict error, got nil")
	}
}

func TestFileLockCoordinator_ReleaseLocksForChange(t *testing.T) {
	registry := filelock.NewMemoryRegistry()
	flc := NewFileLockCoordinator(registry)
	ctx := context.Background()

	// Acquire locks
	change := &ChangeRequest{
		ID:            "agent-1",
		FilesModified: []string{"file1.go", "file2.go"},
	}
	err := flc.AcquireLocksForChange(ctx, change)
	if err != nil {
		t.Fatalf("Failed to acquire locks: %v", err)
	}

	// Release locks
	err = flc.ReleaseLocksForChange(ctx, change)
	if err != nil {
		t.Fatalf("Failed to release locks: %v", err)
	}

	// Verify locks were released
	for _, path := range change.FilesModified {
		absPath, _ := filepath.Abs(path)
		locks := registry.Check(absPath)
		if len(locks) != 0 {
			t.Errorf("Expected 0 locks on %s after release, got %d", absPath, len(locks))
		}
	}
}

func TestFileLockCoordinator_RenewLocksForChange(t *testing.T) {
	registry := filelock.NewMemoryRegistry()
	flc := NewFileLockCoordinatorWithTTL(registry, 1*time.Second)
	ctx := context.Background()

	// Acquire locks
	change := &ChangeRequest{
		ID:            "agent-1",
		FilesModified: []string{"file1.go"},
	}
	err := flc.AcquireLocksForChange(ctx, change)
	if err != nil {
		t.Fatalf("Failed to acquire locks: %v", err)
	}

	// Get initial expiration time
	absPath, _ := filepath.Abs("file1.go")
	initialLocks := registry.Check(absPath)
	if len(initialLocks) == 0 {
		t.Fatal("No locks found after acquisition")
	}
	initialExpiry := initialLocks[0].ExpiresAt

	// Wait a bit then renew
	time.Sleep(100 * time.Millisecond)
	err = flc.RenewLocksForChange(ctx, change)
	if err != nil {
		t.Fatalf("Failed to renew locks: %v", err)
	}

	// Check that expiration time was extended
	renewedLocks := registry.Check(absPath)
	if len(renewedLocks) == 0 {
		t.Fatal("No locks found after renewal")
	}
	renewedExpiry := renewedLocks[0].ExpiresAt

	if !renewedExpiry.After(initialExpiry) {
		t.Errorf("Expected renewed expiry (%v) to be after initial expiry (%v)",
			renewedExpiry, initialExpiry)
	}
}

func TestFileLockCoordinator_CheckConflicts(t *testing.T) {
	registry := filelock.NewMemoryRegistry()
	flc := NewFileLockCoordinator(registry)

	tests := []struct {
		name           string
		change1        *ChangeRequest
		change2        *ChangeRequest
		expectConflict bool
	}{
		{
			name: "same file - should conflict",
			change1: &ChangeRequest{
				ID:            "agent-1",
				FilesModified: []string{"file1.go"},
			},
			change2: &ChangeRequest{
				ID:            "agent-2",
				FilesModified: []string{"file1.go"},
			},
			expectConflict: true,
		},
		{
			name: "files in same directory - should conflict",
			change1: &ChangeRequest{
				ID:            "agent-1",
				FilesModified: []string{"dir/file1.go"},
			},
			change2: &ChangeRequest{
				ID:            "agent-2",
				FilesModified: []string{"dir/file2.go"},
			},
			expectConflict: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasConflict := flc.CheckConflicts(tt.change1, tt.change2)
			if hasConflict != tt.expectConflict {
				t.Errorf("Expected conflict=%v, got %v", tt.expectConflict, hasConflict)
			}
		})
	}
}

func TestFileLockCoordinator_CheckActiveConflicts(t *testing.T) {
	registry := filelock.NewMemoryRegistry()
	flc := NewFileLockCoordinator(registry)
	ctx := context.Background()

	// Acquire locks for change1
	change1 := &ChangeRequest{
		ID:            "agent-1",
		FilesModified: []string{"file1.go"},
	}
	err := flc.AcquireLocksForChange(ctx, change1)
	if err != nil {
		t.Fatalf("Failed to acquire locks for change1: %v", err)
	}

	// Check for conflicts with change2 (same file)
	change2 := &ChangeRequest{
		ID:            "agent-2",
		FilesModified: []string{"file1.go"},
	}
	hasConflict, holders := flc.CheckActiveConflicts(ctx, change2)
	if !hasConflict {
		t.Error("Expected active conflict, got none")
	}
	if len(holders) != 1 || holders[0] != "agent-1" {
		t.Errorf("Expected conflicting holder to be agent-1, got %v", holders)
	}

	// Check for conflicts with change3 (different file and directory)
	change3 := &ChangeRequest{
		ID:            "agent-3",
		FilesModified: []string{"other/file2.go"},
	}
	hasConflict, holders = flc.CheckActiveConflicts(ctx, change3)
	if hasConflict {
		t.Errorf("Expected no conflict for different file in different directory, but got conflicts with %v", holders)
	}
}

func TestFileLockCoordinator_StartLockRenewal(t *testing.T) {
	registry := filelock.NewMemoryRegistry()
	// Use short TTL for testing
	flc := NewFileLockCoordinatorWithTTL(registry, 300*time.Millisecond)
	ctx := context.Background()

	// Acquire locks
	change := &ChangeRequest{
		ID:            "agent-1",
		FilesModified: []string{"file1.go"},
	}
	err := flc.AcquireLocksForChange(ctx, change)
	if err != nil {
		t.Fatalf("Failed to acquire locks: %v", err)
	}

	// Start lock renewal (will renew every 100ms since TTL is 300ms)
	cancel, err := flc.StartLockRenewal(ctx, change)
	if err != nil {
		t.Fatalf("Failed to start lock renewal: %v", err)
	}
	defer cancel()

	// Wait longer than the TTL but with renewal active
	time.Sleep(500 * time.Millisecond)

	// Locks should still be active due to renewal
	absPath, _ := filepath.Abs("file1.go")
	locks := registry.Check(absPath)
	if len(locks) == 0 {
		t.Error("Expected lock to still be active after renewal period")
	}

	// Stop renewal
	cancel()

	// Wait for locks to expire
	time.Sleep(400 * time.Millisecond)

	// Locks should now be expired
	locks = registry.Check(absPath)
	if len(locks) > 0 {
		t.Error("Expected lock to expire after stopping renewal")
	}
}

func TestFileLockCoordinator_GetLockStatus(t *testing.T) {
	registry := filelock.NewMemoryRegistry()
	flc := NewFileLockCoordinator(registry)
	ctx := context.Background()

	// Acquire locks
	change := &ChangeRequest{
		ID:            "agent-1",
		FilesModified: []string{"file1.go", "file2.go"},
	}
	err := flc.AcquireLocksForChange(ctx, change)
	if err != nil {
		t.Fatalf("Failed to acquire locks: %v", err)
	}

	// Get lock status
	status, err := flc.GetLockStatus(ctx, change)
	if err != nil {
		t.Fatalf("Failed to get lock status: %v", err)
	}

	// Should have status for both files
	if len(status) != 2 {
		t.Errorf("Expected status for 2 files, got %d", len(status))
	}

	// Verify each file has a lock
	for _, path := range change.FilesModified {
		absPath, _ := filepath.Abs(path)
		locks, exists := status[absPath]
		if !exists {
			t.Errorf("No status for file %s", absPath)
			continue
		}
		if len(locks) != 1 {
			t.Errorf("Expected 1 lock for %s, got %d", absPath, len(locks))
		}
	}
}

func TestFileLockCoordinator_NilChangeRequest(t *testing.T) {
	registry := filelock.NewMemoryRegistry()
	flc := NewFileLockCoordinator(registry)
	ctx := context.Background()

	// All methods should handle nil gracefully
	err := flc.AcquireLocksForChange(ctx, nil)
	if err == nil {
		t.Error("Expected error for nil change in AcquireLocksForChange")
	}

	err = flc.ReleaseLocksForChange(ctx, nil)
	if err == nil {
		t.Error("Expected error for nil change in ReleaseLocksForChange")
	}

	err = flc.RenewLocksForChange(ctx, nil)
	if err == nil {
		t.Error("Expected error for nil change in RenewLocksForChange")
	}

	hasConflict := flc.CheckConflicts(nil, nil)
	if hasConflict {
		t.Error("Expected no conflict for nil changes")
	}

	hasConflict, _ = flc.CheckActiveConflicts(ctx, nil)
	if hasConflict {
		t.Error("Expected no conflict for nil change in CheckActiveConflicts")
	}

	_, err = flc.StartLockRenewal(ctx, nil)
	if err == nil {
		t.Error("Expected error for nil change in StartLockRenewal")
	}

	_, err = flc.GetLockStatus(ctx, nil)
	if err == nil {
		t.Error("Expected error for nil change in GetLockStatus")
	}
}
