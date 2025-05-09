package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/internal/services/auditlogs"
	"github.com/unkeyed/unkey/go/internal/services/caches"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/internal/services/permissions"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Request = openapi.V2ApisDeleteApiRequestBody
type Response = openapi.V2ApisDeleteApiResponseBody

type Services struct {
	Logger      logging.Logger
	DB          db.Database
	Keys        keys.KeyService
	Permissions permissions.PermissionService
	Auditlogs   auditlogs.AuditLogService
	Caches      caches.Caches
}

func New(svc Services) zen.Route {
	return zen.NewRoute("POST", "/v2/apis.deleteApi", func(ctx context.Context, s *zen.Session) error {
		svc.Logger.Debug("handling request", "requestId", s.RequestID(), "path", "/v2/apis.deleteApi")

		// 1. Authentication
		auth, err := svc.Keys.VerifyRootKey(ctx, s)
		if err != nil {
			return err
		}

		// 2. Request validation
		var req Request
		err = s.BindBody(&req)
		if err != nil {
			return fault.Wrap(err,
				fault.WithDesc("invalid request body", "The request body is invalid."),
			)
		}

		// 3. Permission check
		permissionCheck, err := svc.Permissions.Check(
			ctx,
			auth.KeyID,
			rbac.Or(
				rbac.T(rbac.Tuple{
					ResourceType: rbac.Api,
					ResourceID:   "*",
					Action:       rbac.DeleteAPI,
				}),
				rbac.T(rbac.Tuple{
					ResourceType: rbac.Api,
					ResourceID:   req.ApiId,
					Action:       rbac.DeleteAPI,
				}),
			),
		)
		if err != nil {
			return fault.Wrap(err,
				fault.WithDesc("unable to check permissions", "We're unable to check the permissions of your key."),
			)
		}

		if !permissionCheck.Valid {
			return fault.New("insufficient permissions",
				fault.WithCode(codes.Auth.Authorization.InsufficientPermissions.URN()),
				fault.WithDesc(permissionCheck.Message, permissionCheck.Message),
			)
		}

		// 4. Get API from database (don't use cache for deletion operations)
		api, err := db.Query.FindApiById(ctx, svc.DB.RO(), req.ApiId)
		if err != nil {
			if db.IsNotFound(err) {
				return fault.New("api not found",
					fault.WithCode(codes.Data.Api.NotFound.URN()),
					fault.WithDesc("api not found", "The requested API does not exist or has been deleted."),
				)
			}
			return fault.Wrap(err,
				fault.WithCode(codes.App.Internal.ServiceUnavailable.URN()),
				fault.WithDesc("database error", "Failed to retrieve API information."),
			)
		}

		// Check if API is deleted
		if api.DeletedAtM.Valid {
			return fault.New("api not found",
				fault.WithCode(codes.Data.Api.NotFound.URN()),
				fault.WithDesc("api not found", "The requested API does not exist or has been deleted."),
			)
		}

		// Check if API belongs to the authorized workspace
		if api.WorkspaceID != auth.AuthorizedWorkspaceID {
			return fault.New("wrong workspace",
				fault.WithCode(codes.Data.Api.NotFound.URN()),
				fault.WithDesc("wrong workspace, masking as 404", "The requested API does not exist or has been deleted."),
			)
		}

		// 5. Check delete protection
		if api.DeleteProtection {
			return fault.New("delete protected",
				fault.WithCode(codes.RateLimit.TooManyRequests.URN()),
				fault.WithDesc("api is protected from deletion", "This API has delete protection enabled. Disable it before attempting to delete."),
			)
		}

		now := time.Now()

		// 6. Execute deletion in a transaction
		err = svc.DB.WithTransaction(ctx, func(ctx context.Context, tx db.Tx) error {
			// Soft delete the API
			_, err := db.Query.UpdateApiSetDeletedAtM(ctx, tx, req.ApiId, db.NewNullTime(now))
			if err != nil {
				return fault.Wrap(err,
					fault.WithCode(codes.App.Internal.ServiceUnavailable.URN()),
					fault.WithDesc("database error", "Failed to delete API."),
				)
			}

			// Create audit log for API deletion
			err = svc.Auditlogs.CreateAuditLog(ctx, tx, auditlogs.CreateAuditLogParams{
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Event:       "api.delete",
				ActorType:   "key",
				ActorID:     auth.KeyID,
				Description: "Deleted " + req.ApiId,
				Resources: []auditlogs.Resource{
					{
						Type: "api",
						ID:   req.ApiId,
					},
				},
				Context: auditlogs.Context{
					Location:  s.Location(),
					UserAgent: s.UserAgent(),
				},
			})
			if err != nil {
				return fault.Wrap(err,
					fault.WithCode(codes.App.Internal.ServiceUnavailable.URN()),
					fault.WithDesc("audit log error", "Failed to create audit log for API deletion."),
				)
			}

			// If API has keyAuth, delete associated keys
			if api.KeyAuthID.Valid {
				// Find all keys for this keyAuth
				keys, err := db.Query.FindKeysByKeyAuthId(ctx, tx, api.KeyAuthID.String)
				if err != nil {
					return fault.Wrap(err,
						fault.WithCode(codes.App.Internal.ServiceUnavailable.URN()),
						fault.WithDesc("database error", "Failed to retrieve keys for deletion."),
					)
				}

				// Soft delete all keys
				_, err = db.Query.DeleteKeysByKeyAuthId(ctx, tx, api.KeyAuthID.String, db.NewNullTime(now))
				if err != nil {
					return fault.Wrap(err,
						fault.WithCode(codes.App.Internal.ServiceUnavailable.URN()),
						fault.WithDesc("database error", "Failed to delete keys."),
					)
				}

				// Create audit logs for each key deletion
				for _, key := range keys {
					err = svc.Auditlogs.CreateAuditLog(ctx, tx, auditlogs.CreateAuditLogParams{
						WorkspaceID: auth.AuthorizedWorkspaceID,
						Event:       "key.delete",
						ActorType:   "key",
						ActorID:     auth.KeyID,
						Description: "Deleted " + key.ID + " as part of " + req.ApiId + " deletion",
						Resources: []auditlogs.Resource{
							{
								Type: "keyAuth",
								ID:   api.KeyAuthID.String,
							},
							{
								Type: "key",
								ID:   key.ID,
							},
						},
						Context: auditlogs.Context{
							Location:  s.Location(),
							UserAgent: s.UserAgent(),
						},
					})
					if err != nil {
						return fault.Wrap(err,
							fault.WithCode(codes.App.Internal.ServiceUnavailable.URN()),
							fault.WithDesc("audit log error", "Failed to create audit log for key deletion."),
						)
					}
				}
			}

			return nil
		})
		if err != nil {
			return err
		}

		// 7. Clear caches
		s.ExecutionCtx().WaitUntil(svc.Caches.ApiById.Delete(ctx, req.ApiId))

		return s.JSON(http.StatusOK, Response{
			Meta: openapi.Meta{
				RequestId: s.RequestID(),
			},
		})
	})
}
