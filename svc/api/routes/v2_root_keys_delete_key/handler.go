package handler

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/unkeyed/unkey/internal/services/auditlogs"
	keysdb "github.com/unkeyed/unkey/internal/services/keys/db"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/auditactor"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type Request = openapi.V2RootKeysDeleteKeyRequestBody
type Response = openapi.V2RootKeysDeleteKeyResponseBody

// Handler deletes root keys from both credential stores.
type Handler struct {
	DB           db.Database
	Auditlogs    auditlogs.AuditLogService
	KeyCache     cache.Cache[string, keysdb.CachedKeyData]
	RootKeyCache cache.Cache[string, keysdb.CachedRootKeyData]
	Clock        clock.Clock
}

func (h *Handler) Method() string { return http.MethodPost }

func (h *Handler) Path() string { return "/v2/rootKeys.deleteKey" }

// Handle deletes every live row with the requested ID so a migration twin cannot remain active.
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	p, err := s.GetPrincipal()
	if err != nil {
		return err
	}
	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}
	if err := p.Authorize(rbac.U(urn.New().Workspace(p.AuthorizedWorkspaceID).RootKey(req.KeyId), permissions.Delete)); err != nil {
		return err
	}

	var keys []db.FindRootKeysForManagementRow
	err = db.TxRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		keys, err = db.Query.FindRootKeysForManagement(ctx, tx, db.FindRootKeysForManagementParams{
			ID:             req.KeyId,
			ForWorkspaceID: sql.NullString{String: p.AuthorizedWorkspaceID, Valid: true},
		})
		if err != nil {
			return err
		}
		if len(keys) == 0 {
			return rootKeyNotFound()
		}
		now := h.Clock.Now().UnixMilli()
		for _, key := range keys {
			if key.IsLegacy == 1 {
				_, err = db.Query.SoftDeleteLegacyRootKey(ctx, tx, db.SoftDeleteLegacyRootKeyParams{
					Now:            sql.NullInt64{Int64: now, Valid: true},
					ID:             key.ID,
					ForWorkspaceID: sql.NullString{String: p.AuthorizedWorkspaceID, Valid: true},
				})
			} else {
				_, err = db.Query.SoftDeleteUnkeyRootKey(ctx, tx, db.SoftDeleteUnkeyRootKeyParams{
					Now:            sql.NullInt64{Int64: now, Valid: true},
					ID:             key.ID,
					ForWorkspaceID: p.AuthorizedWorkspaceID,
				})
			}
			if err != nil {
				return err
			}
		}
		key := keys[0]
		name := key.Name.String
		if name == "" {
			name = key.Start
		}
		actor := auditactor.FromPrincipal(p)
		return h.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{{
			WorkspaceID:   p.AuthorizedWorkspaceID,
			Event:         auditlog.KeyDeleteEvent,
			ActorType:     actor.Type,
			ActorID:       actor.ID,
			ActorName:     actor.Name,
			ActorMeta:     actor.Meta,
			Display:       "Deleted root key " + key.ID,
			RemoteIP:      s.Location(),
			UserAgent:     s.UserAgent(),
			CorrelationID: "",
			Resources: []auditlog.AuditLogResource{{
				Type:        auditlog.KeyResourceType,
				ID:          key.ID,
				Name:        name,
				DisplayName: name,
				Meta:        map[string]any{},
			}},
		}})
	})
	if err != nil {
		return err
	}
	for _, key := range keys {
		h.KeyCache.Remove(ctx, key.Hash)
		h.RootKeyCache.Remove(ctx, key.Hash)
	}
	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: openapi.EmptyResponse{},
	})
}

func rootKeyNotFound() error {
	return fault.New("root key not found",
		fault.Code(codes.Data.Key.NotFound.URN()),
		fault.Public("The specified root key was not found."),
	)
}
