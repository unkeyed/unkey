package handler_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_identities_delete_identity"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestDeleteIdentityNotFound(t *testing.T) {
	h := testutil.NewHarness(t)

	route := handler.New(handler.Services{
		DB:          h.DB,
		Keys:        h.Keys,
		Logger:      h.Logger,
		Permissions: h.Permissions,
		Auditlogs:   h.Auditlogs,
	})

	h.Register(route)

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "identity.*.delete_identity")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("delete identity with non-existent ID", func(t *testing.T) {
		nonExistentID := uid.New(uid.IdentityPrefix)
		req := handler.Request{IdentityId: ptr.P(nonExistentID)}
		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/data/identity_not_found", res.Body.Error.Type)
		require.Equal(t, "This identity does not exist.", res.Body.Error.Detail)
		require.Equal(t, http.StatusNotFound, res.Body.Error.Status)
		require.Equal(t, "Not Found", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Nil(t, res.Body.Error.Instance)
	})

	t.Run("delete identity with non-existent external ID", func(t *testing.T) {
		nonExistentExternalID := "non_existent_" + uid.New("test")
		req := handler.Request{ExternalId: ptr.P(nonExistentExternalID)}
		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/data/identity_not_found", res.Body.Error.Type)
		require.Equal(t, "This identity does not exist.", res.Body.Error.Detail)
		require.Equal(t, http.StatusNotFound, res.Body.Error.Status)
		require.Equal(t, "Not Found", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Nil(t, res.Body.Error.Instance)
	})

	t.Run("delete identity from different workspace (masked as 404)", func(t *testing.T) {
		// Create identity in user workspace
		identityId := uid.New(uid.IdentityPrefix)
		err := db.Query.InsertIdentity(t.Context(), h.DB.RW(), db.InsertIdentityParams{
			ID:          identityId,
			ExternalID:  "ext_" + identityId,
			WorkspaceID: h.Resources().UserWorkspace.ID,
			Environment: "default",
			CreatedAt:   time.Now().UnixMilli(),
			Meta:        nil,
		})
		require.NoError(t, err)

		// Create a different workspace
		differentWorkspace := h.CreateWorkspace()

		// Try to delete it using a key from different workspace
		differentWorkspaceKey := h.CreateRootKey(differentWorkspace.ID, "identity.*.delete_identity")
		differentHeaders := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", differentWorkspaceKey)},
		}

		req := handler.Request{IdentityId: ptr.P(identityId)}
		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, differentHeaders, req)
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/data/identity_not_found", res.Body.Error.Type)
		require.Equal(t, "This identity does not exist.", res.Body.Error.Detail)
		require.Equal(t, http.StatusNotFound, res.Body.Error.Status)
		require.Equal(t, "Not Found", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Nil(t, res.Body.Error.Instance)

		// Verify the identity still exists in the original workspace
		identity, err := db.Query.FindIdentityByID(t.Context(), h.DB.RO(), db.FindIdentityByIDParams{
			ID:      identityId,
			Deleted: false,
		})
		require.NoError(t, err)
		require.Equal(t, h.Resources().UserWorkspace.ID, identity.WorkspaceID)
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
			Meta:        nil,
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

		req := handler.Request{ExternalId: ptr.P(externalId)}
		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, differentHeaders, req)
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/data/identity_not_found", res.Body.Error.Type)
		require.Equal(t, "This identity does not exist.", res.Body.Error.Detail)
		require.Equal(t, http.StatusNotFound, res.Body.Error.Status)
		require.Equal(t, "Not Found", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Nil(t, res.Body.Error.Instance)
	})

	t.Run("delete already deleted identity", func(t *testing.T) {
		// Create and soft delete an identity
		identityId := uid.New(uid.IdentityPrefix)
		err := db.Query.InsertIdentity(t.Context(), h.DB.RW(), db.InsertIdentityParams{
			ID:          identityId,
			ExternalID:  "ext_" + identityId,
			WorkspaceID: h.Resources().UserWorkspace.ID,
			Environment: "default",
			CreatedAt:   time.Now().UnixMilli(),
			Meta:        nil,
		})
		require.NoError(t, err)

		// Soft delete the identity directly in DB
		err = db.Query.SoftDeleteIdentity(t.Context(), h.DB.RW(), identityId)
		require.NoError(t, err)

		// Try to delete it again via API
		req := handler.Request{IdentityId: ptr.P(identityId)}
		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/data/identity_not_found", res.Body.Error.Type)
		require.Equal(t, "This identity does not exist.", res.Body.Error.Detail)
		require.Equal(t, http.StatusNotFound, res.Body.Error.Status)
		require.Equal(t, "Not Found", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Nil(t, res.Body.Error.Instance)
	})

	t.Run("delete identity with malformed ID prefix", func(t *testing.T) {
		// Use wrong prefix for identity ID
		wrongPrefixID := uid.New(uid.KeyPrefix) // Using key prefix instead of identity prefix
		req := handler.Request{IdentityId: ptr.P(wrongPrefixID)}
		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/data/identity_not_found", res.Body.Error.Type)
		require.Equal(t, "This identity does not exist.", res.Body.Error.Detail)
		require.Equal(t, http.StatusNotFound, res.Body.Error.Status)
		require.Equal(t, "Not Found", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Nil(t, res.Body.Error.Instance)
	})

	t.Run("delete identity with valid ID format but wrong workspace", func(t *testing.T) {
		// Create a different workspace
		differentWorkspace := h.CreateWorkspace()

		// Create identity in different workspace
		identityId := uid.New(uid.IdentityPrefix)
		err := db.Query.InsertIdentity(t.Context(), h.DB.RW(), db.InsertIdentityParams{
			ID:          identityId,
			ExternalID:  "ext_" + identityId,
			WorkspaceID: differentWorkspace.ID,
			Environment: "default",
			CreatedAt:   time.Now().UnixMilli(),
			Meta:        nil,
		})
		require.NoError(t, err)

		// Try to delete with key from user workspace
		req := handler.Request{IdentityId: ptr.P(identityId)}
		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/data/identity_not_found", res.Body.Error.Type)
		require.Equal(t, "This identity does not exist.", res.Body.Error.Detail)
		require.Equal(t, http.StatusNotFound, res.Body.Error.Status)
		require.Equal(t, "Not Found", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Nil(t, res.Body.Error.Instance)
	})

	t.Run("delete identity using very long non-existent external ID", func(t *testing.T) {
		longExternalID := "very_long_external_id_that_does_not_exist_" + uid.New("test") + "_" + uid.New("test2")
		req := handler.Request{ExternalId: ptr.P(longExternalID)}
		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/data/identity_not_found", res.Body.Error.Type)
		require.Equal(t, "This identity does not exist.", res.Body.Error.Detail)
		require.Equal(t, http.StatusNotFound, res.Status)
		require.Equal(t, "Not Found", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Nil(t, res.Body.Error.Instance)
	})
}
