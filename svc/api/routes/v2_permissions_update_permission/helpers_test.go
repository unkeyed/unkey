package handler_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_permissions_update_permission"
)

func newRoute(h *testutil.Harness) *handler.Handler {
	route := &handler.Handler{
		DB:        h.DB,
		Auditlogs: h.Auditlogs,
	}
	h.Register(route)
	return route
}

func randomSlug() string {
	return strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))
}

func seedPermission(h *testutil.Harness, workspaceID string) db.Permission {
	description := "KEBAP"
	return h.CreatePermission(seed.CreatePermissionRequest{
		WorkspaceID: workspaceID,
		Name:        "documents.read",
		Slug:        randomSlug(),
		Description: &description,
	})
}

func findPermission(t *testing.T, h *testutil.Harness, permission db.Permission) db.FindPermissionByIdOrSlugRow {
	t.Helper()
	stored, err := db.Query.FindPermissionByIdOrSlug(t.Context(), h.DB.RO(), db.FindPermissionByIdOrSlugParams{
		WorkspaceID: permission.WorkspaceID,
		Search:      permission.ID,
	})
	require.NoError(t, err)
	return stored
}

func writeAnyPermission(workspaceID string) string {
	return fmt.Sprintf("unkey:v1:%s:projects/*/rbac/permissions/*#write", workspaceID)
}

func authHeaders(rootKey string) http.Header {
	return http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}
}
