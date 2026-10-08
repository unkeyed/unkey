package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_apis_delete_api"
)

// TestDeleteAPIWithURNPermission guarantees that a URN keyspace
// delete permission authorizes deletion without a legacy API permission.
func TestDeleteAPIWithURNPermission(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		DB:        h.DB,
		Auditlogs: h.Auditlogs,
		Caches:    h.Caches,
	}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
	permission := fmt.Sprintf(
		"unkey:v1:%s:projects/%s/keyspaces/%s#delete",
		workspace.ID,
		api.ProjectID,
		api.KeyAuthID.String,
	)
	rootKey := h.CreateRootKey(workspace.ID, permission)

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}, handler.Request{ApiId: api.ID})
	require.Equal(t, http.StatusOK, res.Status, "received: %s", res.RawBody)

	deleted, err := db.Query.FindApiByID(ctx, h.DB.RO(), api.ID)
	require.NoError(t, err)
	require.True(t, deleted.DeletedAtM.Valid)
}

// TestDeleteAPIRejectsNonmatchingURNPermission guarantees that URN
// permissions cannot delete APIs outside their exact resource, action, or workspace.
func TestDeleteAPIRejectsNonmatchingURNPermission(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		DB:        h.DB,
		Auditlogs: h.Auditlogs,
		Caches:    h.Caches,
	}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	otherWorkspace := h.CreateWorkspace()
	testCases := []struct {
		name       string
		permission func(target db.Api) string
	}{
		{
			name: "resource",
			permission: func(_ db.Api) string {
				otherAPI := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
				return fmt.Sprintf("unkey:v1:%s:projects/%s/keyspaces/%s#delete", workspace.ID, otherAPI.ProjectID, otherAPI.KeyAuthID.String)
			},
		},
		{
			name: "action",
			permission: func(target db.Api) string {
				return fmt.Sprintf("unkey:v1:%s:projects/%s/keyspaces/%s#read", workspace.ID, target.ProjectID, target.KeyAuthID.String)
			},
		},
		{
			name: "workspace",
			permission: func(target db.Api) string {
				return fmt.Sprintf("unkey:v1:%s:projects/%s/keyspaces/%s#delete", otherWorkspace.ID, target.ProjectID, target.KeyAuthID.String)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			target := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
			rootKey := h.CreateRootKey(workspace.ID, tc.permission(target))

			res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
			}, handler.Request{ApiId: target.ID})
			require.Equal(t, http.StatusForbidden, res.Status, "received: %s", res.RawBody)

			notDeleted, err := db.Query.FindApiByID(ctx, h.DB.RO(), target.ID)
			require.NoError(t, err)
			require.False(t, notDeleted.DeletedAtM.Valid)
		})
	}
}

// TestDeleteAPIUsesKeyspaceProjectForURNPermission guarantees that the
// keyspace's project, not the API's denormalized project, owns the resource.
func TestDeleteAPIUsesKeyspaceProjectForURNPermission(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		DB:        h.DB,
		Auditlogs: h.Auditlogs,
		Caches:    h.Caches,
	}
	h.Register(route)

	workspace := h.Resources().UserWorkspace

	createDivergentAPI := func(t *testing.T) (db.Api, string) {
		t.Helper()
		keyspaceProject := h.CreateProject(seed.CreateProjectRequest{
			ID:          uid.New(uid.ProjectPrefix),
			WorkspaceID: workspace.ID,
			Name:        "Keyspace project",
			Slug:        uid.New("project"),
		})
		keyspaceProjectID := keyspaceProject.ID
		apiProject := h.CreateProject(seed.CreateProjectRequest{
			ID:          uid.New(uid.ProjectPrefix),
			WorkspaceID: workspace.ID,
			Name:        "API project",
			Slug:        uid.New("project"),
		})
		keySpaceID := insertKeySpace(t, h, workspace.ID, keyspaceProjectID)
		api := insertAPI(t, h, workspace.ID, apiProject.ID, sql.NullString{String: keySpaceID, Valid: true})
		require.Equal(t, apiProject.ID, api.ProjectID)
		require.NotEqual(t, keyspaceProjectID, api.ProjectID)
		return api, keyspaceProjectID
	}

	t.Run("keyspace project authorizes deletion", func(t *testing.T) {
		api, keyspaceProjectID := createDivergentAPI(t)
		permission := fmt.Sprintf("unkey:v1:%s:projects/%s/keyspaces/%s#delete", workspace.ID, keyspaceProjectID, api.KeyAuthID.String)
		rootKey := h.CreateRootKey(workspace.ID, permission)

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}, handler.Request{ApiId: api.ID})
		require.Equal(t, http.StatusOK, res.Status, "received: %s", res.RawBody)

		deleted, err := db.Query.FindApiByID(ctx, h.DB.RO(), api.ID)
		require.NoError(t, err)
		require.True(t, deleted.DeletedAtM.Valid)
	})

	t.Run("API project does not authorize deletion", func(t *testing.T) {
		api, _ := createDivergentAPI(t)
		permission := fmt.Sprintf("unkey:v1:%s:projects/%s/keyspaces/%s#delete", workspace.ID, api.ProjectID, api.KeyAuthID.String)
		rootKey := h.CreateRootKey(workspace.ID, permission)

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}, handler.Request{ApiId: api.ID})
		require.Equal(t, http.StatusForbidden, res.Status, "received: %s", res.RawBody)

		notDeleted, err := db.Query.FindApiByID(ctx, h.DB.RO(), api.ID)
		require.NoError(t, err)
		require.False(t, notDeleted.DeletedAtM.Valid)
	})
}

// TestDeleteAPIRejectsForeignWorkspaceKeyspace guarantees that inconsistent
// keyspace ownership is masked as not found before URN authorization.
func TestDeleteAPIRejectsForeignWorkspaceKeyspace(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		DB:        h.DB,
		Auditlogs: h.Auditlogs,
		Caches:    h.Caches,
	}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	otherWorkspace := h.CreateWorkspace()
	projectID := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "API project",
		Slug:        uid.New("project"),
	}).ID
	keySpaceID := insertKeySpace(t, h, otherWorkspace.ID, projectID)
	api := insertAPI(t, h, workspace.ID, projectID, sql.NullString{String: keySpaceID, Valid: true})
	permission := fmt.Sprintf("unkey:v1:%s:projects/%s/keyspaces/%s#delete", workspace.ID, projectID, api.KeyAuthID.String)
	rootKey := h.CreateRootKey(workspace.ID, permission)

	res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}, handler.Request{ApiId: api.ID})
	require.Equal(t, http.StatusNotFound, res.Status, "received: %s", res.RawBody)

	notDeleted, err := db.Query.FindApiByID(ctx, h.DB.RO(), api.ID)
	require.NoError(t, err)
	require.False(t, notDeleted.DeletedAtM.Valid)
}

// TestDeleteAPIWithoutKeyspaceRetainsLegacyAuthorization guarantees that old
// API rows with a nullable keyspace remain deletable through legacy permissions.
func TestDeleteAPIWithoutKeyspaceRetainsLegacyAuthorization(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		DB:        h.DB,
		Auditlogs: h.Auditlogs,
		Caches:    h.Caches,
	}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	projectID := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "API project",
		Slug:        uid.New("project"),
	}).ID
	api := insertAPI(t, h, workspace.ID, projectID, sql.NullString{})
	rootKey := h.CreateRootKey(workspace.ID, fmt.Sprintf("api.%s.delete_api", api.ID))

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}, handler.Request{ApiId: api.ID})
	require.Equal(t, http.StatusOK, res.Status, "received: %s", res.RawBody)

	deleted, err := db.Query.FindApiByID(ctx, h.DB.RO(), api.ID)
	require.NoError(t, err)
	require.True(t, deleted.DeletedAtM.Valid)
}

func insertKeySpace(t *testing.T, h *testutil.Harness, workspaceID, projectID string) string {
	t.Helper()
	keySpaceID := uid.New(uid.KeySpacePrefix)
	err := db.Query.InsertKeySpace(context.Background(), h.DB.RW(), db.InsertKeySpaceParams{
		ID:                 keySpaceID,
		WorkspaceID:        workspaceID,
		ProjectID:          projectID,
		CreatedAtM:         time.Now().UnixMilli(),
		StoreEncryptedKeys: false,
		DefaultPrefix:      sql.NullString{},
		DefaultBytes:       sql.NullInt32{},
	})
	require.NoError(t, err)
	return keySpaceID
}

func insertAPI(t *testing.T, h *testutil.Harness, workspaceID, projectID string, keySpaceID sql.NullString) db.Api {
	t.Helper()
	apiID := uid.New(uid.APIPrefix)
	err := db.Query.InsertApi(context.Background(), h.DB.RW(), db.InsertApiParams{
		ID:          apiID,
		Name:        "test-api",
		WorkspaceID: workspaceID,
		ProjectID:   projectID,
		AuthType:    db.NullApisAuthType{Valid: true, ApisAuthType: db.ApisAuthTypeKey},
		IpWhitelist: sql.NullString{},
		KeyAuthID:   keySpaceID,
		CreatedAtM:  time.Now().UnixMilli(),
	})
	require.NoError(t, err)
	api, err := db.Query.FindApiByID(context.Background(), h.DB.RO(), apiID)
	require.NoError(t, err)
	return api
}
