package v2RatelimitLimit

import (
	"github.com/unkeyed/unkey/go/pkg/database/gen"
	"github.com/unkeyed/unkey/go/pkg/fault"
	zen "github.com/unkeyed/unkey/go/pkg/zen"
	"github.com/unkeyed/unkey/go/pkg/zen/openapi"
)

type Request = openapi.V2RatelimitSetOverrideRequestBody
type Response = openapi.V2RatelimitSetOverrideResponseBody

func New(svc *zen.Services) zen.Route {
	return zen.NewRoute("POST", "/v2/ratelimit.setOverride", func(s *zen.Session) error {

		err := svc.Database.InsertOverride(s.Context(), gen.InsertOverrideParams{})
		if err != nil {
			return fault.Wrap(err,
				fault.WithTag(zen.DatabaseError),
				fault.With("database failed", "The database is unavailable."),
			)
		}
		return s.JSON(200, Response{})

	})
}
