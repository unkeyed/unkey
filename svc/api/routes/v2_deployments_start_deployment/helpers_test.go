package handler_test

import (
	"net/http"

	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_start_deployment"
)

func newRoute(h *testutil.Harness, mock *testutil.MockDeploymentClient) *handler.Handler {
	return &handler.Handler{
		DB:         h.DB,
		CtrlClient: mock,
	}
}

func authHeaders(rootKey string) http.Header {
	return http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {"Bearer " + rootKey},
	}
}
