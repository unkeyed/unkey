package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/client"
	"github.com/unkeyed/unkey/go/gen/proto/krane/v1/kranev1connect"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// docker implements kranev1connect.DeploymentServiceHandler using Docker Engine API.
type docker struct {
	logger logging.Logger
	client *client.Client
	kranev1connect.UnimplementedDeploymentServiceHandler
}

var _ kranev1connect.DeploymentServiceHandler = (*docker)(nil)

// New creates a Docker backend instance and validates daemon connectivity.
func New(logger logging.Logger, socketPath string) (*docker, error) {
	// Create Docker client with configurable socket path
	// socketPath must not include protocol (e.g., "var/run/docker.sock")
	dockerClient, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
		client.WithHost(fmt.Sprintf("unix:///%s", socketPath)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	// Test connection to Docker daemon
	ctx := context.Background()
	_, err = dockerClient.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to ping Docker daemon at %s (ensure Docker socket is accessible): %w", socketPath, err)
	}

	logger.Info("Docker client initialized successfully", "socket", socketPath)

	return &docker{
		UnimplementedDeploymentServiceHandler: kranev1connect.UnimplementedDeploymentServiceHandler{},
		logger:                                logger,
		client:                                dockerClient,
	}, nil
}
