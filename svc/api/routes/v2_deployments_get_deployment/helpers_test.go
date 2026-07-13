package handler_test

import (
	"net/http"

	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_get_deployment"
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
