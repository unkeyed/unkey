package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/gen/rpc/vault"
	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/pkg/auditlog"
	authprincipal "github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	dbtype "github.com/unkeyed/unkey/pkg/db/types"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/retry"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/auditactor"
	apierrors "github.com/unkeyed/unkey/svc/api/internal/errors"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2KeysCreateKeyRequestBody
	Response = openapi.V2KeysCreateKeyResponseBody
)

type Handler struct {
	DB        db.Database
	Keys      keys.KeyService
	Auditlogs auditlogs.AuditLogService
	Vault     vault.VaultServiceClient
}

// Method returns the HTTP method this route responds to.
func (h *Handler) Method() string {
	return http.MethodPost
}

// Path returns the URL path pattern this route matches.
func (h *Handler) Path() string {
	return "/v2/keys.createKey"
}

// Handle processes the HTTP request.
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	ctx = auditlog.WithCorrelation(ctx, auditlog.NewCorrelationID())
	auditPreparer, ok := h.Auditlogs.(auditlogs.OutboxPreparer)
	if !ok {
		return fault.New("audit outbox preparation unavailable", fault.Internal("create key requires audit outbox preparation"))
	}

	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}
	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	apiAndKeySpace, err := db.Query.FindApiAndKeySpaceByID(ctx, h.DB.RO(), req.ApiId)
	if err != nil {
		if db.IsNotFound(err) {
			return fault.New("api not found",
				fault.Code(codes.Data.Api.NotFound.URN()),
				fault.Internal("api not found"), fault.Public("The specified API was not found."),
			)
		}
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"), fault.Public("Failed to retrieve API."),
		)
	}
	if apiAndKeySpace.WorkspaceID != principal.WorkspaceID {
		return fault.New("api not found",
			fault.Code(codes.Data.Api.NotFound.URN()),
			fault.Internal("api belongs to different workspace"), fault.Public("The specified API was not found."),
		)
	}

	err = principal.Authorize(rbac.Or(
		rbac.T(rbac.Tuple{ResourceType: rbac.Api, ResourceID: req.ApiId, Action: rbac.CreateKey}),
		rbac.T(rbac.Tuple{ResourceType: rbac.Api, ResourceID: "*", Action: rbac.CreateKey}),
		rbac.U(
			urn.New().Workspace(principal.WorkspaceID).Keyspace(apiAndKeySpace.KeyAuthID.String),
			permissions.CreateKey{},
		),
	))
	if err != nil {
		return apierrors.MaskInsufficientPermissionsAsNotFound(
			err,
			codes.Data.Api.NotFound.URN(),
			"The specified API was not found.",
		)
	}

	switch source := principal.Source.(type) {
	case authprincipal.PortalSessionSource:
		if source.ExternalID == "" {
			return fault.New("portal session missing identity",
				fault.Code(codes.App.Internal.UnexpectedError.URN()),
				fault.Internal("portal session externalId is empty"),
				fault.Public("An internal error occurred."),
			)
		}
		req.ExternalId = &source.ExternalID
	}
	actor := auditactor.FromPrincipal(principal)

	if apiAndKeySpace.KeySpaceID == "" {
		return fault.New("api not set up for keys",
			fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
			fault.Internal("api not set up for keys, keyspace not found"), fault.Public("The requested API is not set up to handle keys."),
		)
	}

	var prefix string
	switch {
	case req.Prefix != nil:
		prefix = *req.Prefix
	case apiAndKeySpace.DefaultPrefix.Valid:
		prefix = apiAndKeySpace.DefaultPrefix.String
	}
	byteLength := ptr.SafeDeref(req.ByteLength)
	if byteLength == 0 && apiAndKeySpace.DefaultBytes.Valid {
		byteLength = int(apiAndKeySpace.DefaultBytes.Int32)
	}
	if byteLength == 0 {
		byteLength = 16
	}

	encrypt := ptr.SafeDeref(req.Recoverable, false)
	if encrypt {
		if h.Vault == nil {
			return fault.New("vault missing",
				fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
				fault.Public("Vault hasn't been set up."),
			)
		}
		err = principal.Authorize(rbac.Or(
			rbac.T(rbac.Tuple{ResourceType: rbac.Api, ResourceID: "*", Action: rbac.EncryptKey}),
			rbac.T(rbac.Tuple{ResourceType: rbac.Api, ResourceID: apiAndKeySpace.ApiID, Action: rbac.EncryptKey}),
			rbac.U(urn.New().Workspace(principal.WorkspaceID).Keyspace(apiAndKeySpace.KeySpaceID).Key("*"), permissions.EncryptKey{}),
		))
		if err != nil {
			return apierrors.MaskInsufficientPermissionsAsNotFound(
				err,
				codes.Data.Api.NotFound.URN(),
				"The specified API was not found.",
			)
		}
		if !apiAndKeySpace.StoreEncryptedKeys {
			return fault.New("api not set up for key encryption",
				fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
				fault.Internal("api not set up for key encryption"), fault.Public("This API does not support key encryption."),
			)
		}
	}

	if req.Credits != nil && req.Credits.Refill != nil {
		if !req.Credits.Remaining.IsSpecified() || req.Credits.Remaining.IsNull() {
			return fault.New("missing credits.remaining",
				fault.Code(codes.App.Validation.InvalidInput.URN()),
				fault.Internal("credits.remaining required when refill is set"),
				fault.Public("`credits.remaining` must be provided when `credits.refill` is set."),
			)
		}
		if req.Credits.Refill.Interval == openapi.KeyCreditsRefillIntervalMonthly && req.Credits.Refill.RefillDay == 0 {
			return fault.New("missing refillDay",
				fault.Code(codes.App.Validation.InvalidInput.URN()),
				fault.Internal("refillDay required for monthly interval"),
				fault.Public("`refillDay` must be provided when the refill interval is `monthly`."),
			)
		}
	}

	permissionSlugs := ptr.SafeDeref(req.Permissions, []string{})
	roleNames := ptr.SafeDeref(req.Roles, []string{})
	existingPermissions := make(map[string]db.Permission, len(permissionSlugs))
	existingRoles := make(map[string]db.FindRolesByNamesRow, len(roleNames))
	identityID := sql.NullString{}
	projectID := ""
	if req.ExternalId != nil || len(permissionSlugs) > 0 || len(roleNames) > 0 {
		findIdentity := int64(0)
		externalID := ""
		if req.ExternalId != nil {
			findIdentity = 1
			externalID = *req.ExternalId
		}
		resources, findErr := db.Query.FindKeyMutationResources(ctx, h.DB.RW(), db.FindKeyMutationResourcesParams{
			FindIdentity:    findIdentity,
			WorkspaceID:     principal.WorkspaceID,
			ExternalID:      externalID,
			PermissionSlugs: permissionSlugs,
			RoleNames:       roleNames,
			KeyID:           sql.NullString{},
			RatelimitNames:  nil,
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
				identityID = sql.NullString{String: resource.IdentityID, Valid: true}
			case "permission":
				existingPermissions[resource.PermissionSlug] = db.Permission{ID: resource.PermissionID, Slug: resource.PermissionSlug} //nolint:exhaustruct
			case "role":
				existingRoles[resource.RoleName] = db.FindRolesByNamesRow{ID: resource.RoleID, Name: resource.RoleName}
			case "project":
				projectID = resource.ProjectID
			}
		}
	}
	for _, roleName := range roleNames {
		if _, ok := existingRoles[roleName]; !ok {
			return fault.New("role not found",
				fault.Code(codes.Data.Role.NotFound.URN()),
				fault.Internal("role not found"), fault.Public(fmt.Sprintf("Role '%s' was not found.", roleName)),
			)
		}
	}

	keyResult, err := h.Keys.CreateKey(ctx, keys.CreateKeyRequest{Prefix: prefix, ByteLength: byteLength})
	if err != nil {
		return err
	}
	var encryption *vaultv1.EncryptResponse
	if encrypt {
		encryption, err = h.Vault.Encrypt(ctx, &vaultv1.EncryptRequest{
			Keyring: principal.WorkspaceID,
			Data:    keyResult.Key,
		})
		if err != nil {
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("vault error"), fault.Public("Failed to encrypt key in vault."),
			)
		}
	}

	now := time.Now().UnixMilli()
	var keyID string
	var preparationErr error
	batchErr := retry.New(
		retry.Attempts(5),
		retry.ShouldRetry(db.IsDuplicateKeyError),
	).DoContext(ctx, func() error {
		keyID = uid.New(uid.KeyPrefix)
		var batch db.CreateKeyBatchParams
		insertKey := db.InsertKeyParams{
			ID:                 keyID,
			KeySpaceID:         apiAndKeySpace.KeyAuthID.String,
			Hash:               keyResult.Hash,
			Start:              keyResult.Start,
			WorkspaceID:        principal.WorkspaceID,
			CreatedAtM:         now,
			Enabled:            true,
			ForWorkspaceID:     sql.NullString{},
			Name:               sql.NullString{},
			IdentityID:         identityID,
			Meta:               sql.NullString{},
			Expires:            sql.NullTime{},
			RemainingRequests:  sql.NullInt64{},
			RefillDay:          sql.NullInt16{},
			RefillAmount:       sql.NullInt64{},
			PendingMigrationID: sql.NullString{},
		}
		if req.Name != nil {
			insertKey.Name = sql.NullString{String: *req.Name, Valid: true}
		}
		if req.Meta != nil {
			meta, marshalErr := json.Marshal(*req.Meta)
			if marshalErr != nil {
				preparationErr = fault.Wrap(marshalErr,
					fault.Code(codes.App.Validation.InvalidInput.URN()),
					fault.Internal("failed to marshal meta"), fault.Public("Invalid metadata format."),
				)
				return preparationErr
			}
			insertKey.Meta = sql.NullString{String: string(meta), Valid: true}
		}
		if req.Expires != nil {
			insertKey.Expires = sql.NullTime{Time: time.UnixMilli(*req.Expires), Valid: true}
		}
		if req.Credits != nil {
			if req.Credits.Remaining.IsSpecified() && !req.Credits.Remaining.IsNull() {
				insertKey.RemainingRequests = sql.NullInt64{Int64: req.Credits.Remaining.MustGet(), Valid: true}
			}
			if req.Credits.Refill != nil {
				insertKey.RefillAmount = sql.NullInt64{Int64: req.Credits.Refill.Amount, Valid: true}
				if req.Credits.Refill.Interval == openapi.KeyCreditsRefillIntervalMonthly {
					insertKey.RefillDay = sql.NullInt16{Int16: int16(req.Credits.Refill.RefillDay), Valid: true} //nolint:gosec
				}
			}
		}
		if req.Enabled != nil {
			insertKey.Enabled = *req.Enabled
		}

		missingResource := req.ExternalId != nil && !identityID.Valid
		for _, slug := range permissionSlugs {
			if _, ok := existingPermissions[slug]; !ok {
				missingResource = true
				break
			}
		}
		resourceProjectID := projectID
		if missingResource && resourceProjectID == "" {
			resourceProjectID = uid.New(uid.ProjectPrefix)
			batch.Project = &db.UpsertDefaultProjectParams{
				ID:          resourceProjectID,
				WorkspaceID: principal.WorkspaceID,
				CreatedAt:   now,
			}
		}
		if req.ExternalId != nil && !identityID.Valid {
			newIdentityID := uid.New(uid.IdentityPrefix)
			insertKey.IdentityID = sql.NullString{String: newIdentityID, Valid: true}
			batch.Identity = &db.UpsertIdentityParams{
				ID:          newIdentityID,
				ExternalID:  *req.ExternalId,
				WorkspaceID: principal.WorkspaceID,
				ProjectID:   resourceProjectID,
				Environment: "default",
				CreatedAt:   now,
				Meta:        json.RawMessage("{}"),
			}
		}
		batch.Key = insertKey
		if encryption != nil {
			batch.Encryption = &db.InsertKeyEncryptionParams{
				WorkspaceID:     principal.WorkspaceID,
				KeyID:           keyID,
				Encrypted:       encryption.GetEncrypted(),
				EncryptionKeyID: encryption.GetKeyId(),
				CreatedAt:       now,
			}
		}
		for _, limit := range ptr.SafeDeref(req.Ratelimits, []openapi.RatelimitRequest{}) {
			batch.Ratelimits = append(batch.Ratelimits, db.InsertKeyRatelimitParams{
				ID:          uid.New(uid.RatelimitPrefix),
				WorkspaceID: principal.WorkspaceID,
				KeyID:       sql.NullString{String: keyID, Valid: true},
				Name:        limit.Name,
				Limit:       uint64(limit.Limit),
				Duration:    uint64(limit.Duration),
				CreatedAt:   now,
				UpdatedAt:   sql.NullInt64{},
				AutoApply:   limit.AutoApply,
			})
		}

		auditLogs := make([]auditlog.AuditLog, 0, len(permissionSlugs)+len(roleNames)+1)
		for _, slug := range permissionSlugs {
			permission, exists := existingPermissions[slug]
			var insertPermission *db.UpsertPermissionParams
			if !exists {
				permission = db.Permission{ID: uid.New(uid.PermissionPrefix), Slug: slug} //nolint:exhaustruct
				insertPermission = &db.UpsertPermissionParams{
					PermissionID: permission.ID,
					WorkspaceID:  principal.WorkspaceID,
					ProjectID:    resourceProjectID,
					Name:         slug,
					Slug:         slug,
					Description:  dbtype.NullString{String: "", Valid: false},
					CreatedAtM:   now,
				}
			}
			batch.Permissions = append(batch.Permissions, db.CreateKeyBatchPermission{
				Permission: insertPermission,
				Link: db.InsertKeyPermissionParams{
					KeyID:        keyID,
					PermissionID: permission.ID,
					WorkspaceID:  principal.WorkspaceID,
					CreatedAt:    now,
					UpdatedAt:    sql.NullInt64{},
				},
				Outbox: db.InsertClickhouseOutboxParams{
					Version:     "",
					WorkspaceID: "",
					EventID:     "",
					Payload:     nil,
					CreatedAt:   0,
				},
			})
			auditLogs = append(auditLogs, permissionAuditLog(principal.WorkspaceID, actor, s, keyID, insertKey.Name.String, permission.ID, slug))
		}
		for _, roleName := range roleNames {
			role := existingRoles[roleName]
			batch.Roles = append(batch.Roles, db.InsertKeyRoleParams{
				KeyID:       keyID,
				RoleID:      role.ID,
				WorkspaceID: principal.WorkspaceID,
				CreatedAtM:  now,
			})
			auditLogs = append(auditLogs, roleAuditLog(principal.WorkspaceID, actor, s, keyID, insertKey.Name.String, role))
		}
		auditLogs = append(auditLogs, createAuditLog(principal.WorkspaceID, actor, s, keyID, req.ApiId, apiAndKeySpace.ApiName))
		outboxRows, prepareErr := auditPreparer.PrepareOutboxRows(ctx, auditLogs)
		if prepareErr != nil {
			preparationErr = prepareErr
			return prepareErr
		}
		if len(outboxRows) != len(auditLogs) {
			preparationErr = fault.New("invalid audit outbox batch size", fault.Internal("create key audit outbox row count does not match event count"))
			return preparationErr
		}
		for i := range batch.Permissions {
			batch.Permissions[i].Outbox = outboxRows[i]
		}
		batch.Outbox = outboxRows[len(batch.Permissions):]
		return h.DB.BatchRW().CreateKeyWithAuditBatch(ctx, batch)
	})
	if preparationErr != nil {
		return preparationErr
	}
	if batchErr != nil {
		return fault.Wrap(batchErr,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to create key."),
		)
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: openapi.V2KeysCreateKeyResponseData{KeyId: keyID, Key: keyResult.Key},
	})
}

func permissionAuditLog(workspaceID string, actor auditactor.Actor, s *zen.Session, keyID, keyName, permissionID, slug string) auditlog.AuditLog {
	return auditlog.AuditLog{
		WorkspaceID:   workspaceID,
		Event:         auditlog.AuthConnectPermissionKeyEvent,
		ActorType:     actor.Type,
		ActorID:       actor.ID,
		ActorName:     actor.Name,
		ActorMeta:     actor.Meta,
		Display:       fmt.Sprintf("Added permission %s to key %s", slug, keyID),
		RemoteIP:      s.Location(),
		UserAgent:     s.UserAgent(),
		CorrelationID: "",
		Resources: []auditlog.AuditLogResource{
			{Type: auditlog.KeyResourceType, ID: keyID, Name: keyName, DisplayName: keyName, Meta: map[string]any{}},
			{Type: auditlog.PermissionResourceType, ID: permissionID, Name: slug, DisplayName: slug, Meta: map[string]any{}},
		},
	}
}

func roleAuditLog(workspaceID string, actor auditactor.Actor, s *zen.Session, keyID, keyName string, role db.FindRolesByNamesRow) auditlog.AuditLog {
	return auditlog.AuditLog{
		WorkspaceID:   workspaceID,
		Event:         auditlog.AuthConnectRoleKeyEvent,
		ActorType:     actor.Type,
		ActorID:       actor.ID,
		ActorName:     actor.Name,
		ActorMeta:     actor.Meta,
		Display:       fmt.Sprintf("Connected role %s to key %s", role.Name, keyID),
		RemoteIP:      s.Location(),
		UserAgent:     s.UserAgent(),
		CorrelationID: "",
		Resources: []auditlog.AuditLogResource{
			{Type: auditlog.KeyResourceType, ID: keyID, Name: keyName, DisplayName: keyName, Meta: map[string]any{}},
			{Type: auditlog.RoleResourceType, ID: role.ID, Name: role.Name, DisplayName: role.Name, Meta: map[string]any{}},
		},
	}
}

func createAuditLog(workspaceID string, actor auditactor.Actor, s *zen.Session, keyID, apiID, apiName string) auditlog.AuditLog {
	return auditlog.AuditLog{
		WorkspaceID:   workspaceID,
		Event:         auditlog.KeyCreateEvent,
		ActorType:     actor.Type,
		ActorID:       actor.ID,
		ActorName:     actor.Name,
		ActorMeta:     actor.Meta,
		Display:       fmt.Sprintf("Created key %s", keyID),
		RemoteIP:      s.Location(),
		UserAgent:     s.UserAgent(),
		CorrelationID: "",
		Resources: []auditlog.AuditLogResource{
			{Type: auditlog.KeyResourceType, ID: keyID, DisplayName: keyID, Name: keyID, Meta: map[string]any{}},
			{Type: auditlog.APIResourceType, ID: apiID, DisplayName: apiName, Name: apiName, Meta: map[string]any{}},
		},
	}
}
