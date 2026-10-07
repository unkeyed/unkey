package source_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	logdrainv1 "github.com/unkeyed/unkey/gen/proto/logdrain/v1"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/logdrain/internal/source"
	"github.com/unkeyed/unkey/svc/logdrain/sink"
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
	require.Equal(t, now-3600000, events[0].Time)
	require.GreaterOrEqual(t, cursor.Time, now)
	require.Equal(t, events[0].EventID, cursor.EventID)
	require.Equal(t, sink.KeyVerificationPayload{
		RequestID:  "req_1",
		KeySpaceID: "ks_1",
		Identity: &sink.KeyVerificationIdentity{
			ID:         "id_1",
			ExternalID: "customer_1",
		},
		KeyID:  "key_1",
		Region: "eu-west-1",
		Source: sink.KeyVerificationSource{
			Type:  "gateway",
			AppID: "app_1",
		},
		Outcome:      "VALID",
		Tags:         []string{"paid", "production"},
		SpentCredits: 7,
	}, events[0].Payload)

	require.NoError(t, client.Conn().Exec(t.Context(), `INSERT INTO key_verifications_raw_v2
		(workspace_id, request_id, time, source, app_id, outcome)
		VALUES (?, 'req_api', ?, 'api', 'ignored_app', 'NOT_FOUND')`, workspaceID, now))
	filter := &logdrainv1.Config{Stream: &logdrainv1.Config_KeyVerifications{KeyVerifications: &logdrainv1.KeyVerificationStreamConfig{Outcomes: []string{"NOT_FOUND"}}}}
	events, _, err = source.NewKeyVerifications(client).Read(t.Context(), workspaceID, source.Cursor{Time: now - 1}, time.Now().UnixMilli()+1000, 10, filter)
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, sink.KeyVerificationPayload{
		RequestID:  "req_api",
		KeySpaceID: "",
		Identity:   nil,
		KeyID:      "",
		Region:     "",
		Source: sink.KeyVerificationSource{
			Type:  "api",
			AppID: "",
		},
		Outcome:      "NOT_FOUND",
		Tags:         []string{},
		SpentCredits: 0,
	}, events[0].Payload)
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
	filter := &logdrainv1.Config{Stream: &logdrainv1.Config_KeyVerifications{KeyVerifications: &logdrainv1.KeyVerificationStreamConfig{Outcomes: []string{"RATE_LIMITED", "EXPIRED"}}}}
	page, next, err := reader.Read(t.Context(), workspaceID, from, now+2, 2, filter)
	require.NoError(t, err)
	require.Len(t, page, 2)
	require.Equal(t, "c", page[0].EventID)
	require.Equal(t, "d", page[1].EventID)
	require.Equal(t, source.Cursor{Time: now, EventID: "d"}, next)
	page, next, err = reader.Read(t.Context(), workspaceID, next, now+2, 2, filter)
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

func TestKeyVerificationsRead_KeySpacesBeforeLimit(t *testing.T) {
	cfg := containers.ClickHouse(t)
	client, err := clickhouse.New(clickhouse.Config{URL: cfg.HTTPDSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	workspaceID := uid.New("workspace")
	now := time.Now().UnixMilli()
	for _, row := range []struct{ id, keyspace, outcome string }{
		{"a", "excluded", "RATE_LIMITED"},
		{"b", "selected", "VALID"},
		{"c", "selected", "RATE_LIMITED"},
		{"d", "also_selected", "RATE_LIMITED"},
		{"e", "excluded", "RATE_LIMITED"},
	} {
		require.NoError(t, client.Conn().Exec(t.Context(), `INSERT INTO key_verifications_raw_v2
			(workspace_id, request_id, inserted_at, time, key_space_id, outcome)
			VALUES (?, ?, ?, ?, ?, ?)`, workspaceID, row.id, now, now, row.keyspace, row.outcome))
	}
	reader := source.NewKeyVerifications(client)
	filter := &logdrainv1.Config{Stream: &logdrainv1.Config_KeyVerifications{KeyVerifications: &logdrainv1.KeyVerificationStreamConfig{
		KeySpaceIds: []string{"selected", "also_selected"},
	}}}
	page, _, err := reader.Read(t.Context(), workspaceID, source.Cursor{Time: now}, now+1, 2, filter)
	require.NoError(t, err)
	require.Len(t, page, 2)
	require.Equal(t, "b", page[0].EventID)
	require.Equal(t, "c", page[1].EventID)
	filter.GetKeyVerifications().Outcomes = []string{"RATE_LIMITED"}
	page, cursor, err := reader.Read(t.Context(), workspaceID, source.Cursor{Time: now}, now+1, 1, filter)
	require.NoError(t, err)
	require.Len(t, page, 1)
	require.Equal(t, "c", page[0].EventID)
	page, cursor, err = reader.Read(t.Context(), workspaceID, cursor, now+1, 1, filter)
	require.NoError(t, err)
	require.Len(t, page, 1)
	require.Equal(t, "d", page[0].EventID)
	page, final, err := reader.Read(t.Context(), workspaceID, cursor, now+1, 1, filter)
	require.NoError(t, err)
	require.Empty(t, page)
	require.Equal(t, cursor, final)
}
