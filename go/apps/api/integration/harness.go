package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/clickhouse"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/port"
	"github.com/unkeyed/unkey/go/pkg/testutil/containers"
	"github.com/unkeyed/unkey/go/pkg/testutil/seed"
)

// Harness is a test harness for creating and managing a cluster of API nodes
type Harness struct {
	t             *testing.T
	ctx           context.Context
	cancel        context.CancelFunc
	instanceAddrs []string
	ports         *port.FreePort
	containerMgr  *containers.Containers
	Seed          *seed.Seeder
	dbDSN         string
	DB            db.Database
	CH            clickhouse.ClickHouse
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

	containerMgr := containers.New(t)

	containerMgr.RunOtel(true)

	// Start ClickHouse container with migrations
	clickhouseHostDSN, clickhouseDockerDSN := containerMgr.RunClickHouse()

	// Create real ClickHouse client
	ch, err := clickhouse.New(clickhouse.Config{
		URL:    clickhouseHostDSN,
		Logger: logging.NewNoop(),
	})
	require.NoError(t, err)

	mysqlHostDSN, mysqlDockerDSN := containerMgr.RunMySQL()
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
		containerMgr:  containerMgr,
		instanceAddrs: []string{},
		Seed:          seed.New(t, db),
		dbDSN:         mysqlHostDSN,
		DB:            db,
		CH:            ch,
	}

	h.Seed.Seed(ctx)

	cluster := containerMgr.RunAPI(containers.ApiConfig{
		Nodes:         config.NumNodes,
		MysqlDSN:      mysqlDockerDSN,
		ClickhouseDSN: clickhouseDockerDSN,
	})
	h.instanceAddrs = cluster.Addrs
	return h
}

func (h *Harness) Resources() seed.Resources {
	return h.Seed.Resources
}
