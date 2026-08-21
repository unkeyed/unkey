package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/unkeyed/unkey/internal/services/auditlogs"
	keysdb "github.com/unkeyed/unkey/internal/services/keys/db"
	"github.com/unkeyed/unkey/internal/services/usagelimiter"
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

// Method returns the HTTP method this route responds to.
func (h *Handler) Method() string {
	return http.MethodPost
}

// Path returns the URL path pattern this route matches.
func (h *Handler) Path() string {
	return "/v2/keys.updateKey"
}

// Handle processes the HTTP request.
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
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

	update := db.UpdateKeyParams{
		ID:                          key.ID,
		Now:                         sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		NameSpecified:               0,
		Name:                        sql.NullString{},
		IdentityIDSpecified:         0,
		IdentityExternalIDSpecified: 0,
		IdentityWorkspaceID:         principal.WorkspaceID,
		IdentityExternalID:          "",
		IdentityID:                  sql.NullString{},
		EnabledSpecified:            0,
		Enabled:                     sql.NullBool{},
		MetaSpecified:               0,
		Meta:                        sql.NullString{},
		ExpiresSpecified:            0,
		Expires:                     sql.NullTime{},
		RemainingRequestsSpecified:  0,
		RemainingRequests:           sql.NullInt64{},
		RefillAmountSpecified:       0,
		RefillAmount:                sql.NullInt64{},
		RefillDaySpecified:          0,
		RefillDay:                   sql.NullInt16{},
	}

	if req.Name.IsSpecified() {
		update.NameSpecified = 1
		if !req.Name.IsNull() {
			update.Name = sql.NullString{String: req.Name.MustGet(), Valid: true}
		}
	}
	if req.ExternalId.IsSpecified() {
		update.IdentityIDSpecified = 1
		if !req.ExternalId.IsNull() {
			update.IdentityExternalIDSpecified = 1
			update.IdentityExternalID = req.ExternalId.MustGet()
		}
	}
	if req.Enabled != nil {
		update.EnabledSpecified = 1
		update.Enabled = sql.NullBool{Bool: *req.Enabled, Valid: true}
	}
	if req.Meta.IsSpecified() {
		update.MetaSpecified = 1
		if !req.Meta.IsNull() {
			meta, marshalErr := json.Marshal(req.Meta.MustGet())
			if marshalErr != nil {
				return fault.Wrap(marshalErr,
					fault.Code(codes.App.Validation.InvalidInput.URN()),
					fault.Internal("failed to marshal meta"),
					fault.Public("Invalid metadata format."),
				)
			}
			update.Meta = sql.NullString{String: string(meta), Valid: true}
		}
	}
	if req.Expires.IsSpecified() {
		update.ExpiresSpecified = 1
		if !req.Expires.IsNull() {
			update.Expires = sql.NullTime{Time: time.UnixMilli(req.Expires.MustGet()), Valid: true}
		}
	}
	if req.Credits.IsSpecified() {
		if req.Credits.IsNull() {
			update.RemainingRequestsSpecified = 1
			update.RefillAmountSpecified = 1
			update.RefillDaySpecified = 1
		} else {
			credits := req.Credits.MustGet()
			if credits.Remaining.IsSpecified() {
				update.RemainingRequestsSpecified = 1
				if credits.Remaining.IsNull() {
					update.RefillAmountSpecified = 1
					update.RefillDaySpecified = 1
				} else {
					update.RemainingRequests = sql.NullInt64{Int64: credits.Remaining.MustGet(), Valid: true}
				}
			}
			if credits.Refill.IsSpecified() {
				update.RefillAmountSpecified = 1
				update.RefillDaySpecified = 1
				if !credits.Refill.IsNull() {
					refill := credits.Refill.MustGet()
					update.RefillAmount = sql.NullInt64{Int64: refill.Amount, Valid: true}
					switch refill.Interval {
					case openapi.UpdateKeyCreditsRefillIntervalMonthly:
						if refill.RefillDay == nil {
							return fault.New("missing refillDay",
								fault.Code(codes.App.Validation.InvalidInput.URN()),
								fault.Internal("refillDay required for monthly interval"),
								fault.Public("`refillDay` must be provided when the refill interval is `monthly`."),
							)
						}
						update.RefillDay = sql.NullInt16{Int16: int16(*refill.RefillDay), Valid: true} //nolint:gosec
					case openapi.UpdateKeyCreditsRefillIntervalDaily:
						if refill.RefillDay != nil {
							return fault.New("invalid refillDay",
								fault.Code(codes.App.Validation.InvalidInput.URN()),
								fault.Internal("refillDay cannot be set for daily interval"),
								fault.Public("`refillDay` must not be provided when the refill interval is `daily`."),
							)
						}
					}
				}
			}
		}
	}

	permissionSlugs := uniqueSortedStrings(req.Permissions)
	roleNames := uniqueSortedStrings(req.Roles)
	requestedRatelimits := make(map[string]openapi.RatelimitRequest)
	if req.Ratelimits != nil {
		requestedRatelimits = make(map[string]openapi.RatelimitRequest, len(*req.Ratelimits))
		for _, ratelimit := range *req.Ratelimits {
			requestedRatelimits[ratelimit.Name] = ratelimit
		}
	}
	ratelimitNames := make([]string, 0, len(requestedRatelimits))
	for name := range requestedRatelimits {
		ratelimitNames = append(ratelimitNames, name)
	}
	sort.Strings(ratelimitNames)
	findIdentity := req.ExternalId.IsSpecified() && !req.ExternalId.IsNull() &&
		(!key.IdentityExternalID.Valid || key.IdentityExternalID.String != req.ExternalId.MustGet())

	existingPermissions := make(map[string]struct{}, len(permissionSlugs))
	existingRoles := make(map[string]db.FindRolesByNamesRow, len(roleNames))
	existingRatelimitIDs := make(map[string]string, len(ratelimitNames))
	identityExists := !findIdentity
	defaultProjectExists := false
	if findIdentity || len(permissionSlugs) > 0 || len(roleNames) > 0 || len(ratelimitNames) > 0 {
		findIdentityArg := int64(0)
		externalID := ""
		if findIdentity {
			findIdentityArg = 1
			externalID = req.ExternalId.MustGet()
		}
		resources, findErr := db.Query.FindKeyMutationResources(ctx, h.DB.RW(), db.FindKeyMutationResourcesParams{
			FindIdentity:    findIdentityArg,
			WorkspaceID:     principal.WorkspaceID,
			ExternalID:      externalID,
			PermissionSlugs: permissionSlugs,
			RoleNames:       roleNames,
			KeyID:           sql.NullString{String: key.ID, Valid: true},
			RatelimitNames:  ratelimitNames,
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
				identityExists = true
			case "permission":
				existingPermissions[resource.PermissionSlug] = struct{}{}
			case "role":
				existingRoles[resource.RoleName] = db.FindRolesByNamesRow{ID: resource.RoleID, Name: resource.RoleName}
			case "project":
				defaultProjectExists = true
			case "ratelimit":
				existingRatelimitIDs[resource.RatelimitName] = resource.RatelimitID
			}
		}
	}

	for _, roleName := range roleNames {
		if _, ok := existingRoles[roleName]; !ok {
			return fault.New("role not found",
				fault.Code(codes.Data.Role.NotFound.URN()),
				fault.Internal("role not found"),
				fault.Public(fmt.Sprintf("Role '%s' was not found.", roleName)),
			)
		}
	}

	batch := db.UpdateKeyBatchParams{
		WorkspaceID:        principal.WorkspaceID,
		WorkspaceLock:      nil,
		Project:            nil,
		Identity:           nil,
		Permissions:        nil,
		Update:             update,
		RatelimitDelete:    nil,
		Ratelimits:         nil,
		ReplacePermissions: req.Permissions != nil,
		ReplaceRoles:       req.Roles != nil,
		KeyPermissions:     nil,
		KeyRoles:           nil,
		Outboxes:           nil,
	}

	now := time.Now().UnixMilli()
	needsProject := findIdentity && !identityExists
	for _, slug := range permissionSlugs {
		if _, ok := existingPermissions[slug]; !ok {
			needsProject = true
		}
	}
	if needsProject && !defaultProjectExists {
		batch.Project = &db.InsertDefaultProjectForUpdateKeyParams{
			ID:          uid.New(uid.ProjectPrefix),
			WorkspaceID: principal.WorkspaceID,
			CreatedAt:   now,
		}
	}
	if findIdentity && !identityExists {
		batch.Identity = &db.InsertIdentityForUpdateKeyParams{
			ID:          uid.New(uid.IdentityPrefix),
			ExternalID:  req.ExternalId.MustGet(),
			WorkspaceID: principal.WorkspaceID,
			CreatedAt:   now,
		}
	}

	auditLogs := make([]auditlog.AuditLog, 0, len(permissionSlugs)+1)
	permissionIDs := make([]string, 0, len(permissionSlugs))
	for _, slug := range permissionSlugs {
		if _, ok := existingPermissions[slug]; !ok {
			permissionID := uid.New(uid.PermissionPrefix)
			permissionIDs = append(permissionIDs, permissionID)
			batch.Permissions = append(batch.Permissions, db.UpdateKeyPermissionBatchWrite{
				Permission: db.InsertPermissionForUpdateKeyParams{
					PermissionID: permissionID,
					WorkspaceID:  principal.WorkspaceID,
					Name:         slug,
					Slug:         slug,
					Description:  dbtype.NullString{String: fmt.Sprintf("Auto-created permission: %s", slug), Valid: true},
					CreatedAtM:   now,
				},
				Outbox: db.InsertClickhouseOutboxForPermissionUpdateKeyParams{}, //nolint:exhaustruct // populated after audit serialization
			})
			auditLogs = append(auditLogs, auditlog.AuditLog{
				WorkspaceID:   principal.WorkspaceID,
				Event:         auditlog.PermissionCreateEvent,
				ActorType:     auditlog.AuditLogActor(principal.Subject.Type),
				ActorID:       principal.Subject.ID,
				ActorName:     principal.Subject.Name,
				ActorMeta:     map[string]any{},
				Display:       fmt.Sprintf("Created %s (%s)", slug, permissionID),
				RemoteIP:      s.Location(),
				UserAgent:     s.UserAgent(),
				CorrelationID: "",
				Resources: []auditlog.AuditLogResource{{
					Type:        auditlog.PermissionResourceType,
					ID:          permissionID,
					Name:        slug,
					DisplayName: slug,
					Meta: map[string]any{
						"name": slug,
						"slug": slug,
					},
				}},
			})
		}
		batch.KeyPermissions = append(batch.KeyPermissions, db.InsertKeyPermissionBySlugForUpdateKeyParams{
			KeyID:          key.ID,
			WorkspaceID:    principal.WorkspaceID,
			CreatedAtM:     now,
			UpdatedAtM:     sql.NullInt64{Int64: now, Valid: true},
			PermissionSlug: slug,
		})
	}
	if batch.Project != nil || batch.Identity != nil || len(batch.Permissions) > 0 {
		batch.WorkspaceLock = &db.LockWorkspaceForUpdateKeyParams{
			WorkspaceID:      principal.WorkspaceID,
			WorkspaceIDCheck: principal.WorkspaceID,
		}
	}
	for _, roleName := range roleNames {
		batch.KeyRoles = append(batch.KeyRoles, db.InsertKeyRoleParams{
			KeyID:       key.ID,
			RoleID:      existingRoles[roleName].ID,
			WorkspaceID: principal.WorkspaceID,
			CreatedAtM:  now,
		})
	}

	if req.Ratelimits != nil {
		batch.RatelimitDelete = &db.DeleteKeyRatelimitsForUpdateKeyParams{
			KeyID:       sql.NullString{String: key.ID, Valid: true},
			WorkspaceID: principal.WorkspaceID,
		}
		for _, name := range ratelimitNames {
			ratelimit := requestedRatelimits[name]
			ratelimitID := existingRatelimitIDs[name]
			if ratelimitID == "" {
				ratelimitID = uid.New(uid.RatelimitPrefix)
			}
			batch.Ratelimits = append(batch.Ratelimits, db.InsertKeyRatelimitParams{
				ID:          ratelimitID,
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

	auditPreparer, ok := h.Auditlogs.(auditlogs.OutboxPreparer)
	if !ok {
		return fault.New("audit service cannot prepare outbox rows", fault.Internal("update key requires audit outbox preparation"))
	}
	outboxRows, err := auditPreparer.PrepareOutboxRows(ctx, auditLogs)
	if err != nil {
		return err
	}
	if len(outboxRows) != len(auditLogs) {
		return fault.New("invalid audit outbox batch size", fault.Internal("update key audit outbox row count does not match events"))
	}
	for i, permissionID := range permissionIDs {
		outbox := outboxRows[i]
		batch.Permissions[i].Outbox = db.InsertClickhouseOutboxForPermissionUpdateKeyParams{
			Version:      outbox.Version,
			WorkspaceID:  outbox.WorkspaceID,
			EventID:      outbox.EventID,
			Payload:      outbox.Payload,
			CreatedAt:    outbox.CreatedAt,
			PermissionID: permissionID,
		}
	}
	batch.Outboxes = append(batch.Outboxes, outboxRows[len(outboxRows)-1])

	batchErr := h.DB.BatchRW().UpdateKeyBatch(ctx, batch)
	h.KeyCache.Remove(ctx, key.Hash)
	if req.Credits.IsSpecified() {
		if invalidateErr := h.UsageLimiter.Invalidate(ctx, key.ID); invalidateErr != nil {
			logger.Error("Failed to invalidate usage limit",
				"error", invalidateErr.Error(),
				"key_id", key.ID,
			)
		}
	}
	if batchErr != nil {
		return fault.Wrap(batchErr,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to update key."),
		)
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: openapi.EmptyResponse{},
	})
}

func uniqueSortedStrings(values *[]string) []string {
	if values == nil {
		return nil
	}
	unique := make(map[string]struct{}, len(*values))
	for _, value := range *values {
		unique[value] = struct{}{}
	}
	result := make([]string, 0, len(unique))
	for value := range unique {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
