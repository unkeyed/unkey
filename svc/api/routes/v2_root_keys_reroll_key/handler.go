package handler

import (
	"context"
	"database/sql"
	"net/http"
	"slices"
	"strings"
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
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type Request = openapi.V2RootKeysRerollKeyRequestBody
type Response = openapi.V2RootKeysRerollKeyResponseBody

// Handler rotates root keys into the new credential store.
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

// Handle creates the replacement before shortening the original so a failure
// never leaves a workspace without its existing credential.
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
	if err := p.Authorize(rbac.U(urn.New().Workspace(p.AuthorizedWorkspaceID).RootKey(req.KeyId), permissions.Write)); err != nil {
		return err
	}

	sources, err := db.Query.FindRootKeysForManagement(ctx, h.DB.RO(), db.FindRootKeysForManagementParams{
		ID:          req.KeyId,
		WorkspaceID: sql.NullString{String: p.AuthorizedWorkspaceID, Valid: true},
	})
	if err != nil {
		return err
	}
	if len(sources) == 0 {
		return fault.New("root key not found",
			fault.Code(codes.Data.Key.NotFound.URN()),
			fault.Public("The specified root key was not found."),
		)
	}
	source := sources[0]
	if err := authorizeLifetime(p, source.Expires); err != nil {
		return err
	}
	storedPermissions, err := db.Query.ListRootKeyPermissions(ctx, h.DB.RO(), db.ListRootKeyPermissionsParams{
		WorkspaceID: sql.NullString{String: p.AuthorizedWorkspaceID, Valid: true},
		KeyIds:      []string{source.ID},
	})
	if err != nil {
		return err
	}
	grants := make([]string, 0, len(storedPermissions))
	for _, permission := range storedPermissions {
		grants = append(grants, permission.Slug)
	}
	slices.Sort(grants)
	grants = slices.Compact(grants)
	for _, grant := range grants {
		if err := p.Authorize(heldPermissionQuery(grant)); err != nil {
			return err
		}
	}

	prefix := source.Prefix
	if prefix == "" {
		prefix = "unkey"
	}
	generated, err := h.Keys.CreateKeyV1(ctx, keys.CreateKeyV1Request{Prefix: prefix})
	if err != nil {
		return err
	}
	keyID := uid.New(uid.KeyPrefix)
	now := h.Clock.Now()
	ctx = auditlog.WithCorrelation(ctx, auditlog.NewCorrelationID())
	err = db.TxRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		current, err := db.Query.FindRootKeysForManagement(ctx, tx, db.FindRootKeysForManagementParams{
			ID:          req.KeyId,
			WorkspaceID: sql.NullString{String: p.AuthorizedWorkspaceID, Valid: true},
		})
		if err != nil {
			return err
		}
		if len(current) == 0 {
			return fault.New("root key not found",
				fault.Code(codes.Data.Key.NotFound.URN()),
				fault.Public("The specified root key was not found."),
			)
		}
		sources = current
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
			for _, key := range sources {
				if key.Expires.Valid && key.Expires.Time.Before(expires) {
					continue
				}
				if err := expireOriginal(ctx, tx, p.AuthorizedWorkspaceID, key, expires, now); err != nil {
					return err
				}
			}
		}
		return h.Auditlogs.Insert(ctx, tx, rerollAuditLogs(s, auditactor.FromPrincipal(p), p.AuthorizedWorkspaceID, source, keyID, permissionRows))
	})
	if err != nil {
		return err
	}
	for _, key := range sources {
		h.KeyCache.Remove(ctx, key.Hash)
		h.RootKeyCache.Remove(ctx, key.Hash)
	}
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

// heldPermissionQuery requires the caller to hold a copied grant. URN grants
// use containment; legacy grants must match exactly because they have no hierarchy.
func heldPermissionQuery(grant string) rbac.PermissionQuery {
	resourceName, actionName, ok := strings.Cut(grant, "#")
	if ok && !strings.Contains(actionName, "#") {
		resource, err := urn.ParseV1(resourceName)
		if err == nil {
			return rbac.U(resource, permissions.Action(actionName))
		}
	}
	return rbac.S(grant)
}

// expireOriginal shortens one store's copy of the original root key.
func expireOriginal(ctx context.Context, tx db.DBTX, workspaceID string, key db.FindRootKeysForManagementRow, expires, now time.Time) error {
	if key.IsLegacy == 1 {
		return db.Query.UpdateLegacyRootKeyExpiration(ctx, tx, db.UpdateLegacyRootKeyExpirationParams{
			Expires:     sql.NullTime{Time: expires, Valid: true},
			Now:         sql.NullInt64{Int64: now.UnixMilli(), Valid: true},
			ID:          key.ID,
			WorkspaceID: sql.NullString{String: workspaceID, Valid: true},
		})
	}
	return db.Query.UpdateUnkeyRootKeyExpiration(ctx, tx, db.UpdateUnkeyRootKeyExpirationParams{
		Expires:     sql.NullTime{Time: expires, Valid: true},
		ID:          key.ID,
		WorkspaceID: workspaceID,
	})
}

// rerollAuditLogs records the rotation and every permission copied to the new key.
func rerollAuditLogs(s *zen.Session, actor auditactor.Actor, workspaceID string, source db.FindRootKeysForManagementRow, keyID string, permissionRows []db.InsertUnkeyPermissionParams) []auditlog.AuditLog {
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
