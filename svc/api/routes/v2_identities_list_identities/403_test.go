package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/testutil"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_identities_list_identities"
)

func TestForbidden(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		Logger: h.Logger,
		DB:     h.DB,
		Keys:   h.Keys,
	}

	// Create a rootKey without any permissions
	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID)
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Create test identities in different environments
	ctx := context.Background()
	tx, err := h.DB.RW().Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback()

	workspaceID := h.Resources().UserWorkspace.ID

	// Insert identity in default environment
	defaultIdentityID := uid.New(uid.IdentityPrefix)
	err = db.Query.InsertIdentity(ctx, tx, db.InsertIdentityParams{
		ID:          defaultIdentityID,
		ExternalID:  "test_user_default",
		WorkspaceID: workspaceID,
		Environment: "default",
		CreatedAt:   time.Now().UnixMilli(),
		Meta:        []byte("{}"),
	})
	require.NoError(t, err)

	// Insert identity in production environment
	prodIdentityID := uid.New(uid.IdentityPrefix)
	err = db.Query.InsertIdentity(ctx, tx, db.InsertIdentityParams{
		ID:          prodIdentityID,
		ExternalID:  "test_user_prod",
		WorkspaceID: workspaceID,
		Environment: "production",
		CreatedAt:   time.Now().UnixMilli(),
		Meta:        []byte("{}"),
	})
	require.NoError(t, err)

	// Insert identity in staging environment
	stagingIdentityID := uid.New(uid.IdentityPrefix)
	err = db.Query.InsertIdentity(ctx, tx, db.InsertIdentityParams{
		ID:          stagingIdentityID,
		ExternalID:  "test_user_staging",
		WorkspaceID: workspaceID,
		Environment: "staging",
		CreatedAt:   time.Now().UnixMilli(),
		Meta:        []byte("{}"),
	})
	require.NoError(t, err)

	err = tx.Commit()
	require.NoError(t, err)

	// Register the route
	h.Register(route)

	t.Run("no permission to read any identity", func(t *testing.T) {
		// With no permissions set, should return 403
		req := handler.Request{}
		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusForbidden, res.Status)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/authorization/insufficient_permissions", res.Body.Error.Type)
		require.Contains(t, res.Body.Error.Detail, "Missing one of these permissions")
	})

	// Create a new key with specific permissions for certain environments
	t.Run("with permission for only specific environment", func(t *testing.T) {
		// Create a new key with production environment permissions
		prodPermKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "identity.*.read_identity")
		prodHeaders := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", prodPermKey)},
		}

		// Attempt to list identities (should see all identities with wildcard permission)
		req := handler.Request{}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, prodHeaders, req)

		// Should get a 200 response with all identities
		require.Equal(t, http.StatusOK, res.Status)

		// Verify we can see all identities including production
		foundProd := false
		for _, identity := range res.Body.Data {
			// Should be able to see production identity
			if identity.ExternalId == "test_user_prod" {
				foundProd = true
			}
		}

		require.True(t, foundProd, "Should find production identity")
	})

	t.Run("with wildcard permission", func(t *testing.T) {
		// Create a new key with wildcard permissions
		wildcardKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "identity.*.read_identity")
		wildcardHeaders := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", wildcardKey)},
		}

		// Attempt to list identities
		req := handler.Request{}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, wildcardHeaders, req)

		// Should get a 200 response with all identities
		require.Equal(t, http.StatusOK, res.Status)

		// Should see at least 3 identities (one from each environment)
		require.GreaterOrEqual(t, len(res.Body.Data), 3)

		// Verify we can find all environment identities
		foundDefault := false
		foundProd := false
		foundStaging := false

		for _, identity := range res.Body.Data {
			if identity.ExternalId == "test_user_default" {
				foundDefault = true
			}
			if identity.ExternalId == "test_user_prod" {
				foundProd = true
			}
			if identity.ExternalId == "test_user_staging" {
				foundStaging = true
			}
		}

		require.True(t, foundDefault, "Should find default environment identity")
		require.True(t, foundProd, "Should find production environment identity")
		require.True(t, foundStaging, "Should find staging environment identity")
	})
}
