package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Request = openapi.V2IdentitiesGetIdentityRequestBody

// Response defines the response body for this endpoint
type Response = openapi.V2IdentitiesGetIdentityResponseBody

// Handler implements zen.Route interface for the v2 identities get identity endpoint
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
	return "/v2/identities.getIdentity"
}

// Handle processes the HTTP request
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	auth, emit, err := h.Keys.GetRootKey(ctx, s)
	defer emit()
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	// Find the identity based on either IdentityId or ExternalId
	type IdentityResult struct {
		Identity   db.Identity
		Ratelimits []db.Ratelimit
	}

	result, err := db.TxWithResult(ctx, h.DB.RO(), func(ctx context.Context, tx db.DBTX) (IdentityResult, error) {
		var identity db.Identity

		identity, err = db.Query.FindIdentity(ctx, tx, db.FindIdentityParams{
			Identity:    req.Identity,
			WorkspaceID: auth.AuthorizedWorkspaceID,
			Deleted:     false,
		})
		if err != nil {
			if db.IsNotFound(err) {
				return IdentityResult{}, fault.New("identity not found",
					fault.Code(codes.Data.Identity.NotFound.URN()),
					fault.Internal("identity not found"),
					fault.Public("This identity does not exist."),
				)
			}

			return IdentityResult{}, fault.Wrap(err,
				fault.Internal("unable to find identity"),
				fault.Public("We're unable to retrieve the identity."),
			)
		}

		// Get the ratelimits for this identity
		ratelimits, listErr := db.Query.ListIdentityRatelimitsByID(ctx, tx, sql.NullString{Valid: true, String: identity.ID})
		if listErr != nil && !db.IsNotFound(listErr) {
			return IdentityResult{}, fault.Wrap(listErr,
				fault.Internal("unable to fetch ratelimits"),
				fault.Public("We're unable to retrieve the identity's ratelimits."),
			)
		}

		return IdentityResult{Identity: identity, Ratelimits: ratelimits}, nil
	})
	if err != nil {
		return err
	}

	identity := result.Identity
	ratelimits := result.Ratelimits

	// Check permissions using either wildcard or the specific identity ID
	err = auth.VerifyRootKey(ctx, keys.WithPermissions(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Identity,
			ResourceID:   "*",
			Action:       rbac.ReadIdentity,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Identity,
			ResourceID:   identity.ID,
			Action:       rbac.ReadIdentity,
		}),
	)))
	if err != nil {
		return err
	}

	// Parse metadata
	var metaMap map[string]any
	if len(identity.Meta) > 0 {
		err = json.Unmarshal(identity.Meta, &metaMap)
		if err != nil {
			return fault.Wrap(err,
				fault.Internal("unable to unmarshal metadata"), fault.Public("We're unable to parse the identity's metadata."),
			)
		}
	} else {
		metaMap = make(map[string]any)
	}

	// Format ratelimits for the response
	responseRatelimits := make([]openapi.RatelimitResponse, 0, len(ratelimits))
	for _, r := range ratelimits {
		responseRatelimits = append(responseRatelimits, openapi.RatelimitResponse{
			Name:      r.Name,
			Limit:     int64(r.Limit),
			Duration:  r.Duration,
			Id:        r.ID,
			AutoApply: r.AutoApply,
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
			Ratelimits: &responseRatelimits,
		},
	})
}
