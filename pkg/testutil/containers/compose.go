package containers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

const (
	composeProjectEnv = "UNKEY_TEST_COMPOSE_PROJECT"
	composeRunDirEnv  = "UNKEY_TEST_RUN_DIR"
)

var (
	isolatedProjectID   atomic.Uint64
	reapIsolatedOnce    sync.Once
	isolatedProjectPID  = os.Getpid()
	sharedServiceCache  sync.Map
	composeServicePorts sync.Map
)

type composePortKey struct {
	project string
	service string
	port    int
}

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

func startService(t testing.TB, service string) Container {
	t.Helper()

	project := composeProjectName()
	cacheKey := project + "\x00" + service
	if cached, ok := sharedServiceCache.Load(cacheKey); ok {
		return cached.(Container)
	}

	lockFile := lockComposeResource(t, project, service)
	defer func() { require.NoError(t, lockFile.Close()) }()

	container := Container{
		Name:    service,
		project: project,
	}
	if composeResourceCompletedForRun(project, service) && composeServiceHealthy(project, service) {
		sharedServiceCache.Store(cacheKey, container)
		return container
	}

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
			markComposeResourceCompleted(t, project, service)
			sharedServiceCache.Store(cacheKey, container)
			return container
		}
		if time.Now().After(deadline) {
			require.NoError(t, err, "docker compose %s failed:\n%s", strings.Join(upArgs, " "), string(out))
		}
		time.Sleep(250 * time.Millisecond)
	}
}

// lockComposeResource coordinates Compose work across concurrently running Go
// test processes. Rask starts package tests in separate processes, so a mutex
// cannot prevent them from issuing the same Docker or schema command at once.
func lockComposeResource(t testing.TB, project string, resource string) *os.File {
	t.Helper()

	digest := sha256.Sum256([]byte(project + "\x00" + resource))
	lockDir := os.TempDir()
	if runDir := os.Getenv(composeRunDirEnv); runDir != "" {
		lockDir = runDir
	}
	path := filepath.Join(lockDir, fmt.Sprintf("unkey-test-%s.lock", hex.EncodeToString(digest[:])[:16]))
	lockFile, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	require.NoError(t, err)
	require.NoError(t, unix.Flock(int(lockFile.Fd()), unix.LOCK_EX))
	return lockFile
}

func composeResourceCompletedForRun(project string, resource string) bool {
	marker := composeResourceMarker(project, resource)
	if marker == "" {
		return false
	}
	_, err := os.Stat(marker)
	return err == nil
}

func markComposeResourceCompleted(t testing.TB, project string, resource string) {
	t.Helper()

	marker := composeResourceMarker(project, resource)
	if marker == "" {
		return
	}
	require.NoError(t, os.WriteFile(marker, nil, 0o600))
}

func composeResourceMarker(project string, resource string) string {
	runDir := os.Getenv(composeRunDirEnv)
	if runDir == "" {
		return ""
	}
	digest := sha256.Sum256([]byte(project + "\x00" + resource))
	return filepath.Join(runDir, fmt.Sprintf("%s.started", hex.EncodeToString(digest[:])[:16]))
}

func composeServiceHealthy(project string, service string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "compose", "-f", composeFile(), "-p", project, "ps", "--format", "json", service)
	out, err := cmd.Output()
	if err != nil || len(out) == 0 {
		return false
	}

	var status struct {
		Health string `json:"Health"`
		State  string `json:"State"`
	}
	if err := json.Unmarshal(out, &status); err != nil {
		return false
	}
	return status.State == "running" && status.Health == "healthy"
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
	cacheKey := composePortKey{project: project, service: service, port: port}
	if cached, ok := composeServicePorts.Load(cacheKey); ok {
		return cached.(int)
	}

	outText := runDockerCompose(t, "-f", composeFile(), "-p", project, "port", service, strconv.Itoa(port))
	hostPort, err := composePort(outText)
	require.NoError(t, err)
	actual, _ := composeServicePorts.LoadOrStore(cacheKey, hostPort)
	return actual.(int)
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

	idx := strings.LastIndex(line, ":")
	if idx == -1 || idx == len(line)-1 {
		return 0, fmt.Errorf("parse docker compose port output %q: %w", output, err)
	}
	return strconv.Atoi(line[idx+1:])
}
