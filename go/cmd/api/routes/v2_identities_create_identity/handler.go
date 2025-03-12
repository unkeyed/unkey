package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/unkeyed/unkey/go/api"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/internal/services/permissions"
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
}

const (
	MAX_META_LENGTH = 64_000
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

		if req.Meta != nil {
			rawMeta, err := json.Marshal(req.Meta)
			if err != nil {
				return fault.Wrap(err,
					fault.WithTag(fault.BAD_REQUEST),
					fault.WithDesc("unable to marshal metadata", "We're unable to use your meta object."),
				)
			}

			if len(rawMeta) > MAX_META_LENGTH {
				return fault.New("metadata is too large",
					fault.WithTag(fault.BAD_REQUEST),
					fault.WithDesc("metadata is too large", fmt.Sprintf("metadata is too large, it must be less than 64k characters when json encoded, got: %d", len(rawMeta))),
				)
			}
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

		identityID := uid.New(uid.IdentityPrefix)
		// TOOD:
		// upsert identity check for duplicate entry
		// insert ratelimits
		// create auditlog
		// response with new identity

		return s.JSON(http.StatusOK, Response{
			IdentityId: identityID,
		})
	})
}
