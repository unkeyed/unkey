package handler

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/internal/services/flags"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type Request = openapi.V2WorkspaceUpdateFlagRequestBody
type Response = openapi.V2WorkspaceUpdateFlagResponseBody

type Handler struct {
	DB        db.Database
	Auditlogs auditlogs.AuditLogService
	Flags     flags.Service
}

func (h *Handler) Method() string { return http.MethodPost }
func (h *Handler) Path() string   { return "/v2/workspace.updateFlag" }
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}
	if err := principal.Authorize(rbac.U(urn.New().Workspace(principal.AuthorizedWorkspaceID).Flags(), permissions.Write)); err != nil {
		return err
	}
	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}
	err = db.TxRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		flag, err := db.Query.FindFlagBySlug(ctx, tx, db.FindFlagBySlugParams{
			WorkspaceID: principal.AuthorizedWorkspaceID,
			Slug:        req.Slug,
		})
		if db.IsNotFound(err) {
			return fault.New("flag not found",
				fault.Code(codes.Data.Flag.NotFound.URN()),
				fault.Public("The requested flag definition does not exist."))
		}
		if err != nil {
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Public("Failed to retrieve the flag definition."))
		}
		if (req.Enabled && !flag.AllowOptIn) || (!req.Enabled && !flag.AllowOptOut) {
			return fault.New("flag enrollment direction forbidden",
				fault.Code(codes.Auth.Authorization.Forbidden.URN()),
				fault.Public("The flag's enrollment policy does not permit this value."))
		}
		before := flag.DefaultEnabled
		if flag.OverrideEnabled.Valid {
			before = flag.OverrideEnabled.Bool
		}
		if err := db.Query.UpsertWorkspaceFlagOverride(ctx, tx, db.UpsertWorkspaceFlagOverrideParams{
			WorkspaceID: principal.AuthorizedWorkspaceID,
			FlagSlug:    req.Slug,
			Enabled:     req.Enabled,
		}); err != nil {
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Public("Failed to update the flag."))
		}
		return h.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{{
			WorkspaceID:   principal.AuthorizedWorkspaceID,
			Event:         auditlog.WorkspaceUpdateEvent,
			ActorType:     auditlog.AuditLogActor(principal.Subject.Type),
			ActorID:       principal.Subject.ID,
			ActorName:     principal.Subject.Name,
			ActorMeta:     map[string]any{},
			Display:       "Updated flag " + req.Slug,
			RemoteIP:      s.Location(),
			UserAgent:     s.UserAgent(),
			CorrelationID: "",
			Targets: []auditlog.AuditLogTarget{{
				Type:        auditlog.WorkspaceTargetType,
				ID:          principal.AuthorizedWorkspaceID,
				Name:        principal.AuthorizedWorkspaceID,
				DisplayName: principal.AuthorizedWorkspaceID,
				Meta:        map[string]any{"slug": req.Slug, "before": before, "after": req.Enabled},
			}},
		}})
	})
	if err != nil {
		return err
	}
	h.Flags.Invalidate(ctx, principal.AuthorizedWorkspaceID)
	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: openapi.EmptyResponse{},
	})
}
