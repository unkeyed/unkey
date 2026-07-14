package handler

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type Request = openapi.V2IdentitiesGetIdentityRequestBody

// Response defines the response body for this endpoint
type Response = openapi.V2IdentitiesGetIdentityResponseBody

// Handler implements zen.Route interface for the v2 identities get identity endpoint
type Handler struct {
	DB db.Database
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
	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	collectionAuthErr := principal.Authorize(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Identity,
			ResourceID:   "*",
			Action:       rbac.ReadIdentity,
		}),
		rbac.U(
			urn.New().Workspace(principal.WorkspaceID).Identity("*"),
			permissions.ReadIdentity{},
		),
	))

	identity, err := db.Query.FindIdentity(ctx, h.DB.RO(), db.FindIdentityParams{
		WorkspaceID: principal.WorkspaceID,
		Identity:    req.Identity,
		Deleted:     false,
	})
	if err != nil {
		if db.IsNotFound(err) {
			if collectionAuthErr != nil {
				return collectionAuthErr
			}
			return fault.New("identity not found",
				fault.Code(codes.Data.Identity.NotFound.URN()),
				fault.Internal("identity not found"), fault.Public("This identity does not exist."),
			)
		}

		return fault.Wrap(err,
			fault.Internal("unable to find identity"),
			fault.Public("We're unable to retrieve the identity."),
		)
	}

	if collectionAuthErr != nil {
		err = principal.Authorize(rbac.Or(
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
			rbac.U(
				urn.New().Workspace(principal.WorkspaceID).Identity(identity.ID),
				permissions.ReadIdentity{},
			),
		))
		if err != nil {
			// Return the wildcard failure so the response never exposes the resolved internal ID.
			return collectionAuthErr
		}
	}

	// Parse ratelimits JSON
	ratelimits, err := db.UnmarshalNullableJSONTo[[]db.RatelimitInfo](identity.Ratelimits)
	if err != nil {
		logger.Error("failed to unmarshal ratelimits",
			"identityId", identity.ID,
			"error", err,
		)
	}

	metaMap, err := db.UnmarshalNullableJSONTo[map[string]any](identity.Meta)
	if err != nil {
		logger.Error("failed to unmarshal identity meta",
			"identityId", identity.ID,
			"error", err,
		)
	}

	// Format ratelimits for the response
	responseRatelimits := make([]openapi.RatelimitResponse, 0, len(ratelimits))
	for _, r := range ratelimits {
		responseRatelimits = append(responseRatelimits, openapi.RatelimitResponse{
			Name:      r.Name,
			Limit:     int64(r.Limit),
			Duration:  int64(r.Duration),
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
			Meta:       metaMap,
			Ratelimits: responseRatelimits,
		},
	})
}
