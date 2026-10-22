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

// InsertRequest is the authorized input for one new root key.
//
// Principal is the caller. Permissions are the slugs to grant, already
// checked against what that caller may delegate. Key holds the secret
// material from the key service. Auditlogs and Clock are the collaborators
// the insert writes with.
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

// InsertResult is the root key that was stored. Key is the plaintext secret
// and is available only from this result.
type InsertResult struct {
	KeyID string
	Key   string
}

// Insert stores a root key, its permission rows, and the audit logs for both.
//
// Insert opens its own transaction. Callers must authorize the request and
// validate the permissions before calling Insert. A failure rolls the key,
// the grants, and the audit logs back together. On failure the result is the
// zero InsertResult.
func Insert(ctx context.Context, database db.Database, req InsertRequest) (InsertResult, error) {
	var result InsertResult
	err := db.TxRetry(ctx, database.RW(), func(ctx context.Context, tx db.DBTX) error {
		created, insertErr := insert(ctx, tx, req)
		if insertErr != nil {
			return insertErr
		}
		result = created
		return nil
	})
	return result, err
}

// insert writes the root key, its grants, and the audit logs inside the
// transaction Insert opened.
func insert(ctx context.Context, tx db.DBTX, req InsertRequest) (InsertResult, error) {
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
	keyTarget := auditlog.AuditLogTarget{
		Type:        auditlog.KeyTargetType,
		ID:          keyID,
		Name:        ptr.SafeDeref(req.Name),
		DisplayName: ptr.SafeDeref(req.Name),
		Meta:        map[string]any{},
	}
	base := auditlog.AuditLog{
		Event:         "",
		WorkspaceID:   req.Principal.AuthorizedWorkspaceID,
		Display:       "",
		ActorType:     actor.Type,
		ActorID:       actor.ID,
		ActorName:     actor.Name,
		ActorMeta:     actor.Meta,
		Targets:       nil,
		RemoteIP:      req.RemoteIP,
		UserAgent:     req.UserAgent,
		CorrelationID: "",
	}
	created := base
	created.Event = auditlog.RootKeyCreateEvent
	created.Display = "Created root key " + keyID
	created.Targets = []auditlog.AuditLogTarget{keyTarget}
	logs := make([]auditlog.AuditLog, 0, 1+len(permissionRows))
	logs = append(logs, created)
	for _, permission := range permissionRows {
		granted := base
		granted.Event = auditlog.AuthConnectPermissionKeyEvent
		granted.Display = "Granted " + permission.Slug
		granted.Targets = []auditlog.AuditLogTarget{
			keyTarget,
			{
				Type:        auditlog.PermissionTargetType,
				ID:          permission.ID,
				Name:        permission.Slug,
				DisplayName: permission.Slug,
				Meta:        map[string]any{},
			},
		}
		logs = append(logs, granted)
	}
	if err := req.Auditlogs.Insert(ctx, tx, logs); err != nil {
		return InsertResult{}, err
	}
	return InsertResult{KeyID: keyID, Key: req.Key.Key}, nil
}
