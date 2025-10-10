package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/client"
	"github.com/unkeyed/unkey/go/gen/proto/krane/v1/kranev1connect"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// docker implements kranev1connect.DeploymentServiceHandler using Docker Engine API.
//
// This backend is designed for single-node deployments where container images
// are built locally using the Docker daemon and deployed on the same host.
// It does not support multi-node clusters or remote image registries.
type docker struct {
	logger logging.Logger
	client *client.Client

	kranev1connect.UnimplementedDeploymentServiceHandler
}

var _ kranev1connect.DeploymentServiceHandler = (*docker)(nil)

// New creates a Docker backend instance and validates daemon connectivity.
//
// The socketPath parameter specifies the Docker daemon socket location.
// Common values are "/var/run/docker.sock" on Linux and macOS.
//
// Returns an error if the Docker daemon is unreachable or the socket
// path is invalid. The socket must be accessible with appropriate permissions.
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
		logger: logger,
		client: dockerClient,
	}, nil
}
