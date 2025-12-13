// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package temporal

import (
	"sync"

	"open-swarm/internal/filelock"
	"open-swarm/internal/infra"
)

var (
	globalPortManager      *infra.PortManager
	globalServerManager    *infra.ServerManager
	globalWorktreeManager  *infra.WorktreeManager
	globalFileLockRegistry *filelock.MemoryRegistry
	initOnce               sync.Once
)

// InitializeGlobals sets up shared infrastructure managers
// Called once per worker process
//
// Since PortManager cannot be serialized across Temporal activities,
// we use global singletons that are shared across all activities
// within the same worker process.
//
// This works because:
// - All workers run in same process (single binary deployment)
// - PortManager state stays in memory
// - No serialization needed
// - Scales to 50 agents per host
func InitializeGlobals(portMin, portMax int, repoDir, worktreeBase string) {
	initOnce.Do(func() {
		globalPortManager = infra.NewPortManager(portMin, portMax)
		globalServerManager = infra.NewServerManager()
		globalWorktreeManager = infra.NewWorktreeManager(repoDir, worktreeBase)
		globalFileLockRegistry = filelock.NewMemoryRegistry()
	})
}

// GetManagers returns the global infrastructure managers
// Must be called after InitializeGlobals
func GetManagers() (*infra.PortManager, *infra.ServerManager, *infra.WorktreeManager) {
	return globalPortManager, globalServerManager, globalWorktreeManager
}

// GetFileLockRegistry returns the global file lock registry
// Must be called after InitializeGlobals
func GetFileLockRegistry() *filelock.MemoryRegistry {
	return globalFileLockRegistry
}
