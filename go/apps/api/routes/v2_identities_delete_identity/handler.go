package handler

import (
	"context"
	"database/sql"
	"errors"
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

type Services struct {
	Logger      logging.Logger
	DB          db.Database
	Keys        keys.KeyService
	Permissions permissions.PermissionService
	Auditlogs   auditlogs.AuditLogService
}

func New(svc Services) zen.Route {
	return zen.NewRoute("POST", "/v2/identities.deleteIdentity", func(ctx context.Context, s *zen.Session) error {
		auth, err := svc.Keys.VerifyRootKey(ctx, s)
		if err != nil {
			return err
		}

		// nolint:exhaustruct
		req := Request{}
		err = s.BindBody(&req)
		if err != nil {
			return fault.Wrap(err,
				fault.WithDesc("invalid request body", "The request body is invalid."),
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

		permissions, err := svc.Permissions.Check(
			ctx,
			auth.KeyID,
			rbac.Or(checks...),
		)
		if err != nil {
			return fault.Wrap(err,
				fault.WithDesc("unable to check permissions", "We're unable to check the permissions of your key."),
			)
		}
		if !permissions.Valid {
			return fault.New("insufficient permissions",
				fault.WithCode(codes.Auth.Authorization.InsufficientPermissions.URN()),
				fault.WithDesc(permissions.Message, permissions.Message),
			)
		}

		tx, err := svc.DB.RW().Begin(ctx)
		if err != nil {
			return fault.Wrap(err,
				fault.WithCode(codes.App.Internal.ServiceUnavailable.URN()),
				fault.WithDesc("database failed to create transaction", "Unable to start database transaction."),
			)
		}

		defer func() {
			rollbackErr := tx.Rollback()
			if rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
				svc.Logger.Error("rollback failed", "requestId", s.RequestID(), "error", rollbackErr)
			}
		}()

		identity, err := getIdentity(ctx, svc, req, auth.AuthorizedWorkspaceID)
		if err != nil {
			if db.IsNotFound(err) {
				return fault.New("identity not found",
					fault.WithCode(codes.Data.Identity.NotFound.URN()),
					fault.WithDesc("identity not found", "This identity does not exist."),
				)
			}

			return fault.Wrap(err,
				fault.WithCode(codes.App.Internal.ServiceUnavailable.URN()),
				fault.WithDesc("database failed to find the identity", "Error finding the identity."),
			)
		}

		if identity.WorkspaceID != auth.AuthorizedWorkspaceID {
			return fault.New("identity not found",
				fault.WithCode(codes.Data.Identity.NotFound.URN()),
				fault.WithDesc("wrong workspace, masking as 404", "This identity does not exist."),
			)
		}

		ratelimits, err := db.Query.FindRatelimitsByIdentityID(ctx, svc.DB.RO(), sql.NullString{String: identity.ID, Valid: true})
		if err != nil && !db.IsNotFound(err) {
			return fault.Wrap(err,
				fault.WithCode(codes.App.Internal.ServiceUnavailable.URN()),
				fault.WithDesc("database failed to load identity ratelimits", "Failed to load Identity ratelimits."),
			)
		}

		if len(ratelimits) > 0 {
			ids := make([]string, 0)
			for _, ratelimit := range ratelimits {
				ids = append(ids, ratelimit.ID)
			}

			err = db.Query.DeleteManyRatelimitsByIDs(ctx, tx, ids)
			if err != nil {
				return fault.Wrap(err,
					fault.WithCode(codes.App.Internal.ServiceUnavailable.URN()),
					fault.WithDesc("database failed to delete identity ratelimits", "Failed to delete Identity ratelimits."),
				)
			}
		}

		err = db.Query.DeleteIdentity(ctx, tx, identity.ID)
		if err != nil {
			return fault.Wrap(err,
				fault.WithCode(codes.App.Internal.ServiceUnavailable.URN()),
				fault.WithDesc("database failed to delete identity", "Failed to delete Identity."),
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
				ActorMeta:   nil,
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

		for _, rl := range ratelimits {
			auditLogs = append(auditLogs, auditlog.AuditLog{
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Event:       auditlog.RatelimitDeleteEvent,
				Display:     fmt.Sprintf("Deleted ratelimit %s.", rl.ID),
				Bucket:      auditlogs.DEFAULT_BUCKET,
				ActorID:     auth.KeyID,
				ActorType:   auditlog.RootKeyActor,
				ActorMeta:   nil,
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

		err = svc.Auditlogs.Insert(ctx, tx, auditLogs)
		if err != nil {
			return fault.Wrap(err,
				fault.WithCode(codes.App.Internal.ServiceUnavailable.URN()),
				fault.WithDesc("database failed to insert audit logs", "Failed to insert audit logs"),
			)
		}

		err = tx.Commit()
		if err != nil {
			return fault.Wrap(err,
				fault.WithCode(codes.App.Internal.ServiceUnavailable.URN()),
				fault.WithDesc("database failed to commit transaction", "Failed to commit changes."),
			)
		}

		return s.JSON(http.StatusOK, Response{})
	})
}

func getIdentity(ctx context.Context, svc Services, req Request, workspaceID string) (db.Identity, error) {
	switch {
	case req.IdentityId != nil:
		return db.Query.FindIdentityByID(ctx, svc.DB.RO(), *req.IdentityId)
	case req.ExternalId != nil:
		return db.Query.FindIdentityByExternalID(ctx, svc.DB.RO(), db.FindIdentityByExternalIDParams{
			WorkspaceID: workspaceID,
			ExternalID:  *req.ExternalId,
		})
	}

	return db.Identity{}, fault.New("missing identity id or external id",
		fault.WithCode(codes.App.Validation.InvalidInput.URN()),
		fault.WithDesc("missing identity id or external id", "You must provide either an identity ID or external ID."),
	)
}
