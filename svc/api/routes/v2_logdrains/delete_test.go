package logdrains_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	logdrains "github.com/unkeyed/unkey/svc/api/routes/v2_logdrains"
	createRoute "github.com/unkeyed/unkey/svc/api/routes/v2_logdrains_create_logdrain"
)

func TestDeleteWorksAfterAllowanceRevoked(t *testing.T) {
	h := testutil.NewHarness(t)
	create := &createRoute.Create{DB: h.DB, Vault: h.Vault, Auditlogs: h.Auditlogs, Clock: h.Clock, LimitsCache: h.Caches.WorkspaceLimits}
	remove := &logdrains.Delete{DB: h.DB, Auditlogs: h.Auditlogs}
	get := &logdrains.Get{DB: h.DB}
	h.Register(create)
	h.Register(remove)
	h.Register(get)
	workspaceID := h.Resources().UserWorkspace.ID
	_, err := h.DB.RW().ExecContext(context.Background(), "UPDATE `limits` SET logdrains_max = 1 WHERE workspace_id = ?", workspaceID)
	require.NoError(t, err)
	key := h.CreateRootKey(workspaceID, "unkey:v1:"+workspaceID+":**#*")
	headers := http.Header{"Authorization": {"Bearer " + key}, "Content-Type": {"application/json"}}
	created := testutil.CallRoute[json.RawMessage, openapi.LogdrainMutationResponse](h, create, headers, json.RawMessage(`{"name":"Delete me","stream":"audit_logs","destination":{"http":{"url":"https://logs.example.com"}}}`))
	require.Equal(t, http.StatusOK, created.Status)
	_, err = h.DB.RW().ExecContext(context.Background(), "UPDATE `limits` SET logdrains_max = 0 WHERE workspace_id = ?", workspaceID)
	require.NoError(t, err)
	input := openapi.LogdrainIdRequest{LogdrainId: created.Body.Data.Id}
	deleted := testutil.CallRoute[openapi.LogdrainIdRequest, openapi.LogdrainMutationResponse](h, remove, headers, input)
	require.Equal(t, http.StatusOK, deleted.Status, "%s", deleted.RawBody)
	require.Equal(t, input.LogdrainId, deleted.Body.Data.Id)
	require.NotEmpty(t, deleted.Body.Meta.RequestId)
	result := testutil.CallRoute[openapi.LogdrainIdRequest, openapi.LogdrainResponse](h, get, headers, input)
	require.Equal(t, http.StatusNotFound, result.Status)
	events := h.FindAuditLogsByTargetID(context.Background(), t, input.LogdrainId)
	require.Len(t, events, 2)
	require.Equal(t, "logdrain.create", events[0].Event)
	require.Equal(t, "logdrain.delete", events[1].Event)
	var count int
	require.NoError(t, h.DB.RW().QueryRowContext(context.Background(), "SELECT COUNT(*) FROM logdrains WHERE id = ?", input.LogdrainId).Scan(&count))
	require.Zero(t, count)
}

func TestDeleteRequiresID(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &logdrains.Delete{DB: h.DB, Auditlogs: h.Auditlogs}
	h.Register(route)
	key := h.CreateRootKey(h.Resources().UserWorkspace.ID, "unkey:v1:"+h.Resources().UserWorkspace.ID+":**#*")
	result := testutil.CallRoute[json.RawMessage, openapi.BadRequestErrorResponse](h, route, http.Header{"Authorization": {"Bearer " + key}, "Content-Type": {"application/json"}}, json.RawMessage(`{}`))
	require.Equal(t, http.StatusBadRequest, result.Status, "%s", result.RawBody)
	require.Contains(t, result.Body.Error.Type, "application/invalid_input")
	require.NotEmpty(t, result.Body.Error.Detail)
	require.NotEmpty(t, result.Body.Meta.RequestId)
}
