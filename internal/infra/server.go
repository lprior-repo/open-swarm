// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package infra

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

const (
	// Default health check timeout duration
	// Increased from 10s to 30s to handle concurrent server bootstraps
	// where resource contention can delay startup
	defaultHealthTimeout = 30 * time.Second
	// Health check interval duration
	healthCheckInterval = 500 * time.Millisecond
	// HTTP status code for OK
	httpStatusOK = 200
	// Graceful shutdown timeout
	gracefulShutdownTimeout = 5 * time.Second
	// HTTP client timeout for health checks
	healthCheckClientTimeout = 2 * time.Second
	// HTTP client timeout for general requests
	generalClientTimeout = 2 * time.Second
	// Settling time after health check succeeds to allow full initialization
	serverSettlingTime = 2 * time.Second
	// Maximum number of concurrent server bootstraps
	// Limits resource contention when starting multiple servers
	maxConcurrentBootstraps = 4
)

var (
	// Global semaphore to limit concurrent server bootstraps
	bootstrapSemaphore = make(chan struct{}, maxConcurrentBootstraps)
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
	mu              sync.Mutex
}

// NewServerManager creates a new server manager
func NewServerManager() *ServerManager {
	return &ServerManager{
		opencodeCommand: "opencode",
		healthTimeout:   defaultHealthTimeout,
		healthInterval:  healthCheckInterval,
	}
}

// waitForHealth polls the health endpoint until ready or timeout.
// Returns nil when server responds with 200 OK, error on timeout.
func waitForHealth(ctx context.Context, baseURL string, timeout time.Duration, interval time.Duration) error {
	healthCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client := &http.Client{Timeout: healthCheckClientTimeout}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-healthCtx.Done():
			return fmt.Errorf("health check timed out after %v", timeout)
		case <-ticker.C:
			req, err := http.NewRequestWithContext(healthCtx, "GET", baseURL+"/health", nil)
			if err != nil {
				continue
			}
			resp, err := client.Do(req)
			if err != nil {
				continue
			}
			resp.Body.Close()
			if resp.StatusCode == httpStatusOK {
				// Health endpoint is responding, but OpenCode needs a moment
				// to fully initialize its session API (ACP server, plugins, etc).
				// Give it time to complete bootstrap before accepting requests.
				// Increased from 3s to 5s to handle concurrent server starts.
				time.Sleep(serverSettlingTime)
				return nil
			}
		}
	}
}

// BootServer starts an opencode server on the specified port and working directory
// INV-002: Agent Server working directory must be set to the Git Worktree
// INV-003: Supervisor must wait for Server Healthcheck (200 OK) before connecting SDK
func (sm *ServerManager) BootServer(ctx context.Context, worktreePath string, worktreeID string, port int) (*ServerHandle, error) {
	// Validate port is within safe range to prevent command injection
	if port < 1 || port > 65535 {
		return nil, fmt.Errorf("invalid port number: %d", port)
	}

	// Acquire semaphore to limit concurrent bootstraps
	// This prevents resource contention when starting multiple servers
	select {
	case bootstrapSemaphore <- struct{}{}:
		// Got semaphore, continue
		defer func() { <-bootstrapSemaphore }()
	case <-ctx.Done():
		return nil, fmt.Errorf("context cancelled while waiting for bootstrap slot: %w", ctx.Err())
	}

	// 1. Prepare Command: opencode serve --port X --hostname localhost
	// IMPORTANT: Use exec.Command (not CommandContext) so the server process
	// survives after the activity context is cancelled by Temporal
	cmd := exec.Command(sm.opencodeCommand, "serve",
		"--port", fmt.Sprintf("%d", port),
		"--hostname", "localhost",
	)

	// Set working directory to worktree (INV-002)
	cmd.Dir = worktreePath

	// 2. Setup logging for diagnostics
	// Create logs directory if it doesn't exist
	logsDir := filepath.Join(worktreePath, ".opencode-logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create logs directory: %w", err)
	}

	// Create log files for stdout and stderr
	stdoutLog, err := os.Create(filepath.Join(logsDir, fmt.Sprintf("server-%d.stdout.log", port)))
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout log: %w", err)
	}
	stderrLog, err := os.Create(filepath.Join(logsDir, fmt.Sprintf("server-%d.stderr.log", port)))
	if err != nil {
		stdoutLog.Close()
		return nil, fmt.Errorf("failed to create stderr log: %w", err)
	}

	// Redirect server output to log files
	// Note: We do NOT capture stdout/stderr directly as that breaks OpenCode's ACP protocol
	// Instead, we redirect to files which the server can still write to
	cmd.Stdout = stdoutLog
	cmd.Stderr = stderrLog

	// 3. Process Group Configuration (for clean kill)
	// This allows us to kill the entire process tree
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	// Start the server process
	if err := cmd.Start(); err != nil {
		stdoutLog.Close()
		stderrLog.Close()
		return nil, fmt.Errorf("failed to start opencode serve: %w", err)
	}

	pid := cmd.Process.Pid
	baseURL := fmt.Sprintf("http://localhost:%d", port)

	// Close log file handles - the process keeps them open
	// We close our handles so we don't leak file descriptors
	stdoutLog.Close()
	stderrLog.Close()

	// 4. Healthcheck (INV-003)
	// Wait for the server to become ready before returning
	if err := waitForHealth(ctx, baseURL, sm.healthTimeout, sm.healthInterval); err != nil {
		_ = sm.killProcess(cmd)
		return nil, fmt.Errorf("opencode server on port %d failed to become ready (check logs at %s): %w", port, logsDir, err)
	}

	return &ServerHandle{
		Port:       port,
		WorktreeID: worktreeID,
		WorkDir:    worktreePath,
		Cmd:        cmd,
		BaseURL:    baseURL,
		PID:        pid,
	}, nil
}

// Shutdown gracefully stops the opencode server
// INV-005: Server Process must be killed when Workflow Activity completes
func (sm *ServerManager) Shutdown(handle *ServerHandle) error {
	if handle == nil || handle.Cmd == nil || handle.Cmd.Process == nil {
		return fmt.Errorf("invalid server handle")
	}

	return sm.killProcess(handle.Cmd)
}

// ShutdownByPID stops the opencode server using only the PID
// This is used when the Cmd object is not available (e.g., after Temporal serialization)
func (sm *ServerManager) ShutdownByPID(pid int) error {
	if pid <= 0 {
		return fmt.Errorf("invalid PID: %d", pid)
	}

	// Try to get the process group ID
	pgid, err := syscall.Getpgid(pid)
	if err != nil {
		// Process might already be dead, that's OK
		return nil
	}

	// Send SIGTERM to process group first for graceful shutdown
	if err := syscall.Kill(-pgid, syscall.SIGTERM); err != nil {
		// If SIGTERM fails, use SIGKILL
		if killErr := syscall.Kill(-pgid, syscall.SIGKILL); killErr != nil {
			return fmt.Errorf("failed to kill process group %d: %w", pgid, killErr)
		}
	}

	return nil
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
		if killErr := cmd.Process.Kill(); killErr != nil {
			return fmt.Errorf("failed to kill process %d: %w", cmd.Process.Pid, killErr)
		}
		return nil
	}

	// Send SIGTERM to process group first for graceful shutdown
	if err := syscall.Kill(-pgid, syscall.SIGTERM); err != nil {
		// If SIGTERM fails, use SIGKILL
		if killErr := syscall.Kill(-pgid, syscall.SIGKILL); killErr != nil {
			return fmt.Errorf("failed to kill process group %d: %w", pgid, killErr)
		}
	}

	// Wait for process to exit (with timeout)
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-time.After(gracefulShutdownTimeout):
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
func (sm *ServerManager) IsHealthy(ctx context.Context, handle *ServerHandle) bool {
	if handle == nil {
		return false
	}

	client := &http.Client{Timeout: generalClientTimeout}
	req, err := http.NewRequestWithContext(ctx, "GET", handle.BaseURL+"/health", nil)
	if err != nil {
		return false
	}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == httpStatusOK
}

// SetOpencodeCommand allows overriding the opencode command (useful for testing)
func (sm *ServerManager) SetOpencodeCommand(cmd string) {
	sm.opencodeCommand = cmd
}

// SetHealthTimeout sets the health check timeout duration
func (sm *ServerManager) SetHealthTimeout(timeout time.Duration) {
	sm.healthTimeout = timeout
}
