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
	"github.com/unkeyed/unkey/svc/api/internal/rootkeys"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type Request = openapi.V2RootKeysRerollKeyRequestBody
type Response = openapi.V2RootKeysRerollKeyResponseBody

// Handler rotates root keys in the new credential store.
type Handler struct {
	DB           db.Database
	Keys         keys.KeyService
	Auditlogs    auditlogs.AuditLogService
	KeyCache     cache.Cache[string, keysdb.CachedKeyData]
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

	source, err := db.Query.FindUnkeyRootKeyByID(ctx, h.DB.RO(), req.KeyId)
	if db.IsNotFound(err) || err == nil && source.WorkspaceID != p.AuthorizedWorkspaceID {
		return rootKeyNotFound()
	}
	if err != nil {
		return err
	}
	if err := authorizeLifetime(p, source.Expires); err != nil {
		return err
	}
	grants, err := db.Query.ListUnkeyPermissionsByPrincipal(ctx, h.DB.RO(), db.ListUnkeyPermissionsByPrincipalParams{
		WorkspaceID:   p.AuthorizedWorkspaceID,
		PrincipalType: db.UnkeyPrincipalPermissionsPrincipalTypeRootKey,
		PrincipalID:   source.ID,
	})
	if err != nil {
		return err
	}
	slices.Sort(grants)
	grants = slices.Compact(grants)
	grants, err = rootkeys.ValidateDelegatedPermissions(ctx, p, grants)
	if err != nil {
		return err
	}

	generated, err := h.Keys.CreateKeyV1(ctx, keys.CreateKeyV1Request{Prefix: source.Prefix})
	if err != nil {
		return err
	}
	keyID := uid.New(uid.KeyPrefix)
	now := h.Clock.Now()
	ctx = auditlog.WithCorrelation(ctx, auditlog.NewCorrelationID())
	err = db.TxRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		current, err := db.Query.FindUnkeyRootKeyByID(ctx, tx, req.KeyId)
		if db.IsNotFound(err) || err == nil && current.WorkspaceID != p.AuthorizedWorkspaceID {
			return rootKeyNotFound()
		}
		if err != nil {
			return err
		}
		source = current
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
		permissionRows := make([]db.InsertUnkeyPermissionParams, 0, len(grants))
		for _, grant := range grants {
			permissionRows = append(permissionRows, db.InsertUnkeyPermissionParams{
				ID:            uid.New(uid.PermissionPrefix),
				WorkspaceID:   p.AuthorizedWorkspaceID,
				PrincipalType: db.UnkeyPrincipalPermissionsPrincipalTypeRootKey,
				PrincipalID:   keyID,
				Slug:          grant,
				CreatedAt:     now.UnixMilli(),
			})
		}
		if err := db.BulkQuery.InsertUnkeyPermissions(ctx, tx, permissionRows); err != nil {
			return err
		}
		if !req.Expiration.IsNull() {
			expires := now.Add(time.Duration(req.Expiration.MustGet()) * time.Millisecond)
			if !source.Expires.Valid || !source.Expires.Time.Before(expires) {
				if err := db.Query.UpdateUnkeyRootKeyExpiration(ctx, tx, db.UpdateUnkeyRootKeyExpirationParams{
					Expires:     sql.NullTime{Time: expires, Valid: true},
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
	h.KeyCache.Remove(ctx, source.Hash)
	h.RootKeyCache.Remove(ctx, source.Hash)
	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: openapi.V2RootKeysRerollKeyResponseData{KeyId: keyID, Key: generated.Key},
	})
}

// authorizeLifetime prevents an expiring caller from minting a longer-lived secret.
func authorizeLifetime(p *principal.Principal, expires sql.NullTime) error {
	source, ok := p.Source.(principal.KeySource)
	if !ok || source.ExpiresAt == nil {
		return nil
	}
	if expires.Valid && !expires.Time.After(*source.ExpiresAt) {
		return nil
	}
	return fault.New("rerolled root key outlives caller",
		fault.Code(codes.App.Validation.InvalidInput.URN()),
		fault.Public("An expiring root key can only reroll root keys that expire no later than itself."),
	)
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
		WorkspaceID: workspaceID, Event: auditlog.KeyRerollEvent,
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
