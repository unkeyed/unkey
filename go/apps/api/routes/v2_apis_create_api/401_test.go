package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_apis_create_api"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestCreateApi_Unauthorized(t *testing.T) {
	h := testutil.NewHarness(t)

	route := handler.New(handler.Services{
		DB:          h.DB,
		Keys:        h.Keys,
		Logger:      h.Logger,
		Permissions: h.Permissions,
		Auditlogs:   h.Auditlogs,
	})

	h.Register(route)

	// Invalid authorization token
	t.Run("invalid auth token", func(t *testing.T) {
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"Bearer invalid_token"},
		}

		req := handler.Request{
			Name: "test-api",
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, http.StatusUnauthorized, res.Status)
	})

	// Test with another workspace (example)
	t.Run("wrong workspace", func(t *testing.T) {
		// Create key for a different workspace
		otherWorkspaceId := uid.New(uid.WorkspacePrefix)
		otherRootKey := h.CreateRootKey(otherWorkspaceId, "api.*.create_api")

		otherHeaders := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", otherRootKey)},
		}

		req := handler.Request{
			Name: "test-api",
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, otherHeaders, req)

		require.Equal(t, http.StatusUnauthorized, res.Status)
	})
}
