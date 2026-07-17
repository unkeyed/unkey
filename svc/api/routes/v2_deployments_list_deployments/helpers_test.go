package handler_test

import (
	"net/http"

	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_list_deployments"
)

func newRoute(h *testutil.Harness) *handler.Handler {
	return &handler.Handler{
		DB: h.DB,
	}
}

func authHeaders(rootKey string) http.Header {
	return http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {"Bearer " + rootKey},
	}
}

// rid wraps a value in a pointer of the request's resource-identifier type.
func rid(s string) *openapi.ResourceIdentifier {
	r := openapi.ResourceIdentifier(s)
	return &r
}
