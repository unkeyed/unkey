package middleware_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/middleware"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestRootAuthenticationCredentialMatrix(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID)
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
	ordinaryKey := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
	})

	type response struct {
		WorkspaceID string `json:"workspaceId"`
	}
	called := false
	route := zen.NewRoute(http.MethodPost, "/authentication-contract", func(ctx context.Context, s *zen.Session) error {
		called = true
		principal, err := s.GetPrincipal()
		if err != nil {
			return err
		}
		return s.JSON(http.StatusOK, response{WorkspaceID: principal.AuthorizedWorkspaceID})
	})
	h.Register(route,
		middleware.WithErrorHandling(),
		middleware.WithAuthentication(middleware.AuthenticationConfig{Auth: h.Auth}),
	)

	for _, tc := range []struct {
		name   string
		header string
		status int
		code   string
		detail string
	}{
		{
			name:   "missing",
			status: http.StatusBadRequest,
			code:   "missing",
		},
		{
			name:   "missing scheme",
			header: rootKey,
			status: http.StatusBadRequest,
			code:   "malformed",
			detail: "You must provide a valid root key in the Authorization header in the format 'Bearer ROOT_KEY'. Your authorization header is missing the 'Bearer ' prefix.",
		},
		{
			name:   "empty bearer",
			header: "Bearer ",
			status: http.StatusBadRequest,
			code:   "malformed",
			detail: "You must provide a valid root key in the Authorization header in the format 'Bearer ROOT_KEY'. Your authorization header is missing the 'Bearer ' prefix.",
		},
		{
			name:   "unknown",
			header: "Bearer " + uid.New(uid.KeyPrefix),
			status: http.StatusUnauthorized,
			code:   "key_not_found",
			detail: "The provided root key is invalid. We could not find the requested key.",
		},
		{
			name:   "ordinary API key",
			header: "Bearer " + ordinaryKey.Key,
			status: http.StatusUnauthorized,
			code:   "key_not_found",
			detail: "The provided root key is invalid. We could not find the requested key.",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			called = false
			res := testutil.CallRoute[struct{}, openapi.BadRequestErrorResponse](
				h, route, http.Header{"Authorization": {tc.header}}, struct{}{},
			)
			require.Equal(t, tc.status, res.Status)
			require.Equal(t, tc.status, res.Body.Error.Status)
			require.Equal(t, "https://unkey.com/docs/errors/unkey/authentication/"+tc.code, res.Body.Error.Type)
			require.Equal(t, tc.detail, res.Body.Error.Detail)
			require.NotEmpty(t, res.Body.Meta.RequestId)
			require.False(t, called)
		})
	}

	t.Run("missing credential workspace", func(t *testing.T) {
		rootKey := h.CreateRootKey(uid.New(uid.WorkspacePrefix))
		called = false
		res := testutil.CallRoute[struct{}, openapi.NotFoundErrorResponse](
			h, route, testutil.Bearer(rootKey), struct{}{},
		)
		require.Equal(t, http.StatusNotFound, res.Status, "%s", res.RawBody)
		require.Equal(t, openapi.BaseError{
			Status: http.StatusNotFound,
			Title:  "Not Found",
			Type:   "https://unkey.com/docs/errors/unkey/data/workspace_not_found",
			Detail: "The provided root key is invalid. The requested workspace does not exist.",
		}, res.Body.Error)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.False(t, called)
	})

	for _, header := range []string{"Bearer " + rootKey, "  Bearer   " + rootKey + "  "} {
		called = false
		res := testutil.CallRoute[struct{}, response](
			h, route, http.Header{"Authorization": {header}}, struct{}{},
		)
		require.Equal(t, http.StatusOK, res.Status)
		require.True(t, called)
		require.Equal(t, workspace.ID, res.Body.WorkspaceID)
	}
}
