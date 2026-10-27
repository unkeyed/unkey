package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_keys_whoami"
)

func TestGetKeyNotFound(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
		Auditlogs: h.Auditlogs,
		Vault:     h.Vault,
	}

	h.Register(route)

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, rootKeyGrant(h.Resources().UserWorkspace.ID, "*", "*", "*", "read"))

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("nonexistent raw key", func(t *testing.T) {
		nonexistentKey := uid.New("api")
		req := handler.Request{
			Key: nonexistentKey,
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
		require.Equal(t, "Resource does not exist.", res.Body.Error.Detail)
	})
}

func TestWhoamiDeniedDoesNotLeakKeyExistenceOrIDs(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
		Auditlogs: h.Auditlogs,
		Vault:     h.Vault,
	}

	h.Register(route)

	workspace := h.Resources().UserWorkspace
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})
	otherAPI := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})
	key := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
	})
	otherWorkspace := h.CreateWorkspace()

	tests := []struct {
		name        string
		workspaceID string
		permissions []string
	}{
		{
			name:        "no permissions",
			workspaceID: workspace.ID,
		},
		{
			name:        "create permission without read",
			workspaceID: workspace.ID,
			permissions: []string{rootKeyGrant(workspace.ID, "*", "*", "*", "write")},
		},
		{
			name:        "URN write permission without read",
			workspaceID: workspace.ID,
			permissions: []string{rootKeyGrant(workspace.ID, api.ProjectID, api.KeyAuthID.String, key.KeyID, "write")},
		},
		{
			name:        "cross workspace access",
			workspaceID: otherWorkspace.ID,
			permissions: []string{rootKeyGrant(otherWorkspace.ID, "*", "*", "*", "read")},
		},
		{
			name:        "cross API access",
			workspaceID: workspace.ID,
			permissions: []string{rootKeyGrant(workspace.ID, otherAPI.ProjectID, otherAPI.KeyAuthID.String, "*", "read")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootKey := h.CreateRootKey(tt.workspaceID, tt.permissions...)
			headers := http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
			}

			missing := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, handler.Request{
				Key: uid.New(uid.KeyPrefix),
			})
			require.Equal(t, http.StatusNotFound, missing.Status)
			require.Equal(t, "Resource does not exist.", missing.Body.Error.Detail)

			denied := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, handler.Request{
				Key: key.Key,
			})
			require.Equal(t, missing.Status, denied.Status)
			require.Equal(t, missing.Body.Error, denied.Body.Error)
			for _, id := range []string{api.ID, workspace.ID, api.ProjectID, api.KeyAuthID.String, key.KeyID} {
				require.NotContains(t, denied.RawBody, id)
			}
		})
	}
}
