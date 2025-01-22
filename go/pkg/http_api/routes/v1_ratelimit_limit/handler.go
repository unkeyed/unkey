package v1RatelimitLimit

import (
	"github.com/unkeyed/unkey/go/pkg/http_Api/openapi"
	httpApi "github.com/unkeyed/unkey/go/pkg/http_api"
)

type Request = openapi.V2RatelimitLimitRequestBody
type Response = openapi.V2RatelimitLimitResponseBody

func New(svc *httpApi.Services) httpApi.Route[Request, Response] {
	return httpApi.NewRoute("POST", "/v2/ratelimit.limit", func(s *httpApi.Session[Request, Response]) error {
		return s.JSON(200, Response{})
	})
}
