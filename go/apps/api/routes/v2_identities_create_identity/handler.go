package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/internal/services/auditlogs"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/internal/services/permissions"
	"github.com/unkeyed/unkey/go/pkg/auditlog"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/uid"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Request = openapi.V2IdentitiesCreateIdentityRequestBody
type Response = openapi.V2IdentitiesCreateIdentityResponseBody

// Handler implements zen.Route interface for the v2 identities create identity endpoint
type Handler struct {
	// Services as public fields
	Logger      logging.Logger
	DB          db.Database
	Keys        keys.KeyService
	Permissions permissions.PermissionService
	Auditlogs   auditlogs.AuditLogService
}

const (
	// Planetscale only allows for 67MB of json data
	// 1MB should be enough for most use cases
	MAX_META_LENGTH_MB = 1
)

// Method returns the HTTP method this route responds to
func (h *Handler) Method() string {
	return "POST"
}

// Path returns the URL path pattern this route matches
func (h *Handler) Path() string {
	return "/v2/identities.createIdentity"
}

// Handle processes the HTTP request
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	auth, err := h.Keys.VerifyRootKey(ctx, s)
	if err != nil {
		return err
	}

	// nolint:exhaustruct
	req := Request{}
	err = s.BindBody(&req)
	if err != nil {
		return fault.Wrap(err,
			fault.Internal("invalid request body"), fault.Public("The request body is invalid."),
		)
	}

	err = h.Permissions.Check(
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
		return err
	}

	var meta []byte
	if req.Meta != nil {
		rawMeta, metaErr := json.Marshal(req.Meta)
		if metaErr != nil {
			return fault.Wrap(metaErr,
				fault.Code(codes.App.Validation.InvalidInput.URN()),
				fault.Internal("unable to marshal metadata"), fault.Public("We're unable to marshal the meta object."),
			)
		}

		sizeInMB := float64(len(rawMeta)) / 1024 / 1024
		if sizeInMB > MAX_META_LENGTH_MB {
			return fault.New("metadata is too large",
				fault.Code(codes.App.Validation.InvalidInput.URN()),
				fault.Internal("metadata is too large"), fault.Public(fmt.Sprintf("Metadata is too large, it must be less than %dMB, got: %.2f", MAX_META_LENGTH_MB, sizeInMB)),
			)
		}

		meta = rawMeta
	}

	// Validate rate limits
	if req.Ratelimits != nil {
		for _, ratelimit := range *req.Ratelimits {
			// Validate rate limit name is provided
			if ratelimit.Name == "" {
				return fault.New("invalid rate limit",
					fault.Code(codes.App.Validation.InvalidInput.URN()),
					fault.Internal("missing rate limit name"), fault.Public("Rate limit name is required."),
				)
			}

			// Validate rate limit value is positive
			if ratelimit.Limit <= 0 {
				return fault.New("invalid rate limit",
					fault.Code(codes.App.Validation.InvalidInput.URN()),
					fault.Internal("invalid rate limit value"), fault.Public("Rate limit value must be greater than zero."),
				)
			}

			// Validate duration is at least 1000ms (1 second)
			if ratelimit.Duration < 1000 {
				return fault.New("invalid rate limit",
					fault.Code(codes.App.Validation.InvalidInput.URN()),
					fault.Internal("invalid rate limit duration"), fault.Public("Rate limit duration must be at least 1000ms (1 second)."),
				)
			}
		}
	}

	identityID, err := db.TxWithResult(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) (string, error) {
		identityID := uid.New(uid.IdentityPrefix)
		args := db.InsertIdentityParams{
			ID:          identityID,
			ExternalID:  req.ExternalId,
			WorkspaceID: auth.AuthorizedWorkspaceID,
			Environment: "default",
			CreatedAt:   time.Now().UnixMilli(),
			Meta:        meta,
		}
		h.Logger.Warn("inserting identity",
			"args", args,
		)
		err = db.Query.InsertIdentity(ctx, tx, args)

		if err != nil {
			if db.IsDuplicateKeyError(err) {
				return "", fault.Wrap(err,
					fault.Code(codes.Data.Identity.Duplicate.URN()),
					fault.Internal("identity already exists"), fault.Public(fmt.Sprintf("Identity with externalId \"%s\" already exists in this workspace.", req.ExternalId)),
				)
			}

			return "", fault.Wrap(err,
				fault.Internal("unable to create identity"), fault.Public("We're unable to create the identity and its ratelimits."),
			)
		}

		auditLogs := []auditlog.AuditLog{
			{
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Event:       auditlog.IdentityCreateEvent,
				Display:     fmt.Sprintf("Created identity %s.", identityID),
				ActorID:     auth.KeyID,
				ActorName:   "root key",
				ActorMeta:   map[string]any{},
				ActorType:   auditlog.RootKeyActor,
				RemoteIP:    s.Location(),
				UserAgent:   s.UserAgent(),
				Resources: []auditlog.AuditLogResource{
					{
						ID:          identityID,
						Type:        auditlog.IdentityResourceType,
						Meta:        nil,
						Name:        req.ExternalId,
						DisplayName: req.ExternalId,
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
					Limit:       int32(ratelimit.Limit), // nolint:gosec
					Duration:    ratelimit.Duration,
					CreatedAt:   time.Now().UnixMilli(),
				})
				if err != nil {
					return "", fault.Wrap(err,
						fault.Internal("unable to create ratelimit"), fault.Public("We're unable to create a ratelimit for the identity."),
					)
				}

				auditLogs = append(auditLogs, auditlog.AuditLog{
					WorkspaceID: auth.AuthorizedWorkspaceID,
					Event:       auditlog.RatelimitCreateEvent,
					Display:     fmt.Sprintf("Created ratelimit %s.", ratelimitID),
					ActorID:     auth.KeyID,
					ActorType:   auditlog.RootKeyActor,
					ActorName:   "root key",
					ActorMeta:   map[string]any{},
					RemoteIP:    s.Location(),
					UserAgent:   s.UserAgent(),
					Resources: []auditlog.AuditLogResource{
						{
							Type:        auditlog.IdentityResourceType,
							ID:          identityID,
							Name:        req.ExternalId,
							Meta:        nil,
							DisplayName: req.ExternalId,
						},
						{
							Type:        auditlog.RatelimitResourceType,
							ID:          ratelimitID,
							DisplayName: ratelimit.Name,
							Name:        ratelimit.Name,
							Meta:        nil,
						},
					},
				})
			}
		}

		err = h.Auditlogs.Insert(ctx, tx, auditLogs)
		if err != nil {
			return "", fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database failed to insert audit logs"), fault.Public("Failed to insert audit logs"),
			)
		}

		return identityID, nil
	})
	if err != nil {
		return err
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.IdentitiesCreateIdentityResponseData{
			IdentityId: identityID,
		},
	})
}
