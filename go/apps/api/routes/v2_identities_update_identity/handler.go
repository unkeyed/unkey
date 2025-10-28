package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/oapi-codegen/nullable"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/internal/services/auditlogs"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/internal/services/usagelimiter"
	"github.com/unkeyed/unkey/go/pkg/array"
	"github.com/unkeyed/unkey/go/pkg/auditlog"
	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/uid"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type (
	Request  = openapi.V2IdentitiesUpdateIdentityRequestBody
	Response = openapi.V2IdentitiesUpdateIdentityResponseBody
)

// Handler implements zen.Route interface for the v2 identities update identity endpoint
type Handler struct {
	// Services as public fields
	Logger       logging.Logger
	DB           db.Database
	Keys         keys.KeyService
	Auditlogs    auditlogs.AuditLogService
	UsageLimiter usagelimiter.Service
	KeyCache     cache.Cache[string, db.CachedKeyData]
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

	identity, err := db.TxWithResult(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) (db.Identity, error) {
		// Find by external ID
		identity, err := db.Query.FindIdentity(ctx, tx, db.FindIdentityParams{
			Identity:    req.Identity,
			WorkspaceID: auth.AuthorizedWorkspaceID,
			Deleted:     false,
		})
		if err != nil {
			if db.IsNotFound(err) {
				// nolint:exhaustruct
				return db.Identity{}, fault.New("identity not found",
					fault.Code(codes.Data.Identity.NotFound.URN()),
					fault.Internal("identity not found"), fault.Public("Identity not found in this workspace"),
				)
			}

			// nolint:exhaustruct
			return db.Identity{}, fault.Wrap(err,
				fault.Internal("unable to find identity"), fault.Public("We're unable to retrieve the identity."),
			)
		}

		if identity.WorkspaceID != auth.AuthorizedWorkspaceID {
			// nolint:exhaustruct
			return db.Identity{}, fault.New("identity not found",
				fault.Code(codes.Data.Identity.NotFound.URN()),
				fault.Internal("wrong workspace, masking as 404"),
				fault.Public("Identity not found in this workspace"),
			)
		}

		var existingRatelimits []db.Ratelimit
		existingRatelimits, err = db.Query.ListIdentityRatelimitsByID(ctx, tx, sql.NullString{String: identity.ID, Valid: true})
		if err != nil && !db.IsNotFound(err) {
			// nolint:exhaustruct
			return db.Identity{}, fault.Wrap(err,
				fault.Internal("unable to fetch ratelimits"), fault.Public("We're unable to retrieve the identity's ratelimits."),
			)
		}

		auditLogs := []auditlog.AuditLog{
			{
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Event:       auditlog.IdentityUpdateEvent,
				Display:     fmt.Sprintf("Updated identity %s", identity.ID),
				ActorID:     auth.Key.ID,
				ActorName:   "root key",
				ActorType:   auditlog.RootKeyActor,
				ActorMeta:   map[string]any{},
				RemoteIP:    s.Location(),
				UserAgent:   s.UserAgent(),
				Resources: []auditlog.AuditLogResource{
					{
						ID:          identity.ID,
						Type:        auditlog.IdentityResourceType,
						Name:        identity.ExternalID,
						DisplayName: identity.ExternalID,
						Meta:        nil,
					},
				},
			},
		}

		if req.Meta != nil {
			err = db.Query.UpdateIdentity(ctx, tx, db.UpdateIdentityParams{
				ID:   identity.ID,
				Meta: metaBytes,
			})
			if err != nil {
				// nolint:exhaustruct
				return db.Identity{}, fault.Wrap(err,
					fault.Internal("unable to update metadata"), fault.Public("We're unable to update the identity's metadata."),
				)
			}
		}

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

			newRatelimitMap := make(map[string]openapi.RatelimitRequest)
			if req.Ratelimits != nil {
				for _, rl := range *req.Ratelimits {
					newRatelimitMap[rl.Name] = rl
				}
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
							ID:          identity.ID,
							Type:        auditlog.IdentityResourceType,
							DisplayName: identity.ExternalID,
							Name:        identity.ExternalID,
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
					return db.Identity{}, fault.Wrap(err,
						fault.Internal("unable to delete ratelimits"), fault.Public("We're unable to delete ratelimits."),
					)
				}
			}

			rateLimitsToInsert := make([]db.InsertIdentityRatelimitParams, 0)
			// Update existing ratelimits or create new ones
			for name, newRL := range newRatelimitMap {
				existingRL, exists := existingRatelimitMap[name]

				if exists {
					rateLimitsToInsert = append(rateLimitsToInsert, db.InsertIdentityRatelimitParams{
						ID:          existingRL.ID,
						WorkspaceID: auth.AuthorizedWorkspaceID,
						IdentityID:  sql.NullString{String: identity.ID, Valid: true},
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
								ID:          identity.ID,
								Type:        auditlog.IdentityResourceType,
								Name:        identity.ExternalID,
								DisplayName: identity.ExternalID,
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

					continue
				}

				// Create new ratelimit
				ratelimitID := uid.New(uid.RatelimitPrefix)
				rateLimitsToInsert = append(rateLimitsToInsert, db.InsertIdentityRatelimitParams{
					ID:          ratelimitID,
					WorkspaceID: auth.AuthorizedWorkspaceID,
					IdentityID:  sql.NullString{String: identity.ID, Valid: true},
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
							ID:          identity.ID,
							Type:        auditlog.IdentityResourceType,
							DisplayName: identity.ExternalID,
							Name:        identity.ExternalID,
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

			if len(rateLimitsToInsert) > 0 {
				err = db.BulkQuery.InsertIdentityRatelimits(ctx, tx, rateLimitsToInsert)
				if err != nil {
					// nolint:exhaustruct
					return db.Identity{}, fault.Wrap(err,
						fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
						fault.Internal("database failed to insert ratelimits"),
						fault.Public("Failed to insert ratelimits"),
					)
				}
			}
		}

		// Handle credit updates
		if req.Credits.IsSpecified() {
			if req.Credits.IsNull() {
				// Delete credit record if it exists
				if identity.CreditID.Valid {
					deleteErr := db.Query.DeleteCredit(ctx, tx, identity.CreditID.String)
					if deleteErr != nil {
						// nolint:exhaustruct
						return db.Identity{}, fault.Wrap(deleteErr,
							fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
							fault.Internal("failed to delete credit"),
							fault.Public("Failed to update credits."),
						)
					}

					auditLogs = append(auditLogs, auditlog.AuditLog{
						WorkspaceID: auth.AuthorizedWorkspaceID,
						Event:       auditlog.IdentityUpdateEvent,
						Display:     fmt.Sprintf("Removed credits from identity %s", identity.ID),
						ActorID:     auth.Key.ID,
						ActorName:   "root key",
						ActorType:   auditlog.RootKeyActor,
						ActorMeta:   map[string]any{},
						RemoteIP:    s.Location(),
						UserAgent:   s.UserAgent(),
						Resources: []auditlog.AuditLogResource{
							{
								ID:          identity.ID,
								Type:        auditlog.IdentityResourceType,
								Name:        identity.ExternalID,
								DisplayName: identity.ExternalID,
								Meta:        nil,
							},
						},
					})
				}
				// If not found, that's fine - already doesn't have credits
			} else {
				// Upsert credits
				creditsValue := req.Credits.MustGet()
				now := time.Now().UnixMilli()

				// Prepare refill configuration
				var refillAmount sql.NullInt32
				var refillDay sql.NullInt16
				var refillAmountSpecified, refillDaySpecified int64

				if creditsValue.Refill.IsSpecified() {
					if creditsValue.Refill.IsNull() {
						// Clear refill configuration
						refillAmountSpecified = 1
						refillDaySpecified = 1
						refillAmount = sql.NullInt32{Valid: false}
						refillDay = sql.NullInt16{Valid: false}
					} else {
						// Set refill configuration
						refillConfig := creditsValue.Refill.MustGet()
						refillAmountSpecified = 1
						refillAmount = sql.NullInt32{
							Valid: true,
							Int32: int32(refillConfig.Amount),
						}

						if refillConfig.Interval == openapi.Monthly {
							refillDaySpecified = 1
							refillDay = sql.NullInt16{
								Valid: true,
								Int16: int16(ptr.SafeDeref(refillConfig.RefillDay, 1)),
							}
						} else {
							refillDaySpecified = 1
							refillDay = sql.NullInt16{Valid: false}
						}
					}
				} else {
					// Don't touch refill configuration
					refillAmountSpecified = 0
					refillDaySpecified = 0
				}

				// Prepare remaining credits
				var remaining int32
				var remainingSpecified int64
				if creditsValue.Remaining.IsSpecified() {
					remainingSpecified = 1
					if creditsValue.Remaining.IsNull() {
						remaining = 0
					} else {
						remaining = int32(creditsValue.Remaining.MustGet())
					}
				} else {
					remainingSpecified = 0
					remaining = 0 // Will be ignored due to specified flag
				}

				creditID := uid.New(uid.CreditPrefix)
				upsertErr := db.Query.UpsertCredit(ctx, tx, db.UpsertCreditParams{
					ID:                    creditID,
					WorkspaceID:           auth.AuthorizedWorkspaceID,
					KeyID:                 sql.NullString{Valid: false},
					IdentityID:            sql.NullString{String: identity.ID, Valid: true},
					Remaining:             remaining,
					RemainingSpecified:    remainingSpecified,
					RefillAmount:          refillAmount,
					RefillAmountSpecified: refillAmountSpecified,
					RefillDay:             refillDay,
					RefillDaySpecified:    refillDaySpecified,
					CreatedAt:             now,
					UpdatedAt:             sql.NullInt64{Valid: true, Int64: now},
					RefilledAt:            sql.NullInt64{Valid: false},
				})
				if upsertErr != nil {
					// nolint:exhaustruct
					return db.Identity{}, fault.Wrap(upsertErr,
						fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
						fault.Internal("failed to upsert credit"),
						fault.Public("Failed to update credits."),
					)
				}

				auditLogs = append(auditLogs, auditlog.AuditLog{
					WorkspaceID: auth.AuthorizedWorkspaceID,
					Event:       auditlog.IdentityUpdateEvent,
					Display:     fmt.Sprintf("Updated credits for identity %s", identity.ID),
					ActorID:     auth.Key.ID,
					ActorName:   "root key",
					ActorType:   auditlog.RootKeyActor,
					ActorMeta:   map[string]any{},
					RemoteIP:    s.Location(),
					UserAgent:   s.UserAgent(),
					Resources: []auditlog.AuditLogResource{
						{
							ID:          identity.ID,
							Type:        auditlog.IdentityResourceType,
							Name:        identity.ExternalID,
							DisplayName: identity.ExternalID,
							Meta:        nil,
						},
					},
				})
			}
		}

		err = h.Auditlogs.Insert(ctx, tx, auditLogs)
		if err != nil {
			// nolint:exhaustruct
			return db.Identity{}, err
		}

		// Convert FindIdentityRow to Identity
		return db.Identity{
			ID:          identity.ID,
			ExternalID:  identity.ExternalID,
			WorkspaceID: identity.WorkspaceID,
			Environment: identity.Environment,
			Meta:        identity.Meta,
			Deleted:     identity.Deleted,
			CreatedAt:   identity.CreatedAt,
			UpdatedAt:   identity.UpdatedAt,
		}, nil
	})
	if err != nil {
		return err
	}

	updatedRatelimits, err := db.Query.ListIdentityRatelimitsByID(ctx, h.DB.RO(), sql.NullString{String: identity.ID, Valid: true})
	if err != nil && !db.IsNotFound(err) {
		return fault.Wrap(err,
			fault.Internal("unable to fetch updated ratelimits"),
			fault.Public("We were able to update the identity but unable to retrieve the updated ratelimits."),
		)
	}

	responseRatelimits := make([]openapi.RatelimitResponse, 0, len(updatedRatelimits))
	for _, r := range updatedRatelimits {
		responseRatelimits = append(responseRatelimits, openapi.RatelimitResponse{
			Id:        r.ID,
			Name:      r.Name,
			Limit:     int64(r.Limit),
			Duration:  r.Duration,
			AutoApply: r.AutoApply,
		})
	}

	// Fetch updated identity with credits to get the latest state
	updatedIdentity, err := db.Query.FindIdentity(ctx, h.DB.RO(), db.FindIdentityParams{
		Identity:    identity.ID,
		WorkspaceID: identity.WorkspaceID,
		Deleted:     false,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return fault.Wrap(err,
				fault.Internal("identity disappeared after update"),
				fault.Public("We were able to update the identity but unable to retrieve the updated state."),
			)
		}

		return fault.Wrap(err,
			fault.Internal("unable to fetch updated identity"),
			fault.Public("We were able to update the identity but unable to retrieve the updated state."),
		)
	}

	// Build credits response from updated identity
	var identityCredits *openapi.Credits
	if updatedIdentity.CreditID.Valid {
		identityCredits = &openapi.Credits{
			Remaining: nullable.NewNullableWithValue(int64(updatedIdentity.CreditRemaining.Int32)),
		}

		if updatedIdentity.CreditRefillAmount.Valid {
			var refillDay *int
			interval := openapi.Daily
			if updatedIdentity.CreditRefillDay.Valid {
				interval = openapi.Monthly
				refillDay = ptr.P(int(updatedIdentity.CreditRefillDay.Int16))
			}

			identityCredits.Refill = &openapi.CreditsRefill{
				Amount:    int64(updatedIdentity.CreditRefillAmount.Int32),
				Interval:  interval,
				RefillDay: refillDay,
			}
		}
	}

	identityData := openapi.Identity{
		Id:         identity.ID,
		ExternalId: identity.ExternalID,
		Meta:       req.Meta,
		Ratelimits: nil,
		Credits:    identityCredits,
	}

	if len(responseRatelimits) > 0 {
		identityData.Ratelimits = ptr.P(responseRatelimits)
	}

	// Invalidate cache for identity credits if they were updated
	if req.Credits.IsSpecified() && updatedIdentity.CreditID.Valid {
		if invalidateErr := h.UsageLimiter.Invalidate(ctx, updatedIdentity.CreditID.String); invalidateErr != nil {
			h.Logger.Error("Failed to invalidate usage limit for identity credit",
				"error", invalidateErr.Error(),
				"credit_id", updatedIdentity.CreditID.String,
				"identity_id", identity.ID,
			)
		}

		// Find and invalidate all keys belonging to this identity
		keys, err := db.Query.ListKeysByIdentityID(ctx, h.DB.RO(), sql.NullString{String: identity.ID, Valid: true})
		if err != nil {
			h.Logger.Error("Failed to find keys for identity",
				"error", err.Error(),
				"identity_id", identity.ID,
			)
		}

		if len(keys) > 0 {
			hashes := array.Map(keys, func(key db.ListKeysByIdentityIDRow) string {
				return key.Hash
			})

			h.KeyCache.Remove(ctx, hashes...)
		}
	}

	response := Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: identityData,
	}

	return s.JSON(http.StatusOK, response)
}
