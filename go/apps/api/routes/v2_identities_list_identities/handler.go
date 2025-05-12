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

type Request struct {
	// Filter identities by environment. Default is "default".
	environment *string `json:"environment,omitempty"`

	// Maximum number of identities to return in a single request.
	// Min: 1, Max: 100, Default: 100
	limit *int `json:"limit,omitempty"`

	// Cursor for pagination. Use the cursor from the previous response to get the next page.
	cursor *string `json:"cursor,omitempty"`
}

type IdentityWithRatelimits struct {
	ID         string              `json:"id"`
	ExternalID string              `json:"externalId"`
	Ratelimits []openapi.Ratelimit `json:"ratelimits"`
}

type Response struct {
	Meta openapi.Meta `json:"meta"`
	Data struct {
		Identities []IdentityWithRatelimits `json:"identities"`
		Total      int                      `json:"total"`
		Cursor     *string                  `json:"cursor,omitempty"`
	} `json:"data"`
}

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
		req := Request{}
		err = s.BindBody(&req)
		if err != nil {
			return fault.Wrap(err,
				fault.WithDesc("invalid request body", "The request body is invalid."),
			)
		}

		// Set default values if not provided
		environment := "default"
		if req.environment != nil && *req.environment != "" {
			environment = *req.environment
		}

		limit := 100
		if req.limit != nil {
			limit = *req.limit
			if limit < 1 {
				limit = 1
			}
			if limit > 100 {
				limit = 100
			}
		}

		// Check permissions
		permissionCheck := rbac.Or(
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Wildcard,
				ResourceID:   "*",
				Action:       rbac.Wildcard,
			}),
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Identity,
				ResourceID:   "*",
				Action:       rbac.ReadIdentity,
			}),
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Identity,
				ResourceID:   environment,
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

		// Begin read-only transaction
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

		// Query to get total count of identities
		total, err := db.Query.CountIdentities(ctx, tx, db.CountIdentitiesParams{
			WorkspaceID: auth.AuthorizedWorkspaceID,
			Environment: environment,
		})
		if err != nil {
			return fault.Wrap(err,
				fault.WithDesc("unable to count identities", "We're unable to count the identities."),
			)
		}

		// Query to get identities with pagination
		identities, err := db.Query.ListIdentities(ctx, tx, db.ListIdentitiesParams{
			WorkspaceID: auth.AuthorizedWorkspaceID,
			Environment: environment,
			Limit:       int32(limit),
			Cursor:      sql.NullString{String: ptr.ValOrZero(req.cursor), Valid: req.cursor != nil},
		})
		if err != nil {
			return fault.Wrap(err,
				fault.WithDesc("unable to list identities", "We're unable to list the identities."),
			)
		}

		// Process the results and get ratelimits for each identity
		result := make([]IdentityWithRatelimits, 0, len(identities))
		for _, identity := range identities {
			// Fetch ratelimits for this identity
			ratelimits, err := db.Query.GetRatelimitsByIdentityID(ctx, tx, identity.ID)
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
					Limit:    int(r.Limit),
					Duration: r.Duration,
				})
			}

			// Add this identity with its ratelimits to results
			result = append(result, IdentityWithRatelimits{
				ID:         identity.ID,
				ExternalID: identity.ExternalID,
				Ratelimits: formattedRatelimits,
			})
		}

		// Build response
		var cursor *string
		if len(identities) > 0 {
			cursor = ptr.P(identities[len(identities)-1].ID)
		}

		response := Response{
			Meta: openapi.Meta{
				RequestId: s.RequestID(),
			},
		}
		response.Data.Identities = result
		response.Data.Total = int(total)
		response.Data.Cursor = cursor

		return s.JSON(http.StatusOK, response)
	})
}
