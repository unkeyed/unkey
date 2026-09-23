package handler

import (
	"context"
	"database/sql"
	"net/http"
	"slices"
	"time"

	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	dbtype "github.com/unkeyed/unkey/pkg/db/types"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/auditactor"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type Request = openapi.V2RootKeysCreateKeyRequestBody
type Response = openapi.V2RootKeysCreateKeyResponseBody

type Handler struct {
	DB                  db.Database
	Keys                keys.KeyService
	Auditlogs           auditlogs.AuditLogService
	Clock               clock.Clock
	InternalWorkspaceID string
	InternalKeyspaceID  string
	InternalProjectID   string
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
	var expires sql.NullTime
	if req.Expires.IsSpecified() && !req.Expires.IsNull() {
		if req.Expires.MustGet() <= h.Clock.Now().UnixMilli() {
			return fault.New("expiration must be in the future", fault.Code(codes.App.Validation.InvalidInput.URN()),
				fault.Public("expires must be a Unix millisecond timestamp in the future."))
		}
		expires = sql.NullTime{
			Time:  time.UnixMilli(req.Expires.MustGet()),
			Valid: true,
		}
	}
	if source, ok := p.Source.(principal.KeySource); ok && source.ExpiresAt != nil {
		if !expires.Valid || expires.Time.After(*source.ExpiresAt) {
			return fault.New("child root key exceeds caller expiration", fault.Code(codes.App.Validation.InvalidInput.URN()),
				fault.Public("expires is required and must not be later than the calling root key's expiration."))
		}
	}
	grants, err := authorizePermissions(ctx, p, req.Permissions)
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
	err = db.Tx(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		keyspace, err := db.Query.FindKeySpaceByID(ctx, tx, h.InternalKeyspaceID)
		if err != nil {
			return fault.Wrap(err, fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Public("Root key creation is not available."))
		}
		if h.InternalWorkspaceID == "" || h.InternalProjectID == "" || keyspace.WorkspaceID != h.InternalWorkspaceID ||
			keyspace.ProjectID != h.InternalProjectID || keyspace.DeletedAtM.Valid {
			return fault.New("invalid internal root key ownership", fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Public("Root key creation is not available."))
		}
		err = db.Query.InsertKey(ctx, tx, db.InsertKeyParams{
			ID:          keyID,
			KeySpaceID:  h.InternalKeyspaceID,
			WorkspaceID: h.InternalWorkspaceID,
			ForWorkspaceID: sql.NullString{
				String: p.AuthorizedWorkspaceID,
				Valid:  true,
			},
			Name: sql.NullString{
				String: ptr.SafeDeref(req.Name),
				Valid:  req.Name != nil,
			},
			Hash:               key.Hash,
			Prefix:             key.Prefix,
			Start:              key.Start,
			End:                key.End,
			Enabled:            true,
			CreatedAtM:         h.Clock.Now().UnixMilli(),
			Expires:            expires,
			IdentityID:         sql.NullString{},
			Meta:               sql.NullString{},
			RemainingRequests:  sql.NullInt64{},
			RefillDay:          sql.NullInt16{},
			RefillAmount:       sql.NullInt64{},
			PendingMigrationID: sql.NullString{},
		})
		if err != nil {
			return err
		}
		permissionRows := make([]db.UpsertPermissionParams, 0, len(grants))
		for _, slug := range grants {
			permissionRows = append(permissionRows, db.UpsertPermissionParams{
				PermissionID: uid.New(uid.PermissionPrefix),
				WorkspaceID:  h.InternalWorkspaceID,
				ProjectID:    h.InternalProjectID,
				Name:         slug,
				Slug:         slug,
				CreatedAtM:   h.Clock.Now().UnixMilli(),
				Description: dbtype.NullString{
					String: "",
					Valid:  false,
				},
			})
		}
		if err := db.BulkQuery.UpsertPermission(ctx, tx, permissionRows); err != nil {
			return err
		}
		permissions, err := db.Query.FindPermissionsBySlugsForUpdate(ctx, tx, db.FindPermissionsBySlugsForUpdateParams{
			WorkspaceID: h.InternalWorkspaceID,
			ProjectID:   h.InternalProjectID,
			Slugs:       grants,
		})
		if err != nil {
			return err
		}
		if len(permissions) != len(grants) {
			return fault.New("root permission belongs to another project", fault.Code(codes.App.Internal.UnexpectedError.URN()))
		}
		for _, permission := range permissions {
			if _, ok := slices.BinarySearch(grants, permission.Slug); !ok {
				return fault.New("stored permission differs from authorized permission", fault.Code(codes.App.Internal.UnexpectedError.URN()))
			}
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
			Event:         auditlog.KeyCreateEvent,
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
		keyPermissions := make([]db.InsertKeyPermissionParams, 0, len(permissions))
		for _, permission := range permissions {
			keyPermissions = append(keyPermissions, db.InsertKeyPermissionParams{
				KeyID:        keyID,
				PermissionID: permission.ID,
				WorkspaceID:  h.InternalWorkspaceID,
				CreatedAt:    h.Clock.Now().UnixMilli(),
				UpdatedAt:    sql.NullInt64{},
			})
			logs = append(logs, auditlog.AuditLog{
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
			})
		}
		if err := db.BulkQuery.InsertKeyPermissions(ctx, tx, keyPermissions); err != nil {
			return err
		}
		return h.Auditlogs.Insert(ctx, tx, logs)
	})
	if err != nil {
		return err
	}
	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: openapi.V2RootKeysCreateKeyResponseData{
			KeyId: keyID,
			Key:   key.Key,
		},
	})
}
