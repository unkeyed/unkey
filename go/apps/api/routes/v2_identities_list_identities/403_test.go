package handler_test

import (
	"context"
	"database/sql"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/apps/api/routes/v2_identities_list_identities/handler"
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

	// Create test identities in different environments
	ctx := context.Background()
	tx, err := h.Database().RW().Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback()

	workspaceID := h.DefaultWorkspaceID()

	// Insert identity in default environment
	defaultIdentityID := uid.New(uid.IdentityPrefix)
	err = db.Query.InsertIdentity(ctx, tx, db.InsertIdentityParams{
		ID:          defaultIdentityID,
		ExternalID:  "test_user_default",
		WorkspaceID: workspaceID,
		Environment: "default",
		CreatedAt:   time.Now().UnixMilli(),
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
	})
	require.NoError(t, err)

	err = tx.Commit()
	require.NoError(t, err)

	t.Run("no permission to read any identity", func(t *testing.T) {
		// Ensure no permissions are set
		h.SetupPermissions(t, rootKeyID, workspaceID, "identity.*.read_identity", false)

		req := handler.Request{}
		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusForbidden, res.Status)
		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/auth/authorization/insufficient_permissions", res.Body.Error.Type)
		require.Contains(t, res.Body.Error.Detail, "insufficient permission")
	})

	t.Run("permission for specific environment only", func(t *testing.T) {
		// Reset permissions and only allow production environment
		h.SetupPermissions(t, rootKeyID, workspaceID, "identity.*.read_identity", false)
		h.SetupPermissions(t, rootKeyID, workspaceID, "identity.production.read_identity", true)

		// Try to access default environment, should fail
		defaultEnv := "default"
		req := handler.Request{
			environment: &defaultEnv,
		}
		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusForbidden, res.Status)
		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/auth/authorization/insufficient_permissions", res.Body.Error.Type)

		// Try to access production environment, should succeed
		prodEnv := "production"
		req = handler.Request{
			environment: &prodEnv,
		}
		successRes := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, http.StatusOK, successRes.Status)
		require.Equal(t, 1, len(successRes.Body.Data.Identities))
		require.Equal(t, prodIdentityID, successRes.Body.Data.Identities[0].ID)
	})
}
