package logdrains_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	logdrainv1 "github.com/unkeyed/unkey/gen/proto/logdrain/v1"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	logdrains "github.com/unkeyed/unkey/svc/api/routes/v2_logdrains_get_logdrain"
	"google.golang.org/protobuf/proto"
)

func TestGetReturnsSecretSafeConfig(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &logdrains.Handler{DB: h.DB}
	h.Register(route)
	workspaceID := h.Resources().UserWorkspace.ID
	id := uid.New("ld")
	config, err := proto.Marshal(&logdrainv1.Config{
		BatchSize: 73,
		Destination: &logdrainv1.Config_Http{
			Http: &logdrainv1.HttpConfig{
				Url:     "https://logs.example.com/ingest",
				Format:  logdrainv1.HttpBodyFormat_HTTP_BODY_FORMAT_HEC,
				Headers: []*logdrainv1.HttpHeader{{Name: "Authorization", EncryptedValue: "must-not-leak"}},
			},
		},
		Stream: &logdrainv1.Config_Ratelimits{
			Ratelimits: &logdrainv1.RatelimitStreamConfig{
				NamespaceIds: []string{"ns_1"},
				Passed:       []bool{false},
			},
		},
	})
	require.NoError(t, err)
	_, err = h.DB.RW().ExecContext(context.Background(), "INSERT INTO logdrains (id, workspace_id, name, stream, config, lease_id, fencing_token, consecutive_failures, committed_offset_inserted_at, created_at) VALUES (?, ?, 'Test drain', 'ratelimits', ?, '', '', 7, 456, 123)", id, workspaceID, config)
	require.NoError(t, err)
	key := h.CreateRootKey(workspaceID, "unkey:v1:"+workspaceID+":**#*")
	response := testutil.CallRoute[openapi.LogdrainIdRequest, openapi.LogdrainResponse](h, route, http.Header{
		"Authorization": {"Bearer " + key},
		"Content-Type":  {"application/json"},
	}, openapi.LogdrainIdRequest{LogdrainId: id})
	require.Equal(t, http.StatusOK, response.Status, "%s", response.RawBody)
	require.NotContains(t, string(response.RawBody), "must-not-leak")
	require.Equal(t, id, response.Body.Data.Id)
	require.Equal(t, "Test drain", response.Body.Data.Name)
	require.Equal(t, openapi.LogdrainStream("ratelimits"), response.Body.Data.Stream)
	require.Equal(t, openapi.LogdrainStatus("running"), response.Body.Data.Status)
	require.Equal(t, int64(73), response.Body.Data.BatchSize)
	require.Equal(t, []string{"ns_1"}, *response.Body.Data.Filters.NamespaceIds)
	require.Equal(t, []bool{false}, *response.Body.Data.Filters.Passed)
	require.NotNil(t, response.Body.Data.Destination.Http)
	require.Nil(t, response.Body.Data.Destination.Axiom)
	require.Equal(t, "https://logs.example.com/ingest", response.Body.Data.Destination.Http.Url)
	require.Equal(t, openapi.LogdrainDestinationHttpFormat("hec"), response.Body.Data.Destination.Http.Format)
	require.Equal(t, []string{"Authorization"}, response.Body.Data.Destination.Http.Headers)
	require.NotContains(t, string(response.RawBody), `"consecutiveFailures"`)
	require.NotContains(t, string(response.RawBody), `"committedOffsetInsertedAt"`)
	require.Equal(t, int64(123), response.Body.Data.CreatedAt)
	require.NotEmpty(t, response.Body.Meta.RequestId)
	var body struct {
		Data json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(response.RawBody), &body))
	require.JSONEq(t, fmt.Sprintf(`{"id":%q,"name":"Test drain","stream":"ratelimits","status":"running","batchSize":73,"createdAt":123,"filters":{"namespaceIds":["ns_1"],"passed":[false]},"destination":{"http":{"url":"https://logs.example.com/ingest","format":"hec","headers":["Authorization"]}}}`, id), string(body.Data))
}

func TestGetReturnsReadableStatusClasses(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &logdrains.Handler{DB: h.DB}
	h.Register(route)
	workspaceID := h.Resources().UserWorkspace.ID
	id := uid.New("ld")
	config, err := proto.Marshal(&logdrainv1.Config{
		Destination: &logdrainv1.Config_Http{
			Http: &logdrainv1.HttpConfig{Url: "https://logs.example.com"},
		},
		Stream: &logdrainv1.Config_GatewayRequests{
			GatewayRequests: &logdrainv1.GatewayRequestStreamConfig{
				StatusClasses: []logdrainv1.HttpStatusClass{4, 2, 5, 3},
			},
		},
	})
	require.NoError(t, err)
	_, err = h.DB.RW().ExecContext(t.Context(), "INSERT INTO logdrains (id, workspace_id, name, stream, config, lease_id, fencing_token, created_at) VALUES (?, ?, 'Gateway', 'gateway_requests', ?, '', '', 123)", id, workspaceID, config)
	require.NoError(t, err)
	key := h.CreateRootKey(workspaceID, "unkey:v1:"+workspaceID+":logdrains/*#read")
	response := testutil.CallRoute[openapi.LogdrainIdRequest, openapi.LogdrainResponse](h, route, http.Header{
		"Authorization": {"Bearer " + key},
		"Content-Type":  {"application/json"},
	}, openapi.LogdrainIdRequest{LogdrainId: id})
	require.Equal(t, http.StatusOK, response.Status, "%s", response.RawBody)
	require.Contains(t, string(response.RawBody), `"statusClasses":["4xx","2xx","5xx","3xx"]`)
}

func TestGetMissingDrainReturnsNotFound(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &logdrains.Handler{DB: h.DB}
	h.Register(route)
	workspaceID := h.Resources().UserWorkspace.ID
	key := h.CreateRootKey(workspaceID, "unkey:v1:"+workspaceID+":logdrains/*#read")
	response := testutil.CallRoute[openapi.LogdrainIdRequest, openapi.NotFoundErrorResponse](h, route, http.Header{
		"Authorization": {"Bearer " + key},
		"Content-Type":  {"application/json"},
	}, openapi.LogdrainIdRequest{LogdrainId: uid.New("ld")})
	require.Equal(t, http.StatusNotFound, response.Status, "%s", response.RawBody)
	require.Equal(t, http.StatusNotFound, response.Body.Error.Status)
	require.Equal(t, "https://unkey.com/docs/errors/unkey/data/logdrain_not_found", response.Body.Error.Type)
	require.NotEmpty(t, response.Body.Meta.RequestId)
}

func TestGetRequiresID(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &logdrains.Handler{DB: h.DB}
	h.Register(route)
	workspaceID := h.Resources().UserWorkspace.ID
	key := h.CreateRootKey(workspaceID, "unkey:v1:"+workspaceID+":logdrains/*#read")
	response := testutil.CallRoute[openapi.LogdrainIdRequest, openapi.BadRequestErrorResponse](h, route, http.Header{
		"Authorization": {"Bearer " + key},
		"Content-Type":  {"application/json"},
	}, openapi.LogdrainIdRequest{})
	require.Equal(t, http.StatusBadRequest, response.Status, "%s", response.RawBody)
	require.Contains(t, response.Body.Error.Type, "application/invalid_input")
}
