package integration

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api"
	"github.com/unkeyed/unkey/go/pkg/clickhouse"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/port"
	"github.com/unkeyed/unkey/go/pkg/testutil/containers"
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

	// Get service configurations
	clickhouseHostDSN := containers.ClickHouse(t)

	// Create real ClickHouse client
	ch, err := clickhouse.New(clickhouse.Config{
		URL:    clickhouseHostDSN,
		Logger: logging.NewNoop(),
	})
	require.NoError(t, err)

	mysqlHostCfg := containers.MySQL(t)
	mysqlHostCfg.DBName = "unkey"
	mysqlHostDSN := mysqlHostCfg.FormatDSN()

	// For docker DSN, use docker service name
	mysqlDockerCfg := containers.MySQL(t)
	mysqlDockerCfg.Addr = "mysql:3306"
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
		instanceAddrs: []string{},
		Seed:          seed.New(t, db),
		dbDSN:         mysqlHostDSN,
		DB:            db,
		CH:            ch,
		apiCluster:    nil, // Will be set later
	}

	h.Seed.Seed(ctx)

	// For docker DSN, use docker service name
	clickhouseDockerDSN := "clickhouse://default:password@clickhouse:9000?secure=false&skip_verify=true&dial_timeout=10s"

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
	cluster := &ApiCluster{
		Addrs:     make([]string, config.Nodes),
		Resources: make([]*dockertest.Resource, config.Nodes), // Not used but kept for compatibility
	}

	// Start each API node as a goroutine
	for i := 0; i < config.Nodes; i++ {
		// Find an available port
		portFinder := port.New()
		nodePort := portFinder.Get()

		cluster.Addrs[i] = fmt.Sprintf("http://localhost:%d", nodePort)

		// Create API config for this node using host connections
		mysqlHostCfg := containers.MySQL(h.t)
		mysqlHostCfg.DBName = "unkey" // Set the database name
		clickhouseHostDSN := containers.ClickHouse(h.t)
		redisHostAddr := containers.Redis(h.t)

		apiConfig := api.Config{
			Platform:                "test",
			Image:                   "test",
			HttpPort:                nodePort,
			DatabasePrimary:         mysqlHostCfg.FormatDSN(),
			DatabaseReadonlyReplica: "",
			ClickhouseURL:           clickhouseHostDSN,
			RedisUrl:                redisHostAddr,
			Region:                  "test",
			InstanceID:              fmt.Sprintf("test-node-%d", i),
			Clock:                   clock.New(),
			TestMode:                true,
			OtelEnabled:             false,
			OtelTraceSamplingRate:   0.0,
			PrometheusPort:          0,
			TLSConfig:               nil,
			VaultMasterKeys:         []string{"Ch9rZWtfMmdqMFBJdVhac1NSa0ZhNE5mOWlLSnBHenFPENTt7an5MRogENt9Si6wms4pQ2XIvqNSIgNpaBenJmXgcInhu6Nfv2U="}, // Test key from docker-compose
		}

		// Start API server in goroutine
		ctx, cancel := context.WithCancel(context.Background())

		// Channel to get startup result
		startupResult := make(chan error, 1)

		go func(nodeID int, cfg api.Config) {
			defer func() {
				if r := recover(); r != nil {
					h.t.Logf("API server %d panicked: %v", nodeID, r)
					startupResult <- fmt.Errorf("panic: %v", r)
				}
			}()

			// Give some time for the server to indicate it's starting
			go func() {
				time.Sleep(500 * time.Millisecond)
				startupResult <- nil // Indicate startup attempt
			}()

			err := api.Run(ctx, cfg)
			if err != nil && ctx.Err() == nil {
				h.t.Logf("API server %d failed: %v", nodeID, err)
				select {
				case startupResult <- err:
				default:
				}
			}
		}(i, apiConfig)

		// Wait for startup indication
		select {
		case err := <-startupResult:
			if err != nil {
				require.NoError(h.t, err, "API server %d startup failed", i)
			}
		case <-time.After(2 * time.Second):
			require.Fail(h.t, "API server %d startup timeout", i)
		}

		// Wait for server to start
		maxAttempts := 30
		for attempt := 0; attempt < maxAttempts; attempt++ {
			resp, err := http.Get(fmt.Sprintf("http://localhost:%d/v2/liveness", nodePort))
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					h.t.Logf("API server %d started on port %d", i, nodePort)
					break
				}
			}
			if attempt == maxAttempts-1 {
				require.NoError(h.t, err, "API server %d failed to start", i)
			}
			time.Sleep(100 * time.Millisecond)
		}

		// Register cleanup
		h.t.Cleanup(func() {
			cancel()
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
