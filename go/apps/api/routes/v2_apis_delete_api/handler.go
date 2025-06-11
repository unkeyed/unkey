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

// Handler implements zen.Route interface for the v2 APIs delete API endpoint
type Handler struct {
	// Services as public fields
	Logger      logging.Logger
	DB          db.Database
	Keys        keys.KeyService
	Permissions permissions.PermissionService
	Auditlogs   auditlogs.AuditLogService
	Caches      caches.Caches
}

// Method returns the HTTP method this route responds to
func (h *Handler) Method() string {
	return "POST"
}

// Path returns the URL path pattern this route matches
func (h *Handler) Path() string {
	return "/v2/apis.deleteApi"
}

// Handle processes the HTTP request
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	auth, err := h.Keys.VerifyRootKey(ctx, s)
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

	err = h.Permissions.Check(
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

	api, err := db.Query.FindApiByID(ctx, h.DB.RO(), req.ApiId)
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

	// 5. Check delete protection
	if api.DeleteProtection.Valid && api.DeleteProtection.Bool {
		return fault.New("delete protected",
			fault.Code(codes.App.Protection.ProtectedResource.URN()),
			fault.Internal("api is protected from deletion"), fault.Public("This API has delete protection enabled. Disable it before attempting to delete."),
		)
	}

	now := time.Now()

	tx, err := h.DB.RW().Begin(ctx)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database failed to create transaction"), fault.Public("Unable to start database transaction."),
		)
	}

	defer func() {
		rollbackErr := tx.Rollback()
		if rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
			h.Logger.Error("rollback failed", "requestId", s.RequestID(), "error", rollbackErr)
		}
	}()

	// Soft delete the API
	err = db.Query.SoftDeleteApi(ctx, tx, db.SoftDeleteApiParams{
		ApiID: req.ApiId,
		Now:   sql.NullInt64{Valid: true, Int64: now.UnixMilli()},
	})
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"), fault.Public("Failed to delete API."),
		)
	}

	// Create audit log for API deletion
	err = h.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{{
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
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("audit log error"), fault.Public("Failed to create audit log for API deletion."),
		)
	}

	err = tx.Commit()
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database failed to commit transaction"), fault.Public("Failed to commit changes."),
		)
	}

	h.Caches.ApiByID.SetNull(ctx, req.ApiId)

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
	})
}
