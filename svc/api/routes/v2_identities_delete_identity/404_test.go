package handler_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_identities_delete_identity"
)

func TestDeleteIdentityNotFound(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		Logger:    h.Logger,
		DB:        h.DB,
		Keys:      h.Keys,
		Auditlogs: h.Auditlogs,
	}

	h.Register(route)

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "identity.*.delete_identity")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("delete identity with non-existent external ID", func(t *testing.T) {
		nonExistentExternalID := "non_existent_" + uid.New("test")
		req := handler.Request{Identity: nonExistentExternalID}
		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/errors/unkey/data/identity_not_found", res.Body.Error.Type)
		require.Equal(t, "This identity does not exist.", res.Body.Error.Detail)
		require.Equal(t, http.StatusNotFound, res.Body.Error.Status)
		require.Equal(t, "Not Found", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	})

	t.Run("delete identity with external ID from different workspace", func(t *testing.T) {
		// Create identity in user workspace
		identityId := uid.New(uid.IdentityPrefix)
		externalId := "ext_" + identityId
		err := db.Query.InsertIdentity(t.Context(), h.DB.RW(), db.InsertIdentityParams{
			ID:          identityId,
			ExternalID:  externalId,
			WorkspaceID: h.Resources().UserWorkspace.ID,
			Environment: "default",
			CreatedAt:   time.Now().UnixMilli(),
			Meta:        []byte("{}"),
		})
		require.NoError(t, err)

		// Create a different workspace
		differentWorkspace := h.CreateWorkspace()

		// Try to delete it using external ID with a key from different workspace
		differentWorkspaceKey := h.CreateRootKey(differentWorkspace.ID, "identity.*.delete_identity")
		differentHeaders := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", differentWorkspaceKey)},
		}

		req := handler.Request{Identity: externalId}
		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, differentHeaders, req)
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/errors/unkey/data/identity_not_found", res.Body.Error.Type)
		require.Equal(t, "This identity does not exist.", res.Body.Error.Detail)
		require.Equal(t, http.StatusNotFound, res.Body.Error.Status)
		require.Equal(t, "Not Found", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	})

	t.Run("delete already deleted identity", func(t *testing.T) {
		// Create and soft delete an identity
		identityId := uid.New(uid.IdentityPrefix)
		externalId := "ext_" + identityId
		err := db.Query.InsertIdentity(t.Context(), h.DB.RW(), db.InsertIdentityParams{
			ID:          identityId,
			ExternalID:  externalId,
			WorkspaceID: h.Resources().UserWorkspace.ID,
			Environment: "default",
			CreatedAt:   time.Now().UnixMilli(),
			Meta:        []byte("{}"),
		})
		require.NoError(t, err)

		// Soft delete the identity directly in DB
		err = db.Query.SoftDeleteIdentity(t.Context(), h.DB.RW(), db.SoftDeleteIdentityParams{
			Identity:    identityId,
			WorkspaceID: h.Resources().UserWorkspace.ID,
		})
		require.NoError(t, err)

		// Try to delete it again via API
		req := handler.Request{Identity: externalId}
		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/errors/unkey/data/identity_not_found", res.Body.Error.Type)
		require.Equal(t, "This identity does not exist.", res.Body.Error.Detail)
		require.Equal(t, http.StatusNotFound, res.Body.Error.Status)
		require.Equal(t, "Not Found", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	})

	t.Run("delete identity using very long non-existent external ID", func(t *testing.T) {
		longExternalID := "very_long_external_id_that_does_not_exist_" + uid.New("test") + "_" + uid.New("test2")
		req := handler.Request{Identity: longExternalID}
		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/errors/unkey/data/identity_not_found", res.Body.Error.Type)
		require.Equal(t, "This identity does not exist.", res.Body.Error.Detail)
		require.Equal(t, http.StatusNotFound, res.Status)
		require.Equal(t, "Not Found", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	})
}
