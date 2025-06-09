package handler

import (
	"context"
	"database/sql"
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

type Request = openapi.V2RatelimitSetOverrideRequestBody
type Response = openapi.V2RatelimitSetOverrideResponseBody

type Services struct {
	Logger      logging.Logger
	DB          db.Database
	Keys        keys.KeyService
	Permissions permissions.PermissionService
	Auditlogs   auditlogs.AuditLogService
}

func New(svc Services) zen.Route {
	return zen.NewRoute("POST", "/v2/ratelimit.setOverride", func(ctx context.Context, s *zen.Session) error {

		auth, err := svc.Keys.VerifyRootKey(ctx, s)
		if err != nil {
			return err
		}

		// nolint:exhaustruct
		req := Request{}
		err = s.BindBody(&req)
		if err != nil {
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.UnexpectedError.URN()),
				fault.Internal("invalid request body"), fault.Public("The request body is invalid."),
			)
		}

		namespace, err := getNamespace(ctx, svc, auth.AuthorizedWorkspaceID, req)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fault.Wrap(err,
					fault.Code(codes.Data.RatelimitNamespace.NotFound.URN()),
					fault.Internal("namespace not found"), fault.Public("This namespace does not exist."),
				)
			}
			return err
		}

		if namespace.WorkspaceID != auth.AuthorizedWorkspaceID {
			return fault.New("namespace not found",
				fault.Code(codes.Data.RatelimitNamespace.NotFound.URN()),
				fault.Internal("wrong workspace, masking as 404"), fault.Public("This namespace does not exist."),
			)
		}

		err = svc.Permissions.Check(
			ctx,
			auth.KeyID,
			rbac.Or(
				rbac.T(rbac.Tuple{
					ResourceType: rbac.Ratelimit,
					ResourceID:   namespace.ID,
					Action:       rbac.SetOverride,
				}),
				rbac.T(rbac.Tuple{
					ResourceType: rbac.Ratelimit,
					ResourceID:   "*",
					Action:       rbac.SetOverride,
				}),
			),
		)
		if err != nil {
			return fault.Wrap(err,
				fault.Internal("unable to check permissions"), fault.Public("We're unable to check the permissions of your key."),
			)
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

		overrideID := uid.New(uid.RatelimitOverridePrefix)
		err = db.Query.InsertRatelimitOverride(ctx, tx, db.InsertRatelimitOverrideParams{
			ID:          overrideID,
			WorkspaceID: auth.AuthorizedWorkspaceID,
			NamespaceID: namespace.ID,
			Identifier:  req.Identifier,
			Limit:       int32(req.Limit),    // nolint:gosec
			Duration:    int32(req.Duration), //nolint:gosec
			CreatedAt:   time.Now().UnixMilli(),
		})
		if err != nil {
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database failed"), fault.Public("The database is unavailable."),
			)
		}

		err = svc.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{
			{
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Event:       auditlog.RatelimitSetOverrideEvent,
				ActorID:     auth.KeyID,
				ActorType:   auditlog.RootKeyActor,
				ActorName:   "root key",
				ActorMeta:   nil,
				Bucket:      auditlogs.DEFAULT_BUCKET,

				RemoteIP:  s.Location(),
				UserAgent: s.UserAgent(),
				Display:   fmt.Sprintf("Set ratelimit override for %s and %s", namespace.ID, req.Identifier),
				Resources: []auditlog.AuditLogResource{
					{
						Type:        auditlog.RatelimitOverrideResourceType,
						ID:          overrideID,
						Name:        req.Identifier,
						DisplayName: req.Identifier,
						Meta:        nil,
					},
				},
			},
		})
		if err != nil {
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database failed to insert audit logs"), fault.Public("Failed to insert audit logs"),
			)
		}

		err = tx.Commit()
		if err != nil {
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database failed to commit transaction"), fault.Public("Failed to commit changes."),
			)
		}

		return s.JSON(http.StatusOK, Response{
			Meta: openapi.Meta{
				RequestId: s.RequestID(),
			},
			Data: openapi.RatelimitSetOverrideResponseData{
				OverrideId: overrideID,
			},
		})
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
		fault.Code(codes.App.Validation.InvalidInput.URN()),
		fault.Internal("missing namespace id or name"), fault.Public("You must provide either a namespace ID or name."),
	)

}
