package handler

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/internal/services/auditlogs"
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
}

func (h *Handler) Method() string { return http.MethodPost }
func (h *Handler) Path() string   { return "/v2/workspace.updateFlag" }
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}
	workspaceID := principal.AuthorizedWorkspaceID
	if err := principal.Authorize(rbac.U(urn.New().Workspace(workspaceID).Flags(), permissions.Write)); err != nil {
		return err
	}
	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}
	data, err := db.TxWithResultRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) (openapi.V2WorkspaceUpdateFlagResponseData, error) {
		flag, err := db.Query.FindFlagBySlug(ctx, tx, req.Slug)
		if db.IsNotFound(err) {
			return openapi.V2WorkspaceUpdateFlagResponseData{}, fault.New("flag not found",
				fault.Code(codes.Data.Flag.NotFound.URN()),
				fault.Public("The requested flag definition does not exist."))
		}
		if err != nil {
			return openapi.V2WorkspaceUpdateFlagResponseData{}, fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Public("Failed to retrieve the flag definition."))
		}
		if (req.Value && !flag.AllowEnable) || (!req.Value && !flag.AllowDisable) {
			return openapi.V2WorkspaceUpdateFlagResponseData{}, fault.New("flag enrollment direction forbidden",
				fault.Code(codes.Auth.Authorization.Forbidden.URN()),
				fault.Public("The flag's enrollment policy does not permit this value."))
		}
		if err := db.Query.UpsertWorkspaceFlagOverride(ctx, tx, db.UpsertWorkspaceFlagOverrideParams{
			WorkspaceID: workspaceID, FlagSlug: req.Slug, Value: req.Value,
		}); err != nil {
			return openapi.V2WorkspaceUpdateFlagResponseData{}, fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Public("Failed to update the flag."))
		}
		if err := h.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{{
			WorkspaceID:   workspaceID,
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
				ID:          workspaceID,
				Name:        workspaceID,
				DisplayName: workspaceID,
				Meta:        map[string]any{"slug": req.Slug, "value": req.Value},
			}},
		}}); err != nil {
			return openapi.V2WorkspaceUpdateFlagResponseData{}, err
		}
		return openapi.V2WorkspaceUpdateFlagResponseData{
			Slug: flag.Slug, Value: req.Value, Description: flag.Description,
			AllowEnable: flag.AllowEnable, AllowDisable: flag.AllowDisable,
		}, nil
	})
	if err != nil {
		return err
	}
	return s.JSON(http.StatusOK, Response{Meta: openapi.Meta{RequestId: s.RequestID()}, Data: data})
}
