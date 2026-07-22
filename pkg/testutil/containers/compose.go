package containers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const composeProjectEnv = "UNKEY_TEST_COMPOSE_PROJECT"

// Container describes a Docker Compose service container started for tests.
type Container struct {
	// Name is the Docker Compose service name.
	Name string
}

// Addr returns the localhost address mapped to a container port.
func (c Container) Addr(t testing.TB, containerPort int) string {
	t.Helper()
	return fmt.Sprintf("localhost:%d", c.Port(t, containerPort))
}

// Port returns the host port mapped to a container port on this service.
func (c Container) Port(t testing.TB, containerPort int) int {
	t.Helper()
	return composeServicePort(t, c.Name, containerPort)
}

func startService(t testing.TB, service string) Container {
	t.Helper()

	compose := dataPath("pkg", "testutil", "docker-compose.test.yaml")
	upArgs := []string{"-f", compose, "-p", composeProjectName(), "up", "-d", "--wait", "--wait-timeout", "60", service}
	var out []byte
	var err error
	deadline := time.Now().Add(90 * time.Second)
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 75*time.Second)
		cmd := exec.CommandContext(ctx, "docker", append([]string{"compose"}, upArgs...)...)
		out, err = cmd.CombinedOutput()
		cancel()
		if err == nil {
			return Container{
				Name: service,
			}
		}
		if time.Now().After(deadline) {
			require.NoError(t, err, "docker compose %s failed:\n%s", strings.Join(upArgs, " "), string(out))
		}
		time.Sleep(250 * time.Millisecond)
	}
}

func composeServicePort(t testing.TB, service string, port int) int {
	t.Helper()

	compose := dataPath("pkg", "testutil", "docker-compose.test.yaml")
	outText := runDockerCompose(t, "-f", compose, "-p", composeProjectName(), "port", service, strconv.Itoa(port))
	hostPort, err := composePort(outText)
	require.NoError(t, err)
	return hostPort
}

func runDockerCompose(t testing.TB, args ...string) string {
	t.Helper()

	cmd := exec.Command("docker", append([]string{"compose"}, args...)...)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "docker compose %s failed:\n%s", strings.Join(args, " "), string(out))
	return strings.TrimSpace(string(out))
}

func composeProjectName() string {
	if project := os.Getenv(composeProjectEnv); project != "" {
		return project
	}
	sum := sha256.Sum256([]byte(sourceRepoRoot()))
	return fmt.Sprintf("unkey-test-%s", hex.EncodeToString(sum[:])[:12])
}

func composePort(output string) (int, error) {
	line := strings.TrimSpace(output)
	_, port, err := net.SplitHostPort(line)
	if err == nil {
		return strconv.Atoi(port)
	}

	idx := strings.LastIndex(line, ":")
	if idx == -1 || idx == len(line)-1 {
		return 0, fmt.Errorf("parse docker compose port output %q: %w", output, err)
	}
	return strconv.Atoi(line[idx+1:])
}
