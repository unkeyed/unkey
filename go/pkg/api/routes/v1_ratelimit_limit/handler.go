package v1RatelimitLimit

import (
	"github.com/unkeyed/unkey/go/pkg/api"
	"github.com/unkeyed/unkey/go/pkg/api/openapi"
	"github.com/unkeyed/unkey/go/pkg/api/routes"
	"github.com/unkeyed/unkey/go/pkg/api/session"
)

type Request = api.Request[openapi.V1RatelimitRatelimitRequestBody]
type Response = api.Request[openapi.V1RatelimitRatelimitResponseBody]

func New(svc *api.Services) routes.Route[Request, Response] {
	return routes.NewRoute("POST", "/v1/ratelimit.limit", func(s session.Session[Request, Response]) error {
		return s.Send(200, []byte("OK"))
	})
}
