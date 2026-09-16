package handler_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	dbtype "github.com/unkeyed/unkey/pkg/db/types"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_permissions_list_permissions"
)

// TestListPermissionsAuthorizesCanonicalReadPermission guarantees a wildcard
// project-scoped permission grant can list permissions in its project.
func TestListPermissionsAuthorizesCanonicalReadPermission(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID
	permission := h.CreatePermission(seed.CreatePermissionRequest{
		WorkspaceID: workspaceID,
		Name:        "canonical.permission",
		Slug:        "canonical.permission",
	})
	otherProjectID := uid.New(uid.ProjectPrefix)
	h.CreateProject(seed.CreateProjectRequest{
		ID:          otherProjectID,
		WorkspaceID: workspaceID,
		Name:        "Other",
		Slug:        uid.New("other"),
	})
	require.NoError(t, db.Query.InsertPermission(t.Context(), h.DB.RW(), db.InsertPermissionParams{
		PermissionID: uid.New(uid.PermissionPrefix),
		WorkspaceID:  workspaceID,
		ProjectID:    otherProjectID,
		Name:         "other.permission",
		Slug:         "other.permission",
		Description:  dbtype.NullString{Valid: false},
		CreatedAtM:   time.Now().UnixMilli(),
	}))
	rootKey := h.CreateRootKey(
		workspaceID,
		fmt.Sprintf("unkey:v1:%s:projects/%s/rbac/permissions/*#read", workspaceID, permission.ProjectID),
	)
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{})

	require.Equal(t, http.StatusOK, res.Status, res.RawBody)
	require.Len(t, res.Body.Data, 1)
	require.Equal(t, permission.ID, res.Body.Data[0].Id)
}

// TestListPermissionsAuthorizesCanonicalReadBeforeDefaultProjectCreation
// guarantees that a project-wildcard grant can authorize a missing default
// project before the route creates it.
func TestListPermissionsAuthorizesCanonicalReadBeforeDefaultProjectCreation(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspace := h.CreateWorkspace()
	rootKey := h.CreateRootKey(
		workspace.ID,
		fmt.Sprintf("unkey:v1:%s:projects/*/rbac/permissions/*#read", workspace.ID),
	)
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{})

	require.Equal(t, http.StatusOK, res.Status, res.RawBody)
	require.Empty(t, res.Body.Data)
}
