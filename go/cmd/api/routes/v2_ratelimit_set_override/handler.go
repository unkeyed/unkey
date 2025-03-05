package handler

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/go/api"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/internal/services/permissions"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/logging"
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/uid"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Request = api.V2RatelimitSetOverrideRequestBody
type Response = api.V2RatelimitSetOverrideResponseBody

type Services struct {
	Logger      logging.Logger
	DB          db.Database
	Keys        keys.KeyService
	Permissions permissions.PermissionService
}

func New(svc Services) zen.Route {
	return zen.NewRoute("POST", "/v2/ratelimit.setOverride", func(s *zen.Session) error {

		auth, err := svc.Keys.VerifyRootKey(s.Context(), s)
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

		namespace, err := getNamespace(s.Context(), svc, auth.AuthorizedWorkspaceID, req)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fault.Wrap(err,
					fault.WithTag(fault.NOT_FOUND),
					fault.WithDesc("namespace not found", "This namespace does not exist."),
				)
			}
			return err
		}

		if namespace.WorkspaceID != auth.AuthorizedWorkspaceID {
			return fault.New("namespace not found",
				fault.WithTag(fault.NOT_FOUND),
				fault.WithDesc("wrong workspace, masking as 404", "This namespace does not exist."),
			)
		}

		permissions, err := svc.Permissions.Check(
			s.Context(),
			auth.KeyID,
			rbac.Or(
				rbac.T(rbac.Tuple{
					ResourceType: rbac.Ratelimit,
					ResourceID:   namespace.ID,
					Action:       rbac.SetOverride,
				}),
				rbac.T(rbac.Tuple{
					ResourceType: rbac.Ratelimit,
					ResourceID:   "*",
					Action:       rbac.SetOverride,
				}),
			),
		)
		if err != nil {
			return fault.Wrap(err,
				fault.WithTag(fault.INTERNAL_SERVER_ERROR),
				fault.WithDesc("unable to check permissions", "We're unable to check the permissions of your key."),
			)
		}

		if !permissions.Valid {
			return fault.New("insufficient permissions",
				fault.WithTag(fault.INSUFFICIENT_PERMISSIONS),
				fault.WithDesc(permissions.Message, permissions.Message),
			)
		}

		overrideID := uid.New(uid.RatelimitOverridePrefix)
		err = db.Query.InsertRatelimitOverride(s.Context(), svc.DB.RO(), db.InsertRatelimitOverrideParams{
			ID:          overrideID,
			WorkspaceID: auth.AuthorizedWorkspaceID,
			NamespaceID: namespace.ID,
			Identifier:  req.Identifier,
			Limit:       int32(req.Limit),    // nolint:gosec
			Duration:    int32(req.Duration), //nolint:gosec
			CreatedAt:   time.Now().UnixMilli(),
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

func getNamespace(ctx context.Context, svc Services, workspaceID string, req Request) (db.RatelimitNamespace, error) {

	switch {
	case req.NamespaceId != nil:
		{
			return db.Query.FindRatelimitNamespaceByID(ctx, svc.DB.RO(), *req.NamespaceId)
		}
	case req.NamespaceName != nil:
		{
			return db.Query.FindRatelimitNamespaceByName(ctx, svc.DB.RO(), db.FindRatelimitNamespaceByNameParams{
				WorkspaceID: workspaceID,
				Name:        *req.NamespaceName,
			})
		}
	}

	return db.RatelimitNamespace{}, fault.New("missing namespace id or name",
		fault.WithTag(fault.BAD_REQUEST),
		fault.WithDesc("missing namespace id or name", "You must provide either a namespace ID or name."),
	)

}
