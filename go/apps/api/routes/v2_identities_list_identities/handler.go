package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/internal/services/permissions"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Request = openapi.V2IdentitiesListIdentitiesRequestBody
type Response = openapi.V2IdentitiesListIdentitiesResponseBody

type Services struct {
	Logger      logging.Logger
	DB          db.Database
	Keys        keys.KeyService
	Permissions permissions.PermissionService
}

func New(svc Services) zen.Route {
	return zen.NewRoute("POST", "/v2/identities.listIdentities", func(ctx context.Context, s *zen.Session) error {
		auth, err := svc.Keys.VerifyRootKey(ctx, s)
		if err != nil {
			return err
		}

		// nolint:exhaustruct
		req, err := zen.BindBody[Request](s)
		if err != nil {
			return err
		}

		limit := ptr.SafeDeref(req.Limit, 100)
		cursor := ptr.SafeDeref(req.Cursor)

		identities, err := db.Query.ListIdentities(ctx, svc.DB.RO(), db.ListIdentitiesParams{
			WorkspaceID: auth.AuthorizedWorkspaceID,
			Deleted:     false,
			IDCursor:    cursor,
			Limit:       int32(limit + 1),
		})

		if err != nil {
			return fault.Wrap(err,
				fault.WithDesc("unable to list identities", "We're unable to list the identities."),
			)
		}
		for _, id := range identities {
			// Check permissions
			permissionCheck := rbac.Or(
				rbac.T(rbac.Tuple{
					ResourceType: rbac.Identity,
					ResourceID:   id.ID,
					Action:       rbac.ReadIdentity,
				}),
				rbac.T(rbac.Tuple{
					ResourceType: rbac.Identity,
					ResourceID:   "*",
					Action:       rbac.ReadIdentity,
				}),
			)

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

		}

		// Process the results and get ratelimits for each identity
		data := make([]openapi.Identity, 0, len(identities))
		for i, identity := range identities {
			// Fetch ratelimits for this identity
			ratelimits, err := db.Query.ListIdentityRatelimits(ctx, svc.DB.RO(), sql.NullString{Valid: true, String: identity.ID})
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return fault.Wrap(err,
					fault.WithDesc("unable to fetch ratelimits", "We're unable to retrieve ratelimits for the identities."),
				)
			}

			// Format ratelimits
			formattedRatelimits := make([]openapi.Ratelimit, 0, len(ratelimits))
			for _, r := range ratelimits {
				formattedRatelimits = append(formattedRatelimits, openapi.Ratelimit{
					Name:     r.Name,
					Limit:    int64(r.Limit),
					Duration: r.Duration,
				})
			}

			// Add this identity with its ratelimits to results
			data[i] = openapi.Identity{

				Id:         identity.ID,
				ExternalId: identity.ExternalID,
				Ratelimits: formattedRatelimits,
			}
			if identity.Meta != nil && len(identity.Meta) > 0 {
				err = json.Unmarshal(identity.Meta, data[i].Meta)
				if err != nil {
					return fault.Wrap(err,
						fault.WithDesc("unable to unmarshal identity metadata", "We're unable to parse the metadata for the identity."),
					)
				}
			}
		}

		response := Response{
			Meta: openapi.Meta{
				RequestId: s.RequestID(),
			},
			Data: data,
			Pagination: openapi.Pagination{
				HasMore: false,
				Cursor:  nil,
			},
		}

		if len(identities) > limit {
			response.Pagination.Cursor = ptr.P(identities[len(identities)-1].ID)
			response.Pagination.HasMore = true
		}

		return s.JSON(http.StatusOK, response)
	})
}
