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

func TestListPagesOnlyAuthorizedWorkspace(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &logdrains.List{DB: h.DB}
	h.Register(route)
	workspaceID := h.Resources().UserWorkspace.ID
	prefix := uid.New("ld")
	config, err := proto.Marshal(&logdrainv1.Config{Destination: &logdrainv1.Config_Axiom{Axiom: &logdrainv1.AxiomConfig{Dataset: "logs", EncryptedToken: "private-ciphertext"}}})
	require.NoError(t, err)
	for _, suffix := range []string{"c", "a", "b"} {
		workspace := workspaceID
		if suffix == "b" {
			workspace = uid.New("ws")
		}
		_, err := h.DB.RW().ExecContext(context.Background(), "INSERT INTO logdrains (id, workspace_id, name, stream, config, lease_id, fencing_token, created_at) VALUES (?, ?, 'Drain', 'audit_logs', ?, '', '', 123)", prefix+suffix, workspace, config)
		require.NoError(t, err)
	}
	key := h.CreateRootKey(workspaceID, "unkey:v1:"+workspaceID+":logdrains/*#read")
	headers := http.Header{"Authorization": {"Bearer " + key}, "Content-Type": {"application/json"}}
	type response struct {
		Data       []openapi.Logdrain
		Pagination openapi.Pagination
	}
	first := testutil.CallRoute[map[string]any, response](h, route, headers, map[string]any{"limit": 1})
	require.Equal(t, http.StatusOK, first.Status, "%s", first.RawBody)
	require.Len(t, first.Body.Data, 1)
	require.Equal(t, prefix+"a", first.Body.Data[0].Id)
	require.True(t, first.Body.Pagination.HasMore)
	require.NotContains(t, first.RawBody, "private-ciphertext")
	second := testutil.CallRoute[map[string]any, response](h, route, headers, map[string]any{"limit": 1, "cursor": first.Body.Pagination.Cursor})
	require.Equal(t, http.StatusOK, second.Status, "%s", second.RawBody)
	require.Len(t, second.Body.Data, 1)
	require.Equal(t, prefix+"c", second.Body.Data[0].Id)
	require.False(t, second.Body.Pagination.HasMore)
}
