package logdrains_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	logdrainv1 "github.com/unkeyed/unkey/gen/proto/logdrain/v1"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	updateRoute "github.com/unkeyed/unkey/svc/api/routes/v2_logdrains_update_logdrain"
	"google.golang.org/protobuf/proto"
)

func TestUpdateAuthorizesOnlyMatchingWriteGrant(t *testing.T) {
	for _, tc := range []struct {
		name, action               string
		wrongDrain, wrongWorkspace bool
		status                     int
	}{
		{name: "exact drain", action: "write", status: http.StatusOK},
		{name: "wrong drain", action: "write", wrongDrain: true, status: http.StatusForbidden},
		{name: "wrong workspace grant", action: "write", wrongWorkspace: true, status: http.StatusForbidden},
		{name: "read only", action: "read", status: http.StatusForbidden},
		{name: "delete only", action: "delete", status: http.StatusForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h, route, id, config := seedUpdateDrain(t)
			workspaceID := h.Resources().UserWorkspace.ID
			grantWorkspace, grantID := workspaceID, id
			if tc.wrongWorkspace {
				grantWorkspace = uid.New("ws")
			}
			if tc.wrongDrain {
				grantID = uid.New("ld")
			}
			key := h.CreateRootKey(workspaceID, urn.New().Workspace(grantWorkspace).Logdrain(grantID).String()+"#"+tc.action)
			res := testutil.CallRoute[json.RawMessage, openapi.ForbiddenErrorResponse](h, route, updateHeaders(key), json.RawMessage(`{"logdrainId":"`+id+`","name":"Renamed"}`))
			require.Equal(t, tc.status, res.Status, "%s", res.RawBody)
			if tc.status == http.StatusOK {
				assertUpdateState(t, h, id, "Renamed", config, true)
			} else {
				require.Equal(t, "https://unkey.com/docs/errors/unkey/authorization/insufficient_permissions", res.Body.Error.Type)
				assertUpdateState(t, h, id, "Original", config, false)
			}
		})
	}
}

func TestUpdatePartialFieldsPreserveOmittedConfiguration(t *testing.T) {
	for _, tc := range []struct{ name, patch string }{
		{"name only", `"name":"  Renamed  "`},
		{"batch only", `"batchSize":23`},
		{"HTTP URL only", `"destination":{"http":{"url":"https://changed.example.com"}}`},
		{"HTTP format only", `"destination":{"http":{"format":"ndjson"}}`},
		{"clear ratelimit namespace", `"filters":{"namespaceIds":[]}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h, route, id, expected := seedUpdateDrain(t)
			key := h.CreateRootKey(h.Resources().UserWorkspace.ID, urn.New().Workspace(h.Resources().UserWorkspace.ID).Logdrain(id).String()+"#write")
			res := testutil.CallRoute[json.RawMessage, openapi.LogdrainMutationResponse](h, route, updateHeaders(key), json.RawMessage(`{"logdrainId":"`+id+`",`+tc.patch+`}`))
			require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
			require.Equal(t, id, res.Body.Data.Id)
			require.NotEmpty(t, res.Body.Meta.RequestId)
			name := "Original"
			switch tc.name {
			case "name only":
				name = "Renamed"
			case "batch only":
				expected.BatchSize = 23
			case "HTTP URL only":
				expected.GetHttp().Url = "https://changed.example.com"
			case "HTTP format only":
				expected.GetHttp().Format = logdrainv1.HttpBodyFormat_HTTP_BODY_FORMAT_NDJSON
			case "clear ratelimit namespace":
				expected.GetRatelimits().NamespaceIds = nil
			}
			assertUpdateState(t, h, id, name, expected, true)
		})
	}
}

func seedUpdateDrain(t *testing.T) (*testutil.Harness, *updateRoute.Handler, string, *logdrainv1.Config) {
	t.Helper()
	h := testutil.NewHarness(t)
	route := &updateRoute.Handler{
		DB:        h.DB,
		Vault:     h.Vault,
		Auditlogs: h.Auditlogs,
		Clock:     h.Clock,
	}
	h.Register(route)
	id := uid.New("ld")
	config := &logdrainv1.Config{
		BatchSize: 47,
		Destination: &logdrainv1.Config_Http{Http: &logdrainv1.HttpConfig{
			Url:    "https://original.example.com",
			Format: logdrainv1.HttpBodyFormat_HTTP_BODY_FORMAT_HEC,
		}},
		Stream: &logdrainv1.Config_Ratelimits{Ratelimits: &logdrainv1.RatelimitStreamConfig{
			NamespaceIds: []string{"ns_keep"},
			Passed:       []bool{false},
		}},
	}
	encoded, err := proto.Marshal(config)
	require.NoError(t, err)
	_, err = h.DB.RW().ExecContext(t.Context(), "INSERT INTO logdrains (id, workspace_id, name, stream, config, status, committed_offset_inserted_at, committed_offset_event_id, lease_id, fencing_token, created_at) VALUES (?, ?, 'Original', 'ratelimits', ?, 'paused_by_user', 4567, 'event_keep', '', '', 123)", id, h.Resources().UserWorkspace.ID, encoded)
	require.NoError(t, err)
	return h, route, id, config
}

func updateHeaders(key string) http.Header {
	return http.Header{"Authorization": {"Bearer " + key}, "Content-Type": {"application/json"}}
}

func assertUpdateState(t *testing.T, h *testutil.Harness, id, name string, config *logdrainv1.Config, updated bool) {
	t.Helper()
	var storedName, status, event string
	var encoded []byte
	var offset int64
	require.NoError(t, h.DB.RW().QueryRowContext(t.Context(), "SELECT name, status, config, committed_offset_inserted_at, committed_offset_event_id FROM logdrains WHERE id = ?", id).Scan(&storedName, &status, &encoded, &offset, &event))
	require.Equal(t, name, storedName)
	require.Equal(t, "paused_by_user", status)
	require.Equal(t, int64(4567), offset)
	require.Equal(t, "event_keep", event)
	var stored logdrainv1.Config
	require.NoError(t, proto.Unmarshal(encoded, &stored))
	require.True(t, proto.Equal(config, &stored), "stored config: %s", &stored)
	logs := h.FindAuditLogsByTargetID(t.Context(), t, id)
	if !updated {
		require.Empty(t, logs)
		return
	}
	require.Len(t, logs, 1)
	require.Equal(t, "logdrain.update", logs[0].Event)
	require.Equal(t, h.Resources().UserWorkspace.ID, logs[0].WorkspaceID)
	var updatedAt int64
	require.NoError(t, h.DB.RW().QueryRowContext(t.Context(), "SELECT updated_at FROM logdrains WHERE id = ?", id).Scan(&updatedAt))
	require.Equal(t, h.Clock.Now().UnixMilli(), updatedAt)
}
