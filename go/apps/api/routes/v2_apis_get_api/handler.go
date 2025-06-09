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

type Request = openapi.V2ApisGetApiRequestBody

type Response = openapi.V2ApisGetApiResponseBody

type Services struct {
	Logger      logging.Logger
	DB          db.Database
	Keys        keys.KeyService
	Permissions permissions.PermissionService
}

func New(svc Services) zen.Route {
	return zen.NewRoute("POST", "/v2/apis.getApi", func(ctx context.Context, s *zen.Session) error {
		svc.Logger.Debug("handling request", "requestId", s.RequestID(), "path", "/v2/apis.getApi")
		auth, err := svc.Keys.VerifyRootKey(ctx, s)
		if err != nil {
			return err
		}

		var req Request
		err = s.BindBody(&req)
		if err != nil {
			return fault.Wrap(err,
				fault.Internal("invalid request body"), fault.Public("The request body is invalid."),
			)
		}

		err = svc.Permissions.Check(
			ctx,
			auth.KeyID,
			rbac.Or(
				rbac.T(rbac.Tuple{
					ResourceType: rbac.Api,
					ResourceID:   "*",
					Action:       rbac.ReadAPI,
				}),
				rbac.T(rbac.Tuple{
					ResourceType: rbac.Api,
					ResourceID:   req.ApiId,
					Action:       rbac.ReadAPI,
				}),
			),
		)
		if err != nil {
			return err
		}

		// Get API from database
		api, err := db.Query.FindApiById(ctx, svc.DB.RO(), req.ApiId)
		if err != nil {
			if db.IsNotFound(err) {
				return fault.New("api not found",
					fault.Code(codes.Data.Api.NotFound.URN()),
					fault.Internal("api not found"), fault.Public("The requested API does not exist or has been deleted."),
				)
			}
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error"), fault.Public("Failed to retrieve API information."),
			)
		}
		// Check if API belongs to the authorized workspace
		if api.WorkspaceID != auth.AuthorizedWorkspaceID {
			return fault.New("wrong workspace",
				fault.Code(codes.Data.Api.NotFound.URN()),
				fault.Internal("wrong workspace, masking as 404"), fault.Public("The requested API does not exist or has been deleted."),
			)
		}

		// Check if API is deleted
		if api.DeletedAtM.Valid {
			return fault.New("api not found",
				fault.Code(codes.Data.Api.NotFound.URN()),
				fault.Internal("api not found"), fault.Public("The requested API does not exist or has been deleted."),
			)
		}

		return s.JSON(http.StatusOK, Response{
			Meta: openapi.Meta{
				RequestId: s.RequestID(),
			},
			Data: openapi.ApisGetApiResponseData{
				Id:   api.ID,
				Name: api.Name,
			},
		})
	})
}
