package handler

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/internal/services/auditlogs"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/pkg/auditlog"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type (
	Request  = openapi.V2IdentitiesDeleteIdentityRequestBody
	Response = openapi.V2IdentitiesDeleteIdentityResponseBody
)

// Handler implements zen.Route interface for the v2 identities delete identity endpoint
type Handler struct {
	// Services as public fields
	Logger    logging.Logger
	DB        db.Database
	Keys      keys.KeyService
	Auditlogs auditlogs.AuditLogService
}

// Method returns the HTTP method this route responds to
func (h *Handler) Method() string {
	return "POST"
}

// Path returns the URL path pattern this route matches
func (h *Handler) Path() string {
	return "/v2/identities.deleteIdentity"
}

// Handle processes the HTTP request
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	auth, emit, err := h.Keys.GetRootKey(ctx, s)
	defer emit()
	if err != nil {
		return err
	}

	// nolint:exhaustruct
	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	err = auth.VerifyRootKey(ctx, keys.WithPermissions(
		rbac.Or(
			rbac.T(
				rbac.Tuple{
					ResourceType: rbac.Identity,
					ResourceID:   "*",
					Action:       rbac.DeleteIdentity,
				},
			),
		),
	))
	if err != nil {
		return err
	}

	identity, err := db.Query.FindIdentity(ctx, h.DB.RO(), db.FindIdentityParams{
		WorkspaceID: auth.AuthorizedWorkspaceID,
		Identity:    req.Identity,
		Deleted:     false,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return fault.New("identity not found",
				fault.Code(codes.Data.Identity.NotFound.URN()),
				fault.Internal("identity not found"), fault.Public("This identity does not exist."),
			)
		}

		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database failed to find the identity"), fault.Public("Error finding the identity."),
		)
	}

	if identity.WorkspaceID != auth.AuthorizedWorkspaceID {
		return fault.New("identity not found",
			fault.Code(codes.Data.Identity.NotFound.URN()),
			fault.Internal("wrong workspace, masking as 404"), fault.Public("This identity does not exist."),
		)
	}

	err = db.Tx(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		err = db.Query.SoftDeleteIdentity(ctx, tx, db.SoftDeleteIdentityParams{
			WorkspaceID: auth.AuthorizedWorkspaceID,
			Identity:    identity.ID,
		})

		// If we hit a duplicate key error, we know that we have an identity that was already soft deleted
		// so we can hard delete the "old" deleted version
		if db.IsDuplicateKeyError(err) {
			// Delete the old soft-deleted identity and its ratelimits
			err = db.Query.DeleteOldIdentityWithRatelimits(ctx, tx, db.DeleteOldIdentityWithRatelimitsParams{
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Identity:    req.Identity,
			})
			if err != nil {
				return fault.Wrap(err,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("database failed to delete old soft-deleted identity"),
					fault.Public("Failed to delete old deleted identity."),
				)
			}

			// Re-apply the soft delete operation
			err = db.Query.SoftDeleteIdentity(ctx, tx, db.SoftDeleteIdentityParams{
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Identity:    identity.ID,
			})
		}

		if err != nil {
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database failed to soft delete identity"), fault.Public("Failed to delete Identity."),
			)
		}

		auditLogs := []auditlog.AuditLog{
			{
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Event:       auditlog.IdentityDeleteEvent,
				Display:     fmt.Sprintf("Deleted identity %s.", identity.ID),
				ActorID:     auth.Key.ID,
				ActorType:   auditlog.RootKeyActor,
				ActorName:   "root key",
				ActorMeta:   map[string]any{},
				RemoteIP:    s.Location(),
				UserAgent:   s.UserAgent(),
				Resources: []auditlog.AuditLogResource{
					{
						ID:          identity.ID,
						Meta:        nil,
						Type:        auditlog.IdentityResourceType,
						DisplayName: identity.ExternalID,
						Name:        identity.ExternalID,
					},
				},
			},
		}

		ratelimits, listErr := db.Query.ListIdentityRatelimitsByID(ctx, tx, sql.NullString{String: identity.ID, Valid: true})
		if listErr != nil {
			return fault.Wrap(listErr,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database failed to load identity ratelimits"), fault.Public("Failed to load Identity ratelimits."),
			)
		}

		for _, rl := range ratelimits {
			auditLogs = append(auditLogs, auditlog.AuditLog{
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Event:       auditlog.RatelimitDeleteEvent,
				Display:     fmt.Sprintf("Deleted ratelimit %s.", rl.ID),
				ActorID:     auth.Key.ID,
				ActorType:   auditlog.RootKeyActor,
				ActorName:   "root key",
				ActorMeta:   map[string]any{},
				RemoteIP:    s.Location(),
				UserAgent:   s.UserAgent(),
				Resources: []auditlog.AuditLogResource{
					{
						Type:        auditlog.IdentityResourceType,
						Meta:        nil,
						ID:          identity.ID,
						DisplayName: identity.ExternalID,
						Name:        identity.ExternalID,
					},
					{
						Type:        auditlog.RatelimitResourceType,
						Meta:        nil,
						ID:          rl.ID,
						DisplayName: rl.Name,
						Name:        rl.Name,
					},
				},
			})
		}

		err = h.Auditlogs.Insert(ctx, tx, auditLogs)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
	})
}
