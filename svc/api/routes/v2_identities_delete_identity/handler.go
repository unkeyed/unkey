package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
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

	// Parse ratelimits JSON
	var ratelimits []db.RatelimitInfo
	if ratelimitBytes, ok := identity.Ratelimits.([]byte); ok && ratelimitBytes != nil {
		if unmarshalErr := json.Unmarshal(ratelimitBytes, &ratelimits); unmarshalErr != nil {
			return fault.Wrap(unmarshalErr,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("failed to parse identity ratelimits"),
				fault.Public("We're unable to process the identity's ratelimits."),
			)
		}
	}

	err = db.TxRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		err = db.Query.SoftDeleteIdentity(ctx, tx, db.SoftDeleteIdentityParams{
			WorkspaceID: auth.AuthorizedWorkspaceID,
			Identity:    identity.ID,
		})

		// If we hit a duplicate key error, we know that we have an identity that was already soft deleted
		// so we can hard delete the "old" deleted version
		if db.IsDuplicateKeyError(err) {
			// Check if this identity is already soft-deleted (could happen with concurrent requests)
			alreadyDeleted, checkErr := db.Query.FindIdentityByID(ctx, tx, db.FindIdentityByIDParams{
				WorkspaceID: auth.AuthorizedWorkspaceID,
				IdentityID:  identity.ID,
				Deleted:     true,
			})
			if checkErr == nil && alreadyDeleted.ID == identity.ID {
				// Identity is already soft-deleted, this is idempotent, return success
				// Skip audit logs - they were already created by the request that deleted it
				return nil
			}

			// Delete the old soft-deleted identity with the same external_id, excluding the current one
			err = db.Query.DeleteOldIdentityByExternalID(ctx, tx, db.DeleteOldIdentityByExternalIDParams{
				WorkspaceID:       auth.AuthorizedWorkspaceID,
				ExternalID:        identity.ExternalID,
				CurrentIdentityID: identity.ID,
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
			if err != nil {
				// If we still get a duplicate key error after deleting the old identity,
				// it means another concurrent request already soft-deleted this identity.
				// This is safe to treat as success (idempotent operation).
				// Skip audit logs - they were already created by the concurrent request
				if db.IsDuplicateKeyError(err) {
					return nil // Skip audit logs
				}
			}
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
