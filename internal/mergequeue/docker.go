package mergequeue

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

const (
	// dockerStopTimeout is the timeout for gracefully stopping a container
	dockerStopTimeout = 10 * time.Second
)

// DockerManager handles Docker container lifecycle operations
type DockerManager struct {
	client *client.Client
}

// NewDockerManager creates a new Docker manager with the default client
func NewDockerManager() (*DockerManager, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	return &DockerManager{
		client: cli,
	}, nil
}

// Close closes the Docker client connection
func (dm *DockerManager) Close() error {
	if dm.client != nil {
		return dm.client.Close()
	}
	return nil
}

// StopAndRemoveContainer stops and removes a Docker container by ID
// This function is idempotent - it will not error if the container doesn't exist
func (dm *DockerManager) StopAndRemoveContainer(ctx context.Context, containerID string) error {
	if containerID == "" {
		return nil // No container to clean up
	}

	// Try to stop the container gracefully
	timeout := int(dockerStopTimeout.Seconds())
	stopOptions := container.StopOptions{
		Timeout: &timeout,
	}
	if err := dm.client.ContainerStop(ctx, containerID, stopOptions); err != nil {
		// Container might already be stopped or doesn't exist - not a fatal error
		// Log but continue to removal
	}

	// Remove the container
	removeOptions := container.RemoveOptions{
		Force:         true, // Force removal even if running
		RemoveVolumes: true, // Remove associated volumes
	}
	if err := dm.client.ContainerRemove(ctx, containerID, removeOptions); err != nil {
		// Check if error is "not found" - that's okay (idempotent)
		if client.IsErrNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to remove container %s: %w", containerID, err)
	}

	return nil
}

// IsContainerRunning checks if a container is currently running
func (dm *DockerManager) IsContainerRunning(ctx context.Context, containerID string) (bool, error) {
	if containerID == "" {
		return false, nil
	}

	inspect, err := dm.client.ContainerInspect(ctx, containerID)
	if err != nil {
		if client.IsErrNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to inspect container %s: %w", containerID, err)
	}

	return inspect.State.Running, nil
}

// GetContainerLogs retrieves the last N lines of logs from a container
func (dm *DockerManager) GetContainerLogs(ctx context.Context, containerID string, tail int) (string, error) {
	if containerID == "" {
		return "", nil
	}

	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       fmt.Sprintf("%d", tail),
	}

	logs, err := dm.client.ContainerLogs(ctx, containerID, options)
	if err != nil {
		if client.IsErrNotFound(err) {
			return "", nil
		}
		return "", fmt.Errorf("failed to get logs for container %s: %w", containerID, err)
	}
	defer logs.Close()

	// Read logs into string
	buf := make([]byte, 4096)
	var result string
	for {
		n, err := logs.Read(buf)
		if n > 0 {
			result += string(buf[:n])
		}
		if err != nil {
			break
		}
	}

	return result, nil
}
