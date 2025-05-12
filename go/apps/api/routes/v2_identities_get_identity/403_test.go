package handler_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/apps/api/routes/v2_identities_get_identity/handler"
	"github.com/unkeyed/unkey/go/internal/testutil"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestForbidden(t *testing.T) {
	h := testutil.NewHarness(t)
	route := handler.New(handler.Services{
		Logger:      h.Logger(),
		DB:          h.Database(),
		Keys:        h.Keys(),
		Permissions: h.Permissions(),
	})

	rootKeyID := h.CreateRootKey()
	headers := testutil.RootKeyAuth(rootKeyID)

	// Create test identity
	ctx := context.Background()
	tx, err := h.Database().RW().Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback()

	workspaceID := h.DefaultWorkspaceID()
	identityID := uid.New(uid.IdentityPrefix)
	otherIdentityID := uid.New(uid.IdentityPrefix)
	externalID := "test_user_403"

	// Insert test identity
	err = db.Query.InsertIdentity(ctx, tx, db.InsertIdentityParams{
		ID:          identityID,
		ExternalID:  externalID,
		WorkspaceID: workspaceID,
		Environment: "default",
		CreatedAt:   time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	// Insert another test identity
	err = db.Query.InsertIdentity(ctx, tx, db.InsertIdentityParams{
		ID:          otherIdentityID,
		ExternalID:  "other_user_403",
		WorkspaceID: workspaceID,
		Environment: "default",
		CreatedAt:   time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	err = tx.Commit()
	require.NoError(t, err)

	t.Run("no permission to read any identity", func(t *testing.T) {
		// Ensure no permissions are set
		h.SetupPermissions(t, rootKeyID, workspaceID, "identity.*.read_identity", false)

		req := handler.Request{
			identityID: &identityID,
		}
		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusForbidden, res.Status)
		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/auth/authorization/insufficient_permissions", res.Body.Error.Type)
		require.Equal(t, res.Body.Error.Detail, "insufficient permission")
	})

	t.Run("permission for specific identity only", func(t *testing.T) {
		// Set up permission for only the other identity
		h.SetupPermissions(t, rootKeyID, workspaceID, "identity.*.read_identity", false)
		h.SetupPermissions(t, rootKeyID, workspaceID, "identity."+otherIdentityID+".read_identity", true)

		// Try to access the first identity, should fail
		req := handler.Request{
			IdentityId: &identityID,
		}
		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusForbidden, res.Status)
		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/auth/authorization/insufficient_permissions", res.Body.Error.Type)
		require.Equal(t, res.Body.Error.Detail, "insufficient permission")

		// Try to access the permitted identity, should succeed
		req = handler.Request{
			identityID: &otherIdentityID,
		}
		successRes := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, http.StatusOK, successRes.Status)
		require.Equal(t, otherIdentityID, successRes.Body.Data.Id)
	})

	t.Run("permission by external ID but not by ID", func(t *testing.T) {
		// Reset permissions
		h.SetupPermissions(t, rootKeyID, workspaceID, "identity.*.read_identity", false)
		h.SetupPermissions(t, rootKeyID, workspaceID, "identity."+otherIdentityID+".read_identity", true)

		// Try to use externalId when only having permission for specific identity IDs
		req := handler.Request{
			externalID: &externalID,
		}
		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusForbidden, res.Status)
		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/auth/authorization/insufficient_permissions", res.Body.Error.Type)
		require.Equal(t, res.Body.Error.Detail, "insufficient permission")
	})
}
