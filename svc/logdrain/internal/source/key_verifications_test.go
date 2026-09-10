package source_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/logdrain/internal/source"
)

func TestKeyVerificationsRead_Payload(t *testing.T) {
	cfg := containers.ClickHouse(t)
	client, err := clickhouse.New(clickhouse.Config{URL: cfg.HTTPDSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	workspaceID := uid.New("workspace")
	now := time.Now().UnixMilli()
	require.NoError(t, client.Conn().Exec(t.Context(), `INSERT INTO key_verifications_raw_v2
		(workspace_id, request_id, time, key_space_id, identity_id, external_id, key_id,
		region, source, app_id, outcome, tags, spent_credits, latency)
		VALUES (?, 'req_1', ?, 'ks_1', 'id_1', 'customer_1', 'key_1', 'eu-west-1',
		'gateway', 'app_1', 'VALID', ['paid', 'production'], 7, 12.5)`, workspaceID, now-3600000))
	events, cursor, err := source.NewKeyVerifications(client).Read(t.Context(), workspaceID, source.Cursor{Time: now - 1}, time.Now().UnixMilli()+1000, 10, nil)
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, "req_1", events[0].EventID)
	require.Equal(t, "key_verifications", events[0].Stream)
	require.GreaterOrEqual(t, cursor.Time, now)
	require.Equal(t, events[0].EventID, cursor.EventID)
	encoded, err := json.Marshal(events[0].Payload)
	require.NoError(t, err)
	require.JSONEq(t, `{"request_id":"req_1","key_space_id":"ks_1","identity":{"id":"id_1","externalId":"customer_1"},"key_id":"key_1","region":"eu-west-1","source":{"type":"gateway","appId":"app_1"},"outcome":"VALID","tags":["paid","production"],"spent_credits":7}`, string(encoded))

	require.NoError(t, client.Conn().Exec(t.Context(), `INSERT INTO key_verifications_raw_v2
		(workspace_id, request_id, time, source, app_id, outcome)
		VALUES (?, 'req_api', ?, 'api', 'ignored_app', 'NOT_FOUND')`, workspaceID, now))
	events, _, err = source.NewKeyVerifications(client).Read(t.Context(), workspaceID, source.Cursor{Time: now - 1}, time.Now().UnixMilli()+1000, 10, []string{"NOT_FOUND"})
	require.NoError(t, err)
	require.Len(t, events, 1)
	encoded, err = json.Marshal(events[0].Payload)
	require.NoError(t, err)
	require.JSONEq(t, `{"request_id":"req_api","key_space_id":"","identity":{"id":"","externalId":""},"key_id":"","region":"","source":{"type":"api"},"outcome":"NOT_FOUND","tags":[],"spent_credits":0}`, string(encoded))
}

func TestKeyVerificationsRead_FilteredCursorBounds(t *testing.T) {
	cfg := containers.ClickHouse(t)
	client, err := clickhouse.New(clickhouse.Config{URL: cfg.HTTPDSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	workspaceID := uid.New("workspace")
	otherWorkspaceID := uid.New("workspace")
	now := time.Now().UnixMilli()
	for _, row := range []struct {
		workspace, id, outcome string
		insertedAt             int64
	}{
		{workspaceID, "z", "RATE_LIMITED", now - 1},
		{workspaceID, "a", "RATE_LIMITED", now},
		{workspaceID, "b", "VALID", now},
		{workspaceID, "c", "RATE_LIMITED", now},
		{workspaceID, "d", "EXPIRED", now},
		{workspaceID, "e", "RATE_LIMITED", now},
		{workspaceID, "a", "EXPIRED", now + 1},
		{workspaceID, "f", "RATE_LIMITED", now + 2},
		{otherWorkspaceID, "g", "RATE_LIMITED", now},
	} {
		require.NoError(t, client.Conn().Exec(t.Context(), `INSERT INTO key_verifications_raw_v2
			(workspace_id, request_id, inserted_at, time, outcome)
			VALUES (?, ?, ?, ?, ?)`, row.workspace, row.id, row.insertedAt, now-60000, row.outcome))
	}
	reader := source.NewKeyVerifications(client)
	from := source.Cursor{Time: now, EventID: "a"}
	page, next, err := reader.Read(t.Context(), workspaceID, from, now+2, 2, []string{"RATE_LIMITED", "EXPIRED"})
	require.NoError(t, err)
	require.Len(t, page, 2)
	require.Equal(t, "c", page[0].EventID)
	require.Equal(t, "d", page[1].EventID)
	require.Equal(t, source.Cursor{Time: now, EventID: "d"}, next)
	page, next, err = reader.Read(t.Context(), workspaceID, next, now+2, 2, []string{"RATE_LIMITED", "EXPIRED"})
	require.NoError(t, err)
	require.Len(t, page, 2)
	require.Equal(t, "e", page[0].EventID)
	require.Equal(t, "a", page[1].EventID)
	require.Equal(t, source.Cursor{Time: now + 1, EventID: "a"}, next)
	page, final, err := reader.Read(t.Context(), workspaceID, next, now+2, 2, nil)
	require.NoError(t, err)
	require.Empty(t, page)
	require.Equal(t, next, final)
	page, _, err = reader.Read(t.Context(), workspaceID, from, now+2, 2, nil)
	require.NoError(t, err)
	require.Len(t, page, 2)
	require.Equal(t, "b", page[0].EventID)
	require.Equal(t, "c", page[1].EventID)
}
