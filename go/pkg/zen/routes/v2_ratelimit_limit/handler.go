package v2RatelimitLimit

import (
	"github.com/unkeyed/unkey/go/pkg/fault"
	zen "github.com/unkeyed/unkey/go/pkg/zen"
	"github.com/unkeyed/unkey/go/pkg/zen/openapi"
)

type Request = openapi.V2RatelimitLimitRequestBody
type Response = openapi.V2RatelimitLimitResponseBody

func New(svc *zen.Services) zen.Route {
	return zen.NewRoute("POST", "/v2/ratelimit.limit", func(s *zen.Session) error {

		req := Request{}
		err := s.BindBody(&req)
		if err != nil {
			return fault.Wrap(err,
				fault.With("binding failed", "We're unable to parse the request body as json."),
			)
		}

		// do stuff

		res := Response{
			// ...
		}
		return s.JSON(200, res)
	})
}
