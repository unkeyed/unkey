package clickhouse_test

import (
	"context"
	"testing"
	"time"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
)

// TestRawEventInsertedAtDefaults guarantees that raw event writers omit
// inserted_at and ClickHouse computes it without changing the event time.
func TestRawEventInsertedAtDefaults(t *testing.T) {
	t.Parallel()

	cfg := containers.ClickHouse(t)
	opts, err := ch.ParseDSN(cfg.DSN)
	require.NoError(t, err)

	conn, err := ch.Open(opts)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })

	ctx := context.Background()
	workspaceID := uid.New(uid.WorkspacePrefix)
	eventTime := time.Now().Add(-24 * time.Hour).UnixMilli()

	t.Run("key verifications", func(t *testing.T) {
		assertServerInsertedAt(t, ctx, conn, schema.KeyVerification{
			RequestID:   uid.New(uid.RequestPrefix),
			Time:        eventTime,
			WorkspaceID: workspaceID,
			Tags:        []string{},
		}, workspaceID, eventTime)
	})

	t.Run("ratelimits", func(t *testing.T) {
		assertServerInsertedAt(t, ctx, conn, schema.Ratelimit{
			RequestID:   uid.New(uid.RequestPrefix),
			Time:        eventTime,
			WorkspaceID: workspaceID,
		}, workspaceID, eventTime)
	})

	t.Run("API requests", func(t *testing.T) {
		assertServerInsertedAt(t, ctx, conn, schema.ApiRequest{
			RequestID:       uid.New(uid.RequestPrefix),
			Time:            eventTime,
			WorkspaceID:     workspaceID,
			QueryParams:     map[string][]string{},
			RequestHeaders:  []string{},
			ResponseHeaders: []string{},
		}, workspaceID, eventTime)
	})

	t.Run("frontline requests", func(t *testing.T) {
		assertServerInsertedAt(t, ctx, conn, schema.FrontlineRequest{
			RequestID:       uid.New(uid.RequestPrefix),
			Time:            eventTime,
			WorkspaceID:     workspaceID,
			QueryParams:     map[string][]string{},
			RequestHeaders:  []string{},
			ResponseHeaders: []string{},
		}, workspaceID, eventTime)
	})
}

// assertServerInsertedAt proves that the insert omits the server-computed
// column and that the generated timestamp falls within the insert.
func assertServerInsertedAt[T schema.Row](
	t *testing.T,
	ctx context.Context,
	conn ch.Conn,
	row T,
	workspaceID string,
	eventTime int64,
) {
	t.Helper()

	query := clickhouse.InsertQuery[T]()
	require.NotContains(t, query, "`inserted_at`")

	before := time.Now().UnixMilli()
	batch, err := conn.PrepareBatch(ctx, query)
	require.NoError(t, err)
	require.NoError(t, batch.AppendStruct(&row))
	require.NoError(t, batch.Send())
	after := time.Now().UnixMilli()

	var gotEventTime, gotInsertedAt int64
	err = conn.QueryRow(ctx,
		"SELECT time, inserted_at FROM "+row.Table()+" WHERE workspace_id = ? AND time = ? LIMIT 1",
		workspaceID,
		eventTime,
	).Scan(&gotEventTime, &gotInsertedAt)
	require.NoError(t, err)
	require.Equal(t, eventTime, gotEventTime)
	require.GreaterOrEqual(t, gotInsertedAt, before)
	require.LessOrEqual(t, gotInsertedAt, after)
}
