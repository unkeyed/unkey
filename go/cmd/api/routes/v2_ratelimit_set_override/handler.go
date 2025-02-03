package handler

import (
	"net/http"
	"time"

	"github.com/unkeyed/unkey/go/api"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/pkg/database"
	"github.com/unkeyed/unkey/go/pkg/entities"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/hash"
	"github.com/unkeyed/unkey/go/pkg/uid"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Request = api.V2RatelimitSetOverrideRequestBody
type Response = api.V2RatelimitSetOverrideResponseBody

type Services struct {
	DB   database.Database
	Keys keys.KeyService
}

func New(svc Services) zen.Route {
	return zen.NewRoute("POST", "/v2/ratelimit.setOverride", func(s *zen.Session) error {

		rootKey, err := zen.Bearer(s)
		if err != nil {
			return err
		}

		auth, err := svc.Keys.Verify(s.Context(), hash.Sha256(rootKey))
		if err != nil {
			return err
		}

		// nolint:exhaustruct
		req := Request{}
		err = s.BindBody(&req)
		if err != nil {
			return fault.Wrap(err,
				fault.WithTag(fault.INTERNAL_SERVER_ERROR),
				fault.WithDesc("invalid request body", "The request body is invalid."),
			)
		}

		overrideID := uid.New(uid.RatelimitOverridePrefix)
		err = svc.DB.InsertRatelimitOverride(s.Context(), entities.RatelimitOverride{
			ID:          overrideID,
			WorkspaceID: auth.AuthorizedWorkspaceID,
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
