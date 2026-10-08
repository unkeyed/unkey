package logdrains_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	logdrainv1 "github.com/unkeyed/unkey/gen/proto/logdrain/v1"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	logdrains "github.com/unkeyed/unkey/svc/api/routes/v2_logdrains_delete_logdrain"
	"google.golang.org/protobuf/proto"
)

func TestDeleteMalformedJSONDoesNotMutate(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &logdrains.Handler{
		DB:        h.DB,
		Auditlogs: h.Auditlogs,
	}
	h.Register(route)
	workspaceID := h.Resources().UserWorkspace.ID
	id := uid.New("ld")
	config, err := proto.Marshal(&logdrainv1.Config{Destination: &logdrainv1.Config_Http{Http: &logdrainv1.HttpConfig{Url: "https://logs.example.com"}}})
	require.NoError(t, err)
	_, err = h.DB.RW().ExecContext(t.Context(), "INSERT INTO logdrains (id, workspace_id, name, stream, config, lease_id, fencing_token, created_at) VALUES (?, ?, 'Unchanged', 'audit_logs', ?, '', '', 123)", id, workspaceID, config)
	require.NoError(t, err)
	key := h.CreateRootKey(workspaceID, "unkey:v1:"+workspaceID+":logdrains/*#delete")
	req, err := http.NewRequest(route.Method(), route.Path(), strings.NewReader(`{"logdrainId":"`+id+`",`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)
	result := testutil.CallRaw[openapi.BadRequestErrorResponse](h, req)
	require.Equal(t, http.StatusBadRequest, result.Status, "%s", result.RawBody)
	require.Equal(t, http.StatusBadRequest, result.Body.Error.Status)
	require.NotEmpty(t, result.Body.Meta.RequestId)
	var name string
	var stored []byte
	require.NoError(t, h.DB.RW().QueryRowContext(t.Context(), "SELECT name, config FROM logdrains WHERE id = ?", id).Scan(&name, &stored))
	require.Equal(t, "Unchanged", name)
	require.Equal(t, config, stored)
	require.Empty(t, h.FindAuditLogsByTargetID(t.Context(), t, id))
}
