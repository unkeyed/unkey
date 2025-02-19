package handler

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/go/api"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/pkg/database"
	"github.com/unkeyed/unkey/go/pkg/entities"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/logging"
	"github.com/unkeyed/unkey/go/pkg/uid"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Request = api.V2RatelimitSetOverrideRequestBody
type Response = api.V2RatelimitSetOverrideResponseBody

type Services struct {
	Logger logging.Logger
	DB     database.Database
	Keys   keys.KeyService
}

func New(svc Services) zen.Route {
	return zen.NewRoute("POST", "/v2/ratelimit.setOverride", func(s *zen.Session) error {

		auth, err := svc.Keys.VerifyRootKey(s.Context(), s)
		if err != nil {
			svc.Logger.Warn(s.Context(), "failed to verify root key", slog.String("error", err.Error()))
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

		namespace, err := getNamespace(s.Context(), svc, auth.AuthorizedWorkspaceID, req)
		if err != nil {
			svc.Logger.Warn(s.Context(), "failed to get namespace", slog.String("error", err.Error()))
			return err
		}

		overrideID := uid.New(uid.RatelimitOverridePrefix)
		err = svc.DB.InsertRatelimitOverride(s.Context(), entities.RatelimitOverride{
			ID:          overrideID,
			WorkspaceID: auth.AuthorizedWorkspaceID,
			NamespaceID: namespace.ID,
			Identifier:  req.Identifier,
			Limit:       int32(req.Limit), // nolint:gosec
			Duration:    time.Duration(req.Duration) * time.Millisecond,
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

func getNamespace(ctx context.Context, svc Services, workspaceID string, req Request) (entities.RatelimitNamespace, error) {

	switch {
	case req.NamespaceId != nil:
		{
			return svc.DB.FindRatelimitNamespaceByID(ctx, *req.NamespaceId)
		}
	case req.NamespaceName != nil:
		{
			return svc.DB.FindRatelimitNamespaceByName(ctx, workspaceID, *req.NamespaceName)
		}
	}

	return entities.RatelimitNamespace{}, fault.New("missing namespace id or name",
		fault.WithTag(fault.BAD_REQUEST),
		fault.WithDesc("missing namespace id or name", "You must provide either a namespace ID or name."),
	)

}
