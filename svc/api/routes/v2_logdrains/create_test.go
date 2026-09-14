package logdrains_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	logdrains "github.com/unkeyed/unkey/svc/api/routes/v2_logdrains"
)

func TestCreatePersistsEncryptedDestination(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &logdrains.Create{DB: h.DB, Vault: h.Vault, Auditlogs: h.Auditlogs, Clock: h.Clock}
	get := &logdrains.Get{DB: h.DB}
	h.Register(route)
	h.Register(get)
	workspaceID := h.Resources().UserWorkspace.ID
	_, err := h.DB.RW().ExecContext(context.Background(), "UPDATE `limits` SET logdrains_max = 1 WHERE workspace_id = ?", workspaceID)
	require.NoError(t, err)
	key := h.CreateRootKey(workspaceID, "unkey:v1:"+workspaceID+":logdrains/*#write", "unkey:v1:"+workspaceID+":logdrains/*#read")
	headers := http.Header{"Authorization": {"Bearer " + key}, "Content-Type": {"application/json"}}
	response := testutil.CallRoute[openapi.CreateLogdrainRequest, openapi.LogdrainMutationResponse](h, route, headers, openapi.CreateLogdrainRequest{
		Name: "My logs", Stream: "audit_logs", BatchSize: ptr.P(int64(17)),
		Destination: openapi.LogdrainDestinationWrite{Axiom: &openapi.LogdrainAxiomWrite{Dataset: ptr.P("production"), Token: ptr.P("secret-token")}},
		Filters:     &openapi.LogdrainFilters{EventTypes: ptr.P([]string{"  key.create  "})},
	})
	require.Equal(t, http.StatusOK, response.Status, "%s", response.RawBody)
	require.NotEmpty(t, response.Body.Data.Id)
	require.NotEmpty(t, response.Body.Meta.RequestId)
	result := testutil.CallRoute[openapi.LogdrainIdRequest, openapi.LogdrainResponse](h, get, headers, openapi.LogdrainIdRequest{LogdrainId: response.Body.Data.Id})
	require.Equal(t, http.StatusOK, result.Status, "%s", result.RawBody)
	require.Equal(t, int64(17), result.Body.Data.BatchSize)
	require.Equal(t, "production", result.Body.Data.Destination.Axiom.Dataset)
	require.Equal(t, []string{"key.create"}, *result.Body.Data.Filters.EventTypes)
	require.NotContains(t, string(result.RawBody), "secret-token")
	var stored []byte
	require.NoError(t, h.DB.RW().QueryRowContext(context.Background(), "SELECT config FROM logdrains WHERE id = ?", response.Body.Data.Id).Scan(&stored))
	require.NotContains(t, string(stored), "secret-token")
	events := h.FindAuditLogsByTargetID(context.Background(), t, response.Body.Data.Id)
	require.Len(t, events, 1)
	require.Equal(t, "logdrain.create", events[0].Event)
}
