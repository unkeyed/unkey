package containers

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"runtime"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/uid"
	"github.com/unkeyed/unkey/go/pkg/vault/keys"
)

type Cluster struct {
	Addrs     []string
	Instances []*dockertest.Resource
}

type ApiConfig struct {
	Nodes         int
	MysqlDSN      string
	ClickhouseDSN string
}

func (c *Containers) RunAPI(config ApiConfig) Cluster {

	// Get the path to the current file
	_, currentFilePath, _, _ := runtime.Caller(0)

	// Navigate from the current file to the project root (go/)
	// We're going from go/pkg/testutil/containers/ up to go/
	projectRoot := filepath.Join(filepath.Dir(currentFilePath), "../../../")

	imageName := "apiv2"

	t0 := time.Now()
	// nolint:exhaustruct
	err := c.pool.Client.BuildImage(docker.BuildImageOptions{
		Name:         imageName,
		Dockerfile:   "Dockerfile",
		ContextDir:   projectRoot,
		OutputStream: io.Discard,
	})
	require.NoError(c.t, err)
	c.t.Logf("building %s took %s", imageName, time.Since(t0))

	_, _, redisUrl := c.RunRedis()

	cluster := Cluster{
		Instances: []*dockertest.Resource{},
		Addrs:     []string{},
	}

	_, vaultMasterKey, err := keys.GenerateMasterKey()
	require.NoError(c.t, err)

	for i := 0; i < config.Nodes; i++ {
		instanceId := uid.New(uid.InstancePrefix)
		// Define run options
		// nolint:exhaustruct
		runOpts := &dockertest.RunOptions{
			Name:         instanceId,
			Repository:   imageName,
			Networks:     []*dockertest.Network{c.network},
			ExposedPorts: []string{"7070"},
			Cmd:          []string{"api"},
			Env: []string{
				"UNKEY_HTTP_PORT=7070",
				"UNKEY_OTEL=true",
				"OTEL_EXPORTER_OTLP_ENDPOINT=http://otel:4318",
				"OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf",
				"UNKEY_TEST_MODE=true",
				"UNKEY_REGION=local_docker",
				fmt.Sprintf("UNKEY_CLICKHOUSE_URL=%s", config.ClickhouseDSN),
				fmt.Sprintf("UNKEY_REDIS_URL=%s", redisUrl),
				fmt.Sprintf("UNKEY_DATABASE_PRIMARY=%s", config.MysqlDSN),
				fmt.Sprintf("UNKEY_VAULT_MASTER_KEYS=%s", vaultMasterKey),
			},
		}

		t0 := time.Now()
		instance, err := c.pool.RunWithOptions(runOpts)
		require.NoError(c.t, err)
		c.t.Logf("starting %s took %s", instanceId, time.Since(t0))

		c.t.Cleanup(func() {
			require.NoError(c.t, c.pool.Client.StopContainer(instance.Container.ID, uint(15)))
			require.NoError(c.t, c.pool.Purge(instance))
		})

		addr := fmt.Sprintf("localhost:%s", instance.GetPort("7070/tcp"))

		require.NoError(c.t, c.pool.Retry(func() error {
			res, err := http.DefaultClient.Get(fmt.Sprintf("http://%s/v2/liveness", addr))
			if err != nil {
				return err
			}
			defer res.Body.Close()
			if res.StatusCode != http.StatusOK {
				return fmt.Errorf("unexpected status code: %d", res.StatusCode)
			}

			return nil
		}))

		cluster.Instances = append(cluster.Instances, instance)
		cluster.Addrs = append(cluster.Addrs, addr)

	}

	return cluster
}
