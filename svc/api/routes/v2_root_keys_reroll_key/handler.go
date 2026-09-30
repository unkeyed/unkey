package handler

import (
	"context"
	"database/sql"
	"net/http"
	"slices"
	"time"

	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/internal/services/keys"
	keysdb "github.com/unkeyed/unkey/internal/services/keys/db"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/auditlog"
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
	"github.com/unkeyed/unkey/svc/api/internal/principal"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type Request = openapi.V2RootKeysRerollKeyRequestBody
type Response = openapi.V2RootKeysRerollKeyResponseBody

// Handler rotates root keys in the new credential store.
type Handler struct {
	DB           db.Database
	Keys         keys.KeyService
	Auditlogs    auditlogs.AuditLogService
	RootKeyCache cache.Cache[string, keysdb.CachedRootKeyData]
	Clock        clock.Clock
}

func (h *Handler) Method() string { return http.MethodPost }

func (h *Handler) Path() string { return "/v2/rootKeys.rerollKey" }

// Handle creates a replacement and optionally shortens the original in one transaction.
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	p, err := s.GetPrincipal()
	if err != nil {
		return err
	}
	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}
	if err := assert.True(req.Expiration.IsSpecified(), "request validation must require expiration"); err != nil {
		return err
	}
	resource := urn.New().Workspace(p.AuthorizedWorkspaceID).RootKey(req.KeyId)
	query := rbac.U(resource, permissions.Write)
	if !req.Expiration.IsNull() {
		query = rbac.And(query, rbac.U(resource, permissions.Delete))
	}
	if err := p.Authorize(query); err != nil {
		return err
	}

	var source db.UnkeyRootKey
	var generated keys.CreateKeyV1Response
	keyID := uid.New(uid.KeyPrefix)
	now := h.Clock.Now()
	ctx = auditlog.WithCorrelation(ctx, auditlog.NewCorrelationID())
	err = db.TxRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		current, err := db.Query.FindUnkeyRootKeyByIDForUpdate(ctx, tx, req.KeyId)
		if db.IsNotFound(err) || err == nil && current.WorkspaceID != p.AuthorizedWorkspaceID {
			return rootKeyNotFound()
		}
		if err != nil {
			return err
		}
		source = current
		rootKeyPermissions, err := db.Query.ListUnkeyPermissionsByPrincipal(ctx, tx, db.ListUnkeyPermissionsByPrincipalParams{
			WorkspaceID:   p.AuthorizedWorkspaceID,
			PrincipalType: db.UnkeyPrincipalPermissionsPrincipalTypeRootKey,
			PrincipalID:   source.ID,
		})
		if err != nil {
			return err
		}
		slices.Sort(rootKeyPermissions)
		rootKeyPermissions = slices.Compact(rootKeyPermissions)
		rootKeyPermissions, err = principal.ValidateDelegatedPermissions(ctx, p, rootKeyPermissions)
		if err != nil {
			return err
		}
		generated, err = h.Keys.CreateKeyV1(ctx, keys.CreateKeyV1Request{Prefix: source.Prefix})
		if err != nil {
			return err
		}
		if err := db.Query.InsertUnkeyRootKey(ctx, tx, db.InsertUnkeyRootKeyParams{
			ID:          keyID,
			WorkspaceID: p.AuthorizedWorkspaceID,
			Hash:        generated.Hash,
			Name:        source.Name,
			Prefix:      generated.Prefix,
			Start:       generated.Start,
			End:         generated.End,
			Enabled:     source.Enabled,
			Expires:     source.Expires,
			CreatedAt:   now.UnixMilli(),
		}); err != nil {
			return err
		}
		permissionRows := make([]db.InsertUnkeyPermissionParams, 0, len(rootKeyPermissions))
		for _, permission := range rootKeyPermissions {
			permissionRows = append(permissionRows, db.InsertUnkeyPermissionParams{
				ID:            uid.New(uid.PermissionPrefix),
				WorkspaceID:   p.AuthorizedWorkspaceID,
				PrincipalType: db.UnkeyPrincipalPermissionsPrincipalTypeRootKey,
				PrincipalID:   keyID,
				Slug:          permission,
				CreatedAt:     now.UnixMilli(),
			})
		}
		if err := db.BulkQuery.InsertUnkeyPermissions(ctx, tx, permissionRows); err != nil {
			return err
		}
		if !req.Expiration.IsNull() {
			expires := now.Add(time.Duration(req.Expiration.MustGet()) * time.Millisecond).UnixMilli()
			if !source.Expires.Valid || source.Expires.Int64 >= expires {
				if err := db.Query.UpdateUnkeyRootKeyExpiration(ctx, tx, db.UpdateUnkeyRootKeyExpirationParams{
					Expires:     sql.NullInt64{Int64: expires, Valid: true},
					ID:          source.ID,
					WorkspaceID: p.AuthorizedWorkspaceID,
				}); err != nil {
					return err
				}
			}
		}
		return h.Auditlogs.Insert(ctx, tx, rerollAuditLogs(s, auditactor.FromPrincipal(p), p.AuthorizedWorkspaceID, source, keyID, permissionRows))
	})
	if err != nil {
		return err
	}
	h.RootKeyCache.Remove(ctx, source.Hash)
	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: openapi.V2RootKeysRerollKeyResponseData{KeyId: keyID, Key: generated.Key},
	})
}

func rootKeyNotFound() error {
	return fault.New("root key not found",
		fault.Code(codes.Data.Key.NotFound.URN()),
		fault.Public("The specified root key was not found."),
	)
}

// rerollAuditLogs records the rotation and every permission copied to the new key.
func rerollAuditLogs(s *zen.Session, actor auditactor.Actor, workspaceID string, source db.UnkeyRootKey, keyID string, permissionRows []db.InsertUnkeyPermissionParams) []auditlog.AuditLog {
	name := source.Name.String
	if name == "" {
		name = source.Start
	}
	newKey := auditlog.AuditLogResource{Type: auditlog.KeyResourceType, ID: keyID, Name: name, DisplayName: name, Meta: map[string]any{}}
	logs := []auditlog.AuditLog{{
		WorkspaceID: workspaceID, Event: auditlog.RootKeyRerollEvent,
		ActorType: actor.Type, ActorID: actor.ID, ActorName: actor.Name, ActorMeta: actor.Meta,
		Display: "Rerolled root key " + source.ID + " to " + keyID, RemoteIP: s.Location(), UserAgent: s.UserAgent(),
		CorrelationID: "",
		Resources: []auditlog.AuditLogResource{newKey, {
			Type: auditlog.KeyResourceType, ID: source.ID, Name: name, DisplayName: name, Meta: map[string]any{},
		}},
	}}
	for _, permission := range permissionRows {
		logs = append(logs, auditlog.AuditLog{
			WorkspaceID: workspaceID, Event: auditlog.AuthConnectPermissionKeyEvent,
			ActorType: actor.Type, ActorID: actor.ID, ActorName: actor.Name, ActorMeta: actor.Meta,
			Display: "Granted " + permission.Slug, RemoteIP: s.Location(), UserAgent: s.UserAgent(),
			CorrelationID: "",
			Resources: []auditlog.AuditLogResource{newKey, {
				Type: auditlog.PermissionResourceType, ID: permission.ID,
				Name: permission.Slug, DisplayName: permission.Slug, Meta: map[string]any{},
			}},
		})
	}
	return logs
}
