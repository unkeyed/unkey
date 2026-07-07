package handler

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/internal/services/auditlogs"
	keysdb "github.com/unkeyed/unkey/internal/services/keys/db"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/auditactor"
	"github.com/unkeyed/unkey/svc/api/internal/portalscope"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type Request = openapi.V2KeysDeleteKeyRequestBody
type Response = openapi.V2KeysDeleteKeyResponseBody

// Handler serves the portal-scoped variant of keys.deleteKey. It authenticates
// only portal sessions and may only delete keys owned by the session's external
// identity.
//
// This is a deliberate copy of the keys.deleteKey handler rather than a wrapper:
// the two share only a small delete-and-audit tail, while the ownership scoping
// is genuinely different behavior. Duplicating a handler this size keeps each
// one readable and independently evolvable; the alternative (threading the
// session and a scope flag through a shared core) is the wrong abstraction.
type Handler struct {
	DB        db.Database
	Auditlogs auditlogs.AuditLogService
	KeyCache  cache.Cache[string, keysdb.CachedKeyData]
}

// Method returns the HTTP method this route responds to.
func (h *Handler) Method() string { return "POST" }

// Path returns the URL path pattern this route matches.
func (h *Handler) Path() string { return "/v2/portal.deleteKey" }

// Handle deletes a key scoped to the portal session's external identity.
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}

	// The portal-only authenticator guarantees a portal session here; this also
	// yields the external identity the delete is scoped to.
	externalID, err := portalscope.ExternalID(s)
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	key, err := db.Query.FindLiveKeyByID(ctx, h.DB.RO(), req.KeyId)
	if err != nil {
		if db.IsNotFound(err) {
			return fault.Wrap(
				err,
				fault.Code(codes.Data.Key.NotFound.URN()),
				fault.Internal("key does not exist"),
				fault.Public("We could not find the requested key."),
			)
		}

		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve Key information."),
		)
	}

	// Validate key belongs to authorized workspace.
	if key.WorkspaceID != principal.WorkspaceID {
		return fault.New("key not found",
			fault.Code(codes.Data.Key.NotFound.URN()),
			fault.Internal("key belongs to different workspace"),
			fault.Public("The specified key was not found."),
		)
	}

	// Fail closed: a portal caller may only delete keys owned by its own external
	// identity. If the key has no identity, or the identity does not match, return
	// 404 so the caller cannot probe for keys it does not own. This scoping is
	// intentionally separate from RBAC: permissions gate what operations a
	// principal may perform; identity scoping gates which keys are visible.
	if !key.IdentityExternalID.Valid || key.IdentityExternalID.String != externalID {
		return fault.New("key not found",
			fault.Code(codes.Data.Key.NotFound.URN()),
			fault.Internal("key identity does not match portal session externalId"),
			fault.Public("The specified key was not found."),
		)
	}

	// Portal-authenticated deletes are attributed to a portalEndUser actor so
	// customers can see end-user activity in their audit logs.
	actor := auditactor.FromPrincipal(principal)

	err = principal.Authorize(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Api,
			ResourceID:   "*",
			Action:       rbac.DeleteKey,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Api,
			ResourceID:   key.Api.ID,
			Action:       rbac.DeleteKey,
		}),
		rbac.U(
			urn.New().Workspace(principal.WorkspaceID).Keyspace(key.KeyAuthID).Key(req.KeyId),
			permissions.DeleteKey{},
		),
	))
	if err != nil {
		return err
	}

	err = db.TxRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) (err error) {
		description := "Deleted"
		if ptr.SafeDeref(req.Permanent) {
			err = db.Query.DeleteKeyByID(ctx, tx, req.KeyId)
			description = "Permanently deleted"
		} else {
			err = db.Query.SoftDeleteKeyByID(ctx, tx, db.SoftDeleteKeyByIDParams{
				Now: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
				ID:  req.KeyId,
			})
		}

		if err != nil {
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error"),
				fault.Public("Failed to delete key."),
			)
		}

		return h.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{
			{
				Event:         auditlog.KeyDeleteEvent,
				WorkspaceID:   principal.WorkspaceID,
				ActorType:     actor.Type,
				ActorID:       actor.ID,
				ActorName:     actor.Name,
				ActorMeta:     actor.Meta,
				Display:       fmt.Sprintf("%s %s", description, key.ID),
				RemoteIP:      s.Location(),
				UserAgent:     s.UserAgent(),
				CorrelationID: "",
				Resources: []auditlog.AuditLogResource{
					{
						ID:          key.ID,
						DisplayName: key.Name.String,
						Name:        key.Name.String,
						Meta:        map[string]any{},
						Type:        auditlog.KeyResourceType,
					},
				},
			},
		})
	})
	if err != nil {
		return err
	}

	h.KeyCache.Remove(ctx, key.Hash)

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.EmptyResponse{},
	})
}
