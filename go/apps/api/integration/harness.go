package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/clickhouse"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/port"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/seed"
)

// ApiConfig holds configuration for dynamic API container creation
type ApiConfig struct {
	Nodes         int
	MysqlDSN      string
	ClickhouseDSN string
}

// ApiCluster represents a cluster of API containers
type ApiCluster struct {
	Addrs     []string
	Resources []*dockertest.Resource
}

// Harness is a test harness for creating and managing a cluster of API nodes
type Harness struct {
	t             *testing.T
	ctx           context.Context
	cancel        context.CancelFunc
	instanceAddrs []string
	ports         *port.FreePort
	services      *testutil.TestServices
	Seed          *seed.Seeder
	dbDSN         string
	DB            db.Database
	CH            clickhouse.ClickHouse
	apiCluster    *ApiCluster
}

// Config contains configuration options for the test harness
type Config struct {
	// NumNodes is the number of API nodes to create in the cluster
	NumNodes int
}

// New creates a new cluster test harness
func New(t *testing.T, config Config) *Harness {
	t.Helper()

	require.Greater(t, config.NumNodes, 0)
	ctx, cancel := context.WithCancel(context.Background())

	services := testutil.NewTestServices()

	// Get ClickHouse connection strings
	clickhouseHostDSN, clickhouseDockerDSN := services.ClickHouse()

	// Create real ClickHouse client
	ch, err := clickhouse.New(clickhouse.Config{
		URL:    clickhouseHostDSN,
		Logger: logging.NewNoop(),
	})
	require.NoError(t, err)

	mysqlHostCfg, mysqlDockerCfg := services.MySQL()
	mysqlHostCfg.DBName = "unkey"
	mysqlHostDSN := mysqlHostCfg.FormatDSN()

	mysqlDockerCfg.DBName = "unkey"
	mysqlDockerDSN := mysqlDockerCfg.FormatDSN()
	db, err := db.New(db.Config{
		Logger:      logging.NewNoop(),
		PrimaryDSN:  mysqlHostDSN,
		ReadOnlyDSN: "",
	})
	require.NoError(t, err)

	h := &Harness{
		t:             t,
		ctx:           ctx,
		cancel:        cancel,
		ports:         port.New(),
		services:      services,
		instanceAddrs: []string{},
		Seed:          seed.New(t, db),
		dbDSN:         mysqlHostDSN,
		DB:            db,
		CH:            ch,
	}

	h.Seed.Seed(ctx)

	// Create dynamic API container cluster for chaos testing
	cluster := h.RunAPI(ApiConfig{
		Nodes:         config.NumNodes,
		MysqlDSN:      mysqlDockerDSN,
		ClickhouseDSN: clickhouseDockerDSN,
	})
	h.apiCluster = cluster
	h.instanceAddrs = cluster.Addrs
	return h
}

func (h *Harness) Resources() seed.Resources {
	return h.Seed.Resources
}

// RunAPI creates a cluster of API containers for chaos testing
func (h *Harness) RunAPI(config ApiConfig) *ApiCluster {
	// Create Docker pool for dynamic container management
	pool, err := dockertest.NewPool("")
	require.NoError(h.t, err)

	err = pool.Client.Ping()
	require.NoError(h.t, err)

	// Get or create the test network
	networks, err := pool.NetworksByName("unkey_default")
	require.NoError(h.t, err)

	var network *dockertest.Network
	for _, found := range networks {
		if found.Network.Name == "unkey_default" {
			network = &found
			break
		}
	}
	if network == nil {
		network, err = pool.CreateNetwork("unkey_default")
		require.NoError(h.t, err)
	}

	cluster := &ApiCluster{
		Addrs:     make([]string, config.Nodes),
		Resources: make([]*dockertest.Resource, config.Nodes),
	}

	// Create API container instances
	for i := 0; i < config.Nodes; i++ {
		containerName := fmt.Sprintf("unkey-api-test-%d", i)

		runOpts := &dockertest.RunOptions{
			Name:       containerName,
			Repository: "apiv2", // Built by make build-docker
			Tag:        "latest",
			Networks:   []*dockertest.Network{network},
			Env: []string{
				"UNKEY_HTTP_PORT=7070",
				fmt.Sprintf("UNKEY_DATABASE_PRIMARY=%s", config.MysqlDSN),
				fmt.Sprintf("UNKEY_CLICKHOUSE_URL=%s", config.ClickhouseDSN),
				"UNKEY_REDIS_URL=redis://redis:6379",
			},
			Cmd: []string{"run", "api"},
		}

		// Try to create container, with retry logic for race conditions
		var resource *dockertest.Resource
		for attempt := 0; attempt < 10; attempt++ {
			existing, exists := pool.ContainerByName(containerName)
			if exists {
				resource = existing
				break
			}

			resource, err = pool.RunWithOptions(runOpts)
			if err == nil {
				break
			}
			time.Sleep(time.Duration(attempt) * time.Second)
		}
		require.NoError(h.t, err, "Failed to create API container %d", i)

		cluster.Resources[i] = resource
		cluster.Addrs[i] = fmt.Sprintf("http://%s:7070", resource.GetIPInNetwork(network))

		// Register cleanup
		h.t.Cleanup(func() {
			_ = pool.Purge(resource)
		})
	}

	return cluster
}

// StopContainer stops a specific API container (for chaos testing)
func (h *Harness) StopContainer(index int) error {
	if h.apiCluster == nil || index >= len(h.apiCluster.Resources) {
		return fmt.Errorf("invalid container index: %d", index)
	}

	pool, err := dockertest.NewPool("")
	if err != nil {
		return err
	}
	return pool.Client.StopContainer(h.apiCluster.Resources[index].Container.ID, 10)
}

// StartContainer starts a stopped API container (for chaos testing)
func (h *Harness) StartContainer(index int) error {
	if h.apiCluster == nil || index >= len(h.apiCluster.Resources) {
		return fmt.Errorf("invalid container index: %d", index)
	}

	pool, err := dockertest.NewPool("")
	if err != nil {
		return err
	}
	return pool.Client.StartContainer(h.apiCluster.Resources[index].Container.ID, nil)
}

// GetClusterAddrs returns the addresses of all API containers
func (h *Harness) GetClusterAddrs() []string {
	if h.apiCluster == nil {
		return []string{}
	}
	return h.apiCluster.Addrs
}
