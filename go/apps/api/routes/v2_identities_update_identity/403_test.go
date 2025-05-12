package handler_test

import (
	"context"
	"database/sql"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/apps/api/routes/v2_identities_update_identity/handler"
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
		Auditlogs:   h.Auditlogs(),
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

	err = tx.Commit()
	require.NoError(t, err)

	t.Run("no permission to update identity", func(t *testing.T) {
		// Ensure no permissions are set
		h.SetupPermissions(t, rootKeyID, workspaceID, "identity.*.update_identity", false)

		req := handler.Request{
			identityID: &identityID,
			meta: map[string]interface{}{
				"test": "value",
			},
		}
		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusForbidden, res.Status)
		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/auth/authorization/insufficient_permissions", res.Body.Error.Type)
		require.Contains(t, res.Body.Error.Detail, "insufficient permission")
	})

	t.Run("with permission to update identity", func(t *testing.T) {
		// Set permission
		h.SetupPermissions(t, rootKeyID, workspaceID, "identity.*.update_identity", true)

		req := handler.Request{
			identityID: &identityID,
			meta: map[string]interface{}{
				"test": "value",
			},
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, http.StatusOK, res.Status, "expected 200, got: %d, response: %s", res.Status, res.RawBody)
		require.Equal(t, identityID, res.Body.Data.ID)
	})
}
