package health

import (
	"context"
	"errors"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
)

type mockDockerClient struct {
	inspectFunc func(ctx context.Context, containerID string) (types.ContainerJSON, error)
}

func (m *mockDockerClient) ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error) {
	return m.inspectFunc(ctx, containerID)
}

func (m *mockDockerClient) Close() error {
	return nil
}

func TestDockerChecker_Execute(t *testing.T) {
	tests := []struct {
		name        string
		container   string
		inspectFunc func(ctx context.Context, containerID string) (types.ContainerJSON, error)
		wantPassed  bool
		wantMessage string
	}{
		{
			name:      "container not found",
			container: "missing",
			inspectFunc: func(ctx context.Context, containerID string) (types.ContainerJSON, error) {
				return types.ContainerJSON{}, errors.New("no such container")
			},
			wantPassed:  false,
			wantMessage: `container "missing" not found or inaccessible`,
		},
		{
			name:      "container not running",
			container: "stopped",
			inspectFunc: func(ctx context.Context, containerID string) (types.ContainerJSON, error) {
				return types.ContainerJSON{
					ContainerJSONBase: &types.ContainerJSONBase{
						State: &types.ContainerState{
							Running: false,
							Status:  "exited",
						},
					},
				}, nil
			},
			wantPassed:  false,
			wantMessage: `container "stopped" is not running (status: exited)`,
		},
		{
			name:      "container running no healthcheck",
			container: "simple",
			inspectFunc: func(ctx context.Context, containerID string) (types.ContainerJSON, error) {
				return types.ContainerJSON{
					ContainerJSONBase: &types.ContainerJSONBase{
						State: &types.ContainerState{
							Running: true,
							Health:  nil,
						},
					},
				}, nil
			},
			wantPassed:  true,
			wantMessage: `container "simple" is running (no healthcheck configured)`,
		},
		{
			name:      "container healthy",
			container: "healthy",
			inspectFunc: func(ctx context.Context, containerID string) (types.ContainerJSON, error) {
				return types.ContainerJSON{
					ContainerJSONBase: &types.ContainerJSONBase{
						State: &types.ContainerState{
							Running: true,
							Health: &types.Health{
								Status: container.Healthy,
							},
						},
					},
				}, nil
			},
			wantPassed:  true,
			wantMessage: `container "healthy" is healthy`,
		},
		{
			name:      "container unhealthy",
			container: "unhealthy",
			inspectFunc: func(ctx context.Context, containerID string) (types.ContainerJSON, error) {
				return types.ContainerJSON{
					ContainerJSONBase: &types.ContainerJSONBase{
						State: &types.ContainerState{
							Running: true,
							Health: &types.Health{
								Status: container.Unhealthy,
							},
						},
					},
				}, nil
			},
			wantPassed:  false,
			wantMessage: `container "unhealthy" is unhealthy`,
		},
		{
			name:      "container unhealthy with log",
			container: "unhealthy-log",
			inspectFunc: func(ctx context.Context, containerID string) (types.ContainerJSON, error) {
				return types.ContainerJSON{
					ContainerJSONBase: &types.ContainerJSONBase{
						State: &types.ContainerState{
							Running: true,
							Health: &types.Health{
								Status: container.Unhealthy,
								Log: []*types.HealthcheckResult{
									{Output: "connection refused"},
								},
							},
						},
					},
				}, nil
			},
			wantPassed:  false,
			wantMessage: `container "unhealthy-log" is unhealthy: connection refused`,
		},
		{
			name:      "container starting",
			container: "starting",
			inspectFunc: func(ctx context.Context, containerID string) (types.ContainerJSON, error) {
				return types.ContainerJSON{
					ContainerJSONBase: &types.ContainerJSONBase{
						State: &types.ContainerState{
							Running: true,
							Health: &types.Health{
								Status: container.Starting,
							},
						},
					},
				}, nil
			},
			wantPassed:  false,
			wantMessage: `container "starting" health check is starting`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := &DockerChecker{
				client: &mockDockerClient{inspectFunc: tt.inspectFunc},
			}

			check := Check{Container: tt.container}
			passed, message, err := checker.Execute(context.Background(), check)

			if err != nil {
				t.Errorf("Execute() error = %v", err)
			}
			if passed != tt.wantPassed {
				t.Errorf("Execute() passed = %v, want %v", passed, tt.wantPassed)
			}
			if message != tt.wantMessage {
				t.Errorf("Execute() message = %q, want %q", message, tt.wantMessage)
			}
		})
	}
}
