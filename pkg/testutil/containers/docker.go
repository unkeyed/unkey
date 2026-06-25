package containers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"
	"github.com/docker/go-connections/nat"
	"github.com/stretchr/testify/require"
)

const (
	containerOwnerLabel      = "owner"
	containerOwnerValue      = "testutil-containers"
	containerReusableLabel   = "unkey.test.reusable"
	containerImageLabel      = "unkey.test.image"
	containerScopeLabel      = "unkey.test.scope"
	containerScopeHashLabel  = "unkey.test.scope_hash"
	containerConfigHashLabel = "unkey.test.config_hash"

	containerScopeEnv = "UNKEY_TEST_CONTAINER_SCOPE"
)

var (
	dockerClient     *client.Client
	dockerClientErr  error
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

	// Tmpfs mounts tmpfs (RAM-backed) filesystems in the container.
	// Keys are mount paths, values are mount options (e.g., "rw,noexec,size=256m").
	Tmpfs map[string]string

	// WaitStrategy determines how to detect container readiness.
	// If nil, the container is considered ready immediately after starting.
	WaitStrategy WaitStrategy

	// WaitTimeout is the maximum time to wait for container readiness.
	// Defaults to 30 seconds if zero.
	WaitTimeout time.Duration

	// Dedicated disables default container reuse and creates a per-test
	// container that is removed with t.Cleanup.
	Dedicated bool
}

// Opt configures a test container.
type Opt func(*containerConfig)

// WithDedicatedContainer creates a per-test container instead of reusing the
// stable shared container for the image.
func WithDedicatedContainer() Opt {
	return func(cfg *containerConfig) {
		cfg.Dedicated = true
	}
}

// getClient returns a shared Docker client instance.
//
// The client is lazily initialized on first call and reused across all tests
// in the same process. Fails the test immediately if Docker is not accessible.
func getClient(t testing.TB) *client.Client {
	t.Helper()

	dockerClientOnce.Do(func() {
		dockerClient, dockerClientErr = client.NewClientWithOpts(
			client.FromEnv,
			client.WithAPIVersionNegotiation(),
		)
	})
	require.NoError(t, dockerClientErr, "failed to create Docker client")

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
func pullImage(t testing.TB, cli *client.Client, imageName string) {
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
// automatically removed when the test completes via t.Cleanup if Dedicated is
// set. Reusable containers intentionally remain after the test so later Bazel
// test processes can attach to the same Docker service.
//
// If WaitStrategy is provided, this function blocks until the container is
// ready or the WaitTimeout is exceeded. If WaitTimeout is zero, defaults to
// 30 seconds.
func startContainer(t testing.TB, cfg containerConfig) *Container {
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
	containerName := reusableContainerName(cfg)
	if cfg.Dedicated {
		containerName = disposableContainerName(t, cfg.Image)
	} else {
		if ctr, ok := findReusableContainer(t, ctx, cli, cfg); ok {
			waitForContainer(t, cfg, ctr)
			return ctr
		}
	}

	// Convert environment map to slice format ("KEY=VALUE")
	var envSlice []string
	for k, v := range cfg.Env {
		envSlice = append(envSlice, fmt.Sprintf("%s=%s", k, v))
	}
	sort.Strings(envSlice)

	labels := containerLabels(cfg)

	resp, err := cli.ContainerCreate(
		ctx,
		&container.Config{
			Image:        cfg.Image,
			ExposedPorts: exposedPorts,
			Env:          envSlice,
			Cmd:          cfg.Cmd,
			Labels:       labels,
		},
		&container.HostConfig{
			PortBindings: portBindings,
			AutoRemove:   false,
			Tmpfs:        cfg.Tmpfs,
			// Ensure host.docker.internal resolves inside containers.
			// Docker Desktop adds this automatically, but alternative runtimes
			// like OrbStack do not. Without it, containers that need to call
			// back to host-bound test servers (e.g. Restate → worker handler)
			// fail with DNS resolution errors.
			ExtraHosts: []string{"host.docker.internal:host-gateway"},
		},
		nil, // NetworkingConfig
		nil, // Platform
		containerName,
	)
	if err != nil {
		if cfg.Dedicated {
			require.NoError(t, err, "failed to create container")
		}
		if errdefs.IsConflict(err) {
			ctr := findReusableContainerAfterConflict(t, ctx, cli, cfg)
			waitForContainer(t, cfg, ctr)
			return ctr
		}
		require.NoError(t, err, "failed to create container")
	}

	containerID := resp.ID

	// Register cleanup before the container starts so failed readiness checks do
	// not leak disposable containers.
	if cfg.Dedicated {
		t.Cleanup(func() {
			require.NoError(t, cli.ContainerRemove(ctx, containerID, container.RemoveOptions{
				Force: true,
			}))
		})
	}

	// Start the container
	err = cli.ContainerStart(ctx, containerID, container.StartOptions{})
	if err != nil {
		if cfg.Dedicated || !errdefs.IsNotModified(err) {
			require.NoError(t, err, "failed to start container")
		}
	}

	// Inspect to get the mapped ports
	inspect, err := cli.ContainerInspect(ctx, containerID)
	require.NoError(t, err, "failed to inspect container")

	ctr := containerFromInspect(t, inspect)
	waitForContainer(t, cfg, ctr)

	return ctr
}

// containerLabels marks containers so reusable ones can be validated before
// attach, and leaked dedicated containers can still be removed by mise test.
func containerLabels(cfg containerConfig) map[string]string {
	labels := map[string]string{
		containerOwnerLabel: containerOwnerValue,
	}
	if cfg.Dedicated {
		return labels
	}

	labels[containerReusableLabel] = "true"
	labels[containerImageLabel] = cfg.Image
	labels[containerScopeLabel] = testContainerScope()
	labels[containerScopeHashLabel] = testContainerScopeHash()
	labels[containerConfigHashLabel] = containerConfigHash(cfg)
	return labels
}

// findReusableContainerAfterConflict waits for the process that won the Docker
// name race to make the container inspectable.
func findReusableContainerAfterConflict(
	t testing.TB,
	ctx context.Context,
	cli *client.Client,
	cfg containerConfig,
) *Container {
	t.Helper()

	containerName := reusableContainerName(cfg)
	var ctr *Container
	require.Eventually(t, func() bool {
		var ok bool
		ctr, ok = findReusableContainer(t, ctx, cli, cfg)
		return ok
	}, 5*time.Second, 100*time.Millisecond, "container %q already exists but could not be inspected after retry", containerName)

	return ctr
}

// findReusableContainer returns a stable test container by name.
//
// Reusable names are shared across Bazel test processes, so the labels guard
// against accidentally attaching to a developer-owned container with the same
// name.
func findReusableContainer(
	t testing.TB,
	ctx context.Context,
	cli *client.Client,
	cfg containerConfig,
) (*Container, bool) {
	t.Helper()

	containerName := reusableContainerName(cfg)
	inspect, err := cli.ContainerInspect(ctx, containerName)
	if err != nil {
		if errdefs.IsNotFound(err) {
			return nil, false
		}
		require.NoError(t, err, "failed to inspect reusable container %q", containerName)
	}

	labels := map[string]string{}
	if inspect.Config != nil && inspect.Config.Labels != nil {
		labels = inspect.Config.Labels
	}
	require.Equal(t, containerOwnerValue, labels[containerOwnerLabel],
		"container %q exists but is not owned by testutil containers", containerName)
	require.Equal(t, "true", labels[containerReusableLabel],
		"container %q exists but is not marked reusable", containerName)
	require.Equal(t, cfg.Image, labels[containerImageLabel],
		"container %q exists for a different image", containerName)
	require.Equal(t, testContainerScopeHash(), labels[containerScopeHashLabel],
		"container %q exists for a different test scope", containerName)
	require.Equal(t, containerConfigHash(cfg), labels[containerConfigHashLabel],
		"container %q exists for a different test container config", containerName)

	if inspect.State == nil || !inspect.State.Running {
		err = cli.ContainerStart(ctx, inspect.ID, container.StartOptions{})
		if err != nil && !errdefs.IsNotModified(err) {
			require.NoError(t, err, "failed to start reusable container %q", containerName)
		}
		inspect, err = cli.ContainerInspect(ctx, inspect.ID)
		require.NoError(t, err, "failed to inspect reusable container %q after start", containerName)
	}

	return containerFromInspect(t, inspect), true
}

// containerFromInspect converts Docker inspect output into the package's
// connection metadata.
func containerFromInspect(t testing.TB, inspect container.InspectResponse) *Container {
	t.Helper()

	require.NotNil(t, inspect.NetworkSettings, "container %s has no network settings", inspect.ID)

	ports := make(map[string]string)
	for port, bindings := range inspect.NetworkSettings.Ports {
		if len(bindings) > 0 {
			ports[string(port)] = bindings[0].HostPort
		}
	}

	return &Container{
		ID:    inspect.ID,
		Host:  "localhost",
		Ports: ports,
	}
}

// waitForContainer applies the configured readiness check.
func waitForContainer(t testing.TB, cfg containerConfig, ctr *Container) {
	t.Helper()

	if cfg.WaitStrategy == nil {
		return
	}

	waitTimeout := cfg.WaitTimeout
	if waitTimeout == 0 {
		waitTimeout = 30 * time.Second
	}

	cfg.WaitStrategy.Wait(t, ctr, waitTimeout)
}

// reusableContainerName returns the stable Docker name for a shared test config.
func reusableContainerName(cfg containerConfig) string {
	return fmt.Sprintf("unkey-test-%s-%s-%s", testContainerScopeHash(), imageSlug(cfg.Image), containerConfigHash(cfg))
}

// imageSlug returns a Docker-safe slug for image names.
func imageSlug(imageName string) string {
	var b strings.Builder
	lastHyphen := false
	for _, r := range strings.ToLower(imageName) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastHyphen = false
			continue
		}
		if !lastHyphen {
			b.WriteByte('-')
			lastHyphen = true
		}
	}

	suffix := strings.Trim(b.String(), "-")
	if suffix == "" {
		suffix = "container"
	}
	return suffix
}

// containerConfigHash returns a short fingerprint for settings that affect the
// running container.
func containerConfigHash(cfg containerConfig) string {
	var b strings.Builder
	b.WriteString("image=")
	b.WriteString(cfg.Image)
	b.WriteString("\nports=")
	for _, port := range sortedStrings(cfg.ExposedPorts) {
		b.WriteString(port)
		b.WriteByte(',')
	}
	b.WriteString("\nenv=")
	for _, key := range sortedMapKeys(cfg.Env) {
		b.WriteString(key)
		b.WriteByte('=')
		b.WriteString(cfg.Env[key])
		b.WriteByte(',')
	}
	b.WriteString("\ncmd=")
	for _, arg := range cfg.Cmd {
		b.WriteString(arg)
		b.WriteByte('\x00')
	}
	b.WriteString("\ntmpfs=")
	for _, key := range sortedMapKeys(cfg.Tmpfs) {
		b.WriteString(key)
		b.WriteByte('=')
		b.WriteString(cfg.Tmpfs[key])
		b.WriteByte(',')
	}

	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])[:12]
}

// sortedStrings returns a sorted copy of values.
func sortedStrings(values []string) []string {
	sorted := append([]string(nil), values...)
	sort.Strings(sorted)
	return sorted
}

// sortedMapKeys returns sorted keys for a string map.
func sortedMapKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// testContainerScope returns the logical owner for reusable containers.
//
// mise sets this to the worktree root. Direct Bazel invocations fall back to
// Bazel's runfiles root, whose output-base component is derived from the
// workspace path.
func testContainerScope() string {
	if scope := os.Getenv(containerScopeEnv); scope != "" {
		return scope
	}
	if scope := os.Getenv("BUILD_WORKSPACE_DIRECTORY"); scope != "" {
		return scope
	}
	if scope := os.Getenv("TEST_SRCDIR"); scope != "" {
		return scope
	}
	wd, err := os.Getwd()
	if err == nil {
		return wd
	}
	return "unknown"
}

// testContainerScopeHash returns a compact Docker-safe scope identifier.
func testContainerScopeHash() string {
	sum := sha256.Sum256([]byte(testContainerScope()))
	return hex.EncodeToString(sum[:])[:12]
}

// disposableContainerName returns a unique Docker name for a non-shared test
// container.
func disposableContainerName(t testing.TB, imageName string) string {
	t.Helper()

	return strings.NewReplacer(
		":", "-",
		"/", "-",
	).Replace(fmt.Sprintf("%s_%s_%d", imageName, t.Name(), time.Now().UnixNano()))
}
