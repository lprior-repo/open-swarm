// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package filelock

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestAcquire_Exclusive tests that exclusive locks prevent other locks
func TestAcquire_Exclusive(t *testing.T) {
	registry := NewMemoryRegistry()

	// Acquire an exclusive lock
	result, err := registry.Acquire(LockRequest{
		Path:      "/file.txt",
		Holder:    "agent1",
		Exclusive: true,
		TTL:       1 * time.Hour,
	})
	assert.NoError(t, err)
	assert.True(t, result.Granted)

	// Verify exclusive lock exists
	locks := registry.Check("/file.txt")
	assert.Len(t, locks, 1)
	assert.True(t, locks[0].Exclusive)
	assert.Equal(t, "agent1", locks[0].Holder)

	// Try to acquire another exclusive lock on same path - should fail
	result, err = registry.Acquire(LockRequest{
		Path:      "/file.txt",
		Holder:    "agent2",
		Exclusive: true,
		TTL:       1 * time.Hour,
	})
	assert.Error(t, err)
	assert.False(t, result.Granted)
	assert.IsType(t, &ConflictError{}, err)

	// Try to acquire shared lock - should fail
	result, err = registry.Acquire(LockRequest{
		Path:      "/file.txt",
		Holder:    "agent3",
		Exclusive: false,
		TTL:       1 * time.Hour,
	})
	assert.Error(t, err)
	assert.False(t, result.Granted)
	assert.IsType(t, &ConflictError{}, err)
}

// TestAcquire_Shared tests that shared locks are compatible with each other
func TestAcquire_Shared(t *testing.T) {
	registry := NewMemoryRegistry()

	// Acquire first shared lock
	result, err := registry.Acquire(LockRequest{
		Path:      "/file.txt",
		Holder:    "agent1",
		Exclusive: false,
		TTL:       1 * time.Hour,
	})
	assert.NoError(t, err)
	assert.True(t, result.Granted)

	// Acquire second shared lock on same path
	result, err = registry.Acquire(LockRequest{
		Path:      "/file.txt",
		Holder:    "agent2",
		Exclusive: false,
		TTL:       1 * time.Hour,
	})
	assert.NoError(t, err)
	assert.True(t, result.Granted)

	// Acquire third shared lock on same path
	result, err = registry.Acquire(LockRequest{
		Path:      "/file.txt",
		Holder:    "agent3",
		Exclusive: false,
		TTL:       1 * time.Hour,
	})
	assert.NoError(t, err)
	assert.True(t, result.Granted)

	// Verify all locks exist
	locks := registry.Check("/file.txt")
	assert.Len(t, locks, 3)
	for _, lock := range locks {
		assert.False(t, lock.Exclusive)
	}
}

// TestAcquire_Conflict tests that conflicts return proper error
func TestAcquire_Conflict(t *testing.T) {
	registry := NewMemoryRegistry()

	// Test exclusive blocking shared
	result, err := registry.Acquire(LockRequest{
		Path:      "/exclusive.txt",
		Holder:    "agent1",
		Exclusive: true,
		TTL:       1 * time.Hour,
	})
	assert.NoError(t, err)
	assert.True(t, result.Granted)

	result, err = registry.Acquire(LockRequest{
		Path:      "/exclusive.txt",
		Holder:    "agent2",
		Exclusive: false,
		TTL:       1 * time.Hour,
	})
	assert.Error(t, err)
	assert.False(t, result.Granted)
	conflictErr, ok := err.(*ConflictError)
	assert.True(t, ok)
	assert.Equal(t, "/exclusive.txt", conflictErr.RequestedPath)

	// Test shared blocking exclusive
	registry2 := NewMemoryRegistry()
	result, err = registry2.Acquire(LockRequest{
		Path:      "/shared.txt",
		Holder:    "agent1",
		Exclusive: false,
		TTL:       1 * time.Hour,
	})
	assert.NoError(t, err)
	assert.True(t, result.Granted)

	result, err = registry2.Acquire(LockRequest{
		Path:      "/shared.txt",
		Holder:    "agent2",
		Exclusive: true,
		TTL:       1 * time.Hour,
	})
	assert.Error(t, err)
	assert.False(t, result.Granted)
	conflictErr, ok = err.(*ConflictError)
	assert.True(t, ok)
	assert.Len(t, conflictErr.ExistingLocks, 1)
}

// TestRelease tests that lock release frees the path
func TestRelease(t *testing.T) {
	registry := NewMemoryRegistry()

	// Acquire locks
	result, err := registry.Acquire(LockRequest{
		Path:      "/file.txt",
		Holder:    "agent1",
		Exclusive: true,
		TTL:       1 * time.Hour,
	})
	assert.NoError(t, err)
	assert.True(t, result.Granted)

	locks := registry.Check("/file.txt")
	assert.Len(t, locks, 1)

	// Release the lock
	err = registry.Release("/file.txt", "agent1")
	assert.NoError(t, err)

	// Verify lock is gone
	locks = registry.Check("/file.txt")
	assert.Len(t, locks, 0)

	// Try to release again - should fail
	err = registry.Release("/file.txt", "agent1")
	assert.Error(t, err)
}

// TestExpiration tests that expired locks are cleaned up
func TestExpiration(t *testing.T) {
	registry := NewMemoryRegistry()

	// Acquire lock with very short TTL
	result, err := registry.Acquire(LockRequest{
		Path:      "/file1.txt",
		Holder:    "agent1",
		Exclusive: false,
		TTL:       1 * time.Millisecond,
	})
	assert.NoError(t, err)
	assert.True(t, result.Granted)

	// Acquire lock with long TTL
	result, err = registry.Acquire(LockRequest{
		Path:      "/file2.txt",
		Holder:    "agent2",
		Exclusive: false,
		TTL:       1 * time.Hour,
	})
	assert.NoError(t, err)
	assert.True(t, result.Granted)

	// Wait for first lock to expire
	time.Sleep(20 * time.Millisecond)

	// Check returns only non-expired locks
	locks := registry.Check("/file1.txt")
	assert.Len(t, locks, 0)

	locks = registry.Check("/file2.txt")
	assert.Len(t, locks, 1)

	// Cleanup expired locks
	count := registry.CleanupExpired()
	assert.Greater(t, count, 0)

	// Verify cleanup removed expired locks
	locks = registry.Check("/file1.txt")
	assert.Len(t, locks, 0)

	locks = registry.Check("/file2.txt")
	assert.Len(t, locks, 1)
}

// TestConcurrent tests thread safety with concurrent operations
func TestConcurrent(t *testing.T) {
	registry := NewMemoryRegistry()

	const numGoroutines = 10
	const numIterations = 10

	var wg sync.WaitGroup
	var successCount atomic.Int32

	// Concurrent acquires
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numIterations; j++ {
				path := "/concurrent/" + string(rune(j%5))
				holder := "agent-" + string(rune(id))
				result, err := registry.Acquire(LockRequest{
					Path:      path,
					Holder:    holder,
					Exclusive: false,
					TTL:       1 * time.Hour,
				})
				if err == nil && result.Granted {
					successCount.Add(1)
				}
			}
		}(i)
	}

	wg.Wait()

	// Should have at least some successful acquisitions
	assert.Greater(t, int(successCount.Load()), 0)

	// Concurrent releases
	registry2 := NewMemoryRegistry()
	for i := 0; i < 5; i++ {
		registry2.Acquire(LockRequest{
			Path:      "/file" + string(rune(i)),
			Holder:    "agent" + string(rune(i)),
			Exclusive: false,
			TTL:       1 * time.Hour,
		})
	}

	successCount.Store(0)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			err := registry2.Release("/file"+string(rune(id)), "agent"+string(rune(id)))
			if err == nil {
				successCount.Add(1)
			}
		}(i)
	}

	wg.Wait()
	assert.Equal(t, int32(5), successCount.Load())
}

// TestGlobPatterns tests pattern matching for file paths
func TestGlobPatterns(t *testing.T) {
	registry := NewMemoryRegistry()

	// Acquire lock with glob pattern
	result, err := registry.Acquire(LockRequest{
		Path:      "/data/*.txt",
		Holder:    "agent1",
		Exclusive: true,
		TTL:       1 * time.Hour,
	})
	assert.NoError(t, err)
	assert.True(t, result.Granted)

	// Matching patterns should conflict
	result, err = registry.Acquire(LockRequest{
		Path:      "/data/file.txt",
		Holder:    "agent2",
		Exclusive: false,
		TTL:       1 * time.Hour,
	})
	assert.Error(t, err)
	assert.False(t, result.Granted)

	// Non-matching patterns should work
	result, err = registry.Acquire(LockRequest{
		Path:      "/data/file.md",
		Holder:    "agent2",
		Exclusive: false,
		TTL:       1 * time.Hour,
	})
	assert.NoError(t, err)
	assert.True(t, result.Granted)

	// Inverse pattern test
	registry2 := NewMemoryRegistry()
	result, err = registry2.Acquire(LockRequest{
		Path:      "/data/file.txt",
		Holder:    "agent1",
		Exclusive: true,
		TTL:       1 * time.Hour,
	})
	assert.NoError(t, err)
	assert.True(t, result.Granted)

	// Glob pattern matching this specific file should conflict
	result, err = registry2.Acquire(LockRequest{
		Path:      "/data/*.txt",
		Holder:    "agent2",
		Exclusive: false,
		TTL:       1 * time.Hour,
	})
	assert.Error(t, err)
	assert.False(t, result.Granted)
}

// TestRenewLock tests lock TTL extension
func TestRenewLock(t *testing.T) {
	registry := NewMemoryRegistry()

	// Acquire lock with short TTL
	result, err := registry.Acquire(LockRequest{
		Path:      "/file.txt",
		Holder:    "agent1",
		Exclusive: false,
		TTL:       100 * time.Millisecond,
	})
	assert.NoError(t, err)
	assert.True(t, result.Granted)

	locks := registry.Check("/file.txt")
	originalExpiry := locks[0].ExpiresAt

	// Renew lock with longer TTL
	err = registry.RenewLock("/file.txt", "agent1", 2*time.Hour)
	assert.NoError(t, err)

	locks = registry.Check("/file.txt")
	assert.Len(t, locks, 1)
	assert.True(t, locks[0].ExpiresAt.After(originalExpiry.Add(1*time.Hour)))

	// Try to renew non-existent lock
	err = registry.RenewLock("/file.txt", "nonexistent", 1*time.Hour)
	assert.Error(t, err)

	// Try to renew expired lock
	registry2 := NewMemoryRegistry()
	registry2.Acquire(LockRequest{
		Path:      "/expired.txt",
		Holder:    "agent1",
		Exclusive: false,
		TTL:       1 * time.Millisecond,
	})
	time.Sleep(20 * time.Millisecond)
	err = registry2.RenewLock("/expired.txt", "agent1", 1*time.Hour)
	assert.Error(t, err)
}
