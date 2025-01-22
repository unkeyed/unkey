package v1RatelimitLimit

import (
	zen "github.com/unkeyed/unkey/go/pkg/zen"
	"github.com/unkeyed/unkey/go/pkg/zen/openapi"
)

type Request = openapi.V2RatelimitLimitRequestBody
type Response = openapi.V2RatelimitLimitResponseBody

func New(svc *zen.Services) zen.Route {
	return zen.NewRoute("POST", "/v2/ratelimit.limit", func(s *zen.Session) error {

		req := openapi.V2RatelimitLimitRequestBody{}
		err := s.BindBody(&req)
		if err != nil {
			return err
		}
		return s.JSON(200, Response{})
	})
}
