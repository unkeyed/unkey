package clickhouse

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auditlog"
)

// TestEncodeAuditLogEvents guarantees that canonical targets map to aligned
// ClickHouse Nested columns without exposing that storage shape to callers.
func TestEncodeAuditLogEvents(t *testing.T) {
	t.Run("event with multiple targets", func(t *testing.T) {
		events := []auditlog.Event{
			{
				EventID:     "log_1",
				Time:        1_000,
				WorkspaceID: "ws_a",
				Bucket:      "unkey_mutations",
				Source:      auditlog.EventSourcePlatform,
				Event:       "key.create",
				Description: "Created key foo",
				Actor: auditlog.EventActor{
					Type: "user",
					ID:   "user_1",
					Name: "Alice",
					Meta: map[string]any{"role": "admin"},
				},
				RemoteIP:  "1.2.3.4",
				UserAgent: "curl",
				Targets: []auditlog.EventTarget{
					{Type: "key", ID: "key_1", Name: "foo", Meta: map[string]any{"k": "v"}},
					{Type: "api", ID: "api_1", Name: "myapi"},
				},
			},
		}

		rows, err := EncodeAuditLogEvents(events)
		require.NoError(t, err)
		require.Len(t, rows, 1, "one event must produce one row regardless of target count")

		row := rows[0]
		require.Equal(t, "log_1", row.EventID)
		require.Equal(t, "ws_a", row.WorkspaceID)
		require.Equal(t, "unkey_mutations", row.Bucket)
		require.Equal(t, auditlog.EventSourcePlatform, row.Source)
		require.Equal(t, "key.create", row.Event)
		require.Equal(t, "Created key foo", row.Description)
		require.Equal(t, "Alice", row.ActorName)
		require.JSONEq(t, `{"role":"admin"}`, string(row.ActorMeta))
		require.Equal(t, "1.2.3.4", row.RemoteIP)
		require.Equal(t, []string{"key", "api"}, row.TargetTypes)
		require.Equal(t, []string{"key_1", "api_1"}, row.TargetIDs)
		require.Equal(t, []string{"foo", "myapi"}, row.TargetNames)
		require.Len(t, row.TargetMetas, 2)
		require.JSONEq(t, `{"k":"v"}`, string(row.TargetMetas[0]))
		require.JSONEq(t, `{}`, string(row.TargetMetas[1]))
	})

	t.Run("event with no targets", func(t *testing.T) {
		rows, err := EncodeAuditLogEvents([]auditlog.Event{
			{
				EventID:     "log_2",
				Time:        2_000,
				WorkspaceID: "ws_a",
				Bucket:      "unkey_mutations",
				Event:       "system.ping",
				Description: "Pinged system",
				Actor:       auditlog.EventActor{Type: "system", ID: "system"},
			},
		})
		require.NoError(t, err)
		require.Len(t, rows, 1)
		require.Equal(t, auditlog.EventSourcePlatform, rows[0].Source)
		require.JSONEq(t, `{}`, string(rows[0].ActorMeta))
		require.JSONEq(t, `{}`, string(rows[0].Meta))
		require.Empty(t, rows[0].TargetTypes)
		require.Empty(t, rows[0].TargetIDs)
		require.Empty(t, rows[0].TargetNames)
		require.Empty(t, rows[0].TargetMetas)
	})

	t.Run("invalid target metadata", func(t *testing.T) {
		_, err := EncodeAuditLogEvents([]auditlog.Event{
			{
				EventID: "log_3",
				Targets: []auditlog.EventTarget{
					{ID: "key_1", Meta: map[string]any{"invalid": make(chan int)}},
				},
			},
		})
		require.ErrorContains(t, err, "encode target_meta event_id=log_3 target_id=key_1")
	})
}

// TestNewAuditLogBuffer_ReportsEncodingErrors guarantees that invalid metadata
// reaches the configured error handler instead of failing on the request path.
func TestNewAuditLogBuffer_ReportsEncodingErrors(t *testing.T) {
	type flushError struct {
		table    string
		rowCount int
		err      error
	}

	errorChannel := make(chan flushError, 1)
	buffer := NewAuditLogBuffer(nil, BufferConfig{
		Name:          "audit_log_test",
		BatchSize:     1,
		BufferSize:    1,
		FlushInterval: time.Hour,
		Consumers:     1,
		Drop:          false,
		OnFlushError: func(_ context.Context, table string, rowCount int, err error) {
			errorChannel <- flushError{table: table, rowCount: rowCount, err: err}
		},
	})
	t.Cleanup(buffer.Close)

	buffer.Buffer(auditlog.Event{
		EventID: "log_invalid",
		Meta:    map[string]any{"invalid": make(chan int)},
	})

	select {
	case flushErr := <-errorChannel:
		require.Equal(t, "default.audit_logs_raw_v1", flushErr.table)
		require.Equal(t, 1, flushErr.rowCount)
		require.ErrorContains(t, flushErr.err, "encode meta event_id=log_invalid")
	case <-time.After(time.Second):
		require.Fail(t, "audit log buffer did not report the encoding error")
	}
}
