package handler

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/internal/services/permissions"
	"github.com/unkeyed/unkey/go/pkg/auditlog"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/uid"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Request = openapi.V2RatelimitDeleteOverrideRequestBody
type Response = openapi.V2RatelimitDeleteOverrideResponseBody

type Services struct {
	Logger      logging.Logger
	DB          db.Database
	Keys        keys.KeyService
	Permissions permissions.PermissionService
}

func New(svc Services) zen.Route {
	return zen.NewRoute("POST", "/v2/ratelimit.deleteOverride", func(ctx context.Context, s *zen.Session) error {
		auth, err := svc.Keys.VerifyRootKey(ctx, s)
		if err != nil {
			return err
		}

		// nolint:exhaustruct
		req := Request{}
		err = s.BindBody(&req)
		if err != nil {
			return fault.Wrap(err,
				fault.WithTag(fault.INTERNAL_SERVER_ERROR),
				fault.WithDesc("invalid request body", "The request body is invalid."),
			)
		}

		namespace, err := getNamespace(ctx, svc, auth.AuthorizedWorkspaceID, req)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fault.Wrap(err,
					fault.WithTag(fault.NOT_FOUND),
					fault.WithDesc("namespace not found", "This namespace does not exist."),
				)
			}
			return err
		}

		if namespace.WorkspaceID != auth.AuthorizedWorkspaceID {
			return fault.New("namespace not found",
				fault.WithTag(fault.NOT_FOUND),
				fault.WithDesc("wrong workspace, masking as 404", "This namespace does not exist."),
			)
		}

		permissions, err := svc.Permissions.Check(
			ctx,
			auth.KeyID,
			rbac.Or(
				rbac.T(rbac.Tuple{
					ResourceType: rbac.Ratelimit,
					ResourceID:   namespace.ID,
					Action:       rbac.DeleteOverride,
				}),
				rbac.T(rbac.Tuple{
					ResourceType: rbac.Ratelimit,
					ResourceID:   "*",
					Action:       rbac.DeleteOverride,
				}),
			),
		)

		if err != nil {
			return fault.Wrap(err,
				fault.WithTag(fault.INTERNAL_SERVER_ERROR),
				fault.WithDesc("unable to check permissions", "We're unable to check the permissions of your key."),
			)
		}

		if !permissions.Valid {
			return fault.New("insufficient permissions",
				fault.WithTag(fault.INSUFFICIENT_PERMISSIONS),
				fault.WithDesc(permissions.Message, permissions.Message),
			)
		}

		tx, err := svc.DB.RW().Begin(ctx)
		if err != nil {
			return fault.Wrap(err,
				fault.WithTag(fault.DATABASE_ERROR),
				fault.WithDesc("database failed to create transaction", "Unable to start database transaction."),
			)
		}
		defer func() {
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				svc.Logger.Error("rollback failed", "requestId", s.RequestID())
			}
		}()

		// Check if the override exists before deleting
		override, err := db.Query.FindRatelimitOverridesByIdentifier(ctx, tx, db.FindRatelimitOverridesByIdentifierParams{
			WorkspaceID: auth.AuthorizedWorkspaceID,
			NamespaceID: namespace.ID,
			Identifier:  req.Identifier,
		})

		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fault.Wrap(err,
					fault.WithTag(fault.NOT_FOUND),
					fault.WithDesc("override not found", "This override does not exist."),
				)
			}
			return err
		}

		// Perform soft delete by updating the DeletedAt field
		err = db.Query.SoftDeleteRatelimitOverride(ctx, tx, db.SoftDeleteRatelimitOverrideParams{
			ID:  override.ID,
			Now: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})

		if err != nil {
			return fault.Wrap(err,
				fault.WithTag(fault.DATABASE_ERROR),
				fault.WithDesc("database failed to soft delete ratelimit override", "The database is unavailable."),
			)
		}

		auditLogID := uid.New(uid.AuditLogPrefix)
		err = db.Query.InsertAuditLog(ctx, tx, db.InsertAuditLogParams{
			ID:          auditLogID,
			WorkspaceID: auth.AuthorizedWorkspaceID,
			BucketID:    "",
			Event:       string(auditlog.RatelimitDeleteOverrideEvent),
			Time:        time.Now().UnixMilli(),
			Display:     fmt.Sprintf("Deleted override %s.", override.ID),
			RemoteIp:    sql.NullString{String: "", Valid: false},
			UserAgent:   sql.NullString{String: "", Valid: false},
			ActorType:   string(auditlog.RootKeyActor),
			ActorID:     auth.KeyID,
			ActorName:   sql.NullString{String: "", Valid: false},
			ActorMeta:   nil,
			CreatedAt:   time.Now().UnixMilli(),
		})
		if err != nil {
			svc.Logger.Error(err.Error())
			return fault.Wrap(err,
				fault.WithTag(fault.DATABASE_ERROR),
				fault.WithDesc("database failed to insert audit log", "Failed to insert audit log."),
			)
		}
		err = db.Query.InsertAuditLogTarget(ctx, tx, db.InsertAuditLogTargetParams{
			ID:          override.ID,
			WorkspaceID: auth.AuthorizedWorkspaceID,
			BucketID:    "",
			AuditLogID:  auditLogID,
			DisplayName: override.Identifier,
			Type:        "ratelimit_override",
			Name:        sql.NullString{String: "", Valid: false},
			Meta:        nil,
			CreatedAt:   time.Now().UnixMilli(),
		})
		if err != nil {
			return fault.Wrap(err,
				fault.WithTag(fault.DATABASE_ERROR),
				fault.WithDesc("database failed to insert audit log target namespace", "Failed to insert audit log target."),
			)
		}
		err = db.Query.InsertAuditLogTarget(ctx, tx, db.InsertAuditLogTargetParams{
			ID:          namespace.ID,
			WorkspaceID: auth.AuthorizedWorkspaceID,
			BucketID:    "",
			AuditLogID:  auditLogID,
			DisplayName: namespace.Name,
			Type:        "ratelimit_namespacee",
			Name:        sql.NullString{String: override.Identifier, Valid: true},
			Meta:        nil,
			CreatedAt:   time.Now().UnixMilli(),
		})
		if err != nil {
			return fault.Wrap(err,
				fault.WithTag(fault.DATABASE_ERROR),
				fault.WithDesc("database failed to insert audit log target override", "Failed to insert audit log target."),
			)
		}

		err = tx.Commit()
		if err != nil {
			return fault.Wrap(err,
				fault.WithTag(fault.DATABASE_ERROR),
				fault.WithDesc("database failed to commit transaction", "Failed to commit changes."),
			)
		}

		return s.JSON(http.StatusOK, Response{})
	})
}

func getNamespace(ctx context.Context, svc Services, workspaceID string, req Request) (db.RatelimitNamespace, error) {
	switch {
	case req.NamespaceId != nil:
		{
			return db.Query.FindRatelimitNamespaceByID(ctx, svc.DB.RO(), *req.NamespaceId)
		}
	case req.NamespaceName != nil:
		{
			return db.Query.FindRatelimitNamespaceByName(ctx, svc.DB.RO(), db.FindRatelimitNamespaceByNameParams{
				WorkspaceID: workspaceID,
				Name:        *req.NamespaceName,
			})
		}
	}

	return db.RatelimitNamespace{}, fault.New("missing namespace id or name",
		fault.WithTag(fault.BAD_REQUEST),
		fault.WithDesc("missing namespace id or name", "You must provide either a namespace ID or name."),
	)
}
