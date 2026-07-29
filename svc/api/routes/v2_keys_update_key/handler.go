package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/internal/services/auditlogs"
	keysdb "github.com/unkeyed/unkey/internal/services/keys/db"
	"github.com/unkeyed/unkey/internal/services/usagelimiter"
	"github.com/unkeyed/unkey/svc/api/internal/projects"
	"github.com/unkeyed/unkey/svc/api/openapi"

	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	dbtype "github.com/unkeyed/unkey/pkg/db/types"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/retry"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
)

type (
	Request  = openapi.V2KeysUpdateKeyRequestBody
	Response = openapi.V2KeysUpdateKeyResponseBody
)

type Handler struct {
	DB           db.Database
	Auditlogs    auditlogs.AuditLogService
	KeyCache     cache.Cache[string, keysdb.CachedKeyData]
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
	// Mint a correlation ID so all audit events from this update (the
	// key.update plus any per-permission / per-role disconnect+connect
	// pairs) share one ID for dashboard drill-down.
	ctx = auditlog.WithCorrelation(ctx, auditlog.NewCorrelationID())

	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	key, err := db.Query.FindLiveKeyForUpdateByID(ctx, h.DB.RO(), req.KeyId)
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

	if key.WorkspaceID != principal.WorkspaceID {
		return fault.New("key not found",
			fault.Code(codes.Data.Key.NotFound.URN()),
			fault.Internal("key belongs to different workspace"),
			fault.Public("The specified key was not found."),
		)
	}

	err = principal.Authorize(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Api,
			ResourceID:   "*",
			Action:       rbac.UpdateKey,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Api,
			ResourceID:   key.ApiID,
			Action:       rbac.UpdateKey,
		}),
		rbac.U(
			urn.New().Workspace(principal.WorkspaceID).Keyspace(key.KeyAuthID).Key(req.KeyId),
			permissions.UpdateKey{},
		),
	))
	if err != nil {
		return err
	}

	needsIdentity := req.ExternalId.IsSpecified() && !req.ExternalId.IsNull()

	// Concurrent requests can race while auto-creating the same identity or permission.
	// Retry the whole transaction so the loser observes and assigns the winner.
	txErr := retry.New(
		retry.Attempts(5),
		retry.ShouldRetry(db.IsDuplicateKeyError),
	).DoContext(ctx, func() error {
		return db.TxRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
			projectID := ""
			attemptIdentityID := sql.NullString{}
			existingPermissionsBySlug := make(map[string]db.Permission)
			existingRolesByName := make(map[string]db.FindRolesByNamesRow)
			findIdentity := int64(0)
			externalID := ""
			if needsIdentity {
				externalID = req.ExternalId.MustGet()
				if key.IdentityID.Valid && key.IdentityExternalID.Valid && key.IdentityExternalID.String == externalID {
					attemptIdentityID = key.IdentityID
				} else {
					findIdentity = 1
				}
			}
			permissionSlugs := []string{}
			if req.Permissions != nil {
				permissionSlugs = *req.Permissions
			}
			roleNames := []string{}
			if req.Roles != nil {
				roleNames = *req.Roles
			}
			if findIdentity == 1 || len(permissionSlugs) > 0 || len(roleNames) > 0 {
				resources, findErr := db.Query.FindKeyMutationResources(ctx, tx, db.FindKeyMutationResourcesParams{
					FindIdentity:    findIdentity,
					WorkspaceID:     principal.WorkspaceID,
					ExternalID:      externalID,
					PermissionSlugs: permissionSlugs,
					RoleNames:       roleNames,
				})
				if findErr != nil {
					return fault.Wrap(findErr,
						fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
						fault.Internal("failed to find key mutation resources"),
						fault.Public("Failed to retrieve key resources."),
					)
				}
				for _, resource := range resources {
					switch resource.ResourceType {
					case "identity":
						attemptIdentityID = sql.NullString{String: resource.IdentityID, Valid: true}
					case "permission":
						existingPermissionsBySlug[resource.PermissionSlug] = db.Permission{ID: resource.PermissionID, Slug: resource.PermissionSlug} //nolint:exhaustruct
					case "role":
						existingRolesByName[resource.RoleName] = db.FindRolesByNamesRow{ID: resource.RoleID, Name: resource.RoleName}
					case "project":
						projectID = resource.ProjectID
					}
				}
			}

			auditLogs := []auditlog.AuditLog{}

			update := db.UpdateKeyParams{
				ID:                         key.ID,
				Now:                        sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
				NameSpecified:              0,
				Name:                       sql.NullString{Valid: false, String: ""},
				IdentityIDSpecified:        0,
				IdentityID:                 sql.NullString{Valid: false, String: ""},
				EnabledSpecified:           0,
				Enabled:                    sql.NullBool{Valid: false, Bool: false},
				MetaSpecified:              0,
				Meta:                       sql.NullString{Valid: false, String: ""},
				ExpiresSpecified:           0,
				Expires:                    sql.NullTime{Valid: false, Time: time.Time{}},
				RemainingRequestsSpecified: 0,
				RemainingRequests:          sql.NullInt64{Valid: false, Int64: 0},
				RefillAmountSpecified:      0,
				RefillAmount:               sql.NullInt64{Valid: false, Int64: 0},
				RefillDaySpecified:         0,
				RefillDay:                  sql.NullInt16{Valid: false, Int16: 0},
			}

			if req.Name.IsSpecified() {
				update.NameSpecified = 1
				if req.Name.IsNull() {
					update.Name = sql.NullString{Valid: false, String: ""}
				} else {
					update.Name = sql.NullString{Valid: true, String: req.Name.MustGet()}
				}
			}

			if req.ExternalId.IsSpecified() {
				update.IdentityIDSpecified = 1
				if req.ExternalId.IsNull() {
					update.IdentityID = sql.NullString{Valid: false, String: ""}
				} else {
					externalID := req.ExternalId.MustGet()

					if !attemptIdentityID.Valid {
						if projectID == "" {
							projectID, err = projects.EnsureDefaultProject(ctx, tx, principal.WorkspaceID)
							if err != nil {
								return err
							}
						}

						identityID := uid.New(uid.IdentityPrefix)
						err = db.Query.InsertIdentity(ctx, tx, db.InsertIdentityParams{
							ID:          identityID,
							ExternalID:  externalID,
							WorkspaceID: principal.WorkspaceID,
							ProjectID:   projectID,
							Environment: "default",
							CreatedAt:   time.Now().UnixMilli(),
							Meta:        []byte("{}"),
						})
						if err != nil {
							return fault.Wrap(err,
								fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
								fault.Internal("failed to insert identity"),
								fault.Public("Failed to create identity."),
							)
						}
						attemptIdentityID = sql.NullString{Valid: true, String: identityID}
					}

					update.IdentityID = attemptIdentityID
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

			//nolint:nestif
			if req.Credits.IsSpecified() {
				if req.Credits.IsNull() {
					update.RemainingRequestsSpecified = 1
					update.RefillAmountSpecified = 1
					update.RefillDaySpecified = 1
					update.RefillAmount = sql.NullInt64{Valid: false, Int64: 0}
					update.RefillDay = sql.NullInt16{Valid: false, Int16: 0}
					update.RemainingRequests = sql.NullInt64{Valid: false, Int64: 0}
				} else {
					credits := req.Credits.MustGet()
					if credits.Remaining.IsSpecified() {
						update.RemainingRequestsSpecified = 1
						if credits.Remaining.IsNull() {
							// This also clears refilling
							update.RefillAmountSpecified = 1
							update.RefillDaySpecified = 1
							update.RemainingRequests = sql.NullInt64{Valid: false, Int64: 0}
							update.RefillAmount = sql.NullInt64{Valid: false, Int64: 0}
							update.RefillDay = sql.NullInt16{Valid: false, Int16: 0}
						} else {
							update.RemainingRequests = sql.NullInt64{
								Valid: true,
								Int64: credits.Remaining.MustGet(),
							}
						}
					}

					if credits.Refill.IsSpecified() {
						if credits.Refill.IsNull() {
							update.RefillAmountSpecified = 1
							update.RefillDaySpecified = 1
							update.RefillAmount = sql.NullInt64{Valid: false, Int64: 0}
							update.RefillDay = sql.NullInt16{Valid: false, Int16: 0}
						} else {
							refill := credits.Refill.MustGet()
							update.RefillAmountSpecified = 1
							update.RefillAmount = sql.NullInt64{
								Valid: true,
								Int64: refill.Amount,
							}

							update.RefillDaySpecified = 1
							switch refill.Interval {
							case openapi.UpdateKeyCreditsRefillIntervalMonthly:
								if refill.RefillDay == nil {
									return fault.New("missing refillDay",
										fault.Code(codes.App.Validation.InvalidInput.URN()),
										fault.Internal("refillDay required for monthly interval"),
										fault.Public("`refillDay` must be provided when the refill interval is `monthly`."),
									)
								}

								update.RefillDay = sql.NullInt16{
									Valid: true,
									Int16: int16(*refill.RefillDay), // nolint:gosec
								}
							case openapi.UpdateKeyCreditsRefillIntervalDaily:
								if refill.RefillDay != nil {
									return fault.New("invalid refillDay",
										fault.Code(codes.App.Validation.InvalidInput.URN()),
										fault.Internal("refillDay cannot be set for daily interval"),
										fault.Public("`refillDay` must not be provided when the refill interval is `daily`."),
									)
								}

								// For daily, refill_day should remain NULL
								update.RefillDay = sql.NullInt16{Valid: false, Int16: 0}
							}
						}
					}
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

			if req.Ratelimits != nil {
				newRatelimitMap := make(map[string]openapi.RatelimitRequest)
				for _, rl := range *req.Ratelimits {
					newRatelimitMap[rl.Name] = rl
				}

				if len(newRatelimitMap) == 0 {
					err = db.Query.DeleteAllRatelimitsByKeyID(ctx, tx, sql.NullString{String: key.ID, Valid: true})
					if err != nil {
						return fault.Wrap(err,
							fault.Internal("unable to delete key ratelimits"),
							fault.Public("Failed to delete ratelimits."),
						)
					}
				} else {
					ratelimitNames := make([]string, 0, len(newRatelimitMap))
					for name := range newRatelimitMap {
						ratelimitNames = append(ratelimitNames, name)
					}
					err = db.Query.DeleteRatelimitsByKeyIDExceptNames(ctx, tx, db.DeleteRatelimitsByKeyIDExceptNamesParams{
						KeyID:          sql.NullString{String: key.ID, Valid: true},
						RatelimitNames: ratelimitNames,
					})
					if err != nil {
						return fault.Wrap(err,
							fault.Internal("unable to delete stale key ratelimits"),
							fault.Public("Failed to delete ratelimits."),
						)
					}
				}

				ratelimitsToInsert := make([]db.InsertKeyRatelimitParams, 0, len(newRatelimitMap))
				now := time.Now().UnixMilli()
				for _, ratelimit := range newRatelimitMap {
					ratelimitsToInsert = append(ratelimitsToInsert, db.InsertKeyRatelimitParams{
						ID:          uid.New(uid.RatelimitPrefix),
						WorkspaceID: principal.WorkspaceID,
						KeyID:       sql.NullString{String: key.ID, Valid: true},
						Name:        ratelimit.Name,
						Limit:       uint64(ratelimit.Limit),
						Duration:    uint64(ratelimit.Duration),
						CreatedAt:   now,
						UpdatedAt:   sql.NullInt64{Int64: now, Valid: true},
						AutoApply:   ratelimit.AutoApply,
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

			clearedKeyPermissionsAndRoles := req.Permissions != nil && req.Roles != nil
			if clearedKeyPermissionsAndRoles {
				err = db.Query.DeleteAllKeyPermissionsAndRolesByKeyID(ctx, tx, key.ID)
				if err != nil {
					return fault.Wrap(err,
						fault.Internal("unable to clear permissions and roles"),
						fault.Public("Failed to clear key permissions and roles."),
					)
				}
			}

			if req.Permissions != nil {
				requestedPermissions := []db.Permission{}
				if len(*req.Permissions) > 0 {
					for _, requestedSlug := range *req.Permissions {
						if _, exists := existingPermissionsBySlug[requestedSlug]; !exists {
							if projectID == "" {
								projectID, err = projects.EnsureDefaultProject(ctx, tx, principal.WorkspaceID)
								if err != nil {
									return err
								}
							}
							break
						}
					}

					permissionsToCreate := []db.InsertPermissionParams{}
					for _, requestedSlug := range *req.Permissions {
						existingPerm, exists := existingPermissionsBySlug[requestedSlug]
						if exists {
							requestedPermissions = append(requestedPermissions, existingPerm)
							continue
						}

						newPermID := uid.New(uid.PermissionPrefix)
						permissionsToCreate = append(permissionsToCreate, db.InsertPermissionParams{
							PermissionID: newPermID,
							WorkspaceID:  principal.WorkspaceID,
							ProjectID:    projectID,
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
								WorkspaceID:   principal.WorkspaceID,
								Event:         auditlog.PermissionCreateEvent,
								ActorType:     auditlog.AuditLogActor(principal.Subject.Type),
								ActorID:       principal.Subject.ID,
								ActorName:     principal.Subject.Name,
								ActorMeta:     map[string]any{},
								Display:       fmt.Sprintf("Created %s (%s)", toCreate.Slug, toCreate.PermissionID),
								RemoteIP:      s.Location(),
								UserAgent:     s.UserAgent(),
								CorrelationID: "",
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
				}

				if !clearedKeyPermissionsAndRoles {
					err = db.Query.DeleteAllKeyPermissionsByKeyID(ctx, tx, key.ID)
					if err != nil {
						return fault.Wrap(err,
							fault.Internal("unable to clear permissions"),
							fault.Public("Failed to clear key permissions."),
						)
					}
				}

				permissionsToInsert := []db.InsertKeyPermissionParams{}
				now := time.Now().UnixMilli()
				for _, reqPerm := range requestedPermissions {
					permissionsToInsert = append(permissionsToInsert, db.InsertKeyPermissionParams{
						KeyID:        key.ID,
						PermissionID: reqPerm.ID,
						WorkspaceID:  principal.WorkspaceID,
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
				requestedRoles := []db.FindRolesByNamesRow{}
				if len(*req.Roles) > 0 {
					for _, requestedName := range *req.Roles {
						existingRole, exists := existingRolesByName[requestedName]
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
				}

				if !clearedKeyPermissionsAndRoles {
					err = db.Query.DeleteAllKeyRolesByKeyID(ctx, tx, key.ID)
					if err != nil {
						return fault.Wrap(err,
							fault.Internal("unable to clear roles"),
							fault.Public("Failed to clear key roles."),
						)
					}
				}

				// Insert all requested roles
				rolesToInsert := []db.InsertKeyRoleParams{}
				for _, reqRole := range requestedRoles {
					rolesToInsert = append(rolesToInsert, db.InsertKeyRoleParams{
						KeyID:       key.ID,
						RoleID:      reqRole.ID,
						WorkspaceID: principal.WorkspaceID,
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
				WorkspaceID:   principal.WorkspaceID,
				Event:         auditlog.KeyUpdateEvent,
				ActorType:     auditlog.AuditLogActor(principal.Subject.Type),
				ActorID:       principal.Subject.ID,
				ActorName:     principal.Subject.Name,
				ActorMeta:     map[string]any{},
				Display:       fmt.Sprintf("Updated key %s", key.ID),
				RemoteIP:      s.Location(),
				UserAgent:     s.UserAgent(),
				CorrelationID: "",
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
						ID:          key.ApiID,
						DisplayName: key.ApiName,
						Name:        key.ApiName,
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
	})

	if txErr != nil {
		return txErr
	}

	h.KeyCache.Remove(ctx, key.Hash)
	if req.Credits.IsSpecified() {
		if err := h.UsageLimiter.Invalidate(ctx, key.ID); err != nil {
			logger.Error("Failed to invalidate usage limit",
				"error", err.Error(),
				"key_id", key.ID,
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
