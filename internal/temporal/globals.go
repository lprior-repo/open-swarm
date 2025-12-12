package temporal

import (
	"sync"

	"open-swarm/internal/infra"
)

var (
	globalPortManager     *infra.PortManager
	globalServerManager   *infra.ServerManager
	globalWorktreeManager *infra.WorktreeManager
	initOnce              sync.Once
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
	})
}

// GetManagers returns the global infrastructure managers
// Must be called after InitializeGlobals
func GetManagers() (*infra.PortManager, *infra.ServerManager, *infra.WorktreeManager) {
	return globalPortManager, globalServerManager, globalWorktreeManager
}
