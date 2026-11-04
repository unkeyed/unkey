package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_keys_update_credits"
)

func TestKeyUpdateCreditsForbidden(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:           h.DB,
		Auditlogs:    h.Auditlogs,
		KeyCache:     h.Caches.VerificationKeyByHash,
		UsageLimiter: h.UsageLimiter,
	}

	h.Register(route)

	// Create API for testing using testutil helper
	apiName := "test-api"
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: h.Resources().UserWorkspace.ID,
		Name:        &apiName,
	})

	diffApi := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: h.Resources().UserWorkspace.ID,
		Name:        &apiName,
	})

	key := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: api.WorkspaceID,
		KeySpaceID:  api.KeyAuthID.String,
		Remaining:   new(int64(100)),
	})

	req := handler.Request{
		KeyId:     key.KeyID,
		Operation: openapi.Increment,
		Value:     nullable.NewNullableWithValue(int64(10)),
	}

	t.Run("no permissions", func(t *testing.T) {
		// Create root key with no permissions
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID)

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
		require.Equal(t, 403, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("read permission cannot update", func(t *testing.T) {
		rootKey := h.CreateRootKey(api.WorkspaceID, rbac.U(urn.New().Workspace(api.WorkspaceID).Project(api.ProjectID).Keyspace(api.KeyAuthID.String).Key(key.KeyID), permissions.Read).Value)

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
		require.Equal(t, 403, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "#write")
	})

	t.Run("cross workspace access", func(t *testing.T) {
		// Create a different workspace
		differentWorkspace := h.CreateWorkspace()

		rootKey := h.CreateRootKey(differentWorkspace.ID, rbac.U(urn.New().Workspace(api.WorkspaceID).Project(api.ProjectID).Keyspace(api.KeyAuthID.String).Key(key.KeyID), permissions.Write).Value)

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
		require.Equal(t, "The specified key was not found.", res.Body.Error.Detail)
	})

	t.Run("cross api access", func(t *testing.T) {
		rootKey := h.CreateRootKey(api.WorkspaceID, rbac.U(urn.New().Workspace(api.WorkspaceID).Project(diffApi.ProjectID).Keyspace(diffApi.KeyAuthID.String).Key("*"), permissions.Write).Value)

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)

		require.Equal(t, 403, res.Status)
		require.NotNil(t, res.Body)
	})
}
