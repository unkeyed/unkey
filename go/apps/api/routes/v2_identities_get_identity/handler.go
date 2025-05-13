package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/internal/services/permissions"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Request = openapi.

type Response struct {
	Meta openapi.Meta     `json:"meta"`
	Data openapi.Identity `json:"data"`
}

type Services struct {
	Logger      logging.Logger
	DB          db.Database
	Keys        keys.KeyService
	Permissions permissions.PermissionService
}

func New(svc Services) zen.Route {
	return zen.NewRoute("POST", "/v2/identities.getIdentity", func(ctx context.Context, s *zen.Session) error {
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

		// Validate that at least one of identityID or externalID is provided
		if req.identityID == nil && req.externalID == nil {
			return fault.New("missing required field",
				fault.WithCode(codes.App.Validation.InvalidInput.URN()),
				fault.WithDesc("missing required field", "Provide either identityId or externalId"),
			)
		}

		// If identityId is provided, we use it to check permissions
		permissionCheck := rbac.Or(
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Identity,
				ResourceID:   "*",
				Action:       rbac.ReadIdentity,
			}),
		)

		// Add specific identity permission if identityID is provided
		if req.identityID != nil {
			permissionCheck = rbac.Or(
				permissionCheck,
				rbac.T(rbac.Tuple{
					ResourceType: rbac.Identity,
					ResourceID:   *req.identityID,
					Action:       rbac.ReadIdentity,
				}),
			)
		}

		permissions, err := svc.Permissions.Check(ctx, auth.KeyID, permissionCheck)
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

		// Find the identity based on either identityId or externalId
		var identity db.Identity
		var ratelimits []db.Ratelimit

		tx, err := svc.DB.RO().Begin(ctx)
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

		// First try to get the identity
		if req.identityID != nil {
			// Find by identityID
			identity, err = db.Query.FindIdentityByID(ctx, tx, db.FindIdentityByIDParams{
				ID:      *req.identityID,
				Deleted: false,
			})
		} else if req.externalID != nil {
			// Find by externalID
			identity, err = db.Query.FindIdentityByExternalID(ctx, tx, db.FindIdentityByExternalIDParams{
				ExternalID:  *req.externalID,
				WorkspaceID: auth.AuthorizedWorkspaceID,
			})
		} else {
			return fault.New("invalid request",
				fault.WithCode(codes.App.Validation.InvalidInput.URN()),
				fault.WithDesc("either identityID or externalID must be provided", "Either identityID or externalID must be provided."),
			)
		}

		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fault.New("identity not found",
					fault.WithCode(codes.Data.Identity.NotFound.URN()),
					fault.WithDesc("identity not found", fmt.Sprintf("Identity not found in this workspace")),
				)
			}
			return fault.Wrap(err,
				fault.WithDesc("unable to find identity", "We're unable to retrieve the identity."),
			)
		}

		// Next, get the ratelimits for this identity
		ratelimits, err = db.Query.FindRatelimitsByIdentityID(ctx, tx, sql.NullString{Valid: true, String: identity.ID})
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fault.Wrap(err,
				fault.WithDesc("unable to fetch ratelimits", "We're unable to retrieve the identity's ratelimits."),
			)
		}

		// Parse metadata
		var metaMap map[string]interface{}
		if identity.Meta != nil && len(identity.Meta) > 0 {
			err = json.Unmarshal(identity.Meta, &metaMap)
			if err != nil {
				return fault.Wrap(err,
					fault.WithDesc("unable to unmarshal metadata", "We're unable to parse the identity's metadata."),
				)
			}
		} else {
			metaMap = make(map[string]interface{})
		}

		// Format ratelimits for the response
		responseRatelimits := make([]openapi.Ratelimit, 0, len(ratelimits))
		for _, r := range ratelimits {
			responseRatelimits = append(responseRatelimits, openapi.Ratelimit{
				Name:     r.Name,
				Limit:    int64(r.Limit),
				Duration: r.Duration,
			})
		}

		return s.JSON(http.StatusOK, Response{
			Meta: openapi.Meta{
				RequestId: s.RequestID(),
			},
			Data: openapi.Identity{
				Id:         identity.ID,
				ExternalId: identity.ExternalID,
				Meta:       &metaMap,
			},
		})
	})
}
