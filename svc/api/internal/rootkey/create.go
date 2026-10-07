package rootkey

import (
	"context"
	"database/sql"

	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/pkg/array"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/auditactor"
)

type InsertRequest struct {
	Principal   *principal.Principal
	Name        *string
	Permissions []string
	Expires     sql.NullInt64
	Key         keys.CreateKeyV1Response
	RemoteIP    string
	UserAgent   string
	Auditlogs   auditlogs.AuditLogService
	Clock       clock.Clock
}

type InsertResult struct {
	KeyID string
	Key   string
}

func Insert(ctx context.Context, tx db.DBTX, req InsertRequest) (InsertResult, error) {
	keyID := uid.New(uid.KeyPrefix)
	err := db.Query.InsertUnkeyRootKey(ctx, tx, db.InsertUnkeyRootKeyParams{
		ID:          keyID,
		WorkspaceID: req.Principal.AuthorizedWorkspaceID,
		Name: sql.NullString{
			String: ptr.SafeDeref(req.Name),
			Valid:  req.Name != nil,
		},
		Hash:      req.Key.Hash,
		Prefix:    req.Key.Prefix,
		Start:     req.Key.Start,
		End:       req.Key.End,
		Enabled:   true,
		CreatedAt: req.Clock.Now().UnixMilli(),
		Expires:   req.Expires,
	})
	if err != nil {
		return InsertResult{}, err
	}
	now := req.Clock.Now().UnixMilli()
	permissionRows := array.Map(req.Permissions, func(slug string) db.InsertUnkeyPermissionParams {
		return db.InsertUnkeyPermissionParams{
			ID:            uid.New(uid.PermissionPrefix),
			WorkspaceID:   req.Principal.AuthorizedWorkspaceID,
			PrincipalType: db.UnkeyPrincipalPermissionsPrincipalTypeRootKey,
			PrincipalID:   keyID,
			Slug:          slug,
			CreatedAt:     now,
		}
	})
	if err := db.BulkQuery.InsertUnkeyPermissions(ctx, tx, permissionRows); err != nil {
		return InsertResult{}, err
	}
	actor := auditactor.FromPrincipal(req.Principal)
	keyResource := auditlog.AuditLogResource{
		Type:        auditlog.KeyResourceType,
		ID:          keyID,
		Name:        ptr.SafeDeref(req.Name),
		DisplayName: ptr.SafeDeref(req.Name),
		Meta:        map[string]any{},
	}
	logs := []auditlog.AuditLog{{
		WorkspaceID:   req.Principal.AuthorizedWorkspaceID,
		Event:         auditlog.RootKeyCreateEvent,
		ActorType:     actor.Type,
		ActorID:       actor.ID,
		ActorName:     actor.Name,
		ActorMeta:     actor.Meta,
		Display:       "Created root key " + keyID,
		RemoteIP:      req.RemoteIP,
		UserAgent:     req.UserAgent,
		CorrelationID: "",
		Resources:     []auditlog.AuditLogResource{keyResource},
	}}
	logs = append(logs, array.Map(permissionRows, func(permission db.InsertUnkeyPermissionParams) auditlog.AuditLog {
		return auditlog.AuditLog{
			WorkspaceID:   req.Principal.AuthorizedWorkspaceID,
			Event:         auditlog.AuthConnectPermissionKeyEvent,
			ActorType:     actor.Type,
			ActorID:       actor.ID,
			ActorName:     actor.Name,
			ActorMeta:     actor.Meta,
			Display:       "Granted " + permission.Slug,
			RemoteIP:      req.RemoteIP,
			UserAgent:     req.UserAgent,
			CorrelationID: "",
			Resources: []auditlog.AuditLogResource{
				keyResource,
				{
					Type:        auditlog.PermissionResourceType,
					ID:          permission.ID,
					Name:        permission.Slug,
					DisplayName: permission.Slug,
					Meta:        map[string]any{},
				},
			},
		}
	})...)
	if err := req.Auditlogs.Insert(ctx, tx, logs); err != nil {
		return InsertResult{}, err
	}
	return InsertResult{KeyID: keyID, Key: req.Key.Key}, nil
}
