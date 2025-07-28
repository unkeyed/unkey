package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Request = openapi.V2IdentitiesListIdentitiesRequestBody
type Response = openapi.V2IdentitiesListIdentitiesResponseBody

// Handler implements zen.Route interface for the v2 identities list identities endpoint
type Handler struct {
	// Services as public fields
	Logger logging.Logger
	DB     db.Database
	Keys   keys.KeyService
}

// Method returns the HTTP method this route responds to
func (h *Handler) Method() string {
	return "POST"
}

// Path returns the URL path pattern this route matches
func (h *Handler) Path() string {
	return "/v2/identities.listIdentities"
}

// Handle processes the HTTP request
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	auth, emit, err := h.Keys.GetRootKey(ctx, s)
	defer emit()
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

	// Query one extra record to check if there are more results
	identities, err := db.Query.ListIdentities(ctx, h.DB.RO(), db.ListIdentitiesParams{
		WorkspaceID: auth.AuthorizedWorkspaceID,
		Deleted:     false,
		IDCursor:    cursor,
		Limit:       int32(limit + 1), // nolint:gosec
	})

	if err != nil {
		return fault.Wrap(err,
			fault.Internal("unable to list identities"), fault.Public("We're unable to list the identities."),
		)
	}

	// Check if we have more results than the requested limit
	hasMore := len(identities) > limit
	var newCursor *string
	if hasMore {
		newCursor = ptr.P(identities[len(identities)-1].ID)
		// Trim the results to the requested limit
		identities = identities[:limit]
	}

	// Check permissions for all identities before processing
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

		err = auth.VerifyRootKey(ctx, keys.WithPermissions(permissionCheck))
		if err != nil {
			return err
		}
	}

	// Process the results and get ratelimits for each identity
	data := make([]openapi.Identity, 0, len(identities))
	for _, identity := range identities {
		// Fetch ratelimits for this identity
		ratelimits, err := db.Query.ListIdentityRatelimits(ctx, h.DB.RO(), sql.NullString{Valid: true, String: identity.ID})
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fault.Wrap(err,
				fault.Internal("unable to fetch ratelimits"), fault.Public("We're unable to retrieve ratelimits for the identities."),
			)
		}

		// Format ratelimits
		formattedRatelimits := make([]openapi.RatelimitResponse, 0, len(ratelimits))
		for _, r := range ratelimits {
			formattedRatelimits = append(formattedRatelimits, openapi.RatelimitResponse{
				Id:        r.ID,
				Name:      r.Name,
				Limit:     int64(r.Limit),
				Duration:  r.Duration,
				AutoApply: r.AutoApply,
			})
		}

		// Create a new identity with its ratelimits
		newIdentity := openapi.Identity{
			ExternalId: identity.ExternalID,
			Ratelimits: formattedRatelimits,
			Meta:       nil,
		}

		// Add metadata if available
		if len(identity.Meta) > 0 {
			// Initialize the Meta field with an empty map
			metaMap := make(map[string]interface{})
			newIdentity.Meta = &metaMap

			// Unmarshal the identity metadata into the map
			err = json.Unmarshal(identity.Meta, &metaMap)
			if err != nil {
				return fault.Wrap(err,
					fault.Internal("unable to unmarshal identity metadata"), fault.Public("We're unable to parse the metadata for the identity."),
				)
			}
		}

		// Append the identity to the results
		data = append(data, newIdentity)
	}

	response := Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: data,
		Pagination: openapi.Pagination{
			HasMore: hasMore,
			Cursor:  newCursor,
		},
	}

	return s.JSON(http.StatusOK, response)
}
