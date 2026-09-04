package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	dbtype "github.com/unkeyed/unkey/pkg/db/types"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_keys_set_permissions"
	"golang.org/x/sync/errgroup"
)

func TestSuccess(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
		Auditlogs: h.Auditlogs,
		KeyCache:  h.Caches.VerificationKeyByHash,
	}

	h.Register(route)

	// Create a workspace and root key
	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "api.*.update_key", "rbac.*.remove_permission_from_key", "rbac.*.add_permission_to_key", "rbac.*.create_permission")

	// Set up request headers
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("urn update_key permission can set permissions", func(t *testing.T) {
		api := h.CreateApi(seed.CreateApiRequest{
			WorkspaceID: workspace.ID,
		})
		key := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
			Name:        ptr.P("urn-set-permission-key"),
		})
		permission := h.CreatePermission(seed.CreatePermissionRequest{
			WorkspaceID: workspace.ID,
			Name:        "documents.write.urn.set",
			Slug:        "documents.write.urn.set",
		})

		updateKeyPermission := fmt.Sprintf("unkey:v1:%s:projects/%s/keyspaces/%s/keys/%s#write", workspace.ID, api.ProjectID, api.KeyAuthID.String, key.KeyID)
		urnRootKey := h.CreateRootKey(workspace.ID, updateKeyPermission)
		urnHeaders := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", urnRootKey)},
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, urnHeaders, handler.Request{
			KeyId:       key.KeyID,
			Permissions: []string{permission.Name},
		})

		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Data)
		require.Len(t, res.Body.Data, 1)
		require.Equal(t, permission.ID, res.Body.Data[0].Id)
	})

	t.Run("set permissions using permission IDs", func(t *testing.T) {
		// Create API with keyring using testutil helper
		defaultPrefix := "test"
		defaultBytes := int32(16)
		api := h.CreateApi(seed.CreateApiRequest{
			WorkspaceID:   workspace.ID,
			DefaultPrefix: &defaultPrefix,
			DefaultBytes:  &defaultBytes,
		})

		// Create a test key using testutil helper
		keyName := "Test Key"
		keyResponse := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
			Name:        &keyName,
		})
		keyID := keyResponse.KeyID

		// Create permissions
		permission1ID := uid.New(uid.TestPrefix)
		permission1Slug := "documents.read.initial"
		err := db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
			PermissionID: permission1ID,
			WorkspaceID:  workspace.ID,
			ProjectID:    api.ProjectID,
			Name:         permission1Slug,
			Slug:         permission1Slug,
			Description:  dbtype.NullString{Valid: true, String: "Initial permission"},
		})
		require.NoError(t, err)

		permission2ID := uid.New(uid.TestPrefix)
		permission2Slug := "documents.write.new"
		err = db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
			PermissionID: permission2ID,
			WorkspaceID:  workspace.ID,
			ProjectID:    api.ProjectID,
			Name:         permission2Slug,
			Slug:         permission2Slug,
			Description:  dbtype.NullString{Valid: true, String: "Write permission"},
		})
		require.NoError(t, err)

		permission3ID := uid.New(uid.TestPrefix)
		permission3Slug := "documents.delete.new"
		err = db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
			PermissionID: permission3ID,
			WorkspaceID:  workspace.ID,
			ProjectID:    api.ProjectID,
			Name:         permission3Slug,
			Slug:         permission3Slug,
			Description:  dbtype.NullString{Valid: true, String: "Delete permission"},
		})
		require.NoError(t, err)

		// Add initial permission to key
		err = db.Query.InsertKeyPermission(ctx, h.DB.RW(), db.InsertKeyPermissionParams{
			KeyID:        keyID,
			PermissionID: permission1ID,
			WorkspaceID:  workspace.ID,
			CreatedAt:    time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		// Verify key has initial permission
		currentPermissions, err := db.Query.ListDirectPermissionsByKeyID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.Len(t, currentPermissions, 1)
		require.Equal(t, permission1ID, currentPermissions[0].ID)

		req := handler.Request{
			KeyId:       keyID,
			Permissions: []string{permission2Slug, permission3Slug},
		}

		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Data)
		require.Len(t, res.Body.Data, 2)

		contains := func(id string) bool {
			for _, p := range res.Body.Data {
				if p.Id == id {
					return true
				}
			}
			return false
		}

		// Verify response contains new permissions
		require.True(t, contains(permission3ID))
		require.True(t, contains(permission2ID))

		// Verify permissions in database
		finalPermissions, err := db.Query.ListDirectPermissionsByKeyID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.Len(t, finalPermissions, 2)

		permissionIDs := make([]string, len(finalPermissions))
		for i, perm := range finalPermissions {
			permissionIDs[i] = perm.ID
		}
		require.Contains(t, permissionIDs, permission2ID)
		require.Contains(t, permissionIDs, permission3ID)
		require.NotContains(t, permissionIDs, permission1ID) // Should be removed
	})

	t.Run("set permissions using permission names", func(t *testing.T) {
		// Create API and key using testutil helpers
		defaultPrefix := "test"
		defaultBytes := int32(16)
		api := h.CreateApi(seed.CreateApiRequest{
			WorkspaceID:   workspace.ID,
			DefaultPrefix: &defaultPrefix,
			DefaultBytes:  &defaultBytes,
		})

		keyName := "Test Key"
		keyResponse := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
			Name:        &keyName,
		})
		keyID := keyResponse.KeyID

		// Create permissions
		permission1ID := uid.New(uid.TestPrefix)
		err := db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
			PermissionID: permission1ID,
			WorkspaceID:  workspace.ID,
			ProjectID:    api.ProjectID,
			Name:         "documents.read.byname",
			Slug:         "documents.read.byname",
			Description:  dbtype.NullString{Valid: true, String: "Read permission"},
		})
		require.NoError(t, err)

		permission2ID := uid.New(uid.TestPrefix)
		err = db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
			PermissionID: permission2ID,
			WorkspaceID:  workspace.ID,
			ProjectID:    api.ProjectID,
			Name:         "documents.write.byname",
			Slug:         "documents.write.byname",
			Description:  dbtype.NullString{Valid: true, String: "Write permission"},
		})
		require.NoError(t, err)

		// Add initial permission to key
		err = db.Query.InsertKeyPermission(ctx, h.DB.RW(), db.InsertKeyPermissionParams{
			KeyID:        keyID,
			PermissionID: permission1ID,
			WorkspaceID:  workspace.ID,
			CreatedAt:    time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		req := handler.Request{
			KeyId:       keyID,
			Permissions: []string{"documents.write.byname"},
		}

		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Data)
		require.Len(t, res.Body.Data, 1)
		require.Equal(t, permission2ID, res.Body.Data[0].Id)
		require.Equal(t, "documents.write.byname", res.Body.Data[0].Name)

		// Verify permissions in database
		finalPermissions, err := db.Query.ListDirectPermissionsByKeyID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.Len(t, finalPermissions, 1)
		require.Equal(t, permission2ID, finalPermissions[0].ID)
	})

	t.Run("set empty permissions (remove all)", func(t *testing.T) {
		// Create API and key using testutil helpers
		defaultPrefix := "test"
		defaultBytes := int32(16)
		api := h.CreateApi(seed.CreateApiRequest{
			WorkspaceID:   workspace.ID,
			DefaultPrefix: &defaultPrefix,
			DefaultBytes:  &defaultBytes,
		})

		keyName := "Test Key"
		keyResponse := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
			Name:        &keyName,
		})
		keyID := keyResponse.KeyID

		// Create and assign permissions
		permission1ID := uid.New(uid.TestPrefix)
		err := db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
			PermissionID: permission1ID,
			WorkspaceID:  workspace.ID,
			ProjectID:    api.ProjectID,
			Name:         "documents.read.empty",
			Slug:         "documents.read.empty",
			Description:  dbtype.NullString{Valid: true, String: "Read permission"},
		})
		require.NoError(t, err)

		permission2ID := uid.New(uid.TestPrefix)
		err = db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
			PermissionID: permission2ID,
			WorkspaceID:  workspace.ID,
			ProjectID:    api.ProjectID,
			Name:         "documents.write.empty",
			Slug:         "documents.write.empty",
			Description:  dbtype.NullString{Valid: true, String: "Write permission"},
		})
		require.NoError(t, err)

		err = db.Query.InsertKeyPermission(ctx, h.DB.RW(), db.InsertKeyPermissionParams{
			KeyID:        keyID,
			PermissionID: permission1ID,
			WorkspaceID:  workspace.ID,
			CreatedAt:    time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		err = db.Query.InsertKeyPermission(ctx, h.DB.RW(), db.InsertKeyPermissionParams{
			KeyID:        keyID,
			PermissionID: permission2ID,
			WorkspaceID:  workspace.ID,
			CreatedAt:    time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		// Verify key has permissions initially
		currentPermissions, err := db.Query.ListDirectPermissionsByKeyID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.Len(t, currentPermissions, 2)

		req := handler.Request{
			KeyId:       keyID,
			Permissions: []string{},
		}

		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Data)
		require.Len(t, res.Body.Data, 0)

		// Verify no permissions in database
		finalPermissions, err := db.Query.ListDirectPermissionsByKeyID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.Len(t, finalPermissions, 0)
	})

	t.Run("set permissions with no changes (idempotent)", func(t *testing.T) {
		// Create API and key using testutil helpers
		defaultPrefix := "test"
		defaultBytes := int32(16)
		api := h.CreateApi(seed.CreateApiRequest{
			WorkspaceID:   workspace.ID,
			DefaultPrefix: &defaultPrefix,
			DefaultBytes:  &defaultBytes,
		})

		keyName := "Test Key"
		keyResponse := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
			Name:        &keyName,
		})
		keyID := keyResponse.KeyID

		// Create permission
		permissionID := uid.New(uid.PermissionPrefix)
		permissionSlugAndName := "documents.read.idempotent"
		err := db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
			PermissionID: permissionID,
			WorkspaceID:  workspace.ID,
			ProjectID:    api.ProjectID,
			Name:         permissionSlugAndName,
			Slug:         permissionSlugAndName,
			Description:  dbtype.NullString{Valid: true, String: "Read permission"},
		})
		require.NoError(t, err)

		// Add permission to key
		err = db.Query.InsertKeyPermission(ctx, h.DB.RW(), db.InsertKeyPermissionParams{
			KeyID:        keyID,
			PermissionID: permissionID,
			WorkspaceID:  workspace.ID,
			CreatedAt:    time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		req := handler.Request{
			KeyId:       keyID,
			Permissions: []string{permissionSlugAndName},
		}

		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Data)
		require.Len(t, res.Body.Data, 1)
		require.Equal(t, permissionID, res.Body.Data[0].Id)

		// Verify permission still exists in database
		finalPermissions, err := db.Query.ListDirectPermissionsByKeyID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.Len(t, finalPermissions, 1)
		require.Equal(t, permissionID, finalPermissions[0].ID)
	})

	t.Run("create permission on-the-fly using slug", func(t *testing.T) {
		// Create API and key using testutil helpers
		defaultPrefix := "test"
		defaultBytes := int32(16)
		api := h.CreateApi(seed.CreateApiRequest{
			WorkspaceID:   workspace.ID,
			DefaultPrefix: &defaultPrefix,
			DefaultBytes:  &defaultBytes,
		})

		keyName := "Test Key"
		keyResponse := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
			Name:        &keyName,
		})
		keyID := keyResponse.KeyID

		// Use a slug that doesn't exist yet
		newPermissionSlug := "documents.create.onthefly"
		req := handler.Request{
			KeyId:       keyID,
			Permissions: []string{newPermissionSlug},
		}

		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Data)
		require.Len(t, res.Body.Data, 1)

		// Verify the permission was created with the slug as both name and slug
		require.Equal(t, newPermissionSlug, res.Body.Data[0].Name)

		// Verify permission exists in database and was assigned to key
		finalPermissions, err := db.Query.ListDirectPermissionsByKeyID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.Len(t, finalPermissions, 1)
		require.Equal(t, newPermissionSlug, finalPermissions[0].Slug)
		require.Equal(t, newPermissionSlug, finalPermissions[0].Name) // Slug used as name too

		// Verify the permission was created in the project
		createdPermission, err := db.Query.FindPermissionBySlugAndProjectID(ctx, h.DB.RO(), db.FindPermissionBySlugAndProjectIDParams{
			Slug:        newPermissionSlug,
			WorkspaceID: workspace.ID,
			ProjectID:   api.ProjectID,
		})
		require.NoError(t, err)
		require.Equal(t, newPermissionSlug, createdPermission.Slug)
		require.Equal(t, newPermissionSlug, createdPermission.Name)
		require.Equal(t, workspace.ID, createdPermission.WorkspaceID)
	})
}

// TestSetPermissionsConcurrent tests that concurrent requests to set permissions
// on the same key don't deadlock. The key-row lock must serialize each replacement
// transaction so the final state is one complete submitted set, never a partial or merge.
func TestSetPermissionsConcurrent(t *testing.T) {
	t.Parallel()

	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
		Auditlogs: h.Auditlogs,
		KeyCache:  h.Caches.VerificationKeyByHash,
	}

	h.Register(route)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "api.*.update_key", "rbac.*.remove_permission_from_key", "rbac.*.add_permission_to_key", "rbac.*.create_permission")

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})

	// Create a single key that all concurrent requests will update
	keyResponse := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
		Name:        ptr.P("concurrent-set-permissions-test-key"),
	})

	// Create permissions that will be set concurrently
	numConcurrent := 10
	permissions := make([]string, numConcurrent)
	for i := range numConcurrent {
		perm := h.CreatePermission(seed.CreatePermissionRequest{
			WorkspaceID: workspace.ID,
			Name:        fmt.Sprintf("concurrent.set.permission.%d", i),
			Slug:        fmt.Sprintf("concurrent.set.permission.%d", i),
		})
		permissions[i] = perm.Name
	}

	// Warm up the validator's schema cache with a single request so the
	// concurrent burst doesn't race on first-time schema rendering.
	warmup := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
		KeyId:       keyResponse.KeyID,
		Permissions: []string{permissions[0]},
	})
	require.Equal(t, 200, warmup.Status, "warmup request should succeed")

	submittedPermissionSets := make([][]string, numConcurrent)
	for i := range numConcurrent {
		submittedPermissionSets[i] = []string{permissions[i], permissions[(i+1)%numConcurrent], permissions[(i+2)%numConcurrent]}
	}

	g := errgroup.Group{}
	for i := range numConcurrent {
		g.Go(func() error {
			body, err := json.Marshal(handler.Request{
				KeyId:       keyResponse.KeyID,
				Permissions: submittedPermissionSets[i],
			})
			if err != nil {
				return fmt.Errorf("request %d: marshal body: %w", i, err)
			}

			req := httptest.NewRequest(route.Method(), route.Path(), bytes.NewReader(body))
			req.Header = headers.Clone()
			recorder := httptest.NewRecorder()
			h.Mux().ServeHTTP(recorder, req)

			if recorder.Code != http.StatusOK {
				return fmt.Errorf("request %d: unexpected status %d: %s", i, recorder.Code, recorder.Body.String())
			}
			return nil
		})
	}

	err := g.Wait()
	require.NoError(t, err, "All concurrent updates should succeed without deadlock")

	finalPermissions, err := db.Query.ListDirectPermissionsByKeyID(t.Context(), h.DB.RO(), keyResponse.KeyID)
	require.NoError(t, err)

	finalPermissionNames := make([]string, len(finalPermissions))
	for i, permission := range finalPermissions {
		finalPermissionNames[i] = permission.Name
	}
	slices.Sort(finalPermissionNames)

	matchesCompleteReplacement := slices.ContainsFunc(submittedPermissionSets, func(submitted []string) bool {
		expected := slices.Clone(submitted)
		slices.Sort(expected)
		return slices.Equal(expected, finalPermissionNames)
	})
	require.Truef(t, matchesCompleteReplacement, "final permissions must exactly match one submitted replacement, got %v", finalPermissionNames)
}
