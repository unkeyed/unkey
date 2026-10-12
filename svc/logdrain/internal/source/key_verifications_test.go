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

func TestKeyVerificationsRead_Payload(t *testing.T) {
	cfg := containers.ClickHouse(t)
	client, err := clickhouse.New(clickhouse.Config{URL: cfg.HTTPDSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	workspaceID := uid.New(uid.WorkspacePrefix)
	requestID := uid.New(uid.RequestPrefix)
	keySpaceID := uid.New(uid.KeySpacePrefix)
	identityID := uid.New(uid.IdentityPrefix)
	externalID := uid.New(uid.TestPrefix)
	keyID := uid.New(uid.KeyPrefix)
	appID := uid.New(uid.AppPrefix)
	now := time.Now().UnixMilli()
	insertKeyVerifications(t, client, schema.KeyVerification{
		RequestID:    requestID,
		Time:         now - 3600000,
		WorkspaceID:  workspaceID,
		KeySpaceID:   keySpaceID,
		IdentityID:   identityID,
		ExternalID:   externalID,
		KeyID:        keyID,
		Region:       "eu-west-1",
		Source:       schema.SourceGateway,
		AppID:        appID,
		Outcome:      "VALID",
		Tags:         []string{"paid", "production"},
		SpentCredits: 7,
		Latency:      12.5,
	})
	events, cursor, err := source.NewKeyVerifications(client).Read(t.Context(), workspaceID, source.Cursor{Time: now - 1}, time.Now().UnixMilli()+1000, 10, nil)
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, requestID, events[0].EventID)
	require.Equal(t, "key_verifications", events[0].Stream)
	require.Equal(t, now-3600000, events[0].Time)
	require.GreaterOrEqual(t, cursor.Time, now)
	require.Equal(t, events[0].EventID, cursor.EventID)
	require.Equal(t, sink.KeyVerificationPayload{
		RequestID:  requestID,
		KeySpaceID: keySpaceID,
		Identity: &sink.KeyVerificationIdentity{
			ID:         identityID,
			ExternalID: externalID,
		},
		KeyID:  keyID,
		Region: "eu-west-1",
		Source: sink.KeyVerificationSource{
			Type:  schema.SourceGateway,
			AppID: appID,
		},
		Outcome:      "VALID",
		Tags:         []string{"paid", "production"},
		SpentCredits: 7,
	}, events[0].Payload)

	apiRequestID := uid.New(uid.RequestPrefix)
	insertKeyVerifications(t, client, schema.KeyVerification{
		RequestID:   apiRequestID,
		Time:        now,
		WorkspaceID: workspaceID,
		Source:      schema.SourceAPI,
		AppID:       uid.New(uid.AppPrefix),
		Outcome:     "NOT_FOUND",
	})
	filter := &logdrainv1.Config{Stream: &logdrainv1.Config_KeyVerifications{KeyVerifications: &logdrainv1.KeyVerificationStreamConfig{Outcomes: []string{"NOT_FOUND"}}}}
	events, _, err = source.NewKeyVerifications(client).Read(t.Context(), workspaceID, source.Cursor{Time: now - 1}, time.Now().UnixMilli()+1000, 10, filter)
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, sink.KeyVerificationPayload{
		RequestID:  apiRequestID,
		KeySpaceID: "",
		Identity:   nil,
		KeyID:      "",
		Region:     "",
		Source: sink.KeyVerificationSource{
			Type:  schema.SourceAPI,
			AppID: "",
		},
		Outcome:      "NOT_FOUND",
		Tags:         []string{},
		SpentCredits: 0,
	}, events[0].Payload)
}

func insertKeyVerifications(t *testing.T, client *clickhouse.Client, rows ...schema.KeyVerification) {
	t.Helper()
	batch, err := client.Conn().PrepareBatch(t.Context(), clickhouse.InsertQuery[schema.KeyVerification]())
	require.NoError(t, err)
	for i := range rows {
		require.NoError(t, batch.AppendStruct(&rows[i]))
	}
	require.NoError(t, batch.Send())
}

func TestKeyVerificationsRead_FilteredCursorBounds(t *testing.T) {
	cfg := containers.ClickHouse(t)
	client, err := clickhouse.New(clickhouse.Config{URL: cfg.HTTPDSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	workspaceID := uid.New(uid.WorkspacePrefix)
	otherWorkspaceID := uid.New(uid.WorkspacePrefix)
	now := time.Now().UnixMilli()
	var rows []keyVerificationRow
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
		rows = append(rows, keyVerificationRow{
			KeyVerification: schema.KeyVerification{
				RequestID:   row.id,
				Time:        now - 60000,
				WorkspaceID: row.workspace,
				Source:      schema.SourceAPI,
				Outcome:     row.outcome,
			},
			InsertedAt: row.insertedAt,
		})
	}
	insertRows(t, client.Conn(), rows...)
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
	workspaceID := uid.New(uid.WorkspacePrefix)
	now := time.Now().UnixMilli()
	var rows []keyVerificationRow
	for _, row := range []struct{ id, keyspace, outcome string }{
		{"a", "excluded", "RATE_LIMITED"},
		{"b", "selected", "VALID"},
		{"c", "selected", "RATE_LIMITED"},
		{"d", "also_selected", "RATE_LIMITED"},
		{"e", "excluded", "RATE_LIMITED"},
	} {
		rows = append(rows, keyVerificationRow{
			KeyVerification: schema.KeyVerification{
				RequestID:   row.id,
				Time:        now,
				WorkspaceID: workspaceID,
				KeySpaceID:  row.keyspace,
				Source:      schema.SourceAPI,
				Outcome:     row.outcome,
			},
			InsertedAt: now,
		})
	}
	insertRows(t, client.Conn(), rows...)
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
