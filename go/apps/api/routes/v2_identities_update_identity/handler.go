package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

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
	"github.com/unkeyed/unkey/go/pkg/uid"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Request = openapi.V2IdentitiesUpdateIdentityRequestBody
type Response = openapi.V2IdentitiesUpdateIdentityResponseBody

type Services struct {
	Logger      logging.Logger
	DB          db.Database
	Keys        keys.KeyService
	Permissions permissions.PermissionService
	Auditlogs   auditlogs.AuditLogService
}

const (
	// Planetscale has a limit on JSON field size
	maxMetaLengthMB = 1
)

func New(svc Services) zen.Route {
	return zen.NewRoute("POST", "/v2/identities.updateIdentity", func(ctx context.Context, s *zen.Session) error {
		auth, err := svc.Keys.VerifyRootKey(ctx, s)
		if err != nil {
			return err
		}

		// Parse request
		req := Request{}
		err = s.BindBody(&req)
		if err != nil {
			return fault.Wrap(err,
				fault.Internal("invalid request body"), fault.Public("The request body is invalid."),
			)
		}

		// Validate that at least one of identityID or externalID is provided
		if req.IdentityId == nil && req.ExternalId == nil {
			return fault.New("missing required field",
				fault.Code(codes.App.Validation.InvalidInput.URN()),
				fault.Internal("missing required field"), fault.Public("Provide either identityId or externalId"),
			)
		}

		// Check permissions
		var identityIdForPermissions string
		if req.IdentityId != nil {
			identityIdForPermissions = *req.IdentityId
		}

		err = svc.Permissions.Check(
			ctx,
			auth.KeyID,
			rbac.Or(
				rbac.T(rbac.Tuple{
					ResourceType: rbac.Identity,
					ResourceID:   "*",
					Action:       rbac.UpdateIdentity,
				}),
				rbac.T(rbac.Tuple{
					ResourceType: rbac.Identity,
					ResourceID:   identityIdForPermissions,
					Action:       rbac.UpdateIdentity,
				}),
			),
		)
		if err != nil {
			return err
		}

		// Check ratelimits for unique names
		if req.Ratelimits != nil {
			nameSet := make(map[string]bool)
			for _, ratelimit := range *req.Ratelimits {
				if _, exists := nameSet[ratelimit.Name]; exists {
					return fault.New("duplicate ratelimit name",
						fault.Code(codes.App.Validation.InvalidInput.URN()),
						fault.Internal("duplicate ratelimit name"), fault.Public(fmt.Sprintf("Ratelimit with name \"%s\" is already defined in the request", ratelimit.Name)),
					)
				}
				nameSet[ratelimit.Name] = true
			}
		}

		// Check metadata size
		var metaBytes []byte
		if req.Meta != nil {
			var metaErr error
			metaBytes, metaErr = json.Marshal(*req.Meta)
			if metaErr != nil {
				return fault.Wrap(metaErr,
					fault.Code(codes.App.Validation.InvalidInput.URN()),
					fault.Internal("unable to marshal metadata"), fault.Public("We're unable to marshal the meta object."),
				)
			}

			sizeInMB := float64(len(metaBytes)) / 1024 / 1024
			if sizeInMB > maxMetaLengthMB {
				return fault.New("metadata is too large",
					fault.Code(codes.App.Validation.InvalidInput.URN()),
					fault.Internal("metadata is too large"), fault.Public(fmt.Sprintf("Metadata is too large, it must be less than %dMB, got: %.2f", maxMetaLengthMB, sizeInMB)),
				)
			}
		}

		tx, err := svc.DB.RW().Begin(ctx)
		if err != nil {
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database failed to create transaction"), fault.Public("Unable to start database transaction."),
			)
		}

		defer func() {
			rollbackErr := tx.Rollback()
			if rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
				svc.Logger.Error("rollback failed", "requestId", s.RequestID(), "error", rollbackErr)
			}
		}()

		// Get the identity
		var identity db.Identity
		var existingRatelimits []db.Ratelimit

		if req.IdentityId != nil {
			// Find by identity ID
			identity, err = db.Query.FindIdentityByID(ctx, tx, db.FindIdentityByIDParams{
				ID:      *req.IdentityId,
				Deleted: false,
			})
		} else {
			// Find by external ID
			identity, err = db.Query.FindIdentityByExternalID(ctx, tx, db.FindIdentityByExternalIDParams{
				ExternalID:  *req.ExternalId,
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Deleted:     false,
			})
		}

		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fault.New("identity not found",
					fault.Code(codes.Data.Identity.NotFound.URN()),
					fault.Internal("identity not found"), fault.Public("Identity not found in this workspace"),
				)
			}
			return fault.Wrap(err,
				fault.Internal("unable to find identity"), fault.Public("We're unable to retrieve the identity."),
			)
		}

		// Verify workspace
		if identity.WorkspaceID != auth.AuthorizedWorkspaceID {
			return fault.New("identity not found",
				fault.Code(codes.Data.Identity.NotFound.URN()),
				fault.Internal("wrong workspace, masking as 404"), fault.Public("Identity not found in this workspace"),
			)
		}

		// Get existing ratelimits
		existingRatelimits, err = db.Query.FindRatelimitsByIdentityID(ctx, tx, sql.NullString{String: identity.ID, Valid: true})
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fault.Wrap(err,
				fault.Internal("unable to fetch ratelimits"), fault.Public("We're unable to retrieve the identity's ratelimits."),
			)
		}

		// Create the base audit log
		auditLogs := []auditlog.AuditLog{
			{
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Event:       auditlog.IdentityUpdateEvent,
				Display:     fmt.Sprintf("Updated identity %s", identity.ID),
				ActorID:     auth.KeyID,
				ActorName:   "root key",
				ActorType:   auditlog.RootKeyActor,
				RemoteIP:    s.Location(),
				UserAgent:   s.UserAgent(),
				Bucket:      auditlogs.DEFAULT_BUCKET,
				Resources: []auditlog.AuditLogResource{
					{
						ID:          identity.ID,
						Type:        auditlog.IdentityResourceType,
						Name:        identity.ExternalID,
						DisplayName: identity.ExternalID,
					},
				},
			},
		}

		// Update metadata if provided
		if req.Meta != nil {
			err = db.Query.UpdateIdentity(ctx, tx, db.UpdateIdentityParams{
				ID:   identity.ID,
				Meta: metaBytes,
			})
			if err != nil {
				return fault.Wrap(err,
					fault.Internal("unable to update metadata"), fault.Public("We're unable to update the identity's metadata."),
				)
			}
		}

		// Handle ratelimits if provided
		if req.Ratelimits != nil {
			// Process ratelimits changes
			// 1. Delete ratelimits that no longer exist
			// 2. Update existing ratelimits
			// 3. Create new ratelimits

			// Create maps to easily find existing and new ratelimits by name
			existingRatelimitMap := make(map[string]db.Ratelimit)
			for _, rl := range existingRatelimits {
				existingRatelimitMap[rl.Name] = rl
			}

			newRatelimitMap := make(map[string]openapi.Ratelimit)
			if req.Ratelimits != nil {
				for _, rl := range *req.Ratelimits {
					newRatelimitMap[rl.Name] = rl
				}
			}

			// Delete ratelimits that are not in the new list
			for _, existingRL := range existingRatelimits {
				if _, exists := newRatelimitMap[existingRL.Name]; !exists {
					// Delete this ratelimit
					err = db.Query.DeleteRatelimit(ctx, tx, existingRL.ID)
					if err != nil {
						return fault.Wrap(err,
							fault.Internal("unable to delete ratelimit"), fault.Public("We're unable to delete a ratelimit."),
						)
					}

					// Add audit log for deletion
					auditLogs = append(auditLogs, auditlog.AuditLog{
						WorkspaceID: auth.AuthorizedWorkspaceID,
						Event:       auditlog.RatelimitDeleteEvent,
						Display:     fmt.Sprintf("Deleted ratelimit %s", existingRL.ID),
						ActorID:     auth.KeyID,
						ActorName:   "root key",
						ActorType:   auditlog.RootKeyActor,
						RemoteIP:    s.Location(),
						UserAgent:   s.UserAgent(),
						Bucket:      auditlogs.DEFAULT_BUCKET,
						Resources: []auditlog.AuditLogResource{
							{
								ID:          identity.ID,
								Type:        auditlog.IdentityResourceType,
								Name:        identity.ExternalID,
								DisplayName: identity.ExternalID,
							},
							{
								ID:          existingRL.ID,
								Type:        auditlog.RatelimitResourceType,
								Name:        existingRL.Name,
								DisplayName: existingRL.Name,
							},
						},
					})
				}
			}

			// Update existing ratelimits or create new ones
			for name, newRL := range newRatelimitMap {
				if existingRL, exists := existingRatelimitMap[name]; exists {
					// Update this ratelimit
					err = db.Query.UpdateRatelimit(ctx, tx, db.UpdateRatelimitParams{
						ID:       existingRL.ID,
						Name:     newRL.Name,
						Limit:    int32(newRL.Limit),
						Duration: newRL.Duration,
					})
					if err != nil {
						return fault.Wrap(err,
							fault.Internal("unable to update ratelimit"), fault.Public("We're unable to update a ratelimit."),
						)
					}

					// Add audit log for update
					auditLogs = append(auditLogs, auditlog.AuditLog{
						WorkspaceID: auth.AuthorizedWorkspaceID,
						Event:       auditlog.RatelimitUpdateEvent,
						Display:     fmt.Sprintf("Updated ratelimit %s", existingRL.ID),
						ActorID:     auth.KeyID,
						ActorName:   "root key",
						ActorType:   auditlog.RootKeyActor,
						RemoteIP:    s.Location(),
						UserAgent:   s.UserAgent(),
						Bucket:      auditlogs.DEFAULT_BUCKET,
						Resources: []auditlog.AuditLogResource{
							{
								ID:          identity.ID,
								Type:        auditlog.IdentityResourceType,
								Name:        identity.ExternalID,
								DisplayName: identity.ExternalID,
							},
							{
								ID:          existingRL.ID,
								Type:        auditlog.RatelimitResourceType,
								Name:        newRL.Name,
								DisplayName: newRL.Name,
							},
						},
					})
				} else {
					// Create new ratelimit
					ratelimitID := uid.New(uid.RatelimitPrefix)
					err = db.Query.InsertIdentityRatelimit(ctx, tx, db.InsertIdentityRatelimitParams{
						ID:          ratelimitID,
						WorkspaceID: auth.AuthorizedWorkspaceID,
						IdentityID:  sql.NullString{String: identity.ID, Valid: true},
						Name:        newRL.Name,
						Limit:       int32(newRL.Limit),
						Duration:    newRL.Duration,
						CreatedAt:   time.Now().UnixMilli(),
					})
					if err != nil {
						return fault.Wrap(err,
							fault.Internal("unable to create ratelimit"), fault.Public("We're unable to create a new ratelimit."),
						)
					}

					// Add audit log for creation
					auditLogs = append(auditLogs, auditlog.AuditLog{
						WorkspaceID: auth.AuthorizedWorkspaceID,
						Event:       auditlog.RatelimitCreateEvent,
						Display:     fmt.Sprintf("Created ratelimit %s", ratelimitID),
						ActorID:     auth.KeyID,
						ActorName:   "root key",
						ActorType:   auditlog.RootKeyActor,
						RemoteIP:    s.Location(),
						UserAgent:   s.UserAgent(),
						Bucket:      auditlogs.DEFAULT_BUCKET,
						Resources: []auditlog.AuditLogResource{
							{
								ID:          identity.ID,
								Type:        auditlog.IdentityResourceType,
								Name:        identity.ExternalID,
								DisplayName: identity.ExternalID,
							},
							{
								ID:          ratelimitID,
								Type:        auditlog.RatelimitResourceType,
								Name:        newRL.Name,
								DisplayName: newRL.Name,
							},
						},
					})
				}
			}
		}

		// Insert audit logs
		err = svc.Auditlogs.Insert(ctx, tx, auditLogs)
		if err != nil {
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database failed to insert audit logs"), fault.Public("Failed to insert audit logs"),
			)
		}

		// Commit transaction
		err = tx.Commit()
		if err != nil {
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database failed to commit transaction"), fault.Public("Failed to commit changes."),
			)
		}

		// Get updated identity with ratelimits
		updatedIdentity, err := db.Query.FindIdentityByID(ctx, svc.DB.RO(), db.FindIdentityByIDParams{
			ID:      identity.ID,
			Deleted: false,
		})
		if err != nil {
			return fault.Wrap(err,
				fault.Internal("unable to get updated identity"), fault.Public("We were able to update the identity but unable to retrieve the updated data."),
			)
		}

		updatedRatelimits, err := db.Query.FindRatelimitsByIdentityID(ctx, svc.DB.RO(), sql.NullString{String: identity.ID, Valid: true})
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fault.Wrap(err,
				fault.Internal("unable to fetch updated ratelimits"), fault.Public("We were able to update the identity but unable to retrieve the updated ratelimits."),
			)
		}

		// Format ratelimits for response
		responseRatelimits := make([]openapi.Ratelimit, 0, len(updatedRatelimits))
		for _, r := range updatedRatelimits {
			responseRatelimits = append(responseRatelimits, openapi.Ratelimit{
				Name:     r.Name,
				Limit:    int64(r.Limit),
				Duration: r.Duration,
			})
		}

		// Parse metadata
		var responseMeta *map[string]interface{}
		if updatedIdentity.Meta != nil && len(updatedIdentity.Meta) > 0 {
			metaMap := make(map[string]interface{})
			err = json.Unmarshal(updatedIdentity.Meta, &metaMap)
			if err != nil {
				return fault.Wrap(err,
					fault.Internal("unable to unmarshal metadata"), fault.Public("We're unable to parse the identity's metadata."),
				)
			}
			responseMeta = &metaMap
		}

		// Build response
		response := Response{
			Meta: openapi.Meta{
				RequestId: s.RequestID(),
			},
			Data: openapi.IdentitiesUpdateIdentityResponseData{
				Id:         updatedIdentity.ID,
				ExternalId: updatedIdentity.ExternalID,
				Meta:       responseMeta,
				Ratelimits: &responseRatelimits,
			},
		}

		return s.JSON(http.StatusOK, response)
	})
}
