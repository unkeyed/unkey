package handler_test

import (
	restateingress "github.com/restatedev/sdk-go/ingress"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deploy_create_deployment"
)

func newRoute(h *testutil.Harness, restate *restateingress.Client) *handler.Handler {
	return &handler.Handler{
		DB:      h.DB,
		Restate: restate,
	}
}
