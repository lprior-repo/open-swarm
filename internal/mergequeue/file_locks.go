// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package mergequeue

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"open-swarm/internal/filelock"
)

const (
	// Default TTL for file locks acquired by merge queue
	defaultLockTTL = 15 * time.Minute

	// Lock renewal interval (should be less than TTL)
	lockRenewalInterval = 5 * time.Minute
)

// FileLockCoordinator manages file locks for merge queue changes
type FileLockCoordinator struct {
	registry filelock.LockRegistry
	lockTTL  time.Duration
}

// NewFileLockCoordinator creates a new file lock coordinator
func NewFileLockCoordinator(registry filelock.LockRegistry) *FileLockCoordinator {
	return &FileLockCoordinator{
		registry: registry,
		lockTTL:  defaultLockTTL,
	}
}

// NewFileLockCoordinatorWithTTL creates a new file lock coordinator with custom TTL
func NewFileLockCoordinatorWithTTL(registry filelock.LockRegistry, ttl time.Duration) *FileLockCoordinator {
	return &FileLockCoordinator{
		registry: registry,
		lockTTL:  ttl,
	}
}

// AcquireLocksForChange attempts to acquire exclusive locks on all files modified by a change
// Returns an error if any locks cannot be acquired due to conflicts
func (flc *FileLockCoordinator) AcquireLocksForChange(ctx context.Context, change *ChangeRequest) error {
	if change == nil {
		return fmt.Errorf("change request is nil")
	}

	// Track which files we successfully locked (for rollback on partial failure)
	var acquiredPaths []string
	defer func() {
		// If we return with an error and have partial locks, release them
		if len(acquiredPaths) > 0 && len(acquiredPaths) < len(change.FilesModified) {
			for _, path := range acquiredPaths {
				_ = flc.registry.Release(path, change.ID)
			}
		}
	}()

	// Acquire locks for each file
	for _, path := range change.FilesModified {
		// Normalize path to absolute
		absPath, err := filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("failed to normalize path %s: %w", path, err)
		}

		req := filelock.LockRequest{
			Path:      absPath,
			Holder:    change.ID,
			Exclusive: true, // Changes need exclusive locks
			TTL:       flc.lockTTL,
		}

		result, err := flc.registry.Acquire(req)
		if err != nil {
			// Lock conflict - return detailed error
			if conflictErr, ok := err.(*filelock.ConflictError); ok {
				return fmt.Errorf("cannot acquire lock on %s: file is locked by %s (exclusive: %v)",
					absPath, conflictErr.Holder, conflictErr.Exclusive)
			}
			return fmt.Errorf("failed to acquire lock on %s: %w", absPath, err)
		}

		if !result.Granted {
			// Should not happen if Acquire returns no error, but check anyway
			return fmt.Errorf("lock not granted for %s", absPath)
		}

		acquiredPaths = append(acquiredPaths, absPath)
	}

	return nil
}

// ReleaseLocksForChange releases all locks held by a change
func (flc *FileLockCoordinator) ReleaseLocksForChange(ctx context.Context, change *ChangeRequest) error {
	if change == nil {
		return fmt.Errorf("change request is nil")
	}

	var firstError error
	for _, path := range change.FilesModified {
		absPath, err := filepath.Abs(path)
		if err != nil {
			if firstError == nil {
				firstError = fmt.Errorf("failed to normalize path %s: %w", path, err)
			}
			continue
		}

		if err := flc.registry.Release(absPath, change.ID); err != nil {
			// Don't fail on ErrLockNotFound or ErrLockNotHeld - might already be released
			if err != filelock.ErrLockNotFound && err != filelock.ErrLockNotHeld {
				if firstError == nil {
					firstError = fmt.Errorf("failed to release lock on %s: %w", absPath, err)
				}
			}
		}
	}

	return firstError
}

// RenewLocksForChange extends the TTL of all locks held by a change
func (flc *FileLockCoordinator) RenewLocksForChange(ctx context.Context, change *ChangeRequest) error {
	if change == nil {
		return fmt.Errorf("change request is nil")
	}

	var firstError error
	for _, path := range change.FilesModified {
		absPath, err := filepath.Abs(path)
		if err != nil {
			if firstError == nil {
				firstError = fmt.Errorf("failed to normalize path %s: %w", path, err)
			}
			continue
		}

		if err := flc.registry.RenewLock(absPath, change.ID, flc.lockTTL); err != nil {
			if firstError == nil {
				firstError = fmt.Errorf("failed to renew lock on %s: %w", absPath, err)
			}
		}
	}

	return firstError
}

// CheckConflicts checks if two changes have conflicting file locks
// Returns true if the changes conflict, false otherwise
func (flc *FileLockCoordinator) CheckConflicts(change1, change2 *ChangeRequest) bool {
	if change1 == nil || change2 == nil {
		return false
	}

	// Build a set of files modified by change2 for fast lookup
	change2Files := make(map[string]bool)
	change2Dirs := make(map[string]bool)

	for _, path := range change2.FilesModified {
		absPath, err := filepath.Abs(path)
		if err != nil {
			// If we can't normalize, fall back to original path
			absPath = path
		}
		change2Files[absPath] = true
		change2Dirs[filepath.Dir(absPath)] = true
	}

	// Check if any files from change1 overlap with change2
	for _, path := range change1.FilesModified {
		absPath, err := filepath.Abs(path)
		if err != nil {
			absPath = path
		}

		// Direct file conflict
		if change2Files[absPath] {
			return true
		}

		// Check if they share a directory (potential conflict)
		dir := filepath.Dir(absPath)
		if change2Dirs[dir] {
			return true
		}
	}

	return false
}

// CheckActiveConflicts checks if a change conflicts with any active locks in the registry
// Returns true if there are conflicts, false otherwise
func (flc *FileLockCoordinator) CheckActiveConflicts(ctx context.Context, change *ChangeRequest) (bool, []string) {
	if change == nil {
		return false, nil
	}

	var conflictingHolders []string
	holderSet := make(map[string]bool)

	for _, path := range change.FilesModified {
		absPath, err := filepath.Abs(path)
		if err != nil {
			absPath = path
		}

		// Check if there are any active locks on this file
		activeLocks := flc.registry.Check(absPath)
		for _, lock := range activeLocks {
			// Ignore locks held by the same change
			if lock.Holder != change.ID {
				if !holderSet[lock.Holder] {
					holderSet[lock.Holder] = true
					conflictingHolders = append(conflictingHolders, lock.Holder)
				}
			}
		}
	}

	return len(conflictingHolders) > 0, conflictingHolders
}

// sharesDirectory checks if two paths share the same directory
func sharesDirectoryPaths(path1, path2 string) bool {
	dir1 := filepath.Dir(filepath.Clean(path1))
	dir2 := filepath.Dir(filepath.Clean(path2))

	// Normalize directory separators
	dir1 = filepath.ToSlash(dir1)
	dir2 = filepath.ToSlash(dir2)

	// Check for exact match
	if dir1 == dir2 {
		return true
	}

	// Check if one is a parent of the other
	return strings.HasPrefix(dir1+"/", dir2+"/") || strings.HasPrefix(dir2+"/", dir1+"/")
}

// StartLockRenewal starts a background goroutine that periodically renews locks for a change
// Returns a cancel function that should be called to stop the renewal
func (flc *FileLockCoordinator) StartLockRenewal(ctx context.Context, change *ChangeRequest) (context.CancelFunc, error) {
	if change == nil {
		return nil, fmt.Errorf("change request is nil")
	}

	renewalCtx, cancel := context.WithCancel(ctx)

	// Calculate renewal interval (use configured interval or 1/3 of TTL, whichever is smaller)
	interval := lockRenewalInterval
	if flc.lockTTL/3 < interval {
		interval = flc.lockTTL / 3
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-renewalCtx.Done():
				return
			case <-ticker.C:
				if err := flc.RenewLocksForChange(renewalCtx, change); err != nil {
					// Log error but continue trying
					// In production, this should use proper logging
					_ = err
				}
			}
		}
	}()

	return cancel, nil
}

// GetLockStatus returns information about locks held by a change
func (flc *FileLockCoordinator) GetLockStatus(ctx context.Context, change *ChangeRequest) (map[string][]filelock.FileLock, error) {
	if change == nil {
		return nil, fmt.Errorf("change request is nil")
	}

	status := make(map[string][]filelock.FileLock)

	for _, path := range change.FilesModified {
		absPath, err := filepath.Abs(path)
		if err != nil {
			absPath = path
		}

		locks := flc.registry.Check(absPath)
		if len(locks) > 0 {
			status[absPath] = locks
		}
	}

	return status, nil
}
