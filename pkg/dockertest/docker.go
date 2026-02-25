package dockertest

import (
	"context"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
)

var (
	dockerClient     *client.Client
	dockerClientOnce sync.Once
)

// Container holds metadata about a running Docker container.
//
// Use the [Container.Port] method to retrieve the host port mapped to a
// specific container port.
type Container struct {
	// ID is the Docker container ID (64-character hex string).
	ID string

	// Host is the hostname to connect to (typically "localhost").
	Host string

	ContainerName string

	// Ports maps container ports to host ports.
	// Key is the container port (e.g., "6379/tcp"), value is the host port.
	Ports map[string]string
}

// HostURL returns a URL for the container using the provided scheme and port.
// The containerPort should be in the format "port/protocol" (e.g., "8080/tcp").
func (c *Container) HostURL(scheme, containerPort string) string {
	port := c.Port(containerPort)
	if port == "" {
		return ""
	}
	return fmt.Sprintf("%s://%s:%s", scheme, c.Host, port)
}

// Port returns the mapped host port for a given container port.
// The containerPort should be in the format "port/protocol" (e.g., "6379/tcp").
// Returns an empty string if the port is not mapped.
func (c *Container) Port(containerPort string) string {
	return c.Ports[containerPort]
}

func containerPortNumber(containerPort string) string {
	for i := 0; i < len(containerPort); i++ {
		if containerPort[i] == '/' {
			return containerPort[:i]
		}
	}
	return containerPort
}

// containerConfig holds the configuration for starting a container.
type containerConfig struct {
	// ContainerName is the Docker container name.
	ContainerName string

	// Image is the Docker image to use (e.g., "redis:8.0").
	Image string

	// ExposedPorts lists container ports to expose (e.g., "6379/tcp").
	ExposedPorts []string

	// Env holds environment variables to set in the container.
	// Keys are variable names, values are variable values.
	Env map[string]string

	// Cmd overrides the default command for the container image.
	// If nil, the image's default CMD is used.
	Cmd []string

	// Tmpfs mounts tmpfs (RAM-backed) filesystems in the container.
	// Keys are mount paths, values are mount options (e.g., "rw,noexec,size=256m").
	Tmpfs map[string]string

	// Binds mounts host paths into the container.
	// Format: "/host/path:/container/path:ro".
	Binds []string

	// NetworkName is the Docker network to attach the container to.
	// Leave empty to use Docker's default networking.
	NetworkName string

	// Keep can be set to true to prevent the testsuite from cleaning up containers.
	// Use it when you need the container to inspect logs or similar.
	Keep bool
}

// getClient returns a shared Docker client instance.
//
// The client is lazily initialized on first call and reused across all tests
// in the same process. Fails the test immediately if Docker is not accessible.
func getClient(t *testing.T) *client.Client {
	t.Helper()

	dockerClientOnce.Do(func() {
		var err error
		dockerClient, err = client.NewClientWithOpts(
			client.FromEnv,
			client.WithAPIVersionNegotiation(),
		)
		require.NoError(t, err, "failed to create Docker client")
	})

	// Verify Docker is accessible
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := dockerClient.Ping(ctx)
	require.NoError(t, err, "Docker is not available. Ensure Docker is running.")

	return dockerClient
}

// pullImage pulls a Docker image if it's not already present locally.
// This is a no-op if the image already exists in the local Docker cache.
func pullImage(cli *client.Client, imageName string) error {
	ctx := context.Background()

	// Check if image exists locally
	_, err := cli.ImageInspect(ctx, imageName)
	if err == nil {
		// Image exists locally
		return nil
	}

	// Pull the image
	// nolint:exhaustruct
	reader, err := cli.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %w", imageName, err)
	}
	defer func() { _ = reader.Close() }()

	// Consume the pull output to ensure the pull completes
	_, err = io.Copy(io.Discard, reader)
	if err != nil {
		return fmt.Errorf("failed to read image pull response: %w", err)
	}

	return nil
}

// startContainer creates and starts a Docker container with the given configuration.
// Returns the container metadata and a cleanup function.
func startContainer(cli *client.Client, cfg containerConfig, testName string) (*Container, func() error, error) {
	if err := pullImage(cli, cfg.Image); err != nil {
		return nil, nil, err
	}

	// Build exposed ports and port bindings
	exposedPorts := nat.PortSet{}
	portBindings := nat.PortMap{}

	for _, port := range cfg.ExposedPorts {
		natPort := nat.Port(port)
		exposedPorts[natPort] = struct{}{}
		// Empty HostPort means Docker will assign a random available port
		portBindings[natPort] = []nat.PortBinding{
			{HostIP: "0.0.0.0", HostPort: ""},
		}
	}

	ctx := context.Background()

	// Create the container
	containerName := cfg.ContainerName
	if containerName == "" {
		containerName = uid.New("")
	}

	// Convert environment map to slice format ("KEY=VALUE")
	var envSlice []string
	for k, v := range cfg.Env {
		envSlice = append(envSlice, fmt.Sprintf("%s=%s", k, v))
	}

	var networkingConfig *network.NetworkingConfig
	if cfg.NetworkName != "" {
		networkingConfig = &network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{
				cfg.NetworkName: {},
			},
		}
	}

	resp, err := cli.ContainerCreate(
		ctx,
		&container.Config{
			Image:        cfg.Image,
			ExposedPorts: exposedPorts,
			Env:          envSlice,
			Cmd:          cfg.Cmd,
			Labels: map[string]string{
				"owner": "dockertest",
				"test":  testName,
			},
		},
		&container.HostConfig{
			PortBindings: portBindings,
			AutoRemove:   false,
			Tmpfs:        cfg.Tmpfs,
			Binds:        cfg.Binds,
		},
		networkingConfig,
		nil,
		containerName,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create container: %w", err)
	}

	containerID := resp.ID

	cleanup := func() error {
		if cfg.Keep {
			return nil
		}
		return cli.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true})
	}

	// Start the container
	if err := cli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return nil, cleanup, fmt.Errorf("failed to start container: %w", err)
	}

	// Inspect to get the mapped ports
	inspect, err := cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, cleanup, fmt.Errorf("failed to inspect container: %w", err)
	}

	// Extract port mappings
	ports := make(map[string]string)
	for port, bindings := range inspect.NetworkSettings.Ports {
		if len(bindings) > 0 {
			ports[string(port)] = bindings[0].HostPort
		}
	}

	ctr := &Container{
		ID:            containerID,
		Host:          "localhost",
		Ports:         ports,
		ContainerName: containerName,
	}

	return ctr, cleanup, nil
}
