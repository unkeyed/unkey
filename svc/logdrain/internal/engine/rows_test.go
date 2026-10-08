package engine_test

import (
	"testing"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
)

// Production rows leave inserted_at to the server default, but sources page
// by (inserted_at, event_id), so these rows set it explicitly.

type auditLogRow struct {
	schema.AuditLogV1
	InsertedAt int64 `ch:"inserted_at"`
}

func (auditLogRow) InsertColumns() string {
	return schema.AuditLogV1{}.InsertColumns() + ", `inserted_at`"
}

type keyVerificationRow struct {
	schema.KeyVerification
	InsertedAt int64 `ch:"inserted_at"`
}

func (keyVerificationRow) InsertColumns() string {
	return schema.KeyVerification{}.InsertColumns() + ", `inserted_at`"
}

type ratelimitRow struct {
	schema.Ratelimit
	InsertedAt int64 `ch:"inserted_at"`
}

func (ratelimitRow) InsertColumns() string {
	return schema.Ratelimit{}.InsertColumns() + ", `inserted_at`"
}

type gatewayRequestRow struct {
	schema.FrontlineRequest
	InsertedAt int64 `ch:"inserted_at"`
}

func (gatewayRequestRow) InsertColumns() string {
	return schema.FrontlineRequest{}.InsertColumns() + ", `inserted_at`"
}

type runtimeLogRow struct {
	schema.RuntimeLogV1
	InsertedAt int64 `ch:"inserted_at"`
}

func (runtimeLogRow) InsertColumns() string {
	return schema.RuntimeLogV1{}.InsertColumns() + ", `inserted_at`"
}

func insertRows[T schema.Row](t *testing.T, conn ch.Conn, rows ...T) {
	t.Helper()
	batch, err := conn.PrepareBatch(t.Context(), clickhouse.InsertQuery[T]())
	require.NoError(t, err)
	for i := range rows {
		require.NoError(t, batch.AppendStruct(&rows[i]))
	}
	require.NoError(t, batch.Send())
}
