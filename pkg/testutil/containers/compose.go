package containers

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

const composeProjectEnv = "UNKEY_TEST_COMPOSE_PROJECT"

var (
	isolatedProjectID  atomic.Uint64
	reapIsolatedOnce   sync.Once
	isolatedProjectPID = os.Getpid()
)

// Container describes a Docker Compose service container started for tests.
type Container struct {
	// Name is the Docker Compose service name.
	Name string
	// project is the Compose project that owns this container. Shared services
	// use the per-worktree project; isolated services get their own.
	project string
}

// Addr returns the localhost address mapped to a container port.
func (c Container) Addr(t testing.TB, containerPort int) string {
	t.Helper()
	return fmt.Sprintf("localhost:%d", c.Port(t, containerPort))
}

// Port returns the host port mapped to a container port on this service.
func (c Container) Port(t testing.TB, containerPort int) int {
	t.Helper()
	return composeServicePort(t, c.project, c.Name, containerPort)
}

// containerState is Compose's view of a service container.
type containerState struct {
	State    string `json:"State"`
	Status   string `json:"Status"`
	Health   string `json:"Health"`
	ExitCode int    `json:"ExitCode"`
}

// state reports what Compose knows about the container, including one that has
// already exited.
//
// Diagnostics must not fail the test they explain, so a Docker error yields
// ok=false rather than an assertion.
func (c Container) state() (containerState, bool) {
	var state containerState

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "docker", "compose",
		"-f", composeFile(), "-p", c.project, "ps", "-a", "--format", "json", c.Name).Output()
	if err != nil {
		return state, false
	}

	// Compose emits one JSON object per container, so a project that somehow
	// holds more than one for this service still parses.
	first, _, _ := strings.Cut(strings.TrimSpace(string(out)), "\n")
	if first == "" {
		return state, false
	}
	if err := json.Unmarshal([]byte(first), &state); err != nil {
		return state, false
	}
	return state, true
}

// logs returns the tail of the container's output, or why it could not be read.
func (c Container) logs(lines int) string {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "docker", "compose",
		"-f", composeFile(), "-p", c.project,
		"logs", "--no-color", "--no-log-prefix", "--tail", strconv.Itoa(lines), c.Name).CombinedOutput()
	if err != nil {
		return fmt.Sprintf("<docker compose logs failed: %v: %s>", err, out)
	}
	if len(bytes.TrimSpace(out)) == 0 {
		return "<no container output>"
	}
	return string(out)
}

func startService(t testing.TB, service string) Container {
	t.Helper()

	project := composeProjectName()
	upArgs := []string{"-f", composeFile(), "-p", project, "up", "-d", "--wait", "--wait-timeout", "60", service}
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
				Name:    service,
				project: project,
			}
		}
		if time.Now().After(deadline) {
			require.NoError(t, err, "docker compose %s failed:\n%s", strings.Join(upArgs, " "), string(out))
		}
		time.Sleep(250 * time.Millisecond)
	}
}

// startIsolatedService starts one service in a Compose project that only this
// caller uses, so the container is never shared with another test or another
// test process. The returned function removes the project; callers own the
// ordering because a container that is torn down before its dependents can
// leave them blocked on connections that never close.
//
// Readiness is deliberately not delegated to `--wait`: Compose polls the
// service healthcheck, which cannot pass before its start_period elapses, so
// waiting on it costs several seconds for a container that is serving in under
// one. Callers poll the service directly instead.
func startIsolatedService(t testing.TB, service string) (Container, func()) {
	t.Helper()

	reapIsolatedOnce.Do(func() { reapAbandonedIsolatedProjects(t, service) })

	project := fmt.Sprintf(
		"%s-%s-%d-%d",
		composeProjectName(),
		service,
		isolatedProjectPID,
		isolatedProjectID.Add(1),
	)

	upArgs := []string{"-f", composeFile(), "-p", project, "up", "-d", service}
	ctx, cancel := context.WithTimeout(context.Background(), 75*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", append([]string{"compose"}, upArgs...)...)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "docker compose %s failed:\n%s", strings.Join(upArgs, " "), string(out))

	return Container{Name: service, project: project}, func() {
		removeComposeProject(project)
	}
}

// reapAbandonedIsolatedProjects removes isolated projects left behind by test
// processes that died without running their cleanup. Ownership is encoded in
// the project name, so a project whose creating process is gone is garbage.
// Projects belonging to live processes are siblings in the same suite run and
// are left alone.
func reapAbandonedIsolatedProjects(t testing.TB, service string) {
	t.Helper()

	prefix := fmt.Sprintf("%s-%s-", composeProjectName(), service)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "docker", "compose", "ls", "--all", "--format", "json").Output()
	if err != nil {
		// Reaping is opportunistic. A listing failure must not fail the test
		// that happened to run first.
		t.Logf("list Compose projects for reaping: %v", err)
		return
	}

	var projects []struct {
		Name string `json:"Name"`
	}
	if err := json.Unmarshal(out, &projects); err != nil {
		t.Logf("parse Compose project listing for reaping: %v", err)
		return
	}

	for _, project := range projects {
		owner, ok := isolatedProjectOwner(project.Name, prefix)
		if !ok || processAlive(owner) {
			continue
		}
		t.Logf("removing abandoned Compose project %s from dead pid %d", project.Name, owner)
		removeComposeProject(project.Name)
	}
}

// isolatedProjectOwner extracts the pid an isolated project name encodes.
func isolatedProjectOwner(projectName string, prefix string) (int, bool) {
	suffix, found := strings.CutPrefix(projectName, prefix)
	if !found {
		return 0, false
	}
	pidText, _, found := strings.Cut(suffix, "-")
	if !found {
		return 0, false
	}
	pid, err := strconv.Atoi(pidText)
	if err != nil || pid <= 0 {
		return 0, false
	}
	return pid, true
}

func processAlive(pid int) bool {
	// Signal 0 performs the permission and existence checks without delivering
	// anything. EPERM means the process exists under another user.
	err := unix.Kill(pid, 0)
	return err == nil || err == unix.EPERM
}

func removeComposeProject(project string) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "compose", "-f", composeFile(), "-p", project, "down", "--volumes")
	_ = cmd.Run()
}

func composeServicePort(t testing.TB, project string, service string, port int) int {
	t.Helper()

	outText := runDockerCompose(t, "-f", composeFile(), "-p", project, "port", service, strconv.Itoa(port))
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

func composeFile() string {
	return dataPath("pkg", "testutil", "docker-compose.test.yaml")
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

	_, port, found := strings.CutLast(line, ":")
	if !found || port == "" {
		return 0, fmt.Errorf("parse docker compose port output %q: %w", output, err)
	}
	return strconv.Atoi(port)
}
