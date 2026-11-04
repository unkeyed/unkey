package containers_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestRedis(t *testing.T) {
	// This test verifies that the Redis container starts correctly
	// and is accessible via the returned URL.
	url := containers.Redis(t)

	// Parse the URL and create a Redis client
	opts, err := redis.ParseURL(url)
	require.NoError(t, err)

	client := redis.NewClient(opts)
	t.Cleanup(func() { require.NoError(t, client.Close()) })

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Verify we can ping Redis
	err = client.Ping(ctx).Err()
	require.NoError(t, err)

	// Verify we can set and get a value
	testKey := fmt.Sprintf("test-key-%d", time.Now().UnixNano())
	err = client.Set(ctx, testKey, "test-value", 0).Err()
	require.NoError(t, err)

	val, err := client.Get(ctx, testKey).Result()
	require.NoError(t, err)
	require.Equal(t, "test-value", val)
}

func TestRedis_ReusesContainer(t *testing.T) {
	url1 := containers.Redis(t)
	url2 := containers.Redis(t)

	require.Equal(t, url1, url2)

	opts1, err := redis.ParseURL(url1)
	require.NoError(t, err)
	client1 := redis.NewClient(opts1)
	t.Cleanup(func() { require.NoError(t, client1.Close()) })

	opts2, err := redis.ParseURL(url2)
	require.NoError(t, err)
	client2 := redis.NewClient(opts2)
	t.Cleanup(func() { require.NoError(t, client2.Close()) })

	ctx := context.Background()
	key := fmt.Sprintf("key-%d", time.Now().UnixNano())

	err = client1.Set(ctx, key, "shared-value", 0).Err()
	require.NoError(t, err)

	val2, err := client2.Get(ctx, key).Result()
	require.NoError(t, err)
	require.Equal(t, "shared-value", val2)
}

func TestRedis_ReusesNamedContextWithoutCompose(t *testing.T) {
	endpoint, err := exec.Command("docker", "context", "inspect", "--format", "{{.Endpoints.docker.Host}}").Output()
	require.NoError(t, err)
	if !strings.HasPrefix(strings.TrimSpace(string(endpoint)), "unix://") {
		t.Skip("named-context fixture requires a local Unix socket")
	}
	contextName := uid.New(uid.TestPrefix)
	out, err := exec.Command("docker", "context", "create", contextName, "--docker", "host="+strings.TrimSpace(string(endpoint))).CombinedOutput()
	require.NoError(t, err, "%s", out)
	t.Cleanup(func() {
		out, err := exec.Command("docker", "context", "rm", contextName).CombinedOutput()
		require.NoError(t, err, "%s", out)
	})
	t.Setenv("DOCKER_CONTEXT", contextName)
	t.Setenv("DOCKER_HOST", "")
	url := containers.Redis(t)
	t.Setenv("PATH", t.TempDir())
	require.Equal(t, url, containers.Redis(t))

	opts, err := redis.ParseURL(url)
	require.NoError(t, err)
	client := redis.NewClient(opts)
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	require.NoError(t, client.Ping(t.Context()).Err())
}

func TestRedis_ConcurrentReuse(t *testing.T) {
	for i := 0; i < 8; i++ {
		t.Run(fmt.Sprintf("client_%d", i), func(t *testing.T) {
			t.Parallel()
			opts, err := redis.ParseURL(containers.Redis(t))
			require.NoError(t, err)
			client := redis.NewClient(opts)
			t.Cleanup(func() { require.NoError(t, client.Close()) })
			require.NoError(t, client.Ping(t.Context()).Err())
		})
	}
}

func TestRedis_ReconcilesConfigurationRollback(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "_main", "pkg", "testutil", "docker-compose.test.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(file), 0o700))
	original, err := os.ReadFile("../docker-compose.test.yaml")
	require.NoError(t, err)
	changed := strings.Replace(string(original), `"10000"`, `"10001"`, 1)
	require.NotEqual(t, string(original), changed)
	project := strings.ToLower(uid.New(uid.TestPrefix))
	t.Setenv("TEST_SRCDIR", root)
	t.Setenv("TEST_WORKSPACE", "_main")
	t.Setenv("UNKEY_TEST_COMPOSE_PROJECT", project)
	t.Cleanup(func() {
		out, err := exec.Command("docker", "compose", "-f", file, "-p", project, "down", "--volumes").CombinedOutput()
		require.NoError(t, err, "%s", out)
	})
	for _, step := range []struct {
		name   string
		config string
		limit  string
	}{
		{"original", string(original), "10000"},
		{"changed", changed, "10001"},
		{"rollback", string(original), "10000"},
	} {
		t.Run(step.name, func(t *testing.T) {
			require.NoError(t, os.WriteFile(file, []byte(step.config), 0o600))
			opts, err := redis.ParseURL(containers.Redis(t))
			require.NoError(t, err)
			client := redis.NewClient(opts)
			t.Cleanup(func() { require.NoError(t, client.Close()) })
			limits, err := client.ConfigGet(t.Context(), "maxclients").Result()
			require.NoError(t, err)
			require.Equal(t, step.limit, limits["maxclients"])
		})
	}
}

func BenchmarkRedisReuse(b *testing.B) {
	containers.Redis(b)
	b.Setenv("PATH", b.TempDir())
	for b.Loop() {
		containers.Redis(b)
	}
}
