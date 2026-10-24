package globalcounter_test

import (
	"database/sql"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/mysql"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
)

// newDatabase returns an isolated MySQL whose read replica holds one connection,
// so tests can block refreshes by occupying it. The pool is returned for that.
func newDatabase(t *testing.T) (mysql.MySQL, *sql.DB) {
	t.Helper()
	config := containers.MySQLIsolated(t)
	database, err := mysql.New(mysql.Config{PrimaryDSN: config.DSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	readPool, err := sql.Open("mysql", config.DSN)
	require.NoError(t, err)
	readPool.SetMaxOpenConns(1)
	return mysql.NewFromReplicas(mysql.NewReplicaFromDB(readPool, "ro", sqlcomment.Static{}), database.RW()), readPool
}

// awaitStartupPasses blocks until the counter's background passes that run right
// after New, eviction and compaction, are over: the pool has used a connection
// and then stayed idle for a while. Tests that hold the pool's only connection or
// insert rows those passes would touch need that first.
func awaitStartupPasses(t *testing.T, pool *sql.DB) {
	t.Helper()
	const quiet = 300 * time.Millisecond
	var idleSince time.Time
	require.Eventually(t, func() bool {
		stats := pool.Stats()
		if stats.OpenConnections == 0 || stats.InUse > 0 {
			idleSince = time.Time{}
			return false
		}
		if idleSince.IsZero() {
			idleSince = time.Now()
		}
		return time.Since(idleSince) >= quiet
	}, 10*time.Second, 10*time.Millisecond)
}

var testRegistry = prometheus.NewRegistry()

func init() {
	lazy.SetRegistry(testRegistry)
}

// metricValue returns a gauge or counter sample, or zero before its first
// use. Tests compare before and after values because the registry is shared.
func metricValue(t *testing.T, name string, labels map[string]string) float64 {
	t.Helper()
	families, err := testRegistry.Gather()
	require.NoError(t, err)
	for _, family := range families {
		if family.GetName() != name {
			continue
		}
	samples:
		for _, sample := range family.GetMetric() {
			for key, value := range labels {
				matched := false
				for _, label := range sample.GetLabel() {
					matched = matched || (label.GetName() == key && label.GetValue() == value)
				}
				if !matched {
					continue samples
				}
			}
			if sample.GetGauge() != nil {
				return sample.GetGauge().GetValue()
			}
			return sample.GetCounter().GetValue()
		}
	}
	return 0
}

func persistedRows(t *testing.T, result string) float64 {
	t.Helper()
	return metricValue(t, "unkey_globalcounter_persist_rows_total", map[string]string{"result": result})
}

func memoryEntries(t *testing.T) float64 {
	t.Helper()
	return metricValue(t, "unkey_globalcounter_memory_entries", nil)
}

func addsTotal(t *testing.T, result string) float64 {
	t.Helper()
	return metricValue(t, "unkey_globalcounter_adds_total", map[string]string{"result": result})
}

func getsTotal(t *testing.T, result string) float64 {
	t.Helper()
	return metricValue(t, "unkey_globalcounter_gets_total", map[string]string{"result": result})
}

// testCompactDelayMs is the delay seeded rows give compaction. Compaction runs
// in every counter a test creates, and a seeded row it merged would come back
// when the test updates the seeded row.
const testCompactDelayMs = int64(time.Hour / time.Millisecond)
