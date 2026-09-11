package logdrains_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	logdrainv1 "github.com/unkeyed/unkey/gen/proto/logdrain/v1"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	logdrains "github.com/unkeyed/unkey/svc/api/routes/v2_logdrains"
	"google.golang.org/protobuf/proto"
)

func TestGetReturnsSecretSafeConfig(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &logdrains.Get{DB: h.DB}
	h.Register(route)
	workspaceID := h.Resources().UserWorkspace.ID
	id := uid.New("ld")
	config, err := proto.Marshal(&logdrainv1.Config{
		BatchSize: 73,
		Destination: &logdrainv1.Config_Http{Http: &logdrainv1.HttpConfig{
			Url: "https://logs.example.com/ingest", Format: logdrainv1.HttpBodyFormat_HTTP_BODY_FORMAT_NDJSON,
			Headers: []*logdrainv1.HttpHeader{{Name: "Authorization", EncryptedValue: "must-not-leak"}},
		}},
		Stream: &logdrainv1.Config_Ratelimits{Ratelimits: &logdrainv1.RatelimitStreamConfig{NamespaceIds: []string{"ns_1"}, Passed: []bool{false}}},
	})
	require.NoError(t, err)
	_, err = h.DB.RW().ExecContext(context.Background(), "INSERT INTO logdrains (id, workspace_id, name, stream, config, lease_id, fencing_token, created_at) VALUES (?, ?, 'Test drain', 'ratelimits', ?, '', '', 123)", id, workspaceID, config)
	require.NoError(t, err)
	key := h.CreateRootKey(workspaceID, "unkey:v1:"+workspaceID+":**#*")
	response := testutil.CallRoute[map[string]string, map[string]any](h, route, http.Header{"Authorization": {"Bearer " + key}, "Content-Type": {"application/json"}}, map[string]string{"logdrainId": id})
	require.Equal(t, http.StatusOK, response.Status, "%s", response.RawBody)
	require.NotContains(t, string(response.RawBody), "must-not-leak")
	require.JSONEq(t, `{"id":"`+id+`","name":"Test drain","stream":"ratelimits","status":"running","batchSize":73,"filters":{"namespaceIds":["ns_1"],"passed":[false]},"destination":{"http":{"url":"https://logs.example.com/ingest","format":"ndjson","headers":["Authorization"]}},"consecutiveFailures":0,"committedOffsetInsertedAt":0,"createdAt":123}`, mustData(t, response.Body))
}

func TestGetMissingDrainReturnsNotFound(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &logdrains.Get{DB: h.DB}
	h.Register(route)
	workspaceID := h.Resources().UserWorkspace.ID
	key := h.CreateRootKey(workspaceID, "unkey:v1:"+workspaceID+":logdrains/*#read")
	response := testutil.CallRoute[map[string]string, map[string]any](h, route, http.Header{"Authorization": {"Bearer " + key}, "Content-Type": {"application/json"}}, map[string]string{"logdrainId": "ld_missing"})
	require.Equal(t, http.StatusNotFound, response.Status, "%s", response.RawBody)
}

func mustData(t *testing.T, body *map[string]any) string {
	t.Helper()
	require.NotNil(t, body)
	data, err := json.Marshal((*body)["data"])
	require.NoError(t, err)
	return string(data)
}
