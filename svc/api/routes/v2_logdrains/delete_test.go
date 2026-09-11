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
)

func TestDeleteWorksAfterAllowanceRevoked(t *testing.T) {
	h := testutil.NewHarness(t)
	create := &logdrains.Create{DB: h.DB, Vault: h.Vault, Auditlogs: h.Auditlogs, Clock: h.Clock}
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
	result := testutil.CallRoute[openapi.LogdrainIdRequest, openapi.LogdrainResponse](h, get, headers, input)
	require.Equal(t, http.StatusNotFound, result.Status)
	events := h.FindAuditLogsByTargetID(context.Background(), t, input.LogdrainId)
	require.Len(t, events, 2)
}
