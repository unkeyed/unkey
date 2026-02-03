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
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_identities_get_identity"
)

func TestNotFound(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		Logger: h.Logger,
		DB:     h.DB,
		Keys:   h.Keys,
	}

	h.Register(route)

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "identity.*.read_identity")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("external ID does not exist", func(t *testing.T) {
		nonExistentExternalID := "non_existent_external_id"
		req := handler.Request{
			Identity: nonExistentExternalID,
		}
		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404, got: %d", res.Status)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/data/identity_not_found", res.Body.Error.Type)
		require.Equal(t, "This identity does not exist.", res.Body.Error.Detail)
		require.Equal(t, http.StatusNotFound, res.Body.Error.Status)
		require.Equal(t, "Not Found", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	})

	t.Run("deleted identity", func(t *testing.T) {
		// Create an identity that we'll mark as deleted
		ctx := context.Background()
		deletedIdentityID := uid.New(uid.IdentityPrefix)
		deletedExternalID := "test_deleted_identity"

		tx, err := h.DB.RW().Begin(ctx)
		require.NoError(t, err)
		defer func() {
			err := tx.Rollback()
			require.True(t, err == nil || errors.Is(err, sql.ErrTxDone), "unexpected rollback error: %v", err)
		}()

		// Insert the identity
		err = db.Query.InsertIdentity(ctx, tx, db.InsertIdentityParams{
			ID:          deletedIdentityID,
			ExternalID:  deletedExternalID,
			WorkspaceID: h.Resources().UserWorkspace.ID,
			Environment: "default",
			CreatedAt:   time.Now().UnixMilli(),
			Meta:        []byte("{}"),
		})
		require.NoError(t, err)

		// Mark it as deleted
		err = db.Query.SoftDeleteIdentity(ctx, tx, db.SoftDeleteIdentityParams{
			IdentityID:    deletedIdentityID,
			WorkspaceID: h.Resources().UserWorkspace.ID,
		})
		require.NoError(t, err)

		err = tx.Commit()
		require.NoError(t, err)

		// Try to retrieve the deleted identity by externalId
		reqByExternalId := handler.Request{
			Identity: deletedExternalID,
		}
		resByExternalId := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, reqByExternalId)
		require.Equal(t, http.StatusNotFound, resByExternalId.Status, "expected 404 for deleted identity (by externalId)")
	})
}
