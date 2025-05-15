package handler

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/internal/services/auditlogs"
	"github.com/unkeyed/unkey/go/internal/services/caches"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/internal/services/permissions"
	"github.com/unkeyed/unkey/go/pkg/auditlog"
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

		auth, err := svc.Keys.VerifyRootKey(ctx, s)
		if err != nil {
			return err
		}

		var req Request
		err = s.BindBody(&req)
		if err != nil {
			return fault.Wrap(err,
				fault.WithDesc("invalid request body", "The request body is invalid."),
			)
		}

		err = svc.Permissions.Check(
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
			return err
		}

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

		// Check if API belongs to the authorized workspace
		if api.WorkspaceID != auth.AuthorizedWorkspaceID {
			return fault.New("wrong workspace",
				fault.WithCode(codes.Data.Api.NotFound.URN()),
				fault.WithDesc("wrong workspace, masking as 404", "The requested API does not exist or has been deleted."),
			)
		}

		// Check if API is deleted
		if api.DeletedAtM.Valid {
			return fault.New("api not found",
				fault.WithCode(codes.Data.Api.NotFound.URN()),
				fault.WithDesc("api not found", "The requested API does not exist or has been deleted."),
			)
		}

		// 5. Check delete protection
		if api.DeleteProtection.Valid && api.DeleteProtection.Bool {
			return fault.New("delete protected",
				fault.WithCode(codes.App.Protection.ProtectedResource.URN()),
				fault.WithDesc("api is protected from deletion", "This API has delete protection enabled. Disable it before attempting to delete."),
			)
		}

		now := time.Now()

		tx, err := svc.DB.RW().Begin(ctx)
		if err != nil {
			return fault.Wrap(err,
				fault.WithCode(codes.App.Internal.ServiceUnavailable.URN()),
				fault.WithDesc("database failed to create transaction", "Unable to start database transaction."),
			)
		}

		defer func() {
			rollbackErr := tx.Rollback()
			if rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
				svc.Logger.Error("rollback failed", "requestId", s.RequestID(), "error", rollbackErr)
			}
		}()

		// Soft delete the API
		err = db.Query.SoftDeleteApi(ctx, tx, db.SoftDeleteApiParams{
			ApiID: req.ApiId,
			Now:   sql.NullInt64{Valid: true, Int64: now.UnixMilli()},
		})
		if err != nil {
			return fault.Wrap(err,
				fault.WithCode(codes.App.Internal.ServiceUnavailable.URN()),
				fault.WithDesc("database error", "Failed to delete API."),
			)
		}

		// Create audit log for API deletion
		err = svc.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{{
			WorkspaceID: auth.AuthorizedWorkspaceID,
			Event:       auditlog.APIDeleteEvent,
			ActorType:   auditlog.RootKeyActor,
			ActorID:     auth.KeyID,
			Display:     fmt.Sprintf("Deleted API %s", req.ApiId),
			Resources: []auditlog.AuditLogResource{
				{
					Type: auditlog.APIResourceType,
					ID:   req.ApiId,
				},
			},
			RemoteIP:  s.Location(),
			UserAgent: s.UserAgent(),
		}})
		if err != nil {
			return fault.Wrap(err,
				fault.WithCode(codes.App.Internal.ServiceUnavailable.URN()),
				fault.WithDesc("audit log error", "Failed to create audit log for API deletion."),
			)
		}

		err = tx.Commit()
		if err != nil {
			return fault.Wrap(err,
				fault.WithCode(codes.App.Internal.ServiceUnavailable.URN()),
				fault.WithDesc("database failed to commit transaction", "Failed to commit changes."),
			)
		}

		svc.Caches.ApiByID.SetNull(ctx, req.ApiId)

		return s.JSON(http.StatusOK, Response{
			Meta: openapi.Meta{
				RequestId: s.RequestID(),
			},
		})
	})
}
