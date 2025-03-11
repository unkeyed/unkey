package v2Liveness

import (
	"context"
	"net/http"

	openapi "github.com/unkeyed/unkey/go/api"
	zen "github.com/unkeyed/unkey/go/pkg/zen"
)

type Response = openapi.V2LivenessResponseBody

func New() zen.Route {
	return zen.NewRoute("GET", "/v2/liveness", func(ctx context.Context, s *zen.Session) error {

		res := Response{
			Message: "we're cooking",
		}
		return s.JSON(http.StatusOK, res)
	})
}
