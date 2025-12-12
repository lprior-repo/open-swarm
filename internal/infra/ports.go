package infra

import (
	"fmt"
	"sync"
)

// PortManager manages the allocation of ports in the range 8000-9000
// Enforces INV-001: Each Agent runs 'opencode serve' on a unique port
type PortManager struct {
	mu        sync.Mutex
	minPort   int
	maxPort   int
	allocated map[int]bool
	nextPort  int
}

// NewPortManager creates a new port manager with the specified range
func NewPortManager(minPort, maxPort int) *PortManager {
	return &PortManager{
		minPort:   minPort,
		maxPort:   maxPort,
		allocated: make(map[int]bool),
		nextPort:  minPort,
	}
}

// Allocate reserves the next available port
func (pm *PortManager) Allocate() (int, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Scan for available port starting from nextPort
	for i := 0; i < (pm.maxPort - pm.minPort + 1); i++ {
		candidatePort := pm.minPort + ((pm.nextPort - pm.minPort + i) % (pm.maxPort - pm.minPort + 1))

		if !pm.allocated[candidatePort] {
			pm.allocated[candidatePort] = true
			pm.nextPort = candidatePort + 1
			if pm.nextPort > pm.maxPort {
				pm.nextPort = pm.minPort
			}
			return candidatePort, nil
		}
	}

	return 0, fmt.Errorf("no available ports in range %d-%d (all %d ports allocated)",
		pm.minPort, pm.maxPort, pm.maxPort-pm.minPort+1)
}

// Release frees a previously allocated port
func (pm *PortManager) Release(port int) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if port < pm.minPort || port > pm.maxPort {
		return fmt.Errorf("port %d is outside valid range %d-%d", port, pm.minPort, pm.maxPort)
	}

	if !pm.allocated[port] {
		return fmt.Errorf("port %d was not allocated", port)
	}

	delete(pm.allocated, port)
	return nil
}

// AllocatedCount returns the number of currently allocated ports
func (pm *PortManager) AllocatedCount() int {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return len(pm.allocated)
}

// IsAllocated checks if a specific port is currently allocated
func (pm *PortManager) IsAllocated(port int) bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.allocated[port]
}

// AvailableCount returns the number of available ports
func (pm *PortManager) AvailableCount() int {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return (pm.maxPort - pm.minPort + 1) - len(pm.allocated)
}
