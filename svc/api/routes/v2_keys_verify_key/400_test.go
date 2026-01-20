package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_keys_verify_key"
)

func TestBadRequest(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:         h.DB,
		Keys:       h.Keys,
		Logger:     h.Logger,
		Auditlogs:  h.Auditlogs,
		ClickHouse: h.ClickHouse,
	}

	h.Register(route)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "api.*.verify_key")
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})

	validHeaders := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("missing required fields", func(t *testing.T) {
		t.Run("missing key", func(t *testing.T) {
			req := handler.Request{
				// Key missing
			}

			res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, validHeaders, req)
			require.Equal(t, 400, res.Status)
			require.NotNil(t, res.Body)
			require.NotNil(t, res.Body.Error)
		})
	})

	t.Run("invalid validation", func(t *testing.T) {
		t.Run("invalid cost value", func(t *testing.T) {
			key := h.CreateKey(seed.CreateKeyRequest{
				WorkspaceID: workspace.ID,
				KeySpaceID:  api.KeyAuthID.String,
			})

			req := handler.Request{
				Key: key.Key,
				Ratelimits: &[]openapi.KeysVerifyKeyRatelimit{
					{
						Name:     "test",
						Cost:     ptr.P(-1), // Invalid negative cost
						Limit:    ptr.P(10),
						Duration: ptr.P(60000),
					},
				},
			}

			res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, validHeaders, req)
			require.Equal(t, 400, res.Status)
			require.NotNil(t, res.Body)
			require.NotNil(t, res.Body.Error)
		})

		t.Run("invalid credits cost", func(t *testing.T) {
			key := h.CreateKey(seed.CreateKeyRequest{
				WorkspaceID: workspace.ID,
				KeySpaceID:  api.KeyAuthID.String,
			})

			req := handler.Request{
				Key: key.Key,
				Credits: &openapi.KeysVerifyKeyCredits{
					Cost: -1, // Invalid negative cost
				},
			}

			res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, validHeaders, req)
			require.Equal(t, 400, res.Status)
			require.NotNil(t, res.Body)
			require.NotNil(t, res.Body.Error)
		})
	})

	t.Run("invalid permissions query syntax", func(t *testing.T) {
		key := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
		})

		testCases := []struct {
			name        string
			permissions string
			expectError string
		}{
			{
				name:        "missing operand",
				permissions: "permission_1 AND",
				expectError: "Unexpected end of query. Expected a permission identifier or opening parenthesis.",
			},
			{
				name:        "unmatched parenthesis",
				permissions: "(permission_1 AND permission_2",
				expectError: "Missing closing parenthesis ')' at position 30. Every opening parenthesis must have a matching closing parenthesis.",
			},
			{
				name:        "empty parentheses",
				permissions: "permission_1 AND ()",
				expectError: "Empty parentheses found at position 18. Parentheses must contain at least one permission or expression.",
			},
			{
				name:        "operator at start",
				permissions: "AND permission_1",
				expectError: "Unexpected token 'AND' at position 0. Expected a permission identifier or opening parenthesis.",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				req := handler.Request{
					Key:         key.Key,
					Permissions: ptr.P(tc.permissions),
				}

				res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, validHeaders, req)
				require.Equal(t, 400, res.Status)
				require.NotNil(t, res.Body)
				require.NotNil(t, res.Body.Error)
				require.Contains(t, res.Body.Error.Detail, tc.expectError)
				require.Equal(t, "https://unkey.com/docs/errors/user/bad_request/permissions_query_syntax_error", res.Body.Error.Type)
			})
		}
	})
}
