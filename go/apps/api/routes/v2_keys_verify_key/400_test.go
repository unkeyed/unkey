package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_verify_key"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/seed"
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
		t.Run("missing apiId", func(t *testing.T) {
			req := handler.Request{
				Key: "test_key",
				// ApiId missing
			}

			res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, validHeaders, req)
			require.Equal(t, 400, res.Status)
			require.NotNil(t, res.Body)
			require.NotNil(t, res.Body.Error)
		})

		t.Run("missing key", func(t *testing.T) {
			req := handler.Request{
				ApiId: api.ID,
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
				KeyAuthID:   api.KeyAuthID.String,
			})

			req := handler.Request{
				ApiId: api.ID,
				Key:   key.Key,
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
				KeyAuthID:   api.KeyAuthID.String,
			})

			req := handler.Request{
				ApiId: api.ID,
				Key:   key.Key,
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
}
