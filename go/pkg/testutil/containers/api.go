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
)

type Cluster struct {
	Addrs     []string
	Instances []*dockertest.Resource
}

func (c *Containers) RunAPI(nodes int, mysqlDSN string) Cluster {

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

	_, _, redisAddr := c.RunRedis()

	cluster := Cluster{
		Instances: []*dockertest.Resource{},
		Addrs:     []string{},
	}
	for i := 0; i < nodes; i++ {
		instanceId := uid.New(uid.InstancePrefix)
		// Define run options
		// nolint:exhaustruct
		runOpts := &dockertest.RunOptions{
			Name:         instanceId,
			Repository:   imageName,
			Networks:     []*dockertest.Network{c.network},
			ExposedPorts: []string{"7070", "9090", "9091"},
			Cmd:          []string{"api"},
			Env: []string{
				"UNKEY_HTTP_PORT=7070",
				"UNKEY_CLUSTER=true",
				"UNKEY_CLUSTER_GOSSIP_PORT=9090",
				"UNKEY_CLUSTER_RPC_PORT=9091",
				fmt.Sprintf("UNKEY_CLUSTER_DISCOVERY_REDIS_URL=redis://%s", redisAddr),
				fmt.Sprintf("UNKEY_DATABASE_PRIMARY_DSN=%s", mysqlDSN),
			},
		}

		t0 := time.Now()
		instance, err := c.pool.RunWithOptions(runOpts)
		require.NoError(c.t, err)
		c.t.Logf("starting %s took %s", instanceId, time.Since(t0))

		c.t.Cleanup(func() {
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
