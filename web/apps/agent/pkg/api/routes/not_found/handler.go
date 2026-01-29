package notFound

import (
	"net/http"

	"github.com/unkeyed/unkey/svc/agent/pkg/api/ctxutil"
	"github.com/unkeyed/unkey/svc/agent/pkg/api/routes"
	"github.com/unkeyed/unkey/svc/agent/pkg/openapi"
)

// This is a hack, because / matches everything, so we need to make sure this is the last route
func New(svc routes.Services) *routes.Route {
	return routes.NewRoute("", "/",
		func(w http.ResponseWriter, r *http.Request) {

			svc.Sender.Send(r.Context(), w, 200, openapi.BaseError{
				Title:     "Not Found",
				Detail:    "This route does not exist",
				Instance:  "https://errors.unkey.com/todo",
				Status:    http.StatusNotFound,
				RequestId: ctxutil.GetRequestID(r.Context()),
				Type:      "TODO docs link",
			})
		},
	)
}
