package handler_test

import (
	"testing"

	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_apis_create_api"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/authz"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

// TestCreateApi_Unauthorized verifies that API creation requests are properly
// rejected when authentication fails. This test ensures that invalid or missing
// authorization tokens result in 401 Unauthorized responses, preventing
// unauthorized access to the API creation endpoint.
func TestCreateApi_Unauthorized(t *testing.T) {
	authz.Test401[handler.Request, handler.Response](t,
		func(h *testutil.Harness) zen.Route {
			return &handler.Handler{
				Logger:    h.Logger,
				DB:        h.DB,
				Keys:      h.Keys,
				Auditlogs: h.Auditlogs,
			}
		},
		func() handler.Request {
			return handler.Request{
				Name: "test-api",
			}
		},
	)
}
