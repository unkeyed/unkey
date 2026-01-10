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
)

var (
	dockerClient     *client.Client
	dockerClientOnce sync.Once
	dockerClientErr  error
)

// Container holds metadata about a running Docker container.
type Container struct {
	// ID is the Docker container ID.
	ID string

	// Host is the hostname to connect to (typically "localhost").
	Host string

	// Ports maps container ports to host ports.
	// Key is the container port (e.g., "6379/tcp"), value is the host port.
	Ports map[string]string
}

// Port returns the mapped host port for a given container port.
// The containerPort should be in the format "port/protocol" (e.g., "6379/tcp").
func (c *Container) Port(containerPort string) string {
	return c.Ports[containerPort]
}

// containerConfig holds the configuration for starting a container.
type containerConfig struct {
	Image        string
	ExposedPorts []string
	Env          []string
	Cmd          []string
	WaitStrategy WaitStrategy
	WaitTimeout  time.Duration
}

// getClient returns a shared Docker client instance.
// The client is lazily initialized on first call and reused thereafter.
func getClient(t *testing.T) *client.Client {
	t.Helper()

	dockerClientOnce.Do(func() {
		dockerClient, dockerClientErr = client.NewClientWithOpts(
			client.FromEnv,
			client.WithAPIVersionNegotiation(),
		)
	})

	if dockerClientErr != nil {
		t.Fatalf("failed to create Docker client: %v", dockerClientErr)
	}

	// Verify Docker is accessible
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := dockerClient.Ping(ctx)
	if err != nil {
		t.Skipf("Docker is not available: %v", err)
	}

	return dockerClient
}

// pullImage pulls a Docker image if it's not already present locally.
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
	reader, err := cli.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		t.Fatalf("failed to pull image %s: %v", imageName, err)
	}
	defer reader.Close()

	// Consume the pull output to ensure the pull completes
	_, err = io.Copy(io.Discard, reader)
	if err != nil {
		t.Fatalf("failed to read image pull response: %v", err)
	}
}

// startContainer creates and starts a Docker container with the given configuration.
// It returns the container metadata including mapped ports.
// The container is automatically cleaned up when the test completes via t.Cleanup.
func startContainer(t *testing.T, cfg containerConfig) *Container {
	t.Helper()

	cli := getClient(t)

	// Pull image if needed
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
	resp, err := cli.ContainerCreate(
		ctx,
		&container.Config{
			Image:        cfg.Image,
			Env:          cfg.Env,
			Cmd:          cfg.Cmd,
			ExposedPorts: exposedPorts,
			Labels: map[string]string{
				"owner": "bazel",
			},
		},
		&container.HostConfig{
			PortBindings: portBindings,
			AutoRemove:   false, // We handle removal in t.Cleanup
		},
		nil, // NetworkingConfig
		nil, // Platform
		strings.NewReplacer(
			":", "-",
			"/", "-",
		).Replace(fmt.Sprintf("%s_%s_%d", cfg.Image, t.Name(), time.Now().UnixNano())),
	)
	if err != nil {
		t.Fatalf("failed to create container: %v", err)
	}

	containerID := resp.ID

	// Register cleanup to ensure container is removed when test completes
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Stop the container (with a short timeout)
		stopTimeout := 5
		_ = cli.ContainerStop(cleanupCtx, containerID, container.StopOptions{
			Timeout: &stopTimeout,
		})

		// Remove the container
		_ = cli.ContainerRemove(cleanupCtx, containerID, container.RemoveOptions{
			Force: true,
		})
	})

	// Start the container
	if err := cli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		t.Fatalf("failed to start container: %v", err)
	}

	// Inspect to get the mapped ports
	inspect, err := cli.ContainerInspect(ctx, containerID)
	if err != nil {
		t.Fatalf("failed to inspect container: %v", err)
	}

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
