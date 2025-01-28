package handler

import (
	"net/http"
	"time"

	"github.com/unkeyed/unkey/go/api"
	"github.com/unkeyed/unkey/go/pkg/entities"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/uid"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Request = api.V2RatelimitSetOverrideRequestBody
type Response = api.V2RatelimitSetOverrideResponseBody

func New(svc *zen.Services) zen.Route {
	return zen.NewRoute("POST", "/v2/ratelimit.setOverride", func(s *zen.Session) error {

		// nolint:exhaustruct
		req := Request{}
		err := s.BindBody(&req)
		if err != nil {
			return fault.Wrap(err,
				fault.WithTag(fault.INTERNAL_SERVER_ERROR),
				fault.WithDesc("invalid request body", "The request body is invalid."),
			)
		}

		overrideID := uid.New(uid.RatelimitOverridePrefix)
		err = svc.Database.InsertRatelimitOverride(s.Context(), entities.RatelimitOverride{
			ID:          overrideID,
			WorkspaceID: "",
			NamespaceID: "",
			Identifier:  "",
			Limit:       0,
			Duration:    0,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Time{},
			DeletedAt:   time.Time{},
			Async:       false,
		})
		if err != nil {
			return fault.Wrap(err,
				fault.WithTag(fault.DATABASE_ERROR),
				fault.WithDesc("database failed", "The database is unavailable."),
			)
		}
		return s.JSON(http.StatusOK, Response{
			OverrideId: overrideID,
		})
	})
}
