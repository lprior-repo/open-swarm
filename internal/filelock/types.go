// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

// Package filelock provides distributed file locking mechanisms for agent coordination.
package filelock

import (
	"errors"
	"time"
)

// FileLock represents a lock held on a file by an agent.
//
// Locking Semantics:
//   - Exclusive locks: Only one agent can hold an exclusive lock on a file.
//     When an exclusive lock is active, no other agent can acquire any lock (exclusive or shared).
//   - Shared locks: Multiple agents can hold shared locks on the same file simultaneously.
//     However, no agent can acquire an exclusive lock while shared locks exist.
//   - Lock expiration: Locks automatically expire at ExpiresAt time. Agents should use
//     RenewLock to extend lock duration before expiration.
type FileLock struct {
	// Path is the absolute path to the locked file
	Path string

	// Holder is the name/identifier of the agent holding this lock
	Holder string

	// Exclusive indicates whether this is an exclusive lock (true) or shared lock (false)
	Exclusive bool

	// ExpiresAt is the time when this lock automatically expires
	ExpiresAt time.Time

	// AcquiredAt is the time when this lock was first acquired
	AcquiredAt time.Time
}

// LockRequest represents a request to acquire a file lock.
type LockRequest struct {
	// Path is the absolute path to the file to lock
	Path string

	// Holder is the name/identifier of the agent requesting the lock
	Holder string

	// Exclusive indicates whether to request an exclusive lock (true) or shared lock (false)
	Exclusive bool

	// TTL is the time-to-live duration for the lock
	TTL time.Duration
}

// LockResult represents the result of an attempt to acquire a lock.
type LockResult struct {
	// Granted indicates whether the lock was successfully acquired
	Granted bool

	// Conflicts contains any existing locks that prevented acquisition (if Granted is false)
	Conflicts []FileLock

	// Lock is the acquired lock (only valid if Granted is true)
	Lock *FileLock
}

// LockRegistry manages file locks for agent coordination.
//
// Locking Semantics:
//   - Acquire: Returns ConflictError if the lock cannot be granted. Exclusive locks
//     prevent any other locks. Shared locks can coexist but prevent exclusive locks.
//   - Release: Removes a lock held by an agent. Only the holding agent can release its own lock.
//   - Check: Returns the current lock state without acquiring or modifying.
//   - RenewLock: Extends the expiration time of an existing lock without requiring re-acquisition.
//   - CleanupExpired: Removes all locks that have passed their expiration time.
type LockRegistry interface {
	// Acquire attempts to acquire a lock on the specified file.
	// Returns ConflictError if the lock cannot be granted.
	Acquire(req LockRequest) (LockResult, error)

	// Release removes a lock held by the specified agent on the given file.
	// Returns an error if the lock is not held by the specified agent or does not exist.
	Release(path, holder string) error

	// Check returns information about locks on a file without acquiring or modifying them.
	Check(path string) []FileLock

	// RenewLock extends the expiration time of an existing lock.
	// Returns an error if the lock does not exist or is not held by the specified agent.
	RenewLock(path, holder string, newTTL time.Duration) error

	// CleanupExpired removes all locks that have passed their expiration time.
	// Returns the number of locks removed.
	CleanupExpired() int
}

// ErrLockNotFound is returned when attempting to operate on a non-existent lock.
var ErrLockNotFound = errors.New("lock not found")

// ErrLockNotHeld is returned when attempting to release a lock not held by the specified agent.
var ErrLockNotHeld = errors.New("lock not held by agent")

// ConflictError represents a conflict when acquiring a lock.
type ConflictError struct {
	// Path is the path that was requested
	Path string

	// Holder is the agent holding the conflicting lock
	Holder string

	// Exclusive indicates whether the conflicting lock is exclusive
	Exclusive bool

	// ExistingLocks are the locks that caused the conflict
	ExistingLocks []FileLock

	// RequestedPath is the path that was requested (compatibility)
	RequestedPath string

	// RequestedHolder is the agent that requested the lock (compatibility)
	RequestedHolder string

	// IsExclusive indicates whether an exclusive lock was requested (compatibility)
	IsExclusive bool
}

// Error implements the error interface.
func (e *ConflictError) Error() string {
	return "lock conflict: cannot acquire lock on " + e.Path
}
