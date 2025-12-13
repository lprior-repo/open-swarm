// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package filelock

import (
	"path/filepath"
	"sync"
	"time"
)

// MemoryRegistry implements LockRegistry with in-memory storage and thread-safe access.
type MemoryRegistry struct {
	mu    sync.RWMutex
	locks map[string][]FileLock // path -> []FileLock
}

// NewMemoryRegistry creates a new in-memory file lock registry.
func NewMemoryRegistry() *MemoryRegistry {
	return &MemoryRegistry{
		locks: make(map[string][]FileLock),
	}
}

// Acquire attempts to acquire a lock on the specified file.
// Returns ConflictError if the lock cannot be granted due to conflicting locks.
func (r *MemoryRegistry) Acquire(req LockRequest) (LockResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check for conflicts with existing locks on the same path
	existingLocks := r.locks[req.Path]
	for _, existingLock := range existingLocks {
		if existingLock.ExpiresAt.After(time.Now()) {
			// Exclusive locks conflict with everything, and anything conflicts with exclusive
			if req.Exclusive || existingLock.Exclusive {
				return LockResult{
						Granted:   false,
						Conflicts: existingLocks,
					}, &ConflictError{
						Path:            req.Path,
						Holder:          existingLock.Holder,
						Exclusive:       existingLock.Exclusive,
						ExistingLocks:   existingLocks,
						RequestedPath:   req.Path,
						RequestedHolder: req.Holder,
						IsExclusive:     req.Exclusive,
					}
			}
		}
	}

	// Check glob patterns against other paths
	for otherPath, otherLocks := range r.locks {
		if otherPath == req.Path {
			continue
		}

		// Check if patterns overlap using filepath.Match
		match1, _ := filepath.Match(req.Path, otherPath)
		match2, _ := filepath.Match(otherPath, req.Path)
		if !match1 && !match2 {
			continue
		}

		// Check for conflicts with locks on overlapping patterns
		for _, otherLock := range otherLocks {
			if otherLock.ExpiresAt.After(time.Now()) {
				if req.Exclusive || otherLock.Exclusive {
					return LockResult{
							Granted:   false,
							Conflicts: otherLocks,
						}, &ConflictError{
							Path:            req.Path,
							Holder:          otherLock.Holder,
							Exclusive:       otherLock.Exclusive,
							ExistingLocks:   otherLocks,
							RequestedPath:   req.Path,
							RequestedHolder: req.Holder,
							IsExclusive:     req.Exclusive,
						}
				}
			}
		}
	}

	// Acquire the lock
	now := time.Now()
	newLock := FileLock{
		Path:       req.Path,
		Holder:     req.Holder,
		Exclusive:  req.Exclusive,
		ExpiresAt:  now.Add(req.TTL),
		AcquiredAt: now,
	}

	r.locks[req.Path] = append(r.locks[req.Path], newLock)

	return LockResult{
		Granted: true,
		Lock:    &newLock,
	}, nil
}

// Release removes a lock held by the specified agent on the given file.
// Returns an error if the lock is not held by the specified agent or does not exist.
func (r *MemoryRegistry) Release(path, holder string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	locks, exists := r.locks[path]
	if !exists {
		return ErrLockNotFound
	}

	// Find and remove the lock held by this holder
	for i, lock := range locks {
		if lock.Holder == holder {
			// Remove the lock by slicing
			r.locks[path] = append(locks[:i], locks[i+1:]...)
			if len(r.locks[path]) == 0 {
				delete(r.locks, path)
			}
			return nil
		}
	}

	return ErrLockNotHeld
}

// Check returns information about locks on a file without acquiring or modifying them.
// Expired locks are filtered out from the result.
func (r *MemoryRegistry) Check(path string) []FileLock {
	r.mu.RLock()
	defer r.mu.RUnlock()

	locks, exists := r.locks[path]
	if !exists {
		return []FileLock{}
	}

	// Filter and return only non-expired locks
	var activeLocks []FileLock
	now := time.Now()
	for _, lock := range locks {
		if lock.ExpiresAt.After(now) {
			activeLocks = append(activeLocks, lock)
		}
	}

	return activeLocks
}

// RenewLock extends the expiration time of an existing lock.
// Returns an error if the lock does not exist or is not held by the specified agent.
func (r *MemoryRegistry) RenewLock(path, holder string, newTTL time.Duration) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	locks, exists := r.locks[path]
	if !exists {
		return ErrLockNotFound
	}

	// Find and renew the lock
	now := time.Now()
	for i, lock := range locks {
		if lock.Holder == holder {
			// Check if lock has already expired
			if lock.ExpiresAt.Before(now) {
				return ErrLockNotFound
			}
			r.locks[path][i].ExpiresAt = now.Add(newTTL)
			return nil
		}
	}

	return ErrLockNotHeld
}

// CleanupExpired removes all locks that have passed their expiration time.
// Returns the number of locks removed.
func (r *MemoryRegistry) CleanupExpired() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	removed := 0

	for path := range r.locks {
		var activeLocks []FileLock
		for _, lock := range r.locks[path] {
			if lock.ExpiresAt.After(now) {
				activeLocks = append(activeLocks, lock)
			} else {
				removed++
			}
		}

		if len(activeLocks) == 0 {
			delete(r.locks, path)
		} else {
			r.locks[path] = activeLocks
		}
	}

	return removed
}

// Registry is an alias for MemoryRegistry for backward compatibility
type Registry = MemoryRegistry

// NewRegistry creates a new in-memory file lock registry
var NewRegistry = NewMemoryRegistry
