package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_apis_create_api"
)

// TestCreateApi_Unauthorized verifies that API creation requests are properly
// rejected when authentication fails. This test ensures that invalid or missing
// authorization tokens result in 401 Unauthorized responses, preventing
// unauthorized access to the API creation endpoint.
func TestCreateApi_Unauthorized(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
		Keys:      h.Keys,
		Auditlogs: h.Auditlogs,
	}

	h.Register(route)

	// This test validates that requests with malformed or invalid authorization
	// tokens are rejected with a 401 status code, ensuring proper security
	// boundaries for the API creation endpoint.
	t.Run("invalid auth token", func(t *testing.T) {
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"Bearer invalid_token"},
		}

		req := handler.Request{
			Name: "test-api",
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, http.StatusUnauthorized, res.Status, "expected 401, sent: %+v, received: %s", req, res.RawBody)
	})

}
