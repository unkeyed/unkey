package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_identities_get_identity"
)

func TestNotFound(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		DB: h.DB,
	}

	h.Register(route)

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "identity.*.read_identity")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("external ID does not exist", func(t *testing.T) {
		nonExistentExternalID := uid.New(uid.TestPrefix)
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
		deletedExternalID := uid.New(uid.TestPrefix)

		deletedIdentityID := h.CreateIdentity(seed.CreateIdentityRequest{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			Environment: "default",
			ExternalID:  deletedExternalID,
		}).ID

		// Mark it as deleted
		err := db.Query.SoftDeleteIdentity(ctx, h.DB.RW(), db.SoftDeleteIdentityParams{
			IdentityID:  deletedIdentityID,
			WorkspaceID: h.Resources().UserWorkspace.ID,
		})
		require.NoError(t, err)

		// Try to retrieve the deleted identity by externalId
		reqByExternalId := handler.Request{
			Identity: deletedExternalID,
		}
		resByExternalId := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, reqByExternalId)
		require.Equal(t, http.StatusNotFound, resByExternalId.Status, "expected 404 for deleted identity (by externalId)")
	})
}
