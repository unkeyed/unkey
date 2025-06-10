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

type Request = openapi.V2ApisCreateApiRequestBody
type Response = openapi.V2ApisCreateApiResponseBody

type Services struct {
	Logger      logging.Logger
	DB          db.Database
	Keys        keys.KeyService
	Permissions permissions.PermissionService
	Auditlogs   auditlogs.AuditLogService
}

func New(svc Services) zen.Route {
	return zen.NewRoute("POST", "/v2/apis.createApi", func(ctx context.Context, s *zen.Session) error {
		auth, err := svc.Keys.VerifyRootKey(ctx, s)
		if err != nil {
			return err
		}

		var req Request
		err = s.BindBody(&req)
		if err != nil {
			return fault.Wrap(err,
				fault.Internal("invalid request body"), fault.Public("The request body is invalid."),
			)
		}

		err = svc.Permissions.Check(
			ctx,
			auth.KeyID,
			rbac.Or(

				rbac.T(rbac.Tuple{
					ResourceType: rbac.Api,
					ResourceID:   "*",
					Action:       rbac.CreateAPI,
				}),
			),
		)
		if err != nil {
			return err
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

		keyAuthId := uid.New(uid.KeyAuthPrefix)
		err = db.Query.InsertKeyring(ctx, tx, db.InsertKeyringParams{
			ID:                 keyAuthId,
			WorkspaceID:        auth.AuthorizedWorkspaceID,
			CreatedAtM:         time.Now().UnixMilli(),
			DefaultPrefix:      sql.NullString{Valid: false, String: ""},
			DefaultBytes:       sql.NullInt32{Valid: false, Int32: 0},
			StoreEncryptedKeys: false,
		})
		if err != nil {
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("unable to create key auth"), fault.Public("We're unable to create key authentication for the API."),
			)
		}

		apiId := uid.New(uid.APIPrefix)
		err = db.Query.InsertApi(ctx, tx, db.InsertApiParams{
			ID:          apiId,
			Name:        req.Name,
			WorkspaceID: auth.AuthorizedWorkspaceID,
			AuthType:    db.NullApisAuthType{Valid: true, ApisAuthType: db.ApisAuthTypeKey},
			KeyAuthID:   sql.NullString{Valid: true, String: keyAuthId},
			CreatedAtM:  time.Now().UnixMilli(),
		})
		if err != nil {
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("unable to create api"), fault.Public("We're unable to create the API."),
			)
		}

		err = svc.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{
			{
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Event:       auditlog.APICreateEvent,
				Display:     fmt.Sprintf("Created API %s", apiId),
				ActorID:     auth.KeyID,
				ActorName:   "root key",
				Bucket:      auditlogs.DEFAULT_BUCKET,
				ActorType:   auditlog.RootKeyActor,
				RemoteIP:    s.Location(),
				UserAgent:   s.UserAgent(),
				Resources: []auditlog.AuditLogResource{
					{
						ID:          apiId,
						Type:        auditlog.APIResourceType,
						Meta:        nil,
						Name:        req.Name,
						DisplayName: req.Name,
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
			Data: openapi.ApisCreateApiResponseData{
				ApiId: apiId,
			},
		})
	})
}
