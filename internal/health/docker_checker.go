package health

import (
	"context"
	"fmt"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

type dockerClient interface {
	ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error)
	Close() error
}

type DockerChecker struct {
	client dockerClient
}

func NewDockerChecker() (*DockerChecker, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}
	return &DockerChecker{client: dockerClientWrapper{cli}}, nil
}

type dockerClientWrapper struct {
	client *client.Client
}

func (w dockerClientWrapper) ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error) {
	return w.client.ContainerInspect(ctx, containerID)
}

func (w dockerClientWrapper) Close() error {
	return w.client.Close()
}

var _ dockerClient = dockerClientWrapper{}
var _ io.Closer = (*DockerChecker)(nil)

func (c *DockerChecker) Execute(ctx context.Context, check Check) (passed bool, message string, err error) {
	info, err := c.client.ContainerInspect(ctx, check.Container)
	if err != nil {
		return false, fmt.Sprintf("container %q not found or inaccessible", check.Container), nil
	}

	if !info.State.Running {
		return false, fmt.Sprintf("container %q is not running (status: %s)", check.Container, info.State.Status), nil
	}

	if info.State.Health == nil {
		return true, fmt.Sprintf("container %q is running (no healthcheck configured)", check.Container), nil
	}

	switch info.State.Health.Status {
	case container.Healthy:
		return true, fmt.Sprintf("container %q is healthy", check.Container), nil
	case container.Unhealthy:
		msg := fmt.Sprintf("container %q is unhealthy", check.Container)
		if len(info.State.Health.Log) > 0 {
			lastLog := info.State.Health.Log[len(info.State.Health.Log)-1]
			if lastLog.Output != "" {
				msg = fmt.Sprintf("%s: %s", msg, lastLog.Output)
			}
		}
		return false, msg, nil
	case container.Starting:
		return false, fmt.Sprintf("container %q health check is starting", check.Container), nil
	default:
		return false, fmt.Sprintf("container %q has unknown health status: %s", check.Container, info.State.Health.Status), nil
	}
}

func (c *DockerChecker) Close() error {
	return c.client.Close()
}
