// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package infra

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"syscall"
	"time"
)

// ServerHandle represents a running opencode server instance
// Enforces INV-002: Working directory must be set to Git Worktree
// Enforces INV-003: Must wait for healthcheck before SDK connection
type ServerHandle struct {
	Port       int
	WorktreeID string
	WorkDir    string
	Cmd        *exec.Cmd
	BaseURL    string
	PID        int
}

// ServerManager handles the lifecycle of opencode serve processes
type ServerManager struct {
	opencodeCommand string
	healthTimeout   time.Duration
	healthInterval  time.Duration
}

// NewServerManager creates a new server manager
func NewServerManager() *ServerManager {
	return &ServerManager{
		opencodeCommand: "opencode",
		healthTimeout:   10 * time.Second,
		healthInterval:  200 * time.Millisecond,
	}
}

// BootServer starts an opencode server on the specified port and working directory
// INV-002: Agent Server working directory must be set to the Git Worktree
// INV-003: Supervisor must wait for Server Healthcheck (200 OK) before connecting SDK
func (sm *ServerManager) BootServer(ctx context.Context, worktreePath string, worktreeID string, port int) (*ServerHandle, error) {
	// 1. Prepare Command: opencode serve --port X --hostname localhost
	cmd := exec.CommandContext(ctx, sm.opencodeCommand, "serve",
		"--port", fmt.Sprintf("%d", port),
		"--hostname", "localhost",
	)

	// Set working directory to worktree (INV-002)
	cmd.Dir = worktreePath

	// 2. Process Group Configuration (for clean kill)
	// This allows us to kill the entire process tree
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	// Start the server process
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start opencode serve: %w", err)
	}

	pid := cmd.Process.Pid
	baseURL := fmt.Sprintf("http://localhost:%d", port)

	// 3. Healthcheck Loop (INV-003)
	// Wait for the server to become ready before returning
	healthCtx, cancel := context.WithTimeout(ctx, sm.healthTimeout)
	defer cancel()

	ready := false
	client := &http.Client{Timeout: 1 * time.Second}
	ticker := time.NewTicker(sm.healthInterval)
	defer ticker.Stop()

	for {
		select {
		case <-healthCtx.Done():
			// Timeout - kill the server and return error
			_ = sm.killProcess(cmd)
			return nil, fmt.Errorf("opencode server on port %d failed to become ready within %v", port, sm.healthTimeout)

		case <-ticker.C:
			// Check health endpoint
			resp, err := client.Get(baseURL + "/health")
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode == 200 {
					ready = true
				}
			}

			if ready {
				handle := &ServerHandle{
					Port:       port,
					WorktreeID: worktreeID,
					WorkDir:    worktreePath,
					Cmd:        cmd,
					BaseURL:    baseURL,
					PID:        pid,
				}
				return handle, nil
			}
		}
	}
}

// Shutdown gracefully stops the opencode server
// INV-005: Server Process must be killed when Workflow Activity completes
func (sm *ServerManager) Shutdown(handle *ServerHandle) error {
	if handle == nil || handle.Cmd == nil || handle.Cmd.Process == nil {
		return fmt.Errorf("invalid server handle")
	}

	return sm.killProcess(handle.Cmd)
}

// killProcess terminates the process and its entire process group
func (sm *ServerManager) killProcess(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}

	// Kill the entire process group (negative PID)
	// This ensures all child processes are terminated
	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err != nil {
		// Fallback to killing just the main process
		return cmd.Process.Kill()
	}

	// Send SIGTERM to process group first for graceful shutdown
	if err := syscall.Kill(-pgid, syscall.SIGTERM); err != nil {
		// If SIGTERM fails, use SIGKILL
		return syscall.Kill(-pgid, syscall.SIGKILL)
	}

	// Wait for process to exit (with timeout)
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-time.After(5 * time.Second):
		// Force kill if graceful shutdown takes too long
		_ = syscall.Kill(-pgid, syscall.SIGKILL)
		return fmt.Errorf("server shutdown timed out, force killed")
	case err := <-done:
		if err != nil && err.Error() != "signal: terminated" && err.Error() != "signal: killed" {
			return err
		}
		return nil
	}
}

// IsHealthy checks if the server is still responsive
func (sm *ServerManager) IsHealthy(handle *ServerHandle) bool {
	if handle == nil {
		return false
	}

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(handle.BaseURL + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200
}

// SetOpencodeCommand allows overriding the opencode command (useful for testing)
func (sm *ServerManager) SetOpencodeCommand(cmd string) {
	sm.opencodeCommand = cmd
}

// SetHealthTimeout sets the health check timeout duration
func (sm *ServerManager) SetHealthTimeout(timeout time.Duration) {
	sm.healthTimeout = timeout
}
