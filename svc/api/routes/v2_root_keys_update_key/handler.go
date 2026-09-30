package handler

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/unkeyed/unkey/internal/services/auditlogs"
	keysdb "github.com/unkeyed/unkey/internal/services/keys/db"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/auditactor"
	principalpermissions "github.com/unkeyed/unkey/svc/api/internal/principal"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type Request = openapi.V2RootKeysUpdateKeyRequestBody
type Response = openapi.V2RootKeysUpdateKeyResponseBody

// Handler updates root keys in the new credential store.
type Handler struct {
	DB           db.Database
	Auditlogs    auditlogs.AuditLogService
	RootKeyCache cache.Cache[string, keysdb.CachedRootKeyData]
	Clock        clock.Clock
}

func (h *Handler) Method() string { return http.MethodPost }

func (h *Handler) Path() string { return "/v2/rootKeys.updateKey" }

// Handle updates one new root key in the authenticated customer workspace.
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	p, err := s.GetPrincipal()
	if err != nil {
		return err
	}
	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}
	if err := p.Authorize(rbac.U(urn.New().Workspace(p.AuthorizedWorkspaceID).RootKey(req.KeyId), permissions.Write)); err != nil {
		return err
	}
	var validatedPermissions []string
	if req.Permissions != nil {
		validatedPermissions, err = principalpermissions.ValidateDelegatedPermissions(ctx, p, *req.Permissions)
		if err != nil {
			return err
		}
	}

	var key db.UnkeyRootKey
	ctx = auditlog.WithCorrelation(ctx, auditlog.NewCorrelationID())
	err = db.TxRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		key, err = db.Query.FindUnkeyRootKeyByIDForUpdate(ctx, tx, req.KeyId)
		if db.IsNotFound(err) || err == nil && key.WorkspaceID != p.AuthorizedWorkspaceID {
			return fault.New("root key not found",
				fault.Code(codes.Data.Key.NotFound.URN()),
				fault.Public("The specified root key was not found."),
			)
		}
		if err != nil {
			return err
		}
		if source, ok := p.Source.(principal.KeySource); ok && source.ExpiresAt != nil {
			if !key.Expires.Valid || key.Expires.Int64 > source.ExpiresAt.UnixMilli() {
				return fault.New("target root key outlives caller",
					fault.Code(codes.App.Validation.InvalidInput.URN()),
					fault.Public("An expiring root key can only update root keys that expire no later than itself."),
				)
			}
		}
		if err := h.updateFields(ctx, tx, p.AuthorizedWorkspaceID, req); err != nil {
			return err
		}
		permissionRows, err := h.replacePermissions(ctx, tx, p.AuthorizedWorkspaceID, req.KeyId, validatedPermissions, req.Permissions != nil)
		if err != nil {
			return err
		}
		return h.Auditlogs.Insert(ctx, tx, updateAuditLogs(s, auditactor.FromPrincipal(p), p.AuthorizedWorkspaceID, key, permissionRows))
	})
	if err != nil {
		return err
	}
	h.RootKeyCache.Remove(ctx, key.Hash)
	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: openapi.EmptyResponse{},
	})
}

// updateFields applies optional scalar changes to a new root key.
func (h *Handler) updateFields(ctx context.Context, tx db.DBTX, workspaceID string, req Request) error {
	var nameSpecified, enabledSpecified int64
	name := sql.NullString{}
	if req.Name.IsSpecified() {
		nameSpecified = 1
		if !req.Name.IsNull() {
			name = sql.NullString{String: req.Name.MustGet(), Valid: true}
		}
	}
	enabled := sql.NullBool{}
	if req.Enabled != nil {
		enabledSpecified = 1
		enabled = sql.NullBool{Bool: *req.Enabled, Valid: true}
	}
	if nameSpecified == 0 && enabledSpecified == 0 {
		return nil
	}
	return db.Query.UpdateUnkeyRootKey(ctx, tx, db.UpdateUnkeyRootKeyParams{
		NameSpecified: nameSpecified, Name: name, EnabledSpecified: enabledSpecified, Enabled: enabled,
		ID: req.KeyId, WorkspaceID: workspaceID,
	})
}

// replacePermissions makes principal permissions the complete permission set.
func (h *Handler) replacePermissions(ctx context.Context, tx db.DBTX, workspaceID, keyID string, slugs []string, specified bool) ([]db.InsertUnkeyPermissionParams, error) {
	if !specified {
		return nil, nil
	}
	if err := db.Query.DeleteUnkeyPermissionsByPrincipal(ctx, tx, db.DeleteUnkeyPermissionsByPrincipalParams{
		WorkspaceID:   workspaceID,
		PrincipalType: db.UnkeyPrincipalPermissionsPrincipalTypeRootKey,
		PrincipalID:   keyID,
	}); err != nil {
		return nil, err
	}
	rows := make([]db.InsertUnkeyPermissionParams, 0, len(slugs))
	now := h.Clock.Now().UnixMilli()
	for _, slug := range slugs {
		rows = append(rows, db.InsertUnkeyPermissionParams{
			ID:            uid.New(uid.PermissionPrefix),
			WorkspaceID:   workspaceID,
			PrincipalType: db.UnkeyPrincipalPermissionsPrincipalTypeRootKey,
			PrincipalID:   keyID,
			Slug:          slug,
			CreatedAt:     now,
		})
	}
	return rows, db.BulkQuery.InsertUnkeyPermissions(ctx, tx, rows)
}

// updateAuditLogs records the key update and every newly granted permission.
func updateAuditLogs(s *zen.Session, actor auditactor.Actor, workspaceID string, key db.UnkeyRootKey, permissionRows []db.InsertUnkeyPermissionParams) []auditlog.AuditLog {
	name := key.Name.String
	if name == "" {
		name = key.Start
	}
	keyResource := auditlog.AuditLogResource{Type: auditlog.KeyResourceType, ID: key.ID, Name: name, DisplayName: name, Meta: map[string]any{}}
	logs := []auditlog.AuditLog{{
		WorkspaceID: workspaceID, Event: auditlog.RootKeyUpdateEvent,
		ActorType: actor.Type, ActorID: actor.ID, ActorName: actor.Name, ActorMeta: actor.Meta,
		Display: "Updated root key " + key.ID, RemoteIP: s.Location(), UserAgent: s.UserAgent(),
		CorrelationID: "",
		Resources:     []auditlog.AuditLogResource{keyResource},
	}}
	for _, permission := range permissionRows {
		logs = append(logs, auditlog.AuditLog{
			WorkspaceID: workspaceID, Event: auditlog.AuthConnectPermissionKeyEvent,
			ActorType: actor.Type, ActorID: actor.ID, ActorName: actor.Name, ActorMeta: actor.Meta,
			Display: "Granted " + permission.Slug, RemoteIP: s.Location(), UserAgent: s.UserAgent(),
			CorrelationID: "",
			Resources: []auditlog.AuditLogResource{keyResource, {
				Type: auditlog.PermissionResourceType, ID: permission.ID,
				Name: permission.Slug, DisplayName: permission.Slug, Meta: map[string]any{},
			}},
		})
	}
	return logs
}
