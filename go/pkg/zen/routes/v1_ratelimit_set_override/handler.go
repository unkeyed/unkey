package v1RatelimitLimit

import (
	"github.com/unkeyed/unkey/go/pkg/database/gen"
	zen "github.com/unkeyed/unkey/go/pkg/zen"
	apierrors "github.com/unkeyed/unkey/go/pkg/zen/errors"
	"github.com/unkeyed/unkey/go/pkg/zen/openapi"
)

type Request = openapi.V2RatelimitSetOverrideRequestBody
type Response = openapi.V2RatelimitSetOverrideResponseBody

func New(svc *zen.Services) zen.Route {
	return zen.NewRoute("POST", "/v2/ratelimit.setOverride", func(s *zen.Session) error {

		err := svc.Database.InsertOverride(s.Context(), gen.InsertOverrideParams{})
		if err != nil {
			return s.Error(apierrors.NewInternalServerError("unable to insert override", "detail"))
		}
		return s.JSON(200, Response{})

	})
}
