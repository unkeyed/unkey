package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2IdentitiesUpdateIdentityRequestBody
	Response = openapi.V2IdentitiesUpdateIdentityResponseBody
)

// Handler implements zen.Route interface for the v2 identities update identity endpoint
type Handler struct {
	DB        db.Database
	Keys      keys.KeyService
	Auditlogs auditlogs.AuditLogService
}

const (
	// Planetscale has a limit on JSON field size
	maxMetaLengthMB = 1
)

// Method returns the HTTP method this route responds to
func (h *Handler) Method() string {
	return "POST"
}

// Path returns the URL path pattern this route matches
func (h *Handler) Path() string {
	return "/v2/identities.updateIdentity"
}

// Handle processes the HTTP request
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	auth, emit, err := h.Keys.GetRootKey(ctx, s)
	defer emit()
	if err != nil {
		return err
	}

	// Parse request
	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	err = auth.VerifyRootKey(ctx, keys.WithPermissions(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Identity,
			ResourceID:   "*",
			Action:       rbac.UpdateIdentity,
		}),
	)))
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
					fault.Internal("duplicate ratelimit name"),
					fault.Public(fmt.Sprintf("Ratelimit with name '%s' is already defined in the request", ratelimit.Name)),
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

	identityRow, err := db.Query.FindIdentity(ctx, h.DB.RO(), db.FindIdentityParams{
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
			fault.Internal("unable to find identity"),
			fault.Public("We're unable to retrieve the identity."),
		)
	}

	// Parse existing ratelimits from JSON
	var existingRatelimits []db.RatelimitInfo
	if ratelimitBytes, ok := identityRow.Ratelimits.([]byte); ok && ratelimitBytes != nil {
		_ = json.Unmarshal(ratelimitBytes, &existingRatelimits) // Ignore error, default to empty array
	}

	type txResult struct {
		identity        db.FindIdentityRow
		finalRatelimits []openapi.RatelimitResponse
	}

	result, err := db.TxWithResultRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) (txResult, error) {
		// Lock the identity row to prevent concurrent modifications and deadlocks.
		// This is necessary because UpdateIdentity is only called when req.Meta != nil,
		// so without this lock, concurrent ratelimit updates could deadlock.
		_, err := db.Query.LockIdentityForUpdate(ctx, tx, identityRow.ID)
		if err != nil {
			return txResult{}, fault.Wrap(err,
				fault.Internal("unable to lock identity"),
				fault.Public("We're unable to update the identity."),
			)
		}

		auditLogs := []auditlog.AuditLog{
			{
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Event:       auditlog.IdentityUpdateEvent,
				Display:     fmt.Sprintf("Updated identity %s", identityRow.ID),
				ActorID:     auth.Key.ID,
				ActorName:   "root key",
				ActorType:   auditlog.RootKeyActor,
				ActorMeta:   map[string]any{},
				RemoteIP:    s.Location(),
				UserAgent:   s.UserAgent(),
				Resources: []auditlog.AuditLogResource{
					{
						ID:          identityRow.ID,
						Type:        auditlog.IdentityResourceType,
						Name:        identityRow.ExternalID,
						DisplayName: identityRow.ExternalID,
						Meta:        nil,
					},
				},
			},
		}

		if req.Meta != nil {
			err = db.Query.UpdateIdentity(ctx, tx, db.UpdateIdentityParams{
				ID:   identityRow.ID,
				Meta: metaBytes,
			})
			if err != nil {
				// nolint:exhaustruct
				return txResult{}, fault.Wrap(err,
					fault.Internal("unable to update metadata"), fault.Public("We're unable to update the identity's metadata."),
				)
			}
		}

		// Build final ratelimits list (what will exist after this transaction)
		finalRatelimits := make([]openapi.RatelimitResponse, 0)

		if req.Ratelimits != nil {
			// Process ratelimits changes
			// 1. Delete ratelimits that no longer exist
			// 2. Update existing ratelimits
			// 3. Create new ratelimits

			// Create maps to easily find existing and new ratelimits by name
			existingRatelimitMap := make(map[string]db.RatelimitInfo)
			for _, rl := range existingRatelimits {
				existingRatelimitMap[rl.Name] = rl
			}

			newRatelimitMap := make(map[string]openapi.RatelimitRequest)
			for _, rl := range *req.Ratelimits {
				newRatelimitMap[rl.Name] = rl
			}

			rateLimitsToDelete := make([]string, 0)
			// Delete ratelimits that are not in the new list
			for _, existingRL := range existingRatelimits {
				_, exists := newRatelimitMap[existingRL.Name]
				if exists {
					continue
				}

				rateLimitsToDelete = append(rateLimitsToDelete, existingRL.ID)

				// Add audit log for deletion
				auditLogs = append(auditLogs, auditlog.AuditLog{
					WorkspaceID: auth.AuthorizedWorkspaceID,
					Event:       auditlog.RatelimitDeleteEvent,
					Display:     fmt.Sprintf("Deleted ratelimit %s", existingRL.ID),
					ActorID:     auth.Key.ID,
					ActorName:   "root key",
					ActorType:   auditlog.RootKeyActor,
					ActorMeta:   map[string]any{},
					RemoteIP:    s.Location(),
					UserAgent:   s.UserAgent(),
					Resources: []auditlog.AuditLogResource{
						{
							ID:          identityRow.ID,
							Type:        auditlog.IdentityResourceType,
							DisplayName: identityRow.ExternalID,
							Name:        identityRow.ExternalID,
							Meta:        nil,
						},
						{
							ID:          existingRL.ID,
							Type:        auditlog.RatelimitResourceType,
							DisplayName: existingRL.Name,
							Name:        existingRL.Name,
							Meta:        nil,
						},
					},
				})
			}

			if len(rateLimitsToDelete) > 0 {
				err = db.Query.DeleteManyRatelimitsByIDs(ctx, tx, rateLimitsToDelete)
				if err != nil {
					// nolint:exhaustruct
					return txResult{}, fault.Wrap(err,
						fault.Internal("unable to delete ratelimits"), fault.Public("We're unable to delete ratelimits."),
					)
				}
			}

			rateLimitsToInsert := make([]db.InsertIdentityRatelimitParams, 0)
			// Update existing ratelimits or create new ones
			for name, newRL := range newRatelimitMap {
				existingRL, exists := existingRatelimitMap[name]

				var ratelimitID string
				if exists {
					ratelimitID = existingRL.ID
					rateLimitsToInsert = append(rateLimitsToInsert, db.InsertIdentityRatelimitParams{
						ID:          existingRL.ID,
						WorkspaceID: auth.AuthorizedWorkspaceID,
						IdentityID:  sql.NullString{String: identityRow.ID, Valid: true},
						Name:        newRL.Name,
						Limit:       int32(newRL.Limit), // nolint:gosec
						Duration:    newRL.Duration,
						AutoApply:   newRL.AutoApply,
						CreatedAt:   time.Now().UnixMilli(),
					})

					auditLogs = append(auditLogs, auditlog.AuditLog{
						WorkspaceID: auth.AuthorizedWorkspaceID,
						Event:       auditlog.RatelimitUpdateEvent,
						Display:     fmt.Sprintf("Updated ratelimit %s", existingRL.ID),
						ActorID:     auth.Key.ID,
						ActorName:   "root key",
						ActorType:   auditlog.RootKeyActor,
						ActorMeta:   map[string]any{},
						RemoteIP:    s.Location(),
						UserAgent:   s.UserAgent(),
						Resources: []auditlog.AuditLogResource{
							{
								ID:          identityRow.ID,
								Type:        auditlog.IdentityResourceType,
								Name:        identityRow.ExternalID,
								DisplayName: identityRow.ExternalID,
								Meta:        nil,
							},
							{
								ID:          existingRL.ID,
								Type:        auditlog.RatelimitResourceType,
								Name:        newRL.Name,
								DisplayName: newRL.Name,
								Meta:        nil,
							},
						},
					})
				} else {
					// Create new ratelimit
					ratelimitID = uid.New(uid.RatelimitPrefix)
					rateLimitsToInsert = append(rateLimitsToInsert, db.InsertIdentityRatelimitParams{
						ID:          ratelimitID,
						WorkspaceID: auth.AuthorizedWorkspaceID,
						IdentityID:  sql.NullString{String: identityRow.ID, Valid: true},
						Name:        newRL.Name,
						Limit:       int32(newRL.Limit), // nolint:gosec
						Duration:    newRL.Duration,
						CreatedAt:   time.Now().UnixMilli(),
						AutoApply:   newRL.AutoApply,
					})

					// Add audit log for creation
					auditLogs = append(auditLogs, auditlog.AuditLog{
						WorkspaceID: auth.AuthorizedWorkspaceID,
						Event:       auditlog.RatelimitCreateEvent,
						Display:     fmt.Sprintf("Created ratelimit %s", ratelimitID),
						ActorID:     auth.Key.ID,
						ActorName:   "root key",
						ActorType:   auditlog.RootKeyActor,
						ActorMeta:   map[string]any{},
						RemoteIP:    s.Location(),
						UserAgent:   s.UserAgent(),
						Resources: []auditlog.AuditLogResource{
							{
								ID:          identityRow.ID,
								Type:        auditlog.IdentityResourceType,
								DisplayName: identityRow.ExternalID,
								Name:        identityRow.ExternalID,
								Meta:        nil,
							},
							{
								ID:          ratelimitID,
								Type:        auditlog.RatelimitResourceType,
								DisplayName: newRL.Name,
								Name:        newRL.Name,
								Meta:        nil,
							},
						},
					})
				}

				// Add to final ratelimits list (no DB query needed!)
				finalRatelimits = append(finalRatelimits, openapi.RatelimitResponse{
					Id:        ratelimitID,
					Name:      newRL.Name,
					Limit:     newRL.Limit,
					Duration:  newRL.Duration,
					AutoApply: newRL.AutoApply,
				})
			}

			if len(rateLimitsToInsert) > 0 {
				err = db.BulkQuery.InsertIdentityRatelimits(ctx, tx, rateLimitsToInsert)
				if err != nil {
					// nolint:exhaustruct
					return txResult{}, fault.Wrap(err,
						fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
						fault.Internal("database failed to insert ratelimits"),
						fault.Public("Failed to insert ratelimits"),
					)
				}
			}
		} else {
			// No ratelimit changes - keep existing ones
			for _, rl := range existingRatelimits {
				finalRatelimits = append(finalRatelimits, openapi.RatelimitResponse{
					Id:        rl.ID,
					Name:      rl.Name,
					Limit:     int64(rl.Limit),
					Duration:  rl.Duration,
					AutoApply: rl.AutoApply,
				})
			}
		}

		err = h.Auditlogs.Insert(ctx, tx, auditLogs)
		if err != nil {
			// nolint:exhaustruct
			return txResult{}, err
		}

		return txResult{
			identity:        identityRow,
			finalRatelimits: finalRatelimits,
		}, nil
	})
	if err != nil {
		return err
	}

	// No extra SELECT query needed - we built the ratelimits list during the transaction!
	identityData := openapi.Identity{
		Id:         result.identity.ID,
		ExternalId: result.identity.ExternalID,
		Meta:       ptr.SafeDeref(req.Meta),
		Ratelimits: nil,
	}

	identityData.Ratelimits = result.finalRatelimits

	response := Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: identityData,
	}

	return s.JSON(http.StatusOK, response)
}
