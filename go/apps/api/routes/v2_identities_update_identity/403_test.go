package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_identities_update_identity"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestForbidden(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		Logger:    h.Logger,
		DB:        h.DB,
		Keys:      h.Keys,
		Auditlogs: h.Auditlogs,
	}

	h.Register(route)

	t.Run("no permission to update identity", func(t *testing.T) {
		// Create root key without permissions
		rootKeyID := h.CreateRootKey(h.Resources().UserWorkspace.ID)
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKeyID)},
		}

		identityID := uid.New(uid.IdentityPrefix)
		meta := map[string]interface{}{
			"test": "value",
		}
		req := handler.Request{
			IdentityId: &identityID,
			Meta:       &meta,
		}
		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusForbidden, res.Status)
		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/authorization/insufficient_permissions", res.Body.Error.Type)
		require.Contains(t, res.Body.Error.Detail, "permission")
	})

	t.Run("wrong permission type", func(t *testing.T) {
		// Create root key with wrong permission
		rootKeyID := h.CreateRootKey(h.Resources().UserWorkspace.ID, "identity.*.create_identity")
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKeyID)},
		}

		identityID := uid.New(uid.IdentityPrefix)
		meta := map[string]interface{}{
			"test": "value",
		}
		req := handler.Request{
			IdentityId: &identityID,
			Meta:       &meta,
		}
		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusForbidden, res.Status)
		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/authorization/insufficient_permissions", res.Body.Error.Type)
		require.Contains(t, res.Body.Error.Detail, "permission")
	})

	t.Run("with permission to update identity", func(t *testing.T) {
		// Create test identity first
		ctx := context.Background()
		tx, err := h.DB.RW().Begin(ctx)
		require.NoError(t, err)
		defer tx.Rollback()

		workspaceID := h.Resources().UserWorkspace.ID
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

		// Create root key with correct permission
		rootKeyID := h.CreateRootKey(workspaceID, "identity.*.update_identity")
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKeyID)},
		}

		meta := map[string]interface{}{
			"test": "value",
		}
		req := handler.Request{
			IdentityId: &identityID,
			Meta:       &meta,
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, http.StatusOK, res.Status, "expected 200, got: %d, response: %s", res.Status, res.RawBody)
		require.Equal(t, identityID, res.Body.Data.Id)
	})

	t.Run("specific identity permission for wrong identity", func(t *testing.T) {
		// Create a different identity ID
		differentIdentityId := uid.New(uid.IdentityPrefix)

		// Create root key with permission for the different identity
		rootKeyID := h.CreateRootKey(h.Resources().UserWorkspace.ID, fmt.Sprintf("identity.%s.update_identity", differentIdentityId))
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKeyID)},
		}

		identityID := uid.New(uid.IdentityPrefix)
		meta := map[string]interface{}{
			"test": "value",
		}
		req := handler.Request{
			IdentityId: &identityID,
			Meta:       &meta,
		}
		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusForbidden, res.Status)
		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/authorization/insufficient_permissions", res.Body.Error.Type)
		require.Contains(t, res.Body.Error.Detail, "permission")
	})
}
