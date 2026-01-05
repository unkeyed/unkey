package openapi

import (
	"net/http"

	"github.com/unkeyed/unkey/svc/agent/pkg/api/routes"
	"github.com/unkeyed/unkey/svc/agent/pkg/openapi"
)

func New(svc routes.Services) *routes.Route {
	return routes.NewRoute("GET", "/openapi.json",
		func(w http.ResponseWriter, r *http.Request) {

			w.WriteHeader(200)
			w.Header().Set("Content-Type", "application/json")
			_, err := w.Write(openapi.Spec)
			if err != nil {
				http.Error(w, "failed to write response", http.StatusInternalServerError)
			}
		},
	)
}
