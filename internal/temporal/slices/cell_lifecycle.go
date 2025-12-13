// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

// Package slices provides vertical slice architecture for Open Swarm workflows.
//
// cell_lifecycle.go: Complete vertical slice for cell lifecycle management
// - Cell bootstrap (port allocation, worktree creation, server startup)
// - Cell teardown (cleanup and resource release)
// - Health monitoring
//
// This slice follows CUPID principles:
// - Composable: Self-contained cell management operations
// - Unix philosophy: Does cell lifecycle management and nothing else
// - Predictable: Clear input/output, guaranteed cleanup
// - Idiomatic: Go error handling, Temporal patterns
// - Domain-centric: Organized around cell capability
package slices

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/activity"

	"open-swarm/internal/agent"
	"open-swarm/internal/infra"
)

// ============================================================================
// ACTIVITIES
// ============================================================================

// CellLifecycleActivities handles all cell lifecycle operations
type CellLifecycleActivities struct {
	portManager     infra.PortManagerInterface
	serverManager   infra.ServerManagerInterface
	worktreeManager infra.WorktreeManagerInterface
}

// NewCellLifecycleActivities creates a new cell lifecycle activities instance
func NewCellLifecycleActivities(
	portMgr infra.PortManagerInterface,
	serverMgr infra.ServerManagerInterface,
	worktreeMgr infra.WorktreeManagerInterface,
) *CellLifecycleActivities {
	return &CellLifecycleActivities{
		portManager:     portMgr,
		serverManager:   serverMgr,
		worktreeManager: worktreeMgr,
	}
}

// BootstrapCell creates an isolated OpenCode cell with allocated resources
//
// This activity orchestrates the complete cell bootstrap sequence:
// 1. Port allocation (INV-001: unique port in 8000-9000 range)
// 2. Worktree creation (isolated filesystem)
// 3. Server startup (INV-002: health check before return)
// 4. SDK client setup (INV-004: HTTP connection)
//
// On error, all allocated resources are automatically cleaned up.
// Returns serializable BootstrapOutput for workflow state persistence.
func (c *CellLifecycleActivities) BootstrapCell(ctx context.Context, input BootstrapInput) (*BootstrapOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Bootstrapping cell", "cellID", input.CellID, "branch", input.Branch)

	activity.RecordHeartbeat(ctx, "allocating port")

	// 1. Allocate Port (INV-001: PortManager guarantees uniqueness)
	port, err := c.portManager.Allocate()
	if err != nil {
		return nil, fmt.Errorf("failed to allocate port for cell %q: %w", input.CellID, err)
	}

	// Track port for cleanup on error
	var cleanupPort = true
	defer func() {
		if cleanupPort {
			if releaseErr := c.portManager.Release(port); releaseErr != nil {
				logger.Error("Failed to release port during cleanup", "port", port, "error", releaseErr)
			}
		}
	}()

	activity.RecordHeartbeat(ctx, "creating worktree")

	// 2. Create Worktree (isolated filesystem)
	worktreeID := generateWorktreeID(input.CellID)
	worktree, err := c.worktreeManager.CreateWorktree(worktreeID, input.Branch)
	if err != nil {
		return nil, fmt.Errorf("failed to create worktree for cell %q: %w", input.CellID, err)
	}

	// Track worktree for cleanup on error
	var cleanupWorktree = true
	defer func() {
		if cleanupWorktree {
			if removeErr := c.worktreeManager.RemoveWorktree(worktreeID); removeErr != nil {
				logger.Error("Failed to remove worktree during cleanup", "worktreeID", worktreeID, "error", removeErr)
			}
		}
	}()

	activity.RecordHeartbeat(ctx, "booting server")

	// 3. Boot Server (INV-002: health check, INV-003: timeout)
	serverHandle, err := c.serverManager.BootServer(ctx, worktree.Path, worktreeID, port)
	if err != nil {
		return nil, fmt.Errorf("failed to boot server for cell %q: %w", input.CellID, err)
	}

	// Track server for cleanup on error
	var cleanupServer = true
	defer func() {
		if cleanupServer {
			if shutdownErr := c.serverManager.Shutdown(serverHandle); shutdownErr != nil {
				logger.Error("Failed to shutdown server during cleanup", "port", port, "error", shutdownErr)
			}
		}
	}()

	// Success - disable cleanup
	cleanupPort = false
	cleanupWorktree = false
	cleanupServer = false

	logger.Info("Cell bootstrapped successfully",
		"cellID", input.CellID,
		"port", port,
		"worktreeID", worktreeID,
		"pid", serverHandle.PID)

	// Return serializable output for workflow persistence
	return &BootstrapOutput{
		CellID:       input.CellID,
		Port:         port,
		WorktreeID:   worktreeID,
		WorktreePath: worktree.Path,
		BaseURL:      serverHandle.BaseURL,
		ServerPID:    serverHandle.PID,
	}, nil
}

// TeardownCell destroys a cell and releases all allocated resources
//
// This activity implements the Saga pattern cleanup:
// 1. Shutdown server (INV-005: kill process)
// 2. Remove worktree (cleanup filesystem)
// 3. Release port (return to pool)
//
// All cleanup steps are attempted even if earlier steps fail.
// Multiple errors are collected and returned together.
func (c *CellLifecycleActivities) TeardownCell(ctx context.Context, output BootstrapOutput) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Tearing down cell", "cellID", output.CellID)

	var errs []error

	// 1. Shutdown Server (INV-005: process must be killed)
	if output.ServerPID != 0 {
		serverHandle := &infra.ServerHandle{
			Port:    output.Port,
			BaseURL: output.BaseURL,
			PID:     output.ServerPID,
		}
		if err := c.serverManager.Shutdown(serverHandle); err != nil {
			errs = append(errs, fmt.Errorf("failed to shutdown server (PID %d): %w", output.ServerPID, err))
		} else {
			logger.Info("Server shutdown completed", "cellID", output.CellID, "pid", output.ServerPID)
		}
	}

	// 2. Remove Worktree (cleanup filesystem)
	if output.WorktreeID != "" {
		if err := c.worktreeManager.RemoveWorktree(output.WorktreeID); err != nil {
			errs = append(errs, fmt.Errorf("failed to remove worktree %q: %w", output.WorktreeID, err))
		} else {
			logger.Info("Worktree removed", "cellID", output.CellID, "worktreeID", output.WorktreeID)
		}
	}

	// 3. Release Port (return to pool)
	if output.Port != 0 {
		if err := c.portManager.Release(output.Port); err != nil {
			errs = append(errs, fmt.Errorf("failed to release port %d: %w", output.Port, err))
		} else {
			logger.Info("Port released", "cellID", output.CellID, "port", output.Port)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("teardown errors for cell %q: %v", output.CellID, errs)
	}

	logger.Info("Cell teardown completed successfully", "cellID", output.CellID)
	return nil
}

// HealthCheckCell verifies that a cell is healthy and ready for operations
//
// This activity checks:
// - Server process is running
// - Server responds to health endpoint
// - SDK connection is working
func (c *CellLifecycleActivities) HealthCheckCell(ctx context.Context, output BootstrapOutput) (bool, error) {
	logger := activity.GetLogger(ctx)

	serverHandle := &infra.ServerHandle{
		Port:    output.Port,
		BaseURL: output.BaseURL,
		PID:     output.ServerPID,
	}

	healthy := c.serverManager.IsHealthy(ctx, serverHandle)
	if !healthy {
		logger.Warn("Health check failed", "cellID", output.CellID, "port", output.Port)
		return false, fmt.Errorf("cell %q is not healthy", output.CellID)
	}

	logger.Info("Health check passed", "cellID", output.CellID)
	return true, nil
}

// ============================================================================
// BUSINESS LOGIC
// ============================================================================

// generateWorktreeID creates a unique worktree identifier
// Format: cell-{cellID}-{timestamp}
func generateWorktreeID(cellID string) string {
	return fmt.Sprintf("cell-%s-%d", cellID, time.Now().Unix())
}

// ReconstructClient creates an agent client from bootstrap output
// This is used by other activities that need to interact with the cell
func ReconstructClient(output BootstrapOutput) agent.ClientInterface {
	return agent.NewClient(output.BaseURL, output.Port)
}

// ============================================================================
// WORKFLOW HELPERS
// ============================================================================

// ValidateBootstrapInput checks if bootstrap input is valid
func ValidateBootstrapInput(input BootstrapInput) error {
	if input.CellID == "" {
		return fmt.Errorf("cellID cannot be empty")
	}
	if input.Branch == "" {
		return fmt.Errorf("branch cannot be empty")
	}
	return nil
}

// ValidateBootstrapOutput checks if bootstrap output is valid
func ValidateBootstrapOutput(output BootstrapOutput) error {
	if output.CellID == "" {
		return fmt.Errorf("cellID cannot be empty")
	}
	if output.Port == 0 {
		return fmt.Errorf("port cannot be zero")
	}
	if output.WorktreeID == "" {
		return fmt.Errorf("worktreeID cannot be empty")
	}
	if output.WorktreePath == "" {
		return fmt.Errorf("worktreePath cannot be empty")
	}
	if output.BaseURL == "" {
		return fmt.Errorf("baseURL cannot be empty")
	}
	if output.ServerPID == 0 {
		return fmt.Errorf("serverPID cannot be zero")
	}
	return nil
}
