package source_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	logdrainv1 "github.com/unkeyed/unkey/gen/proto/logdrain/v1"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/logdrain/internal/source"
	"github.com/unkeyed/unkey/svc/logdrain/sink"
)

func TestRatelimitsRead_PayloadAndRepeatedChecks(t *testing.T) {
	cfg := containers.ClickHouse(t)
	client, err := clickhouse.New(clickhouse.Config{URL: cfg.HTTPDSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	workspace := uid.New(uid.WorkspacePrefix)
	requestID := uid.New(uid.RequestPrefix)
	namespaceID := uid.New(uid.RatelimitNamespacePrefix)
	now := time.Now().UnixMilli()
	check := schema.Ratelimit{
		RequestID:   requestID,
		Time:        now - 3600000,
		WorkspaceID: workspace,
		NamespaceID: namespaceID,
		Identifier:  "customer@example.com",
		Passed:      false,
		Latency:     1.5,
		OverrideID:  "",
		Limit:       100,
		Remaining:   0,
		ResetAt:     now + 60000,
		Tokens:      3,
	}
	batch, err := client.Conn().PrepareBatch(t.Context(), clickhouse.InsertQuery[schema.Ratelimit]())
	require.NoError(t, err)
	for range 2 {
		require.NoError(t, batch.AppendStruct(&check))
	}
	require.NoError(t, batch.Send())
	reader := source.NewRatelimits(client)
	events, cursor, err := reader.Read(t.Context(), workspace, source.Cursor{Time: now - 1}, time.Now().UnixMilli()+1000, 2, nil)
	require.NoError(t, err)
	require.Len(t, events, 2)
	require.Equal(t, "ratelimits", events[0].Stream)
	require.Equal(t, now-3600000, events[0].Time)
	require.GreaterOrEqual(t, cursor.Time, now)
	require.Equal(t, requestID, cursor.EventID)
	require.Equal(t, sink.RatelimitPayload{
		RequestID:   requestID,
		NamespaceID: namespaceID,
		Identifier:  "customer@example.com",
		Passed:      false,
		OverrideID:  "",
		Limit:       100,
		Remaining:   0,
		ResetAt:     now + 60000,
		Tokens:      3,
		Source:      "api",
	}, events[0].Payload)
	events, cursor, err = reader.Read(t.Context(), workspace, cursor, time.Now().UnixMilli()+1000, 1, nil)
	require.NoError(t, err)
	require.Empty(t, events)
	require.Equal(t, requestID, cursor.EventID)
}

func TestRatelimitsRead_CombinedFiltersBeforeLimit(t *testing.T) {
	cfg := containers.ClickHouse(t)
	client, err := clickhouse.New(clickhouse.Config{URL: cfg.HTTPDSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	workspace := uid.New(uid.WorkspacePrefix)
	now := time.Now().UnixMilli()
	var rows []ratelimitRow
	for _, row := range []struct {
		workspace, request, namespace, identifier string
		passed                                    bool
		insertedAt                                int64
	}{
		{workspace, "a", "other", "customer", false, now},
		{workspace, "b", "ns", "other", true, now},
		{workspace, "c", "ns", "customer", true, now},
		{uid.New(uid.WorkspacePrefix), "d", "ns", "customer", false, now},
		{workspace, "e", "ns", "other", false, now},
		{workspace, "f", "ns2", "customer2", false, now},
		{workspace, "g", "ns", "customer", false, now + 1},
	} {
		rows = append(rows, ratelimitRow{
			Ratelimit: schema.Ratelimit{
				RequestID:   row.request,
				Time:        now - 60000,
				WorkspaceID: row.workspace,
				NamespaceID: row.namespace,
				Identifier:  row.identifier,
				Passed:      row.passed,
				OverrideID:  "override_1",
			},
			InsertedAt: row.insertedAt,
		})
	}
	insertRows(t, client.Conn(), rows...)
	filter := &logdrainv1.Config{Stream: &logdrainv1.Config_Ratelimits{Ratelimits: &logdrainv1.RatelimitStreamConfig{
		NamespaceIds: []string{"ns", "ns2"}, Passed: []bool{false},
	}}}
	reader := source.NewRatelimits(client)
	page, next, err := reader.Read(t.Context(), workspace, source.Cursor{Time: now}, now+1, 1, filter)
	require.NoError(t, err)
	require.Len(t, page, 1)
	require.Equal(t, "e", page[0].EventID)
	require.Equal(t, sink.RatelimitPayload{
		RequestID:   "e",
		NamespaceID: "ns",
		Identifier:  "other",
		Passed:      false,
		OverrideID:  "override_1",
		Limit:       0,
		Remaining:   0,
		ResetAt:     0,
		Tokens:      0,
		Source:      "api",
	}, page[0].Payload)
	page, next, err = reader.Read(t.Context(), workspace, next, now+1, 1, filter)
	require.NoError(t, err)
	require.Len(t, page, 1)
	require.Equal(t, source.Cursor{Time: now, EventID: "f"}, next)
	page, final, err := reader.Read(t.Context(), workspace, next, now+1, 1, filter)
	require.NoError(t, err)
	require.Empty(t, page)
	require.Equal(t, next, final)
	page, next, err = reader.Read(t.Context(), workspace, next, now+2, 1, filter)
	require.NoError(t, err)
	require.Len(t, page, 1)
	require.Equal(t, source.Cursor{Time: now + 1, EventID: "g"}, next)
}
