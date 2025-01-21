package v1RatelimitLimit

import (
	"github.com/unkeyed/unkey/go/pkg/api"
	"github.com/unkeyed/unkey/go/pkg/api/openapi"
	"github.com/unkeyed/unkey/go/pkg/api/routes"
	"github.com/unkeyed/unkey/go/pkg/api/session"
)

type Request = openapi.V2RatelimitLimitRequestBody
type Response = openapi.V2RatelimitLimitResponseBody

func New(svc *api.Services) routes.Route[Request, Response] {
	return routes.NewRoute("POST", "/v2/ratelimit.limit", func(s session.Session[Request, Response]) error {
		return s.Send(200, []byte("OK"))
	})
}
