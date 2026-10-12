package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_permissions_update_role"
)

func newRoute(h *testutil.Harness) *handler.Handler {
	route := &handler.Handler{
		DB:        h.DB,
		Auditlogs: h.Auditlogs,
	}
	h.Register(route)
	return route
}

func seedRole(h *testutil.Harness, workspaceID string) db.Role {
	description := "KEBAP"
	return h.CreateRole(seed.CreateRoleRequest{
		WorkspaceID: workspaceID,
		Name:        uid.New("role.name"),
		Description: &description,
		Permissions: nil,
	})
}

func findRole(t *testing.T, h *testutil.Harness, role db.Role) db.Role {
	t.Helper()
	stored, err := db.Query.FindRoleByID(t.Context(), h.DB.RO(), role.ID)
	require.NoError(t, err)
	return stored
}

func writeAnyRole(workspaceID string) string {
	return fmt.Sprintf("unkey:v1:%s:projects/*/rbac/roles/*#write", workspaceID)
}

func authHeaders(rootKey string) http.Header {
	return http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}
}
