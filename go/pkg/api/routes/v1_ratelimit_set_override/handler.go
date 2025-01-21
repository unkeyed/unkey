package v1RatelimitLimit

import (
	"github.com/unkeyed/unkey/go/pkg/api"
	"github.com/unkeyed/unkey/go/pkg/api/openapi"
	"github.com/unkeyed/unkey/go/pkg/api/routes"
	"github.com/unkeyed/unkey/go/pkg/api/session"
	"github.com/unkeyed/unkey/go/pkg/database/gen"
)

type Request = api.Request[openapi.V2RatelimitSetOverrideRequestBody]
type Response = api.Request[openapi.V2RatelimitSetOverrideResponseBody]

func New(svc *api.Services) routes.Route[Request, Response] {
	return routes.NewRoute("POST", "/v2/ratelimit.setOverride", func(s session.Session[Request, Response]) error {

		svc.Database.InsertOverride(s.Context(), gen.InsertOverrideParams{})
		return s.Send(200, []byte("OK"))

	})
}
