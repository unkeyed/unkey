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
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
	"google.golang.org/protobuf/proto"
)

type Create struct {
	DB        db.Database
	Vault     vault.VaultServiceClient
	Auditlogs auditlogs.AuditLogService
	Clock     clock.Clock
}

func (h *Create) Method() string { return http.MethodPost }
func (h *Create) Path() string   { return "/v2/logdrains.createLogdrain" }
func (h *Create) Handle(ctx context.Context, s *zen.Session) error {
	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}
	req, err := zen.BindBody[openapi.CreateLogdrainRequest](s)
	if err != nil {
		return err
	}
	if err := principal.Authorize(rbac.U(urn.V1{WorkspaceID: principal.AuthorizedWorkspaceID, Resource: "logdrains/*"}, permissions.Write)); err != nil {
		return err
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return invalid("Name must not be empty.")
	}
	config := &logdrainv1.Config{
		BatchSize: uint32(ptr.SafeDeref(req.BatchSize)),
	}
	if err := setStream(config, string(req.Stream), ptr.SafeDeref(req.Filters)); err != nil {
		return err
	}
	if err := setDestination(ctx, h.Vault, principal.AuthorizedWorkspaceID, config, req.Destination); err != nil {
		return err
	}
	encoded, err := proto.Marshal(config)
	if err != nil {
		return err
	}
	id := uid.New("ld")
	now := h.Clock.Now().UnixMilli()
	err = db.TxRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		limit, err := db.Query.LockLogdrainLimit(ctx, tx, principal.AuthorizedWorkspaceID)
		if err != nil && !db.IsNotFound(err) {
			return err
		}
		if db.IsNotFound(err) || limit == 0 {
			return fault.New("log drains disabled", fault.Code(codes.Auth.Authorization.Forbidden.URN()), fault.Public("Contact support to enable log drains for this workspace."))
		}
		drains, err := db.Query.LockWorkspaceLogdrains(ctx, tx, principal.AuthorizedWorkspaceID)
		if err != nil {
			return err
		}
		if len(drains) >= int(limit) {
			return fault.New("log drain limit reached", fault.Code(codes.Auth.Authorization.Forbidden.URN()), fault.Public("Contact support to increase this workspace's log drain allowance."))
		}
		if err := db.Query.InsertLogdrain(ctx, tx, db.InsertLogdrainParams{ID: id, WorkspaceID: principal.AuthorizedWorkspaceID, Name: name, Stream: db.LogdrainsStream(req.Stream), Config: encoded, CreatedAt: now, UpdatedAt: sql.NullInt64{Int64: now, Valid: true}}); err != nil {
			return err
		}
		return h.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{{
			WorkspaceID: principal.AuthorizedWorkspaceID, Event: "logdrain.create", Display: "Created log drain " + id,
			ActorID: principal.Subject.ID, ActorType: auditlog.AuditLogActor(principal.Subject.Type), ActorName: principal.Subject.Name,
			ActorMeta: map[string]any{}, CorrelationID: "",
			RemoteIP: s.Location(), UserAgent: s.UserAgent(),
			Resources: []auditlog.AuditLogResource{{ID: id, Type: "logdrain", Name: name, DisplayName: "", Meta: map[string]any{}}},
		}})
	})
	if err != nil {
		return err
	}
	var response openapi.LogdrainMutationResponse
	response.Meta.RequestId = s.RequestID()
	response.Data.Id = id
	return s.JSON(http.StatusOK, response)
}

func invalid(message string) error {
	return fault.New("invalid log drain configuration", fault.Code(codes.App.Validation.InvalidInput.URN()), fault.Public(message))
}
