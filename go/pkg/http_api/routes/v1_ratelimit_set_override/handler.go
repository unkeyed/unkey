package v1RatelimitLimit

import (
	"github.com/unkeyed/unkey/go/pkg/database/gen"
	httpApi "github.com/unkeyed/unkey/go/pkg/http_api"
	apierrors "github.com/unkeyed/unkey/go/pkg/http_api/errors"
	"github.com/unkeyed/unkey/go/pkg/http_api/openapi"
	"github.com/unkeyed/unkey/go/pkg/http_api/routes"
)

type Request = openapi.V2RatelimitSetOverrideRequestBody
type Response = openapi.V2RatelimitSetOverrideResponseBody

func New(svc *httpApi.Services) routes.Route[Request, Response] {
	return routes.NewRoute("POST", "/v2/ratelimit.setOverride", func(s *httpApi.Session[Request, Response]) error {

		err := svc.Database.InsertOverride(s.Context(), gen.InsertOverrideParams{})
		if err != nil {
			return s.Error(apierrors.NewInternalServerError("unable to insert override", "detail"))
		}
		return s.JSON(200, Response{})

	})
}
