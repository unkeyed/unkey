package handler_test

import (
	"context"
	"database/sql"
	"errors"
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
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_identities_update_identity"
)

func TestForbidden(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		DB:        h.DB,
		Auditlogs: h.Auditlogs,
	}

	h.Register(route)
	workspaceID := h.Resources().UserWorkspace.ID
	forbiddenIdentity := h.CreateIdentity(seed.CreateIdentityRequest{
		WorkspaceID: workspaceID,
		ExternalID:  uid.New(uid.TestPrefix),
	})

	t.Run("no permission to update identity", func(t *testing.T) {
		// Create root key without permissions
		rootKeyID := h.CreateRootKey(workspaceID)
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKeyID)},
		}

		meta := map[string]interface{}{
			"test": "value",
		}
		req := handler.Request{
			Identity: forbiddenIdentity.ExternalID,
			Meta:     &meta,
		}
		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusForbidden, res.Status)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/authorization/insufficient_permissions", res.Body.Error.Type)
		require.Contains(t, res.Body.Error.Detail, "permission")
		require.NotContains(t, res.Body.Error.Detail, forbiddenIdentity.ID)
	})

	t.Run("wrong permission type", func(t *testing.T) {
		// Create root key with wrong permission
		rootKeyID := h.CreateRootKey(workspaceID, "identity.*.create_identity")
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKeyID)},
		}

		meta := map[string]interface{}{
			"test": "value",
		}
		req := handler.Request{
			Identity: forbiddenIdentity.ExternalID,
			Meta:     &meta,
		}
		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusForbidden, res.Status)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/authorization/insufficient_permissions", res.Body.Error.Type)
		require.Contains(t, res.Body.Error.Detail, "permission")
	})

	t.Run("with permission to update identity", func(t *testing.T) {
		// Create test identity first
		ctx := context.Background()
		tx, err := h.DB.RW().Begin(ctx)
		require.NoError(t, err)
		defer func() {
			err := tx.Rollback()
			require.True(t, err == nil || errors.Is(err, sql.ErrTxDone), "unexpected rollback error: %v", err)
		}()

		identityID := uid.New(uid.IdentityPrefix)
		externalID := "test_user_403"

		// Insert test identity
		err = db.Query.InsertIdentity(ctx, tx, db.InsertIdentityParams{
			ID:          identityID,
			ExternalID:  externalID,
			WorkspaceID: workspaceID,
			Environment: "default",
			CreatedAt:   time.Now().UnixMilli(),
			Meta:        []byte("{}"),
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
			Identity: externalID,
			Meta:     &meta,
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, http.StatusOK, res.Status, "expected 200, got: %d, response: %s", res.Status, res.RawBody)
		require.Equal(t, externalID, res.Body.Data.ExternalId)
	})

	t.Run("missing identity does not reveal existence without permission", func(t *testing.T) {
		rootKeyID := h.CreateRootKey(workspaceID)
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKeyID)},
		}
		meta := map[string]any{"test": "value"}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](
			h,
			route,
			headers,
			handler.Request{Identity: uid.New(uid.TestPrefix), Meta: &meta},
		)
		require.Equal(t, http.StatusForbidden, res.Status)
	})
}
