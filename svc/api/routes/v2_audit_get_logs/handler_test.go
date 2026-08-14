package handler

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestCursorRoundTrip(t *testing.T) {
	original := auditCursor{InsertedAt: 1786581490123, EventID: "log_abc123"}
	encoded := encodeCursor(original)
	require.NotEmpty(t, encoded)

	decoded, err := decodeCursor(encoded)
	require.NoError(t, err)
	require.Equal(t, original, decoded)
}

func TestDecodeCursor_Invalid(t *testing.T) {
	t.Run("not base64", func(t *testing.T) {
		_, err := decodeCursor("!!!not base64!!!")
		require.Error(t, err)
	})
	t.Run("not json", func(t *testing.T) {
		_, err := decodeCursor("bm90anNvbg") // base64("notjson")
		require.Error(t, err)
	})
	t.Run("missing event id", func(t *testing.T) {
		_, err := decodeCursor(encodeCursor(auditCursor{InsertedAt: 123}))
		require.Error(t, err)
	})
}

func TestMapAuditLog(t *testing.T) {
	row := schema.AuditLogV1{
		EventID:       "log_1",
		Time:          1786581490123,
		InsertedAt:    1786581491456,
		WorkspaceID:   "ws_1",
		Bucket:        "unkey_mutations",
		Source:        "platform",
		Event:         "key.create",
		Description:   "created a key",
		ActorType:     "root_key",
		ActorID:       "root_1",
		ActorName:     "CI root key",
		ActorMeta:     json.RawMessage(`{"role":"admin"}`),
		RemoteIP:      "1.2.3.4",
		UserAgent:     "unkey-cli/1.0",
		Meta:          json.RawMessage(`{"foo":"bar"}`),
		TargetTypes:   []string{"key", "api"},
		TargetIDs:     []string{"key_1", "api_1"},
		TargetNames:   []string{"a key", "an api"},
		TargetMetas:   []json.RawMessage{json.RawMessage(`{"k":"v"}`), json.RawMessage(`{}`)},
		CorrelationID: "corr_1",
	}

	got := mapAuditLog(row)

	require.Equal(t, "log_1", got.AuditLogId)
	require.Equal(t, "v1", got.Version)
	require.Equal(t, openapi.Success, got.Outcome)
	require.Equal(t, "platform", got.Source)

	// epoch-ms -> RFC3339 UTC
	require.Equal(t, time.UnixMilli(1786581490123).UTC(), got.Time)
	require.Equal(t, time.UnixMilli(1786581491456).UTC(), got.InsertedAt)
	require.Equal(t, "2026", got.Time.UTC().Format("2006"))

	// actor
	require.Equal(t, "root_key", got.Actor.Type)
	require.Equal(t, "root_1", got.Actor.Id)
	require.NotNil(t, got.Actor.Name)
	require.Equal(t, "CI root key", *got.Actor.Name)
	require.NotNil(t, got.Actor.Meta)
	require.Equal(t, "admin", (*got.Actor.Meta)["role"])

	// context
	require.NotNil(t, got.Context.IpAddress)
	require.Equal(t, "1.2.3.4", *got.Context.IpAddress)
	require.NotNil(t, got.Context.UserAgent)

	// resources from parallel arrays
	require.Len(t, got.Resources, 2)
	require.Equal(t, "key", got.Resources[0].Type)
	require.Equal(t, "key_1", got.Resources[0].Id)
	require.NotNil(t, got.Resources[0].Name)
	require.Equal(t, "a key", *got.Resources[0].Name)
	require.NotNil(t, got.Resources[0].Meta)
	require.Equal(t, "v", (*got.Resources[0].Meta)["k"])
	require.Equal(t, "api", got.Resources[1].Type)
	// {} meta collapses to nil (omitted)
	require.Nil(t, got.Resources[1].Meta)

	// event meta + correlation
	require.NotNil(t, got.Meta)
	require.Equal(t, "bar", (*got.Meta)["foo"])
	require.NotNil(t, got.CorrelationId)
	require.Equal(t, "corr_1", *got.CorrelationId)
}

func TestMapAuditLog_EmptyOptionalsOmitted(t *testing.T) {
	row := schema.AuditLogV1{
		EventID:     "log_2",
		Time:        1,
		InsertedAt:  2,
		WorkspaceID: "ws_1",
		Source:      "platform",
		Event:       "workspace.update",
		ActorType:   "system",
		ActorID:     "system",
		// No name, no meta, no ip/ua, no targets, no correlation.
	}

	got := mapAuditLog(row)
	require.Nil(t, got.Actor.Name)
	require.Nil(t, got.Actor.Meta)
	require.Nil(t, got.Context.IpAddress)
	require.Nil(t, got.Context.UserAgent)
	require.Nil(t, got.CorrelationId)
	require.Nil(t, got.Meta)
	// resources must be an empty slice, never nil, so it marshals to [].
	require.NotNil(t, got.Resources)
	require.Len(t, got.Resources, 0)

	b, err := json.Marshal(got)
	require.NoError(t, err)
	require.Contains(t, string(b), `"resources":[]`)
}
