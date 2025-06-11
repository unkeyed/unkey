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
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Request = openapi.V2IdentitiesGetIdentityRequestBody

// Response defines the response body for this endpoint
type Response = openapi.V2IdentitiesGetIdentityResponseBody

// Handler implements zen.Route interface for the v2 identities get identity endpoint
type Handler struct {
	// Services as public fields
	Logger      logging.Logger
	DB          db.Database
	Keys        keys.KeyService
	Permissions permissions.PermissionService
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
	auth, err := h.Keys.VerifyRootKey(ctx, s)
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	// Find the identity based on either IdentityId or ExternalId
	var identity db.Identity
	var ratelimits []db.Ratelimit

	tx, err := h.DB.RO().Begin(ctx)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database failed to create transaction"), fault.Public("Unable to start database transaction."),
		)
	}
	defer func() {
		rollbackErr := tx.Rollback()
		if rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
			h.Logger.Error("rollback failed", "requestId", s.RequestID(), "error", rollbackErr)
		}
	}()

	// First try to get the identity
	if req.IdentityId != nil {
		// Find by IdentityId
		identity, err = db.Query.FindIdentityByID(ctx, tx, db.FindIdentityByIDParams{
			ID:      *req.IdentityId,
			Deleted: false,
		})
	} else if req.ExternalId != nil {
		// Find by ExternalId
		identity, err = db.Query.FindIdentityByExternalID(ctx, tx, db.FindIdentityByExternalIDParams{
			ExternalID:  *req.ExternalId,
			WorkspaceID: auth.AuthorizedWorkspaceID,
			Deleted:     false,
		})
	} else {
		return fault.New("invalid request",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Internal("either identityId or externalId must be provided"), fault.Public("Either identityId or externalId must be provided."),
		)
	}

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fault.New("identity not found",
				fault.Code(codes.Data.Identity.NotFound.URN()),
				fault.Internal("identity not found"), fault.Public("This identity does not exist."),
			)
		}
		return fault.Wrap(err,
			fault.Internal("unable to find identity"), fault.Public("We're unable to retrieve the identity."),
		)
	}

	// Check permissions using either wildcard or the specific identity ID
	permissionCheck := rbac.Or(
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
	)

	err = h.Permissions.Check(ctx, auth.KeyID, permissionCheck)
	if err != nil {
		return err
	}

	// Next, get the ratelimits for this identity
	ratelimits, err = db.Query.ListIdentityRatelimitsByID(ctx, tx, sql.NullString{Valid: true, String: identity.ID})
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fault.Wrap(err,
			fault.Internal("unable to fetch ratelimits"), fault.Public("We're unable to retrieve the identity's ratelimits."),
		)
	}

	// Parse metadata
	var metaMap map[string]interface{}
	if identity.Meta != nil && len(identity.Meta) > 0 {
		err = json.Unmarshal(identity.Meta, &metaMap)
		if err != nil {
			return fault.Wrap(err,
				fault.Internal("unable to unmarshal metadata"), fault.Public("We're unable to parse the identity's metadata."),
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
		Data: openapi.IdentitiesGetIdentityResponseData{
			Id:         identity.ID,
			ExternalId: identity.ExternalID,
			Meta:       &metaMap,
			Ratelimits: &responseRatelimits,
		},
	})
}
