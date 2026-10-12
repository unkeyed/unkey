package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	apierrors "github.com/unkeyed/unkey/svc/api/internal/errors"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2IdentitiesDeleteIdentityRequestBody
	Response = openapi.V2IdentitiesDeleteIdentityResponseBody
)

// Handler implements zen.Route interface for the v2 identities delete identity endpoint
type Handler struct {
	DB        db.Database
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
	// Mint a correlation ID so the identity.delete audit event plus any
	// cascade events (per-ratelimit cleanup, per-key disconnects) share
	// one ID for dashboard drill-down.
	ctx = auditlog.WithCorrelation(ctx, auditlog.NewCorrelationID())

	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}

	// nolint:exhaustruct
	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	identity, err := db.Query.FindIdentity(ctx, h.DB.RO(), db.FindIdentityParams{
		WorkspaceID: principal.AuthorizedWorkspaceID,
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

	err = principal.Authorize(
		rbac.Or(
			rbac.T(
				rbac.Tuple{
					ResourceType: rbac.Identity,
					ResourceID:   "*",
					Action:       rbac.DeleteIdentity,
				},
			),
			rbac.U(
				urn.New().Workspace(principal.AuthorizedWorkspaceID).Project(identity.ProjectID).Identity(identity.ID),
				permissions.Delete,
			),
		),
	)
	if err != nil {
		return apierrors.MaskInsufficientPermissionsAsNotFound(
			err,
			codes.Data.Identity.NotFound.URN(),
			"This identity does not exist.",
		)
	}

	// Parse ratelimits JSON
	var ratelimits []db.RatelimitInfo
	if ratelimitBytes, ok := identity.Ratelimits.([]byte); ok && ratelimitBytes != nil {
		_ = json.Unmarshal(ratelimitBytes, &ratelimits) // Ignore error, default to empty array
	}

	err = db.TxRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		_, err := db.Query.LockIdentityForUpdate(ctx, tx, identity.ID)
		if err != nil {
			return fault.Wrap(err, fault.Internal("unable to lock identity for deletion"))
		}
		_, err = db.Query.FindIdentityByID(ctx, tx, db.FindIdentityByIDParams{
			WorkspaceID: principal.AuthorizedWorkspaceID,
			IdentityID:  identity.ID,
			Deleted:     true,
		})
		if err == nil {
			return nil
		}
		if !db.IsNotFound(err) {
			return fault.Wrap(err, fault.Internal("unable to check identity deletion state"))
		}

		var oldResources []db.FindOldIdentityResourcesForUpdateRow
		err = db.Query.SoftDeleteIdentity(ctx, tx, db.SoftDeleteIdentityParams{
			WorkspaceID: principal.AuthorizedWorkspaceID,
			IdentityID:  identity.ID,
		})

		// If we hit a duplicate key error, we know that we have an identity that was already soft deleted
		// so we can hard delete the "old" deleted version
		if db.IsDuplicateKeyError(err) {
			oldResources, err = db.Query.FindOldIdentityResourcesForUpdate(ctx, tx, db.FindOldIdentityResourcesForUpdateParams{
				WorkspaceID:       principal.AuthorizedWorkspaceID,
				ExternalID:        identity.ExternalID,
				CurrentIdentityID: identity.ID,
			})
			if err != nil {
				return fault.Wrap(err, fault.Internal("unable to lock old identity resources"))
			}
			var expectedDeleted int64
			if len(oldResources) > 0 {
				expectedDeleted = 1
			}
			for _, resource := range oldResources {
				if resource.RatelimitID.Valid {
					expectedDeleted++
				}
			}

			// Delete the old soft-deleted identity with the same external_id, excluding the current one
			deleted, deleteErr := db.Query.DeleteOldIdentityByExternalID(ctx, tx, db.DeleteOldIdentityByExternalIDParams{
				WorkspaceID:       principal.AuthorizedWorkspaceID,
				ExternalID:        identity.ExternalID,
				CurrentIdentityID: identity.ID,
			})
			if deleteErr != nil {
				return fault.Wrap(deleteErr,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("database failed to delete old soft-deleted identity"),
					fault.Public("Failed to delete old deleted identity."),
				)
			}
			if err = assert.Equal(deleted, expectedDeleted, "identity cleanup must match locked audit targets"); err != nil {
				return err
			}

			// Re-apply the soft delete operation
			err = db.Query.SoftDeleteIdentity(ctx, tx, db.SoftDeleteIdentityParams{
				WorkspaceID: principal.AuthorizedWorkspaceID,
				IdentityID:  identity.ID,
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
				WorkspaceID:   principal.AuthorizedWorkspaceID,
				Event:         auditlog.IdentityDeleteEvent,
				Display:       fmt.Sprintf("Deleted identity %s.", identity.ID),
				ActorID:       principal.Subject.ID,
				ActorType:     auditlog.AuditLogActor(principal.Subject.Type),
				ActorName:     principal.Subject.Name,
				ActorMeta:     map[string]any{},
				RemoteIP:      s.Location(),
				UserAgent:     s.UserAgent(),
				CorrelationID: "",
				Targets: []auditlog.AuditLogTarget{
					{
						ID:          identity.ID,
						Meta:        nil,
						Type:        auditlog.IdentityTargetType,
						DisplayName: identity.ExternalID,
						Name:        identity.ExternalID,
					},
				},
			},
		}

		for _, rl := range ratelimits {
			auditLogs = append(auditLogs, auditlog.AuditLog{
				WorkspaceID:   principal.AuthorizedWorkspaceID,
				Event:         auditlog.RatelimitDeleteEvent,
				Display:       fmt.Sprintf("Deleted ratelimit %s.", rl.ID),
				ActorID:       principal.Subject.ID,
				ActorType:     auditlog.AuditLogActor(principal.Subject.Type),
				ActorName:     principal.Subject.Name,
				ActorMeta:     map[string]any{},
				RemoteIP:      s.Location(),
				UserAgent:     s.UserAgent(),
				CorrelationID: "",
				Targets: []auditlog.AuditLogTarget{
					{
						Type:        auditlog.IdentityTargetType,
						Meta:        nil,
						ID:          identity.ID,
						DisplayName: identity.ExternalID,
						Name:        identity.ExternalID,
					},
					{
						Type:        auditlog.RatelimitTargetType,
						Meta:        nil,
						ID:          rl.ID,
						DisplayName: rl.Name,
						Name:        rl.Name,
					},
				},
			})
		}

		if len(oldResources) > 0 {
			oldIdentity := auditlog.AuditLogTarget{
				ID:          oldResources[0].ID,
				Meta:        nil,
				Type:        auditlog.IdentityTargetType,
				DisplayName: oldResources[0].ExternalID,
				Name:        oldResources[0].ExternalID,
			}
			cleanup := auditLogs[0]
			cleanup.Display = fmt.Sprintf("Deleted identity %s.", oldIdentity.ID)
			cleanup.Targets = []auditlog.AuditLogTarget{oldIdentity}
			auditLogs = append(auditLogs, cleanup)
			for _, resource := range oldResources {
				if !resource.RatelimitID.Valid {
					continue
				}
				cleanup.Event = auditlog.RatelimitDeleteEvent
				cleanup.Display = fmt.Sprintf("Deleted ratelimit %s.", resource.RatelimitID.String)
				cleanup.Targets = []auditlog.AuditLogTarget{
					oldIdentity,
					{
						ID:          resource.RatelimitID.String,
						Meta:        nil,
						Type:        auditlog.RatelimitTargetType,
						DisplayName: resource.RatelimitName.String,
						Name:        resource.RatelimitName.String,
					},
				}
				auditLogs = append(auditLogs, cleanup)
			}
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
