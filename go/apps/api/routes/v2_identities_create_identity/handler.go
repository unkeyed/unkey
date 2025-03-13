package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/unkeyed/unkey/go/api"
	"github.com/unkeyed/unkey/go/internal/services/auditlogs"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/internal/services/permissions"
	"github.com/unkeyed/unkey/go/pkg/auditlog"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/logging"
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/uid"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Request = api.V2IdentitiesCreateIdentityRequestBody
type Response = api.V2IdentitiesCreateIdentityResponseBody

type Services struct {
	Logger      logging.Logger
	DB          db.Database
	Keys        keys.KeyService
	Permissions permissions.PermissionService
	Auditlogs   auditlogs.AuditLogService
}

const (
	// Planetscale only allows for 67MB of json data
	// 5MB should be enough for most use cases
	MAX_META_LENGTH_MB = 5
)

func New(svc Services) zen.Route {
	return zen.NewRoute("POST", "/v2/identities.createIdentity", func(ctx context.Context, s *zen.Session) error {
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

		permissions, err := svc.Permissions.Check(
			ctx,
			auth.KeyID,
			rbac.Or(
				rbac.S("*"),
				rbac.T(rbac.Tuple{
					ResourceType: rbac.Identity,
					ResourceID:   "*",
					Action:       rbac.CreateIdentity,
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

		var meta []byte
		if req.Meta != nil {
			rawMeta, err := json.Marshal(req.Meta)
			if err != nil {
				return fault.Wrap(err,
					fault.WithTag(fault.BAD_REQUEST),
					fault.WithDesc("unable to marshal metadata", "We're unable to use your meta object."),
				)
			}

			sizeInMB := float64(len(rawMeta)) / 1024 / 1024
			if sizeInMB > MAX_META_LENGTH_MB {
				return fault.New("metadata is too large",
					fault.WithTag(fault.BAD_REQUEST),
					fault.WithDesc("metadata is too large", fmt.Sprintf("Metadata is too large, it must be less than %dMB, got: %.2f", MAX_META_LENGTH_MB, sizeInMB)),
				)
			}

			meta = rawMeta
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
			if rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
				svc.Logger.Error("rollback failed", "requestId", s.RequestID(), "error", rollbackErr)
			}
		}()

		identityID := uid.New(uid.IdentityPrefix)
		err = db.Query.InsertIdentity(ctx, tx, db.InsertIdentityParams{
			ID:          identityID,
			ExternalID:  req.ExternalId,
			WorkspaceID: auth.AuthorizedWorkspaceID,
			Environment: "default",
			CreatedAt:   time.Now().UnixMilli(),
			Meta:        meta,
		})
		if err != nil {
			if strings.Contains(err.Error(), "Duplicate entry") {
				return fault.Wrap(err,
					fault.WithTag(fault.CONFLICT),
					fault.WithDesc("identity already exists", fmt.Sprintf("Identity with externalId \"%s\" already exists in this workspace.", req.ExternalId)),
				)
			}

			return fault.Wrap(err,
				fault.WithTag(fault.INTERNAL_SERVER_ERROR),
				fault.WithDesc("unable to create identity", "We're unable to create the identity and it's ratelimits."),
			)
		}

		auditLogs := []auditlog.AuditLog{
			{
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Event:       auditlog.IdentityCreateEvent,
				Display:     fmt.Sprintf("Created identity %s.", identityID),
				Actor: auditlog.AuditLogActorData{
					ID:   auth.KeyID,
					Type: auditlog.RootKeyActor,
				},
				RemoteIP:  s.Location(),
				UserAgent: s.UserAgent(),
				Resources: []auditlog.AuditLogResource{
					{
						ID:   identityID,
						Type: auditlog.IdentityResourceType,
					},
				},
			},
		}

		if req.Ratelimits != nil {
			for _, ratelimit := range *req.Ratelimits {
				ratelimitID := uid.New(uid.RatelimitPrefix)
				err = db.Query.InsertIdentityRatelimit(ctx, tx, db.InsertIdentityRatelimitParams{
					ID:          ratelimitID,
					WorkspaceID: auth.AuthorizedWorkspaceID,
					IdentityID:  sql.NullString{String: identityID, Valid: true},
					Name:        ratelimit.Name,
					Limit:       int32(ratelimit.Limit),
					Duration:    int64(ratelimit.Duration),
					CreatedAt:   time.Now().UnixMilli(),
				})
				if err != nil {
					return fault.Wrap(err,
						fault.WithTag(fault.INTERNAL_SERVER_ERROR),
						fault.WithDesc("unable to create ratelimit", "We're unable to create a ratelimit for the identity."),
					)
				}

				auditLogs = append(auditLogs, auditlog.AuditLog{
					WorkspaceID: auth.AuthorizedWorkspaceID,
					Event:       auditlog.RatelimitCreateEvent,
					Display:     fmt.Sprintf("Created ratelimit %s.", ratelimitID),
					Actor: auditlog.AuditLogActorData{
						Type: auditlog.RootKeyActor,
						ID:   auth.KeyID,
					},
					RemoteIP:  s.Location(),
					UserAgent: s.UserAgent(),
					Resources: []auditlog.AuditLogResource{
						{
							Type: auditlog.IdentityResourceType,
							ID:   identityID,
						},
						{
							Type:        auditlog.RatelimitResourceType,
							ID:          ratelimitID,
							DisplayName: ratelimit.Name,
						},
					},
				})
			}
		}

		err = svc.Auditlogs.Insert(ctx, tx, auditLogs)
		if err != nil {
			return fault.Wrap(err,
				fault.WithTag(fault.DATABASE_ERROR),
				fault.WithDesc("database failed to insert audit logs", "Failed to insert audit logs"),
			)
		}

		err = tx.Commit()
		if err != nil {
			return fault.Wrap(err,
				fault.WithTag(fault.DATABASE_ERROR),
				fault.WithDesc("database failed to commit transaction", "Failed to commit changes."),
			)
		}

		return s.JSON(http.StatusOK, Response{IdentityId: identityID})
	})
}
