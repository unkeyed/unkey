// Package handler implements the API endpoint for deleting an identity in the Unkey system.
//
// OVERVIEW:
// This handler implements the POST /v2/identities.deleteIdentity endpoint which allows
// authorized users to delete identities from the system. The deletion is performed as a
// soft delete to maintain data integrity and audit history.
//
// FLOW DIAGRAM:
//
//	+----------------+     +-------------+     +----------------+     +--------------+
//	|  Verify Key    |---->| Check Perms |---->| Get Identity   |---->| Begin Tx     |
//	+----------------+     +-------------+     +----------------+     +--------------+
//	                                                                         |
//	                                                                         v
//	+----------------+     +-------------+     +----------------+     +--------------+
//	| Return 200    |<----| Commit Tx   |<----| Create Audits  |<----| Soft Delete  |
//	+----------------+     +-------------+     +----------------+     +--------------+
//
// DETAILED PROCESS:
// 1. Authentication: Verifies the root key for API access
// 2. Permission Verification: Checks if the key has permission to delete identities
//   - Checks for general identity deletion permission (*)
//   - Checks for specific identity deletion permission if ID provided
//
// 3. Identity Retrieval: Gets the identity by either:
//
//	+---------------+     +-----------------------+
//	| Identity ID   |---->| FindIdentityByID     |
//	+---------------+     +-----------------------+
//	+---------------+     +-----------------------+
//	| External ID   |---->| FindIdentityByExtID  |
//	+---------------+     +-----------------------+
//
// 4. Soft Deletion Process:
//
//	+----------------+
//	| Soft Delete    |
//	+--------+-------+
//	         |
//	         v
//	+---------------------------+       +------------------------+
//	| Duplicate Key Error?      |--Yes->| Delete Old Soft-       |
//	+-----------------+---------+       | Deleted Identity       |
//	                |                   +----------+-------------+
//	                No                            |
//	                |                             |
//	                v                             v
//	+--------------------------+       +------------------------+
//	| Create Audit Logs         |<-----| Retry Soft Delete      |
//	+---------------------------+      +------------------------+
//
// 5. Audit Logging: Creates logs for:
//   - The deleted identity
//   - Any rate limits associated with the identity
//
// 6. Transaction Management:
//   - All database operations are wrapped in a transaction
//   - Rollback occurs automatically if any operation fails
//   - Commit only happens after all operations succeed
//
// ERROR HANDLING:
// - Authentication failures result in auth errors
// - Permission failures result in authorization errors
// - Database errors are wrapped with appropriate error codes and descriptions
// - Not Found errors are returned when identity doesn't exist or belongs to wrong workspace
package handler

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/internal/services/auditlogs"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/internal/services/permissions"
	"github.com/unkeyed/unkey/go/pkg/auditlog"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Request = openapi.V2IdentitiesDeleteIdentityRequestBody
type Response = openapi.V2IdentitiesDeleteIdentityResponseBody

// Handler implements zen.Route interface for the v2 identities delete identity endpoint
type Handler struct {
	// Services as public fields
	Logger      logging.Logger
	DB          db.Database
	Keys        keys.KeyService
	Permissions permissions.PermissionService
	Auditlogs   auditlogs.AuditLogService
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
	auth, err := h.Keys.VerifyRootKey(ctx, s)
	if err != nil {
		return err
	}

	// nolint:exhaustruct
	req := Request{}
	err = s.BindBody(&req)
	if err != nil {
		return fault.Wrap(err,
			fault.Internal("invalid request body"), fault.Public("The request body is invalid."),
		)
	}

	checks := []rbac.PermissionQuery{
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Identity,
			ResourceID:   "*",
			Action:       rbac.DeleteIdentity,
		}),
	}

	if req.IdentityId != nil {
		checks = append(checks, rbac.T(rbac.Tuple{
			ResourceType: rbac.Identity,
			ResourceID:   *req.IdentityId,
			Action:       rbac.DeleteIdentity,
		}))
	}

	err = h.Permissions.Check(
		ctx,
		auth.KeyID,
		rbac.Or(checks...),
	)
	if err != nil {
		return err
	}

	identity, err := h.getIdentity(ctx, req, auth.AuthorizedWorkspaceID)
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

	_, err = db.Tx(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) (interface{}, error) {
		err := db.Query.SoftDeleteIdentity(ctx, tx, identity.ID)

		// If we hit a duplicate key error, we know that we have an identity that was already soft deleted
		// so we can hard delete the "old" deleted version
		if db.IsDuplicateKeyError(err) {
			err = deleteOldIdentity(ctx, tx, auth.AuthorizedWorkspaceID, identity.ExternalID)
			if err != nil {
				return nil, err
			}

			// Re-apply the soft delete operation
			err = db.Query.SoftDeleteIdentity(ctx, tx, identity.ID)

		}
		if err != nil {

			return nil, fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database failed to soft delete identity"), fault.Public("Failed to delete Identity."),
			)
		}

		auditLogs := []auditlog.AuditLog{
			{
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Event:       auditlog.IdentityDeleteEvent,
				Display:     fmt.Sprintf("Deleted identity %s.", identity.ID),
				Bucket:      auditlogs.DEFAULT_BUCKET,
				ActorID:     auth.KeyID,
				ActorType:   auditlog.RootKeyActor,
				ActorName:   "root key",
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

		ratelimits, err := db.Query.ListIdentityRatelimitsByID(ctx, tx, sql.NullString{String: identity.ID, Valid: true})
		if err != nil {
			return nil, fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database failed to load identity ratelimits"), fault.Public("Failed to load Identity ratelimits."),
			)
		}

		for _, rl := range ratelimits {
			auditLogs = append(auditLogs, auditlog.AuditLog{
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Event:       auditlog.RatelimitDeleteEvent,
				Display:     fmt.Sprintf("Deleted ratelimit %s.", rl.ID),
				Bucket:      auditlogs.DEFAULT_BUCKET,
				ActorID:     auth.KeyID,
				ActorType:   auditlog.RootKeyActor,
				ActorName:   "root key",
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
			return nil, fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database failed to insert audit logs"), fault.Public("Failed to insert audit logs"),
			)
		}

		return nil, nil
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

func deleteOldIdentity(ctx context.Context, tx db.DBTX, workspaceID, externalID string) error {
	oldIdentity, err := db.Query.FindIdentityByExternalID(ctx, tx, db.FindIdentityByExternalIDParams{
		WorkspaceID: workspaceID,
		ExternalID:  externalID,
		Deleted:     true,
	})
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database failed to load old identity"), fault.Public("Failed to load Identity."),
		)
	}

	err = db.Query.DeleteManyRatelimitsByIdentityID(ctx, tx, sql.NullString{String: oldIdentity.ID, Valid: true})
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database failed to delete identity ratelimits"), fault.Public("Failed to delete Identity ratelimits."),
		)
	}

	err = db.Query.DeleteIdentity(ctx, tx, oldIdentity.ID)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database failed to delete identity"), fault.Public("Failed to delete Identity."),
		)
	}

	return nil
}

func (h *Handler) getIdentity(ctx context.Context, req Request, workspaceID string) (db.Identity, error) {
	switch {
	case req.IdentityId != nil:
		return db.Query.FindIdentityByID(ctx, h.DB.RO(), db.FindIdentityByIDParams{
			ID:      *req.IdentityId,
			Deleted: false,
		})
	case req.ExternalId != nil:
		return db.Query.FindIdentityByExternalID(ctx, h.DB.RO(), db.FindIdentityByExternalIDParams{
			WorkspaceID: workspaceID,
			ExternalID:  *req.ExternalId,
			Deleted:     false,
		})
	}

	return db.Identity{}, fault.New("missing identity id or external id",
		fault.Code(codes.App.Validation.InvalidInput.URN()),
		fault.Internal("missing identity id or external id"), fault.Public("You must provide either an identity ID or external ID."),
	)
}
