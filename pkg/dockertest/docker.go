package dockertest

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/stretchr/testify/require"
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

	// Ports maps container ports to host ports.
	// Key is the container port (e.g., "6379/tcp"), value is the host port.
	Ports map[string]string
}

// Port returns the mapped host port for a given container port.
// The containerPort should be in the format "port/protocol" (e.g., "6379/tcp").
// Returns an empty string if the port is not mapped.
func (c *Container) Port(containerPort string) string {
	return c.Ports[containerPort]
}

// containerConfig holds the configuration for starting a container.
type containerConfig struct {
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

	// WaitStrategy determines how to detect container readiness.
	// If nil, the container is considered ready immediately after starting.
	WaitStrategy WaitStrategy

	// WaitTimeout is the maximum time to wait for container readiness.
	// Defaults to 30 seconds if zero.
	WaitTimeout time.Duration
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
	if err != nil {
		t.Fatalf("Docker is not available: %v. Ensure Docker is running.", err)
	}

	return dockerClient
}

// pullImage pulls a Docker image if it's not already present locally.
// This is a no-op if the image already exists in the local Docker cache.
func pullImage(t *testing.T, cli *client.Client, imageName string) {
	t.Helper()

	ctx := context.Background()

	// Check if image exists locally
	_, err := cli.ImageInspect(ctx, imageName)
	if err == nil {
		// Image exists locally
		return
	}

	// Pull the image
	// nolint:exhaustruct
	reader, err := cli.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		t.Fatalf("failed to pull image %s: %v", imageName, err)
	}
	defer func() { require.NoError(t, reader.Close()) }()

	// Consume the pull output to ensure the pull completes
	_, err = io.Copy(io.Discard, reader)
	if err != nil {
		t.Fatalf("failed to read image pull response: %v", err)
	}
}

// startContainer creates and starts a Docker container with the given configuration.
//
// Returns the container metadata including mapped ports. The container is
// automatically removed when the test completes via t.Cleanup, ensuring no
// orphaned containers remain even if the test fails.
//
// If WaitStrategy is provided, this function blocks until the container is
// ready or the WaitTimeout is exceeded. If WaitTimeout is zero, defaults to
// 30 seconds.
func startContainer(t *testing.T, cfg containerConfig) *Container {
	t.Helper()

	cli := getClient(t)

	// Pull image if not present locally
	pullImage(t, cli, cfg.Image)

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
	containerName := strings.NewReplacer(
		":", "-",
		"/", "-",
	).Replace(fmt.Sprintf("%s_%s_%d", cfg.Image, t.Name(), time.Now().UnixNano()))

	// Convert environment map to slice format ("KEY=VALUE")
	var envSlice []string
	for k, v := range cfg.Env {
		envSlice = append(envSlice, fmt.Sprintf("%s=%s", k, v))
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
			},
		},
		&container.HostConfig{
			PortBindings: portBindings,
			AutoRemove:   false, // We handle removal in t.Cleanup
		},
		nil, // NetworkingConfig
		nil, // Platform
		containerName,
	)
	require.NoError(t, err, "failed to create container")

	containerID := resp.ID

	// Register cleanup to ensure container is removed when test completes
	t.Cleanup(func() {
		require.NoError(t, cli.ContainerRemove(ctx, containerID, container.RemoveOptions{
			Force: true,
		}))
	})

	// Start the container
	require.NoError(t, cli.ContainerStart(ctx, containerID, container.StartOptions{}))

	// Inspect to get the mapped ports
	inspect, err := cli.ContainerInspect(ctx, containerID)
	require.NoError(t, err, "failed to inspect container")

	// Extract port mappings
	ports := make(map[string]string)
	for port, bindings := range inspect.NetworkSettings.Ports {
		if len(bindings) > 0 {
			ports[string(port)] = bindings[0].HostPort
		}
	}

	ctr := &Container{
		ID:    containerID,
		Host:  "localhost",
		Ports: ports,
	}

	// Wait for the container to be ready
	if cfg.WaitStrategy != nil {
		waitTimeout := cfg.WaitTimeout
		if waitTimeout == 0 {
			waitTimeout = 30 * time.Second
		}

		cfg.WaitStrategy.Wait(t, ctr, waitTimeout)
	}

	return ctr
}
