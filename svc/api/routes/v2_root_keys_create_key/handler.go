package handler

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	principalpermissions "github.com/unkeyed/unkey/svc/api/internal/principal"
	"github.com/unkeyed/unkey/svc/api/internal/rootkey"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type Request = openapi.V2RootKeysCreateKeyRequestBody
type Response = openapi.V2RootKeysCreateKeyResponseBody

type Handler struct {
	DB        db.Database
	Keys      keys.KeyService
	Auditlogs auditlogs.AuditLogService
	Clock     clock.Clock
}

func (h *Handler) Method() string { return "POST" }

func (h *Handler) Path() string { return "/v2/rootKeys.createKey" }

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	p, err := s.GetPrincipal()
	if err != nil {
		return err
	}
	if err := p.Authorize(rbac.U(urn.New().Workspace(p.AuthorizedWorkspaceID).RootKey("*"), permissions.Write)); err != nil {
		return err
	}
	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}
	var expires sql.NullInt64
	if req.Expires.IsSpecified() && !req.Expires.IsNull() {
		if req.Expires.MustGet() <= h.Clock.Now().UnixMilli() {
			return fault.New("expiration must be in the future", fault.Code(codes.App.Validation.InvalidInput.URN()),
				fault.Public("Expires must be a Unix millisecond timestamp in the future."))
		}
		expires = sql.NullInt64{Int64: req.Expires.MustGet(), Valid: true}
	}
	if source, ok := p.Source.(principal.KeySource); ok && source.ExpiresAt != nil {
		if !expires.Valid || expires.Int64 > source.ExpiresAt.UnixMilli() {
			return fault.New("child root key exceeds caller expiration", fault.Code(codes.App.Validation.InvalidInput.URN()),
				fault.Public("Expires is required and must not be later than the calling root key's expiration."))
		}
	}
	validatedPermissions, err := principalpermissions.ValidateDelegatedPermissions(ctx, p, req.Permissions)
	if err != nil {
		return err
	}
	key, err := h.Keys.CreateKeyV1(ctx, keys.CreateKeyV1Request{
		Prefix: "unkey",
	})
	if err != nil {
		return err
	}
	ctx = auditlog.WithCorrelation(ctx, auditlog.NewCorrelationID())
	var created rootkey.InsertResult
	err = db.TxRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		var insertErr error
		created, insertErr = rootkey.Insert(ctx, tx, rootkey.InsertRequest{
			Principal:   p,
			Name:        req.Name,
			Permissions: validatedPermissions,
			Expires:     expires,
			Key:         key,
			RemoteIP:    s.Location(),
			UserAgent:   s.UserAgent(),
			Auditlogs:   h.Auditlogs,
			Clock:       h.Clock,
		})
		return insertErr
	})
	if err != nil {
		return err
	}
	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: openapi.V2RootKeysCreateKeyResponseData{
			KeyId: created.KeyID,
			Key:   created.Key,
		},
	})
}
