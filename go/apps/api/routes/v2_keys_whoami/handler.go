package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/oapi-codegen/nullable"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/internal/services/auditlogs"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/hash"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/vault"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Request = openapi.V2KeysWhoamiRequestBody
type Response = openapi.V2KeysWhoamiResponseBody

type Handler struct {
	// Services as public fields
	Logger    logging.Logger
	DB        db.Database
	Keys      keys.KeyService
	Auditlogs auditlogs.AuditLogService
	Vault     *vault.Service
}

// Method returns the HTTP method this route responds to
func (h *Handler) Method() string {
	return "POST"
}

// Path returns the URL path pattern this route matches
func (h *Handler) Path() string {
	return "/v2/keys.whoami"
}

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {

	// Authentication
	auth, err := h.Keys.GetRootKey(ctx, s)
	if err != nil {
		return err
	}

	// Request validation
	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	args := db.FindKeyByIdOrHashParams{
		ID:   sql.NullString{String: "", Valid: false},
		Hash: sql.NullString{String: hash.Sha256(req.Key), Valid: true},
	}

	key, err := db.Query.FindKeyByIdOrHash(ctx, h.DB.RO(), args)
	if err != nil {
		if db.IsNotFound(err) {
			return fault.Wrap(
				err,
				fault.Code(codes.Data.Key.NotFound.URN()),
				fault.Internal("key does not exist"),
				fault.Public("We could not find the requested key."),
			)
		}

		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve Key information."),
		)
	}

	// Validate key belongs to authorized workspace
	if key.WorkspaceID != auth.AuthorizedWorkspaceID {
		return fault.New("key not found",
			fault.Code(codes.Data.Key.NotFound.URN()),
			fault.Internal("key belongs to different workspace"),
			fault.Public("The specified key was not found."),
		)
	}

	// Permission check
	err = auth.Verify(ctx, keys.WithPermissions(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Api,
			ResourceID:   "*",
			Action:       rbac.ReadKey,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Api,
			ResourceID:   key.Api.ID,
			Action:       rbac.ReadKey,
		}),
	)))
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.Data.Key.NotFound.URN()),
			fault.Internal("user doesn't have permissions and we don't want to leak the existance of the key"),
			fault.Public("The specified key was not found."),
		)
	}

	k := openapi.KeyResponseData{
		CreatedAt:   key.CreatedAtM,
		Enabled:     key.Enabled,
		KeyId:       key.ID,
		Start:       key.Start,
		Plaintext:   nil,
		Name:        nil,
		Meta:        nil,
		Identity:    nil,
		Credits:     nil,
		Expires:     nil,
		Permissions: nil,
		Ratelimits:  nil,
		Roles:       nil,
		UpdatedAt:   nil,
	}

	if key.Name.Valid {
		k.Name = ptr.P(key.Name.String)
	}

	if key.UpdatedAtM.Valid {
		k.UpdatedAt = ptr.P(key.UpdatedAtM.Int64)
	}

	if key.Expires.Valid {
		k.Expires = ptr.P(key.Expires.Time.UnixMilli())
	}

	if key.RemainingRequests.Valid {
		k.Credits = &openapi.KeyCreditsData{
			Remaining: nullable.NewNullableWithValue(int64(key.RemainingRequests.Int32)),
			Refill:    nil,
		}

		if key.RefillAmount.Valid {
			var refillDay *int
			interval := openapi.Daily
			if key.RefillDay.Valid {
				interval = openapi.Monthly
				refillDay = ptr.P(int(key.RefillDay.Int16))
			}

			k.Credits.Refill = &openapi.KeyCreditsRefill{
				Amount:    int64(key.RefillAmount.Int32),
				Interval:  interval,
				RefillDay: refillDay,
			}
		}
	}

	if key.IdentityID.Valid {
		identity, idErr := db.Query.FindIdentityByID(ctx, h.DB.RO(), db.FindIdentityByIDParams{ID: key.IdentityID.String, Deleted: false})
		if idErr != nil {
			if db.IsNotFound(idErr) {
				return fault.New("identity not found for key",
					fault.Code(codes.Data.Identity.NotFound.URN()),
					fault.Internal("identity not found"),
					fault.Public("The requested identity does not exist or has been deleted."),
				)
			}

			return fault.Wrap(idErr,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error"),
				fault.Public("Failed to retrieve Identity information."),
			)
		}

		k.Identity = &openapi.Identity{
			ExternalId: identity.ExternalID,
			Meta:       nil,
			Ratelimits: nil,
		}

		if len(identity.Meta) > 0 {
			err = json.Unmarshal(identity.Meta, &k.Identity.Meta)
			if err != nil {
				return fault.Wrap(err, fault.Code(codes.App.Internal.UnexpectedError.URN()),
					fault.Internal("unable to unmarshal identity meta"),
					fault.Public("We encountered an error while trying to unmarshal the identity meta data."),
				)
			}
		}

		ratelimits, rlErr := db.Query.ListIdentityRatelimitsByID(ctx, h.DB.RO(), sql.NullString{Valid: true, String: identity.ID})
		if rlErr != nil && !db.IsNotFound(rlErr) {
			return fault.Wrap(rlErr, fault.Code(codes.App.Internal.UnexpectedError.URN()),
				fault.Internal("unable to retrieve identity ratelimits"),
				fault.Public("We encountered an error while trying to retrieve the identity ratelimits."),
			)
		}

		for _, ratelimit := range ratelimits {
			k.Identity.Ratelimits = append(k.Identity.Ratelimits, openapi.RatelimitResponse{
				Id:        ratelimit.ID,
				Duration:  ratelimit.Duration,
				Limit:     int64(ratelimit.Limit),
				Name:      ratelimit.Name,
				AutoApply: ratelimit.AutoApply,
			})
		}
	}

	ratelimits, err := db.Query.ListRatelimitsByKeyID(ctx, h.DB.RO(), sql.NullString{String: key.ID, Valid: true})
	if err != nil && !db.IsNotFound(err) {
		return fault.Wrap(err, fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Internal("unable to retrieve key ratelimits"),
			fault.Public("We encountered an error while trying to retrieve the key ratelimits."),
		)
	}

	ratelimitsResponse := make([]openapi.RatelimitResponse, len(ratelimits))
	for idx, ratelimit := range ratelimits {
		ratelimitsResponse[idx] = openapi.RatelimitResponse{
			Id:        ratelimit.ID,
			Duration:  ratelimit.Duration,
			Limit:     int64(ratelimit.Limit),
			Name:      ratelimit.Name,
			AutoApply: ratelimit.AutoApply,
		}
	}

	k.Ratelimits = ptr.P(ratelimitsResponse)

	if key.Meta.Valid {
		err = json.Unmarshal([]byte(key.Meta.String), &k.Meta)
		if err != nil {
			return fault.Wrap(err, fault.Code(codes.App.Internal.UnexpectedError.URN()),
				fault.Internal("unable to unmarshal key meta"),
				fault.Public("We encountered an error while trying to unmarshal the key meta data."),
			)
		}
	}

	permissionSlugs, err := db.Query.ListPermissionsByKeyID(ctx, h.DB.RO(), db.ListPermissionsByKeyIDParams{
		KeyID: k.KeyId,
	})
	if err != nil {
		return fault.Wrap(err, fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Internal("unable to find permissions for key"), fault.Public("Could not load permissions for key."))
	}
	k.Permissions = ptr.P(permissionSlugs)

	// Get roles for the key
	roles, err := db.Query.ListRolesByKeyID(ctx, h.DB.RO(), k.KeyId)
	if err != nil {
		return fault.Wrap(err, fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Internal("unable to find roles for key"), fault.Public("Could not load roles for key."))
	}

	roleNames := make([]string, len(roles))
	for i, role := range roles {
		roleNames[i] = role.Name
	}

	k.Roles = ptr.P(roleNames)

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: k,
	})
}
