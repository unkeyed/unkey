// Package handler implements POST /v2/rootKeys.createAgentKey.
//
// The route is the only API path the dashboard uses to mint a root key during
// agent signup. It accepts a JWT whose role is exactly agent_signup and
// rejects every permission outside that role's allowlist, including when the
// caller would otherwise be allowed to delegate it.
package handler

import (
	"context"
	"database/sql"
	"net/http"
	"slices"

	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/pkg/array"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/auth/agentsignup"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/auditactor"
	principalpermissions "github.com/unkeyed/unkey/svc/api/internal/principal"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

// Request is the JSON body for an agent-signup root key.
type Request struct {
	Name        *string  `json:"name"`
	Permissions []string `json:"permissions"`
}

// Response is the JSON body returned after a key is stored.
type Response struct {
	Meta openapi.Meta `json:"meta"`
	Data ResponseData `json:"data"`
}

// ResponseData carries the new key id and the plaintext secret.
type ResponseData struct {
	KeyId string `json:"keyId"`
	Key   string `json:"key"`
}

// Handler creates a root key bounded to the agent-signup allowlist.
type Handler struct {
	DB        db.Database
	Keys      keys.KeyService
	Auditlogs auditlogs.AuditLogService
	Clock     clock.Clock
}

func (h *Handler) Method() string { return "POST" }

func (h *Handler) Path() string { return "/v2/rootKeys.createAgentKey" }

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	p, err := s.GetPrincipal()
	if err != nil {
		return err
	}
	source, ok := p.Source.(principal.JWTSource)
	if p.Type != principal.TypeJWT || !ok || !slices.Equal(source.Roles, []string{agentsignup.Role}) {
		return fault.New("agent signup root key requires the agent_signup role",
			fault.Code(codes.Auth.Authorization.Forbidden.URN()),
			fault.Public("This credential cannot create an agent root key."),
		)
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}
	for _, permission := range req.Permissions {
		if !agentsignup.PermissionAllowed(p.AuthorizedWorkspaceID, permission) {
			return fault.New("permission outside agent allowlist",
				fault.Code(codes.Auth.Authorization.InsufficientPermissions.URN()),
				fault.Public("That permission is outside the agent allowlist."),
			)
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
	keyID := uid.New(uid.KeyPrefix)
	ctx = auditlog.WithCorrelation(ctx, auditlog.NewCorrelationID())
	err = db.TxRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		err := db.Query.InsertUnkeyRootKey(ctx, tx, db.InsertUnkeyRootKeyParams{
			ID:          keyID,
			WorkspaceID: p.AuthorizedWorkspaceID,
			Name: sql.NullString{
				String: ptr.SafeDeref(req.Name),
				Valid:  req.Name != nil,
			},
			Hash:      key.Hash,
			Prefix:    key.Prefix,
			Start:     key.Start,
			End:       key.End,
			Enabled:   true,
			Expires:   sql.NullInt64{},
			CreatedAt: h.Clock.Now().UnixMilli(),
		})
		if err != nil {
			return err
		}
		permissionRows := array.Map(validatedPermissions, func(slug string) db.InsertUnkeyPermissionParams {
			return db.InsertUnkeyPermissionParams{
				ID:            uid.New(uid.PermissionPrefix),
				WorkspaceID:   p.AuthorizedWorkspaceID,
				PrincipalType: db.UnkeyPrincipalPermissionsPrincipalTypeRootKey,
				PrincipalID:   keyID,
				Slug:          slug,
				CreatedAt:     h.Clock.Now().UnixMilli(),
			}
		})
		if err := db.BulkQuery.InsertUnkeyPermissions(ctx, tx, permissionRows); err != nil {
			return err
		}
		actor := auditactor.FromPrincipal(p)
		keyResource := auditlog.AuditLogResource{
			Type:        auditlog.KeyResourceType,
			ID:          keyID,
			Name:        ptr.SafeDeref(req.Name),
			DisplayName: ptr.SafeDeref(req.Name),
			Meta:        map[string]any{},
		}
		logs := []auditlog.AuditLog{{
			WorkspaceID:   p.AuthorizedWorkspaceID,
			Event:         auditlog.RootKeyCreateEvent,
			ActorType:     actor.Type,
			ActorID:       actor.ID,
			ActorName:     actor.Name,
			ActorMeta:     actor.Meta,
			Display:       "Created root key " + keyID,
			RemoteIP:      s.Location(),
			UserAgent:     s.UserAgent(),
			CorrelationID: "",
			Resources:     []auditlog.AuditLogResource{keyResource},
		}}
		logs = append(logs, array.Map(permissionRows, func(permission db.InsertUnkeyPermissionParams) auditlog.AuditLog {
			return auditlog.AuditLog{
				WorkspaceID:   p.AuthorizedWorkspaceID,
				Event:         auditlog.AuthConnectPermissionKeyEvent,
				ActorType:     actor.Type,
				ActorID:       actor.ID,
				ActorName:     actor.Name,
				ActorMeta:     actor.Meta,
				Display:       "Granted " + permission.Slug,
				RemoteIP:      s.Location(),
				UserAgent:     s.UserAgent(),
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
		return h.Auditlogs.Insert(ctx, tx, logs)
	})
	if err != nil {
		return err
	}
	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: ResponseData{
			KeyId: keyID,
			Key:   key.Key,
		},
	})
}
