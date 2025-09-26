package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/client"
	"github.com/unkeyed/unkey/go/gen/proto/krane/v1/kranev1connect"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

type docker struct {
	logger logging.Logger
	client *client.Client
	kranev1connect.UnimplementedDeploymentServiceHandler
}

func New(logger logging.Logger, socketPath string) (*docker, error) {
	// Create Docker client with configurable socket path
	// socketPath should include protocol (e.g., "unix:///var/run/docker.sock")
	dockerClient, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
		client.WithHost(socketPath),
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
