package integration

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clock"
	sharedconfig "github.com/unkeyed/unkey/pkg/config"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/dockertest"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/svc/api"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
)

// ApiConfig holds configuration for dynamic API container creation
type ApiConfig struct {
	Nodes         int
	MysqlDSN      string
	ClickhouseDSN string
	KafkaBrokers  []string
}

// ApiCluster represents a cluster of API containers
type ApiCluster struct {
	Addrs []string
}

// Harness is a test harness for creating and managing a cluster of API nodes
type Harness struct {
	t             *testing.T
	ctx           context.Context
	cancel        context.CancelFunc
	instanceAddrs []string
	Seed          *seed.Seeder
	dbDSN         string
	DB            db.Database
	CH            clickhouse.ClickHouse
	apiCluster    *ApiCluster
	redisUrl      string
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
		URL: clickhouseHostDSN,
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
		PrimaryDSN:  mysqlHostDSN,
		ReadOnlyDSN: "",
	})
	require.NoError(t, err)

	h := &Harness{
		t:             t,
		ctx:           ctx,
		cancel:        cancel,
		instanceAddrs: []string{},
		Seed:          seed.New(t, db, nil),
		dbDSN:         mysqlHostDSN,
		DB:            db,
		CH:            ch,
		apiCluster:    nil, // Will be set later
		redisUrl:      dockertest.Redis(t),
	}

	h.Seed.Seed(ctx)

	// For docker DSN, use docker service name
	clickhouseDockerDSN := "clickhouse://default:password@clickhouse:9000?secure=false&skip_verify=true&dial_timeout=10s"

	// Create dynamic API container cluster for chaos testing
	kafkaBrokers := containers.Kafka(t)

	cluster := h.RunAPI(ApiConfig{
		Nodes:         config.NumNodes,
		MysqlDSN:      mysqlDockerDSN,
		ClickhouseDSN: clickhouseDockerDSN,
		KafkaBrokers:  kafkaBrokers,
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
		Addrs: make([]string, config.Nodes),
	}

	// Start each API node as a goroutine
	for i := 0; i < config.Nodes; i++ {
		// Create ephemeral listener
		ln, err := net.Listen("tcp", ":0") //nolint: gosec
		require.NoError(h.t, err, "Failed to create ephemeral listener")

		cluster.Addrs[i] = fmt.Sprintf("http://%s", ln.Addr().String())

		// Create API config for this node using host connections
		mysqlHostCfg := containers.MySQL(h.t)
		mysqlHostCfg.DBName = "unkey" // Set the database name
		clickhouseHostDSN := containers.ClickHouse(h.t)
		kafkaBrokers := containers.Kafka(h.t)
		vaultURL, vaultToken := containers.Vault(h.t)
		apiConfig := api.Config{
			HttpPort:           7070,
			Platform:           "test",
			Image:              "test",
			Listener:           ln,
			RedisURL:           h.redisUrl,
			Region:             "test",
			InstanceID:         fmt.Sprintf("test-node-%d", i),
			Clock:              clock.New(),
			TestMode:           true,
			PrometheusPort:     0,
			TLSConfig:          nil,
			MaxRequestBodySize: 0,
			Database: sharedconfig.DatabaseConfig{
				Primary:         mysqlHostCfg.FormatDSN(),
				ReadonlyReplica: "",
			},
			ClickHouse: api.ClickHouseConfig{
				URL:          clickhouseHostDSN,
				AnalyticsURL: "",
				ProxyToken:   "",
			},
			Otel: sharedconfig.OtelConfig{
				Enabled:           false,
				TraceSamplingRate: 0.0,
			},
			TLS: sharedconfig.TLSFiles{
				CertFile: "",
				KeyFile:  "",
			},
			Vault: sharedconfig.VaultConfig{
				URL:   vaultURL,
				Token: vaultToken,
			},
			Kafka: &api.KafkaConfig{
				Brokers:                kafkaBrokers, // Use host brokers for test runner connections
				CacheInvalidationTopic: "",
			},
			Ctrl: api.CtrlConfig{
				URL:   "http://ctrl:7091",
				Token: "your-local-dev-key",
			},
			Pprof: &api.PprofConfig{
				Username: "unkey",
				Password: "password",
			},
			Logging: sharedconfig.LoggingConfig{
				SampleRate:    1.0,
				SlowThreshold: time.Second,
			},
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
		healthURL := fmt.Sprintf("http://%s/v2/liveness", ln.Addr().String())
		for attempt := range maxAttempts {
			//nolint:gosec // Health check URL is constructed from controlled Docker container address
			resp, err := http.Get(healthURL)
			if err == nil {
				_ = resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					h.t.Logf("API server %d started on %s", i, ln.Addr().String())
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
			// Note: Don't call ln.Close() here as the zen server
			// will properly close the listener during graceful shutdown
		})
	}

	return cluster
}

// GetClusterAddrs returns the addresses of all API containers
func (h *Harness) GetClusterAddrs() []string {
	if h.apiCluster == nil {
		return []string{}
	}
	return h.apiCluster.Addrs
}
