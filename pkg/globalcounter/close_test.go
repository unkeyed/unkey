package globalcounter_test

import (
	"database/sql"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/globalcounter"
	"github.com/unkeyed/unkey/pkg/mysql"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestCounter_ClosePersistsEveryLocalCounter(t *testing.T) {
	config := containers.MySQLIsolated(t)
	database, err := mysql.New(mysql.Config{PrimaryDSN: config.DSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	droppedName := uid.New(uid.RatelimitNamespacePrefix)
	queuedName := uid.New(uid.RatelimitNamespacePrefix)

	writePool, err := sql.Open("mysql", config.DSN)
	require.NoError(t, err)
	writePool.SetMaxOpenConns(1)
	t.Cleanup(func() { require.NoError(t, writePool.Close()) })
	writer, err := globalcounter.New(globalcounter.Config{
		DB:    mysql.NewFromReplicas(database.RO(), mysql.NewReplicaFromDB(writePool, "rw", sqlcomment.Static{})),
		Clock: clock.NewTestClock(),
	})
	require.NoError(t, err)
	conn, err := writePool.Conn(t.Context())
	require.NoError(t, err)
	release := sync.OnceFunc(func() { require.NoError(t, conn.Close()) })
	t.Cleanup(release)

	waited := writePool.Stats().WaitDuration
	require.NoError(t, writer.Add(globalcounter.AddParams{Name: droppedName, Delta: 7}))
	require.Eventually(t, func() bool { return writePool.Stats().WaitDuration > waited }, 10*time.Second, 10*time.Millisecond)
	release()
	require.NoError(t, writer.Add(globalcounter.AddParams{Name: queuedName, Delta: -4}))
	require.NoError(t, writer.Add(globalcounter.AddParams{Name: queuedName, Delta: -5}))

	writer.Close()
	writer.Close()

	observer, err := globalcounter.New(globalcounter.Config{DB: database, Clock: clock.NewTestClock()})
	require.NoError(t, err)
	t.Cleanup(observer.Close)
	for name, want := range map[string]int64{droppedName: 7, queuedName: -9} {
		got, err := observer.Get(t.Context(), globalcounter.GetParams{Name: name})
		require.NoError(t, err)
		require.Equal(t, want, got, name)
	}
	require.NotPanics(t, func() {
		_ = writer.Add(globalcounter.AddParams{Name: queuedName, Delta: 1})
		_, _ = writer.Get(t.Context(), globalcounter.GetParams{Name: queuedName})
	}, "calls after Close must not crash the process")
}

func TestCounter_CloseSkipsRetiredWriterIDs(t *testing.T) {
	config := containers.MySQLIsolated(t)
	database, err := mysql.New(mysql.Config{PrimaryDSN: config.DSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	retiredName := uid.New(uid.RatelimitNamespacePrefix)
	currentName := uid.New(uid.RatelimitNamespacePrefix)

	writePool, err := sql.Open("mysql", config.DSN)
	require.NoError(t, err)
	writePool.SetMaxOpenConns(1)
	t.Cleanup(func() { require.NoError(t, writePool.Close()) })
	writer, err := globalcounter.New(globalcounter.Config{
		DB:    mysql.NewFromReplicas(database.RO(), mysql.NewReplicaFromDB(writePool, "rw", sqlcomment.Static{})),
		Clock: clock.NewTestClock(),
	})
	require.NoError(t, err)
	conn, err := writePool.Conn(t.Context())
	require.NoError(t, err)
	release := sync.OnceFunc(func() { require.NoError(t, conn.Close()) })
	t.Cleanup(release)

	waited := writePool.Stats().WaitDuration
	require.NoError(t, writer.Add(globalcounter.AddParams{Name: retiredName, Delta: 7}))
	require.Eventually(t, func() bool { return writePool.Stats().WaitDuration > waited }, 10*time.Second, 10*time.Millisecond)
	// The snapshot was dropped. After two rotations its writer id is older than
	// the previous one, so a compaction may already have merged the rows of that id.
	globalcounter.RotateOnce(writer)
	globalcounter.RotateOnce(writer)
	release()
	require.NoError(t, writer.Add(globalcounter.AddParams{Name: currentName, Delta: 3}))

	writer.Close()

	observer, err := globalcounter.New(globalcounter.Config{DB: database, Clock: clock.NewTestClock()})
	require.NoError(t, err)
	t.Cleanup(observer.Close)
	for name, want := range map[string]int64{retiredName: 0, currentName: 3} {
		got, err := observer.Get(t.Context(), globalcounter.GetParams{Name: name})
		require.NoError(t, err)
		require.Equal(t, want, got, name)
	}
}
