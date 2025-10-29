package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/internal/services/auditlogs"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/internal/services/usagelimiter"

	"github.com/unkeyed/unkey/go/pkg/auditlog"
	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/db"
	dbtype "github.com/unkeyed/unkey/go/pkg/db/types"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/uid"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type (
	Request  = openapi.V2KeysUpdateKeyRequestBody
	Response = openapi.V2KeysUpdateKeyResponseBody
)

type Handler struct {
	Logger       logging.Logger
	DB           db.Database
	Keys         keys.KeyService
	Auditlogs    auditlogs.AuditLogService
	KeyCache     cache.Cache[string, db.CachedKeyData]
	UsageLimiter usagelimiter.Service
}

// Method returns the HTTP method this route responds to
func (h *Handler) Method() string {
	return "POST"
}

// Path returns the URL path pattern this route matches
func (h *Handler) Path() string {
	return "/v2/keys.updateKey"
}

// Handle processes the HTTP request
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	h.Logger.Debug("handling request", "requestId", s.RequestID(), "path", "/v2/keys.updateKey")

	auth, emit, err := h.Keys.GetRootKey(ctx, s)
	defer emit()
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

	if key.WorkspaceID != auth.AuthorizedWorkspaceID {
		return fault.New("key not found",
			fault.Code(codes.Data.Key.NotFound.URN()),
			fault.Internal("key belongs to different workspace"),
			fault.Public("The specified key was not found."),
		)
	}

	// Convert to KeyData to access credits
	keyData := db.ToKeyData(key)

	// Check all credit sources: new credits table, or legacy fields
	hasNewCredits := keyData.KeyCredits != nil
	hasLegacyCredits := key.RemainingRequests.Valid

	// Save the old credit ID for cache invalidation later (before it potentially gets deleted/changed in the transaction)
	oldCreditID := ""
	if hasNewCredits {
		oldCreditID = keyData.KeyCredits.ID
	}

	// TODO: We should actually check if the user has permission to set/remove roles.
	err = auth.VerifyRootKey(ctx, keys.WithPermissions(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Api,
			ResourceID:   "*",
			Action:       rbac.UpdateKey,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Api,
			ResourceID:   key.Api.ID,
			Action:       rbac.UpdateKey,
		}),
	)))
	if err != nil {
		return err
	}

	// Retry transaction up to 2 times on deadlock or identity creation race
	var txErr error
	for attempt := range 3 {
		txErr = db.Tx(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
			auditLogs := []auditlog.AuditLog{}

			update := db.UpdateKeyParams{
				ID:                  key.ID,
				Now:                 sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
				NameSpecified:       0,
				Name:                sql.NullString{Valid: false, String: ""},
				IdentityIDSpecified: 0,
				IdentityID:          sql.NullString{Valid: false, String: ""},
				EnabledSpecified:    0,
				Enabled:             sql.NullBool{Valid: false, Bool: false},
				MetaSpecified:       0,
				Meta:                sql.NullString{Valid: false, String: ""},
				ExpiresSpecified:    0,
				Expires:             sql.NullTime{Valid: false, Time: time.Time{}},
			}

			if req.Name.IsSpecified() {
				update.NameSpecified = 1
				if req.Name.IsNull() {
					update.Name = sql.NullString{Valid: false, String: ""}
				} else {
					update.Name = sql.NullString{Valid: true, String: req.Name.MustGet()}
				}
			}

			//nolint:nestif
			if req.ExternalId.IsSpecified() {
				update.IdentityIDSpecified = 1
				if req.ExternalId.IsNull() {
					update.IdentityID = sql.NullString{Valid: false, String: ""}
				} else {
					externalID := req.ExternalId.MustGet()

					// Try to find existing identity
					identity, err := db.Query.FindIdentity(ctx, tx, db.FindIdentityParams{
						WorkspaceID: auth.AuthorizedWorkspaceID,
						Identity:    externalID,
						Deleted:     false,
					})

					if err != nil {
						if !db.IsNotFound(err) {
							return fault.Wrap(err,
								fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
								fault.Internal("failed to find identity"),
								fault.Public("Failed to find identity."),
							)
						}

						// Create new identity
						identityID := uid.New(uid.IdentityPrefix)
						err = db.Query.InsertIdentity(ctx, tx, db.InsertIdentityParams{
							ID:          identityID,
							ExternalID:  externalID,
							WorkspaceID: auth.AuthorizedWorkspaceID,
							Environment: "default",
							CreatedAt:   time.Now().UnixMilli(),
							Meta:        []byte("{}"),
						})
						if err != nil {
							// Don't wrap duplicate key or deadlock errors - let retry loop handle them
							if db.IsDuplicateKeyError(err) || db.IsDeadlockError(err) {
								return err
							}

							return fault.Wrap(err,
								fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
								fault.Internal("failed to create identity"),
								fault.Public("Failed to create identity."),
							)
						}

						update.IdentityID = sql.NullString{Valid: true, String: identityID}
					} else {
						// Use existing identity
						update.IdentityID = sql.NullString{Valid: true, String: identity.ID}
					}
				}
			}

			if req.Enabled != nil {
				update.EnabledSpecified = 1
				update.Enabled = sql.NullBool{Valid: true, Bool: *req.Enabled}
			}

			if req.Meta.IsSpecified() {
				update.MetaSpecified = 1
				if req.Meta.IsNull() {
					update.Meta = sql.NullString{Valid: false, String: ""}
				} else {
					metaBytes, marshalErr := json.Marshal(req.Meta.MustGet())
					if marshalErr != nil {
						return fault.Wrap(marshalErr,
							fault.Code(codes.App.Validation.InvalidInput.URN()),
							fault.Internal("failed to marshal meta"),
							fault.Public("Invalid metadata format."),
						)
					}
					update.Meta = sql.NullString{Valid: true, String: string(metaBytes)}
				}
			}

			if req.Expires.IsSpecified() {
				update.ExpiresSpecified = 1
				if req.Expires.IsNull() {
					update.Expires = sql.NullTime{Valid: false, Time: time.Time{}}
				} else {
					update.Expires = sql.NullTime{Valid: true, Time: time.UnixMilli(req.Expires.MustGet())}
				}
			}

			err = db.Query.UpdateKey(ctx, tx, update)
			if err != nil {
				return fault.Wrap(err,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("database error"),
					fault.Public("Failed to update key."),
				)
			}

			// Handle credits updates
			if req.Credits.IsSpecified() {
				now := time.Now().UnixMilli()

				// Work with whichever credit system the key is using (no migration)
				if hasNewCredits {
					// Key uses new credits table
					if req.Credits.IsNull() {
						// Setting credits to null means unlimited - delete the credit record
						if err := db.Query.DeleteCredit(ctx, tx, keyData.KeyCredits.ID); err != nil {
							return fault.Wrap(err,
								fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
								fault.Internal("database error"),
								fault.Public("Failed to update key credits."),
							)
						}
					} else {
						credits := req.Credits.MustGet()
						if credits.Remaining.IsSpecified() && credits.Remaining.IsNull() {
							// Setting remaining to null means unlimited - delete the credit record
							if err := db.Query.DeleteCredit(ctx, tx, keyData.KeyCredits.ID); err != nil {
								return fault.Wrap(err,
									fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
									fault.Internal("database error"),
									fault.Public("Failed to update key credits."),
								)
							}
						} else {
							// Upsert credits with three-state logic
							var remaining int32
							var remainingSpecified int64
							var refillDay sql.NullInt16
							var refillDaySpecified int64
							var refillAmount sql.NullInt32
							var refillAmountSpecified int64

							if credits.Remaining.IsSpecified() {
								remainingSpecified = 1
								remaining = int32(credits.Remaining.MustGet()) // nolint:gosec
							}

							if credits.Refill.IsSpecified() {
								if credits.Refill.IsNull() {
									refillDaySpecified = 1
									refillAmountSpecified = 1
									refillDay = sql.NullInt16{Valid: false}
									refillAmount = sql.NullInt32{Valid: false}
								} else {
									refill := credits.Refill.MustGet()
									refillAmountSpecified = 1
									refillAmount = sql.NullInt32{
										Valid: true,
										Int32: int32(refill.Amount), // nolint:gosec
									}

									refillDaySpecified = 1
									switch refill.Interval {
									case openapi.Monthly:
										if refill.RefillDay == nil {
											return fault.New("missing refillDay",
												fault.Code(codes.App.Validation.InvalidInput.URN()),
												fault.Internal("refillDay required for monthly interval"),
												fault.Public("`refillDay` must be provided when the refill interval is `monthly`."),
											)
										}

										refillDay = sql.NullInt16{
											Valid: true,
											Int16: int16(*refill.RefillDay), // nolint:gosec
										}
									case openapi.Daily:
										if refill.RefillDay != nil {
											return fault.New("invalid refillDay",
												fault.Code(codes.App.Validation.InvalidInput.URN()),
												fault.Internal("refillDay cannot be set for daily interval"),
												fault.Public("`refillDay` must not be provided when the refill interval is `daily`."),
											)
										}

										refillDay = sql.NullInt16{Valid: false}
									}
								}
							}

							err = db.Query.UpsertCredit(ctx, tx, db.UpsertCreditParams{
								ID:                    keyData.KeyCredits.ID,
								WorkspaceID:           key.WorkspaceID,
								KeyID:                 sql.NullString{Valid: true, String: key.ID},
								IdentityID:            sql.NullString{Valid: false},
								Remaining:             remaining,
								RefillDay:             refillDay,
								RefillAmount:          refillAmount,
								CreatedAt:             now,
								UpdatedAt:             sql.NullInt64{Valid: true, Int64: now},
								RefilledAt:            sql.NullInt64{Valid: false},
								RemainingSpecified:    remainingSpecified,
								RefillDaySpecified:    refillDaySpecified,
								RefillAmountSpecified: refillAmountSpecified,
							})
							if err != nil {
								return fault.Wrap(err,
									fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
									fault.Internal("database error"),
									fault.Public("Failed to update key credits."),
								)
							}
						}
					}
				} else if hasLegacyCredits {
					// Key uses legacy fields - update them directly
					credits := req.Credits.MustGet()

					updateKeyParams := db.UpdateKeyParams{
						ID:                         key.ID,
						Now:                        sql.NullInt64{Valid: true, Int64: now},
						NameSpecified:              0,
						IdentityIDSpecified:        0,
						EnabledSpecified:           0,
						MetaSpecified:              0,
						ExpiresSpecified:           0,
						RemainingRequestsSpecified: 0,
						RefillAmountSpecified:      0,
						RefillDaySpecified:         0,
					}

					if credits.Remaining.IsSpecified() {
						updateKeyParams.RemainingRequestsSpecified = 1
						if credits.Remaining.IsNull() {
							// Set to unlimited (NULL)
							updateKeyParams.RemainingRequests = sql.NullInt32{Valid: false}
						} else {
							updateKeyParams.RemainingRequests = sql.NullInt32{
								Valid: true,
								Int32: int32(credits.Remaining.MustGet()), // nolint:gosec
							}
						}
					}

					if credits.Refill.IsSpecified() {
						updateKeyParams.RefillAmountSpecified = 1
						updateKeyParams.RefillDaySpecified = 1
						if credits.Refill.IsNull() {
							// Clear refill
							updateKeyParams.RefillAmount = sql.NullInt32{Valid: false}
							updateKeyParams.RefillDay = sql.NullInt16{Valid: false}
						} else {
							refill := credits.Refill.MustGet()
							updateKeyParams.RefillAmount = sql.NullInt32{
								Valid: true,
								Int32: int32(refill.Amount), // nolint:gosec
							}

							switch refill.Interval {
							case openapi.Monthly:
								if refill.RefillDay == nil {
									return fault.New("missing refillDay",
										fault.Code(codes.App.Validation.InvalidInput.URN()),
										fault.Internal("refillDay required for monthly interval"),
										fault.Public("`refillDay` must be provided when the refill interval is `monthly`."),
									)
								}

								updateKeyParams.RefillDay = sql.NullInt16{
									Valid: true,
									Int16: int16(*refill.RefillDay), // nolint:gosec
								}
							case openapi.Daily:
								if refill.RefillDay != nil {
									return fault.New("invalid refillDay",
										fault.Code(codes.App.Validation.InvalidInput.URN()),
										fault.Internal("refillDay cannot be set for daily interval"),
										fault.Public("`refillDay` must not be provided when the refill interval is `daily`."),
									)
								}

								updateKeyParams.RefillDay = sql.NullInt16{Valid: false}
							}
						}
					}

					// Apply the update
					err = db.Query.UpdateKey(ctx, tx, updateKeyParams)
					if err != nil {
						return fault.Wrap(err,
							fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
							fault.Internal("database error"),
							fault.Public("Failed to update key credits."),
						)
					}
				} else {
					// No existing credits - create new credits in new table
					credits := req.Credits.MustGet()
					if !credits.Remaining.IsNull() {
						creditID := uid.New(uid.CreditPrefix)
						remaining := int32(credits.Remaining.MustGet())

						var refillDay sql.NullInt16
						var refillAmount sql.NullInt32

						if credits.Refill != nil && !credits.Refill.IsNull() {
							refill := credits.Refill.MustGet()
							refillAmount = sql.NullInt32{
								Valid: true,
								Int32: int32(refill.Amount), // nolint:gosec
							}

							switch refill.Interval {
							case openapi.Monthly:
								if refill.RefillDay == nil {
									return fault.New("missing refillDay",
										fault.Code(codes.App.Validation.InvalidInput.URN()),
										fault.Internal("refillDay required for monthly interval"),
										fault.Public("`refillDay` must be provided when the refill interval is `monthly`."),
									)
								}

								refillDay = sql.NullInt16{
									Valid: true,
									Int16: int16(*refill.RefillDay), // nolint:gosec
								}
							case openapi.Daily:
								if refill.RefillDay != nil {
									return fault.New("invalid refillDay",
										fault.Code(codes.App.Validation.InvalidInput.URN()),
										fault.Internal("refillDay cannot be set for daily interval"),
										fault.Public("`refillDay` must not be provided when the refill interval is `daily`."),
									)
								}

								refillDay = sql.NullInt16{Valid: false}
							}
						}

						err = db.Query.InsertCredit(ctx, tx, db.InsertCreditParams{
							ID:           creditID,
							WorkspaceID:  key.WorkspaceID,
							KeyID:        sql.NullString{Valid: true, String: key.ID},
							IdentityID:   sql.NullString{Valid: false},
							Remaining:    remaining,
							RefillDay:    refillDay,
							RefillAmount: refillAmount,
							CreatedAt:    now,
							UpdatedAt:    sql.NullInt64{Valid: true, Int64: now},
							RefilledAt:   sql.NullInt64{Valid: false},
						})
						if err != nil {
							return fault.Wrap(err,
								fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
								fault.Internal("database error"),
								fault.Public("Failed to create key credits."),
							)
						}
					}
				}
			}

			if req.Ratelimits != nil {
				var existingRatelimits []db.ListRatelimitsByKeyIDRow
				existingRatelimits, err = db.Query.ListRatelimitsByKeyID(ctx, tx, sql.NullString{String: key.ID, Valid: true})
				if err != nil && !db.IsNotFound(err) {
					return fault.Wrap(err,
						fault.Internal("unable to fetch ratelimits"),
						fault.Public("Failed to retrieve key ratelimits."),
					)
				}

				// Create map of existing ratelimits by name
				existingRatelimitMap := make(map[string]db.ListRatelimitsByKeyIDRow)
				for _, rl := range existingRatelimits {
					existingRatelimitMap[rl.Name] = rl
				}

				// Create map of new ratelimits
				newRatelimitMap := make(map[string]openapi.RatelimitRequest)
				for _, rl := range *req.Ratelimits {
					newRatelimitMap[rl.Name] = rl
				}

				// Delete ratelimits that are not in the new list
				rateLimitsToDelete := []string{}
				for _, existingRL := range existingRatelimits {
					if _, exists := newRatelimitMap[existingRL.Name]; !exists {
						rateLimitsToDelete = append(rateLimitsToDelete, existingRL.ID)
					}
				}

				if len(rateLimitsToDelete) > 0 {
					err = db.Query.DeleteManyRatelimitsByIDs(ctx, tx, rateLimitsToDelete)
					if err != nil {
						return fault.Wrap(err,
							fault.Internal("unable to delete ratelimits"),
							fault.Public("Failed to delete ratelimits."),
						)
					}
				}

				// Insert or update ratelimits
				ratelimitsToInsert := []db.InsertKeyRatelimitParams{}
				now := time.Now().UnixMilli()
				for name, newRL := range newRatelimitMap {
					_, exists := existingRatelimitMap[name]

					var rlID string
					if exists {
						rlID = existingRatelimitMap[name].ID
					} else {
						rlID = uid.New(uid.RatelimitPrefix)
					}

					ratelimitsToInsert = append(ratelimitsToInsert, db.InsertKeyRatelimitParams{
						ID:          rlID,
						WorkspaceID: auth.AuthorizedWorkspaceID,
						KeyID:       sql.NullString{String: key.ID, Valid: true},
						Name:        newRL.Name,
						Limit:       int32(newRL.Limit), // nolint:gosec
						Duration:    newRL.Duration,
						CreatedAt:   now,
						UpdatedAt:   sql.NullInt64{Int64: now, Valid: true},
						AutoApply:   newRL.AutoApply,
					})
				}

				if len(ratelimitsToInsert) > 0 {
					err = db.BulkQuery.InsertKeyRatelimits(ctx, tx, ratelimitsToInsert)
					if err != nil {
						return fault.Wrap(err,
							fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
							fault.Internal("database error"),
							fault.Public("Failed to update rate limits."),
						)
					}
				}
			}

			if req.Permissions != nil {
				var existingPermissions []db.Permission
				existingPermissions, err = db.Query.FindPermissionsBySlugs(ctx, tx, db.FindPermissionsBySlugsParams{
					WorkspaceID: auth.AuthorizedWorkspaceID,
					Slugs:       *req.Permissions,
				})
				if err != nil {
					return fault.Wrap(err,
						fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
						fault.Internal("database error"),
						fault.Public("Failed to retrieve permissions."),
					)
				}

				existingPermMap := make(map[string]db.Permission)
				for _, p := range existingPermissions {
					existingPermMap[p.Slug] = p
				}

				permissionsToCreate := []db.InsertPermissionParams{}
				requestedPermissions := []db.Permission{}

				for _, requestedSlug := range *req.Permissions {
					existingPerm, exists := existingPermMap[requestedSlug]
					if exists {
						requestedPermissions = append(requestedPermissions, existingPerm)
						continue
					}

					newPermID := uid.New(uid.PermissionPrefix)
					permissionsToCreate = append(permissionsToCreate, db.InsertPermissionParams{
						PermissionID: newPermID,
						WorkspaceID:  auth.AuthorizedWorkspaceID,
						Name:         requestedSlug,
						Slug:         requestedSlug,
						Description:  dbtype.NullString{String: fmt.Sprintf("Auto-created permission: %s", requestedSlug), Valid: true},
						CreatedAtM:   time.Now().UnixMilli(),
					})

					//nolint: exhaustruct
					requestedPermissions = append(requestedPermissions, db.Permission{
						ID:   newPermID,
						Slug: requestedSlug,
					})
				}

				if len(permissionsToCreate) > 0 {
					for _, toCreate := range permissionsToCreate {
						auditLogs = append(auditLogs, auditlog.AuditLog{
							WorkspaceID: auth.AuthorizedWorkspaceID,
							Event:       auditlog.PermissionCreateEvent,
							ActorType:   auditlog.RootKeyActor,
							ActorID:     auth.Key.ID,
							ActorName:   "root key",
							ActorMeta:   map[string]any{},
							Display:     fmt.Sprintf("Created %s (%s)", toCreate.Slug, toCreate.PermissionID),
							RemoteIP:    s.Location(),
							UserAgent:   s.UserAgent(),
							Resources: []auditlog.AuditLogResource{
								{
									Type:        auditlog.PermissionResourceType,
									ID:          toCreate.PermissionID,
									Name:        toCreate.Slug,
									DisplayName: toCreate.Name,
									Meta: map[string]any{
										"name": toCreate.Name,
										"slug": toCreate.Slug,
									},
								},
							},
						})
					}

					err = db.BulkQuery.InsertPermissions(ctx, tx, permissionsToCreate)
					if err != nil {
						return fault.Wrap(err,
							fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
							fault.Internal("database error"),
							fault.Public("Failed to create permissions."),
						)
					}
				}

				err = db.Query.DeleteAllKeyPermissionsByKeyID(ctx, tx, key.ID)
				if err != nil {
					return fault.Wrap(err,
						fault.Internal("unable to clear permissions"),
						fault.Public("Failed to clear key permissions."),
					)
				}

				permissionsToInsert := []db.InsertKeyPermissionParams{}
				now := time.Now().UnixMilli()
				for _, reqPerm := range requestedPermissions {
					permissionsToInsert = append(permissionsToInsert, db.InsertKeyPermissionParams{
						KeyID:        key.ID,
						PermissionID: reqPerm.ID,
						WorkspaceID:  auth.AuthorizedWorkspaceID,
						CreatedAt:    now,
						UpdatedAt:    sql.NullInt64{Int64: now, Valid: true},
					})
				}

				if len(permissionsToInsert) > 0 {
					err = db.BulkQuery.InsertKeyPermissions(ctx, tx, permissionsToInsert)
					if err != nil {
						return fault.Wrap(err,
							fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
							fault.Internal("database error"),
							fault.Public("Failed to assign permissions."),
						)
					}
				}
			}

			if req.Roles != nil {
				var existingRoles []db.FindRolesByNamesRow
				existingRoles, err = db.Query.FindRolesByNames(ctx, tx, db.FindRolesByNamesParams{
					WorkspaceID: auth.AuthorizedWorkspaceID,
					Names:       *req.Roles,
				})
				if err != nil {
					return fault.Wrap(err,
						fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
						fault.Internal("database error"),
						fault.Public("Failed to retrieve roles."),
					)
				}

				// Find which roles need to be created
				existingRoleMap := make(map[string]db.FindRolesByNamesRow)
				for _, r := range existingRoles {
					existingRoleMap[r.Name] = r
				}

				requestedRoles := []db.FindRolesByNamesRow{}
				for _, requestedName := range *req.Roles {
					existingRole, exists := existingRoleMap[requestedName]
					if exists {
						requestedRoles = append(requestedRoles, existingRole)
						continue
					}

					return fault.New("role not found",
						fault.Code(codes.Data.Role.NotFound.URN()),
						fault.Internal("role not found"),
						fault.Public(fmt.Sprintf("Role '%s' was not found.", requestedName)),
					)
				}

				err = db.Query.DeleteAllKeyRolesByKeyID(ctx, tx, key.ID)
				if err != nil {
					return fault.Wrap(err,
						fault.Internal("unable to clear roles"),
						fault.Public("Failed to clear key roles."),
					)
				}

				// Insert all requested roles
				rolesToInsert := []db.InsertKeyRoleParams{}
				for _, reqRole := range requestedRoles {
					rolesToInsert = append(rolesToInsert, db.InsertKeyRoleParams{
						KeyID:       key.ID,
						RoleID:      reqRole.ID,
						WorkspaceID: auth.AuthorizedWorkspaceID,
						CreatedAtM:  time.Now().UnixMilli(),
					})
				}

				if len(rolesToInsert) > 0 {
					err = db.BulkQuery.InsertKeyRoles(ctx, tx, rolesToInsert)
					if err != nil {
						return fault.Wrap(err,
							fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
							fault.Internal("database error"),
							fault.Public("Failed to assign roles."),
						)
					}
				}
			}

			auditLogs = append(auditLogs, auditlog.AuditLog{
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Event:       auditlog.KeyUpdateEvent,
				ActorType:   auditlog.RootKeyActor,
				ActorID:     auth.Key.ID,
				ActorName:   "root key",
				ActorMeta:   map[string]any{},
				Display:     fmt.Sprintf("Updated key %s", key.ID),
				RemoteIP:    s.Location(),
				UserAgent:   s.UserAgent(),
				Resources: []auditlog.AuditLogResource{
					{
						Type:        auditlog.KeyResourceType,
						ID:          key.ID,
						DisplayName: key.Name.String,
						Name:        key.Name.String,
						Meta:        map[string]any{},
					},
					{
						Type:        auditlog.APIResourceType,
						ID:          key.Api.ID,
						DisplayName: key.Api.Name,
						Name:        key.Api.Name,
						Meta:        map[string]any{},
					},
				},
			})

			err = h.Auditlogs.Insert(ctx, tx, auditLogs)
			if err != nil {
				return err
			}

			return nil
		})

		// Break if successful
		if txErr == nil {
			break
		}

		// Check if error is retryable (deadlock or identity race condition)
		isRetryable := db.IsDeadlockError(txErr) || (db.IsDuplicateKeyError(txErr) && attempt < 2)

		if !isRetryable {
			break
		}
	}

	if txErr != nil {
		// Wrap retryable errors with appropriate message after exhausting retries
		if db.IsDuplicateKeyError(txErr) || db.IsDeadlockError(txErr) {
			return fault.Wrap(txErr,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("failed to update key after retries"),
				fault.Public("Failed to update key."),
			)
		}
		return txErr
	}

	h.KeyCache.Remove(ctx, key.Hash)
	if req.Credits.IsSpecified() && oldCreditID != "" {
		// Invalidate the old credit ID (it may have been deleted or updated)
		if err := h.UsageLimiter.Invalidate(ctx, oldCreditID); err != nil {
			h.Logger.Error("Failed to invalidate usage limit",
				"error", err.Error(),
				"credit_id", oldCreditID,
			)
		}
	}

	// Return success response
	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.EmptyResponse{},
	})
}
