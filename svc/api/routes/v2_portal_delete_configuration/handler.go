package handler

import (
	"context"
	"fmt"
	"net/http"

	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2PortalDeleteConfigurationRequestBody
	Response = openapi.V2PortalDeleteConfigurationResponseBody
)

// Handler implements zen.Route for deleting a portal configuration and its
// branding. It is an operator action authenticated by a root key and scoped to
// the root key's workspace, so a caller can never delete another workspace's
// configuration. The schema has no cascading delete, so branding is removed
// explicitly.
type Handler struct {
	DB        db.Database
	Auditlogs auditlogs.AuditLogService
}

func (h *Handler) Method() string { return "POST" }
func (h *Handler) Path() string   { return "/v2/portal.deleteConfiguration" }

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	workspaceID := principal.WorkspaceID

	// Verify the config exists and belongs to the caller's workspace so an
	// unknown or cross-workspace id yields a 404 rather than a silent no-op.
	existing, err := db.Query.FindPortalConfigByID(ctx, h.DB.RW(), db.FindPortalConfigByIDParams{
		ID:          req.ConfigId,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return fault.New("portal config not found",
				fault.Code(codes.Data.PortalConfig.NotFound.URN()),
				fault.Internal("no portal config found for the given id"),
				fault.Public("Portal configuration not found."),
			)
		}
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error looking up portal config"),
			fault.Public("Failed to look up portal configuration."),
		)
	}

	err = db.Tx(ctx, h.DB.RW(), func(txCtx context.Context, tx db.DBTX) error {
		if txErr := db.Query.DeletePortalBranding(txCtx, tx, req.ConfigId); txErr != nil {
			return fault.Wrap(txErr,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("failed to delete portal branding"),
				fault.Public("Failed to delete portal configuration."),
			)
		}

		if txErr := db.Query.DeletePortalConfig(txCtx, tx, db.DeletePortalConfigParams{
			ID:          req.ConfigId,
			WorkspaceID: workspaceID,
		}); txErr != nil {
			return fault.Wrap(txErr,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("failed to delete portal config"),
				fault.Public("Failed to delete portal configuration."),
			)
		}

		return h.Auditlogs.Insert(txCtx, tx, []auditlog.AuditLog{
			{
				Event:         auditlog.PortalConfigDeleteEvent,
				WorkspaceID:   workspaceID,
				ActorType:     auditlog.AuditLogActor(principal.Subject.Type),
				ActorID:       principal.Subject.ID,
				ActorName:     principal.Subject.Name,
				ActorMeta:     map[string]any{},
				Display:       fmt.Sprintf("Deleted portal configuration %s", existing.Slug),
				RemoteIP:      s.Location(),
				UserAgent:     s.UserAgent(),
				CorrelationID: "",
				Resources: []auditlog.AuditLogResource{
					{
						ID:          req.ConfigId,
						DisplayName: existing.Slug,
						Name:        existing.Slug,
						Meta:        map[string]any{"slug": existing.Slug},
						Type:        auditlog.PortalConfigResourceType,
					},
				},
			},
		})
	})
	if err != nil {
		return err
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: openapi.EmptyResponse{},
	})
}
