package handler

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/internal/services/permissions"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Request = openapi.V2RatelimitGetOverrideRequestBody
type Response = openapi.V2RatelimitGetOverrideResponseBody

type Services struct {
	Logger      logging.Logger
	DB          db.Database
	Keys        keys.KeyService
	Permissions permissions.PermissionService
}

func New(svc Services) zen.Route {
	return zen.NewRoute("POST", "/v2/ratelimit.getOverride", func(ctx context.Context, s *zen.Session) error {
		auth, err := svc.Keys.VerifyRootKey(ctx, s)
		if err != nil {
			return err
		}

		// nolint:exhaustruct
		req := Request{}
		err = s.BindBody(&req)
		if err != nil {
			return fault.Wrap(err,
				fault.WithDesc("invalid request body", "The request body is invalid."),
			)
		}

		namespace, err := getNamespace(ctx, svc, auth.AuthorizedWorkspaceID, req)

		if err != nil {
			// already handled correctly in getNamespace
			return err
		}

		if namespace.WorkspaceID != auth.AuthorizedWorkspaceID {
			return fault.New("namespace not found",
				fault.WithCode(codes.Data.RatelimitNamespace.NotFound.URN()),
				fault.WithDesc("namespace was deleted", "This namespace does not exist."),
			)
		}

		permissions, err := svc.Permissions.Check(
			ctx,
			auth.KeyID,
			rbac.Or(
				rbac.T(rbac.Tuple{
					ResourceType: rbac.Ratelimit,
					ResourceID:   namespace.ID,
					Action:       rbac.ReadOverride,
				}),
				rbac.T(rbac.Tuple{
					ResourceType: rbac.Ratelimit,
					ResourceID:   "*",
					Action:       rbac.ReadOverride,
				}),
			),
		)
		if err != nil {
			return fault.Wrap(err,
				fault.WithDesc("unable to check permissions", "We're unable to check the permissions of your key."),
			)
		}

		if !permissions.Valid {
			return fault.New("insufficient permissions",
				fault.WithCode(codes.Auth.Authorization.InsufficientPermissions.URN()),
				fault.WithDesc(permissions.Message, permissions.Message),
			)
		}

		override, err := db.Query.FindRatelimitOverridesByIdentifier(ctx, svc.DB.RO(), db.FindRatelimitOverridesByIdentifierParams{
			WorkspaceID: auth.AuthorizedWorkspaceID,
			NamespaceID: namespace.ID,
			Identifier:  req.Identifier,
		})

		if db.IsNotFound(err) {
			return fault.New("override not found",
				fault.WithCode(codes.Data.RatelimitOverride.NotFound.URN()),
				fault.WithDesc("override not found", "This override does not exist."),
			)
		}
		if err != nil {
			return fault.Wrap(err,
				fault.WithCode(codes.App.Internal.ServiceUnavailable.URN()),
				fault.WithDesc("database failed to find the override", "Error finding the ratelimit override."),
			)
		}

		return s.JSON(http.StatusOK, Response{
			Meta: openapi.Meta{
				RequestId: s.RequestID(),
			},
			Data: openapi.RatelimitOverride{

				OverrideId:  override.ID,
				Duration:    int64(override.Duration),
				Identifier:  override.Identifier,
				NamespaceId: override.NamespaceID,
				Limit:       int64(override.Limit),
			},
		})
	})
}

func getNamespace(ctx context.Context, svc Services, workspaceID string, req Request) (namespace db.RatelimitNamespace, err error) {

	switch {
	case req.NamespaceId != nil:
		{
			namespace, err = db.Query.FindRatelimitNamespaceByID(ctx, svc.DB.RO(), *req.NamespaceId)
			break
		}
	case req.NamespaceName != nil:
		{
			namespace, err = db.Query.FindRatelimitNamespaceByName(ctx, svc.DB.RO(), db.FindRatelimitNamespaceByNameParams{
				WorkspaceID: workspaceID,
				Name:        *req.NamespaceName,
			})
			break
		}
	default:
		return db.RatelimitNamespace{}, fault.New("namespace id or name required",
			fault.WithCode(codes.App.Validation.InvalidInput.URN()),
			fault.WithDesc("namespace id or name required", "You must provide either a namespace ID or name."),
		)
	}

	if err != nil {

		if db.IsNotFound(err) {
			return db.RatelimitNamespace{}, fault.New("namespace not found",
				fault.WithCode(codes.Data.RatelimitNamespace.NotFound.URN()),
				fault.WithDesc("namespace not found", "The namespace was not found."),
			)
		}

		return db.RatelimitNamespace{}, fault.Wrap(err,
			fault.WithCode(codes.App.Internal.ServiceUnavailable.URN()),
			fault.WithDesc("database failed to find the namespace", "Error finding the ratelimit namespace."),
		)
	}
	return namespace, nil
}
