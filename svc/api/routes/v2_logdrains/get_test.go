package logdrains_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	logdrainv1 "github.com/unkeyed/unkey/gen/proto/logdrain/v1"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
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
	response := testutil.CallRoute[openapi.LogdrainIdRequest, openapi.LogdrainResponse](h, route, http.Header{"Authorization": {"Bearer " + key}, "Content-Type": {"application/json"}}, openapi.LogdrainIdRequest{LogdrainId: id})
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
	require.Equal(t, openapi.LogdrainDestinationHttpFormat("ndjson"), response.Body.Data.Destination.Http.Format)
	require.Equal(t, []string{"Authorization"}, response.Body.Data.Destination.Http.Headers)
	require.Zero(t, response.Body.Data.ConsecutiveFailures)
	require.Zero(t, response.Body.Data.CommittedOffsetInsertedAt)
	require.Equal(t, int64(123), response.Body.Data.CreatedAt)
	require.NotEmpty(t, response.Body.Meta.RequestId)
}

func TestGetMissingDrainReturnsNotFound(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &logdrains.Get{DB: h.DB}
	h.Register(route)
	workspaceID := h.Resources().UserWorkspace.ID
	key := h.CreateRootKey(workspaceID, "unkey:v1:"+workspaceID+":logdrains/*#read")
	response := testutil.CallRoute[openapi.LogdrainIdRequest, openapi.NotFoundErrorResponse](h, route, http.Header{"Authorization": {"Bearer " + key}, "Content-Type": {"application/json"}}, openapi.LogdrainIdRequest{LogdrainId: uid.New("ld")})
	require.Equal(t, http.StatusNotFound, response.Status, "%s", response.RawBody)
	require.Equal(t, http.StatusNotFound, response.Body.Error.Status)
	require.Equal(t, "https://unkey.com/docs/errors/unkey/data/logdrain_not_found", response.Body.Error.Type)
	require.NotEmpty(t, response.Body.Meta.RequestId)
}

func TestGetRequiresID(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &logdrains.Get{DB: h.DB}
	h.Register(route)
	workspaceID := h.Resources().UserWorkspace.ID
	key := h.CreateRootKey(workspaceID, "unkey:v1:"+workspaceID+":logdrains/*#read")
	response := testutil.CallRoute[openapi.LogdrainIdRequest, openapi.BadRequestErrorResponse](h, route, http.Header{"Authorization": {"Bearer " + key}, "Content-Type": {"application/json"}}, openapi.LogdrainIdRequest{})
	require.Equal(t, http.StatusBadRequest, response.Status, "%s", response.RawBody)
	require.Contains(t, response.Body.Error.Type, "application/invalid_input")
}
