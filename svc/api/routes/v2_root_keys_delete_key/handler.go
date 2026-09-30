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

// Handler deletes root keys from the new credential store.
type Handler struct {
	DB           db.Database
	Auditlogs    auditlogs.AuditLogService
	KeyCache     cache.Cache[string, keysdb.CachedKeyData]
	RootKeyCache cache.Cache[string, keysdb.CachedRootKeyData]
	Clock        clock.Clock
}

func (h *Handler) Method() string { return http.MethodPost }

func (h *Handler) Path() string { return "/v2/rootKeys.deleteKey" }

// Handle tombstones one new root key in the authenticated customer workspace.
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

	var key db.UnkeyRootKey
	err = db.TxRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		key, err = db.Query.FindUnkeyRootKeyByID(ctx, tx, req.KeyId)
		if db.IsNotFound(err) || err == nil && key.WorkspaceID != p.AuthorizedWorkspaceID {
			return rootKeyNotFound()
		}
		if err != nil {
			return err
		}
		if _, err := db.Query.SoftDeleteUnkeyRootKey(ctx, tx, db.SoftDeleteUnkeyRootKeyParams{
			Now:         sql.NullInt64{Int64: h.Clock.Now().UnixMilli(), Valid: true},
			ID:          key.ID,
			WorkspaceID: p.AuthorizedWorkspaceID,
		}); err != nil {
			return err
		}
		name := key.Name.String
		if name == "" {
			name = key.Start
		}
		actor := auditactor.FromPrincipal(p)
		return h.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{{
			WorkspaceID:   p.AuthorizedWorkspaceID,
			Event:         auditlog.RootKeyDeleteEvent,
			ActorType:     actor.Type,
			ActorID:       actor.ID,
			ActorName:     actor.Name,
			ActorMeta:     actor.Meta,
			Display:       "Deleted root key " + key.ID,
			RemoteIP:      s.Location(),
			UserAgent:     s.UserAgent(),
			CorrelationID: "",
			Resources: []auditlog.AuditLogResource{{
				Type: auditlog.KeyResourceType, ID: key.ID, Name: name, DisplayName: name, Meta: map[string]any{},
			}},
		}})
	})
	if err != nil {
		return err
	}
	h.KeyCache.Remove(ctx, key.Hash)
	h.RootKeyCache.Remove(ctx, key.Hash)
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
