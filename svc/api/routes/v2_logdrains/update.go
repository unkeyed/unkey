package logdrains

import (
	"context"
	"database/sql"
	"net/http"
	"strings"

	logdrainv1 "github.com/unkeyed/unkey/gen/proto/logdrain/v1"
	"github.com/unkeyed/unkey/gen/rpc/vault"
	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
	"google.golang.org/protobuf/proto"
)

type Update struct {
	DB        db.Database
	Vault     vault.VaultServiceClient
	Auditlogs auditlogs.AuditLogService
	Clock     clock.Clock
}

func (h *Update) Method() string { return http.MethodPost }
func (h *Update) Path() string   { return "/v2/logdrains.updateLogdrain" }
func (h *Update) Handle(ctx context.Context, s *zen.Session) error {
	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}
	req, err := zen.BindBody[openapi.UpdateLogdrainRequest](s)
	if err != nil {
		return err
	}
	if err := principal.Authorize(rbac.U(urn.V1{WorkspaceID: principal.AuthorizedWorkspaceID, Resource: "logdrains/" + req.LogdrainId}, permissions.Write)); err != nil {
		return err
	}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return invalid("Name must not be empty.")
		}
		req.Name = &name
	}
	err = db.TxRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		row, err := db.Query.LockLogdrain(ctx, tx, db.LockLogdrainParams{WorkspaceID: principal.AuthorizedWorkspaceID, ID: req.LogdrainId})
		if err != nil {
			if db.IsNotFound(err) {
				return fault.Wrap(err, fault.Code(codes.Data.Logdrain.NotFound.URN()), fault.Public("Log drain not found."))
			}
			return err
		}
		changesDelivery := req.Destination != nil || req.BatchSize != nil || req.Filters != nil
		if changesDelivery {
			config := &logdrainv1.Config{}
			if err := proto.Unmarshal(row.Config, config); err != nil {
				return err
			}
			if req.BatchSize != nil {
				config.BatchSize = uint32(*req.BatchSize)
			}
			if req.Filters != nil {
				if err := setStream(config, string(row.Stream), *req.Filters); err != nil {
					return err
				}
			}
			if req.Destination != nil {
				if err := setDestination(ctx, h.Vault, principal.AuthorizedWorkspaceID, config, *req.Destination); err != nil {
					return err
				}
			}
			row.Config, err = proto.Marshal(config)
			if err != nil {
				return err
			}
			if row.Status == "paused_by_failure" {
				row.Status = "running"
			}
		}
		if req.Status != nil {
			row.Status = db.LogdrainsStatus(*req.Status)
		}
		if changesDelivery || (req.Status != nil && *req.Status == "running") {
			row.ConsecutiveFailures = 0
			row.NextAttemptAt = 0
		}
		if changesDelivery || req.Status != nil {
			row.LeaseExpiresAt = 0
		}
		if req.Name != nil {
			row.Name = *req.Name
		}
		if err := db.Query.UpdateLogdrain(ctx, tx, db.UpdateLogdrainParams{ID: row.ID, WorkspaceID: principal.AuthorizedWorkspaceID, Name: row.Name, Config: row.Config, Status: row.Status, LeaseExpiresAt: row.LeaseExpiresAt, ConsecutiveFailures: row.ConsecutiveFailures, NextAttemptAt: row.NextAttemptAt, UpdatedAt: sql.NullInt64{Int64: h.Clock.Now().UnixMilli(), Valid: true}}); err != nil {
			return err
		}
		return h.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{{
			WorkspaceID: principal.AuthorizedWorkspaceID, Event: "logdrain.update", Display: "Updated log drain " + row.ID,
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
