package v1Liveness

import (
	"net/http"

	"github.com/unkeyed/unkey/apps/agent/gen/openapi"
	"github.com/unkeyed/unkey/apps/agent/pkg/api/routes"
)

func New(svc routes.Services) *routes.Route {
	return routes.NewRoute("GET", "/v1/liveness",
		func(w http.ResponseWriter, r *http.Request) {

			svc.Logger.Debug().Msg("incoming liveness check")

			svc.Sender.Send(r.Context(), w, 200, openapi.V1LivenessResponseBody{
				Message: "OK",
			})
		},
	)
}
