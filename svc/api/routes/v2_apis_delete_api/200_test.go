package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_apis_delete_api"
)

func TestSuccess(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
		Auditlogs: h.Auditlogs,
		Caches:    h.Caches,
	}

	h.Register(route)

	// Create a workspace
	workspace := h.Resources().UserWorkspace

	// Create a root key with appropriate permissions
	rootKey := h.CreateRootKey(workspace.ID, "api.*.delete_api")

	// Set up request headers
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Test case for deleting an API without keys
	t.Run("delete api without keys", func(t *testing.T) {
		api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: h.Resources().UserWorkspace.ID})

		// Ensure API exists before deletion
		apiBeforeDelete, err := db.Query.FindApiByID(ctx, h.DB.RO(), api.ID)
		require.NoError(t, err)

		require.Equal(t, api.ID, apiBeforeDelete.ID)
		require.False(t, apiBeforeDelete.DeletedAtM.Valid)

		// Delete the API
		req := handler.Request{
			ApiId: api.ID,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.NotEmpty(t, res.Body.Meta.RequestId)

		// Verify API is marked as deleted
		apiAfterDelete, err := db.Query.FindApiByID(ctx, h.DB.RO(), api.ID)
		require.NoError(t, err) // Should still find it, just marked as deleted
		require.True(t, apiAfterDelete.DeletedAtM.Valid)
	})

	// Test case for deleting an API with active keys
	t.Run("delete api with active keys", func(t *testing.T) {
		api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: h.Resources().UserWorkspace.ID})

		createKey := h.CreateKey(seed.CreateKeyRequest{
			KeySpaceID:  api.KeyAuthID.String,
			WorkspaceID: h.Resources().UserWorkspace.ID,
		})

		// Ensure API exists before deletion
		apiBeforeDelete, err := db.Query.FindApiByID(ctx, h.DB.RO(), api.ID)
		require.NoError(t, err)
		require.Equal(t, api.ID, apiBeforeDelete.ID)
		require.False(t, apiBeforeDelete.DeletedAtM.Valid)

		// Delete the API
		req := handler.Request{
			ApiId: api.ID,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.NotEmpty(t, res.Body.Meta.RequestId)

		// Verify API is marked as deleted
		apiAfterDelete, err := db.Query.FindApiByID(ctx, h.DB.RO(), api.ID)
		require.NoError(t, err)
		require.True(t, apiAfterDelete.DeletedAtM.Valid)

		// Check that the key is still accessible (soft delete doesn't cascade to keys)
		key, err := db.Query.FindKeyByID(ctx, h.DB.RO(), createKey.KeyID)
		require.NoError(t, err)
		require.Equal(t, createKey.KeyID, key.ID)
	})

	// Test case for deleting an API immediately after creation
	t.Run("delete api immediately after creation", func(t *testing.T) {
		api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: h.Resources().UserWorkspace.ID})

		// Verify the API was created
		apiBeforeDelete, err := db.Query.FindApiByID(ctx, h.DB.RO(), api.ID)
		require.NoError(t, err)
		require.Equal(t, api.ID, apiBeforeDelete.ID)
		require.Equal(t, api.Name, apiBeforeDelete.Name)
		require.False(t, apiBeforeDelete.DeletedAtM.Valid)

		// Immediately delete the API without any delay
		req := handler.Request{
			ApiId: api.ID,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.NotEmpty(t, res.Body.Meta.RequestId)

		// Verify API is marked as deleted
		apiAfterDelete, err := db.Query.FindApiByID(ctx, h.DB.RO(), api.ID)
		require.NoError(t, err)
		require.True(t, apiAfterDelete.DeletedAtM.Valid)
	})
}

// TestDeleteApiWithKeyspaceUrnGrant verifies that a caller holding only the
// canonical keyspace URN delete grant can delete an API. The dashboard proxy
// mints URN grants exclusively, so without this arm every dashboard role is
// rejected by the legacy api.*.delete_api tuple.
func TestDeleteApiWithKeyspaceUrnGrant(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
		Auditlogs: h.Auditlogs,
		Caches:    h.Caches,
	}
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID
	grant := rbac.U(
		urn.New().Workspace(workspaceID).Project("*").Keyspace("*"),
		permissions.Delete,
	).Value
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", h.CreateRootKey(workspaceID, grant))},
	}

	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspaceID})
	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{ApiId: api.ID})
	require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)

	deleted, err := db.Query.FindApiByID(ctx, h.DB.RO(), api.ID)
	require.NoError(t, err)
	require.True(t, deleted.DeletedAtM.Valid)
}

// TestDeleteApiCascadesToKeyspaceAndKeys pins the cascade the dashboard used to
// get from the tRPC route: the keyspace and every live key go down with the API.
// The audit trail stays at one event on purpose, see the handler.
func TestDeleteApiCascadesToKeyspaceAndKeys(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
		Auditlogs: h.Auditlogs,
		Caches:    h.Caches,
	}
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", h.CreateRootKey(workspaceID, "api.*.delete_api"))},
	}

	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspaceID})
	keySpaceID := api.KeyAuthID.String

	keyIDs := make([]string, 0, 3)
	for range 3 {
		keyIDs = append(keyIDs, h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspaceID,
			KeySpaceID:  keySpaceID,
		}).KeyID)
	}

	// A second keyspace in the same workspace pins the blast radius: without
	// the key_auth_id predicate the UPDATE would take this key down too.
	bystander := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspaceID})
	bystanderKey := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspaceID,
		KeySpaceID:  bystander.KeyAuthID.String,
	}).KeyID

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{ApiId: api.ID})
	require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)

	survivor, err := db.Query.FindKeyByID(ctx, h.DB.RO(), bystanderKey)
	require.NoError(t, err)
	require.False(t, survivor.DeletedAtM.Valid, "a key in another keyspace must survive")

	survivingSpace, err := db.Query.FindKeySpaceByID(ctx, h.DB.RO(), bystander.KeyAuthID.String)
	require.NoError(t, err)
	require.False(t, survivingSpace.DeletedAtM.Valid, "another keyspace must survive")

	keySpace, err := db.Query.FindKeySpaceByID(ctx, h.DB.RO(), keySpaceID)
	require.NoError(t, err)
	require.True(t, keySpace.DeletedAtM.Valid, "keyspace should be soft deleted")

	for _, keyID := range keyIDs {
		key, keyErr := db.Query.FindKeyByID(ctx, h.DB.RO(), keyID)
		require.NoError(t, keyErr)
		require.True(t, key.DeletedAtM.Valid, "key %s should be soft deleted", keyID)
	}

	events := h.FindAuditLogsByTargetID(ctx, t, api.ID)
	require.Len(t, events, 1, "one api.delete event, no per-key cascade")
	require.Equal(t, string(auditlog.APIDeleteEvent), events[0].Event)
	for _, keyID := range keyIDs {
		require.Empty(t, h.FindAuditLogsByTargetID(ctx, t, keyID), "key %s must not get its own event", keyID)
	}

	targets := map[string]auditlog.EventTarget{}
	for _, target := range events[0].Targets {
		targets[target.Type] = target
	}
	require.Contains(t, targets, string(auditlog.KeySpaceResourceType), "keyspace must be a target")
	require.Equal(t, keySpaceID, targets[string(auditlog.KeySpaceResourceType)].ID)
	require.EqualValues(t, len(keyIDs), targets[string(auditlog.APIResourceType)].Meta["keysDeleted"])
}

// TestDeleteApiCountsOnlyLiveKeys pins what the batch loop reports. Seeding
// past the 1000-row batch size is too slow for this suite, so the multi-batch
// path is not covered here; what is covered is that the loop drains to
// completion and that a key deleted earlier keeps its own tombstone.
func TestDeleteApiCountsOnlyLiveKeys(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
		Auditlogs: h.Auditlogs,
		Caches:    h.Caches,
	}
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", h.CreateRootKey(workspaceID, "api.*.delete_api"))},
	}

	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspaceID})

	const liveKeys = 5
	for range liveKeys {
		h.CreateKey(seed.CreateKeyRequest{WorkspaceID: workspaceID, KeySpaceID: api.KeyAuthID.String})
	}

	// An already-deleted key must not be recounted or re-stamped.
	alreadyDead := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspaceID,
		KeySpaceID:  api.KeyAuthID.String,
		Deleted:     true,
	}).KeyID
	before, err := db.Query.FindKeyByID(ctx, h.DB.RO(), alreadyDead)
	require.NoError(t, err)
	require.True(t, before.DeletedAtM.Valid)

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{ApiId: api.ID})
	require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)

	after, err := db.Query.FindKeyByID(ctx, h.DB.RO(), alreadyDead)
	require.NoError(t, err)
	require.Equal(t, before.DeletedAtM.Int64, after.DeletedAtM.Int64, "earlier tombstone must be preserved")

	events := h.FindAuditLogsByTargetID(ctx, t, api.ID)
	require.Len(t, events, 1)
	var apiTarget auditlog.EventTarget
	for _, target := range events[0].Targets {
		if target.Type == string(auditlog.APIResourceType) {
			apiTarget = target
		}
	}
	require.EqualValues(t, liveKeys, apiTarget.Meta["keysDeleted"], "only live keys are counted")
	require.NotEmpty(t, apiTarget.Meta["deletedAtM"])
}

// TestDeleteApiWithScopedKeyspaceGrant verifies that a delete grant on the
// API's own project and keyspace is enough to delete it.
func TestDeleteApiWithScopedKeyspaceGrant(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
		Auditlogs: h.Auditlogs,
		Caches:    h.Caches,
	}
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspaceID})
	keySpace, err := db.Query.FindKeySpaceByID(ctx, h.DB.RO(), api.KeyAuthID.String)
	require.NoError(t, err)

	grant := rbac.U(
		urn.New().Workspace(workspaceID).Project(keySpace.ProjectID).Keyspace(keySpace.ID),
		permissions.Delete,
	).Value
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", h.CreateRootKey(workspaceID, grant))},
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{ApiId: api.ID})
	require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)

	deleted, err := db.Query.FindApiByID(ctx, h.DB.RO(), api.ID)
	require.NoError(t, err)
	require.True(t, deleted.DeletedAtM.Valid)
}
