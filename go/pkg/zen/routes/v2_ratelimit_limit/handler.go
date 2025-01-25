package v2RatelimitLimit

import (
	"net/http"

	openapi "github.com/unkeyed/unkey/go/api"
	"github.com/unkeyed/unkey/go/pkg/fault"
	zen "github.com/unkeyed/unkey/go/pkg/zen"
)

type Request = openapi.V2RatelimitLimitRequestBody
type Response = openapi.V2RatelimitLimitResponseBody

func New(svc *zen.Services) zen.Route {
	return zen.NewRoute("POST", "/v2/ratelimit.limit", func(s *zen.Session) error {
		req := new(Request)
		err := s.BindBody(req)
		if err != nil {
			return fault.Wrap(err,
				fault.WithDesc("binding failed", "We're unable to parse the request body as json."),
			)
		}

		// do stuff

		res := Response{
			Limit:     -1,
			Remaining: -1,
			Reset:     -1,
			Success:   true,
		}
		return s.JSON(http.StatusOK, res)
	})
}
