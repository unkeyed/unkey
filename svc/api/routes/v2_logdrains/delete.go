package logdrains

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

type Delete struct {
	DB        db.Database
	Auditlogs auditlogs.AuditLogService
}

func (h *Delete) Method() string { return http.MethodPost }
func (h *Delete) Path() string   { return "/v2/logdrains.deleteLogdrain" }
func (h *Delete) Handle(ctx context.Context, s *zen.Session) error {
	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}
	req, err := zen.BindBody[openapi.LogdrainIdRequest](s)
	if err != nil {
		return err
	}
	if err := principal.Authorize(rbac.U(urn.V1{WorkspaceID: principal.AuthorizedWorkspaceID, Resource: "logdrains/" + req.LogdrainId}, permissions.Delete)); err != nil {
		return err
	}
	err = db.TxRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		row, err := db.Query.LockLogdrain(ctx, tx, db.LockLogdrainParams{WorkspaceID: principal.AuthorizedWorkspaceID, ID: req.LogdrainId})
		if err != nil {
			if db.IsNotFound(err) {
				return fault.Wrap(err, fault.Code(codes.Data.Logdrain.NotFound.URN()), fault.Public("Log drain not found."))
			}
			return err
		}
		if err := db.Query.DeleteLogdrain(ctx, tx, db.DeleteLogdrainParams{WorkspaceID: principal.AuthorizedWorkspaceID, ID: row.ID}); err != nil {
			return err
		}
		return h.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{{
			WorkspaceID: principal.AuthorizedWorkspaceID, Event: "logdrain.delete", Display: "Deleted log drain " + row.ID,
			ActorID: principal.Subject.ID, ActorType: auditlog.AuditLogActor(principal.Subject.Type), ActorName: principal.Subject.Name,
			ActorMeta: map[string]any{}, CorrelationID: "", RemoteIP: s.Location(), UserAgent: s.UserAgent(),
			Resources: []auditlog.AuditLogResource{{ID: row.ID, Type: "logdrain", Name: row.Name, DisplayName: "", Meta: map[string]any{}}},
		}})
	})
	if err != nil {
		return err
	}
	var response openapi.LogdrainMutationResponse
	response.Meta.RequestId = s.RequestID()
	response.Data.Id = req.LogdrainId
	return s.JSON(http.StatusOK, response)
}
