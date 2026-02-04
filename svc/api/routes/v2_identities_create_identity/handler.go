package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/wide"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type Request = openapi.V2IdentitiesCreateIdentityRequestBody
type Response = openapi.V2IdentitiesCreateIdentityResponseBody

// Handler implements zen.Route interface for the v2 identities create identity endpoint
type Handler struct {
	// Services as public fields
	Logger    logging.Logger
	DB        db.Database
	Keys      keys.KeyService
	Auditlogs auditlogs.AuditLogService
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
	auth, emit, err := h.Keys.GetRootKey(ctx, s)
	defer emit()
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	err = auth.VerifyRootKey(ctx, keys.WithPermissions(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Identity,
			ResourceID:   "*",
			Action:       rbac.CreateIdentity,
		}),
	)))
	if err != nil {
		return err
	}

	meta := []byte("{}")
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

	identityID := uid.New(uid.IdentityPrefix)
	wide.Set(ctx, wide.FieldIdentityID, identityID)

	err = db.TxRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		args := db.InsertIdentityParams{
			ID:          identityID,
			ExternalID:  req.ExternalId,
			WorkspaceID: auth.AuthorizedWorkspaceID,
			Environment: "default",
			CreatedAt:   time.Now().UnixMilli(),
			Meta:        meta,
		}

		err = db.Query.InsertIdentity(ctx, tx, args)
		if err != nil {
			if db.IsDuplicateKeyError(err) {
				return fault.Wrap(err,
					fault.Code(codes.Data.Identity.Duplicate.URN()),
					fault.Internal("identity already exists"), fault.Public(fmt.Sprintf("Identity with externalId '%s' already exists in this workspace.", req.ExternalId)),
				)
			}

			return fault.Wrap(err,
				fault.Internal("unable to create identity"), fault.Public("We're unable to create the identity and its ratelimits."),
			)
		}

		auditLogs := []auditlog.AuditLog{
			{
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Event:       auditlog.IdentityCreateEvent,
				Display:     fmt.Sprintf("Created identity %s.", identityID),
				ActorID:     auth.Key.ID,
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
			rateLimitsToInsert := make([]db.InsertIdentityRatelimitParams, len(*req.Ratelimits))
			for i, ratelimit := range *req.Ratelimits {
				ratelimitID := uid.New(uid.RatelimitPrefix)
				rateLimitsToInsert[i] = db.InsertIdentityRatelimitParams{
					ID:          ratelimitID,
					WorkspaceID: auth.AuthorizedWorkspaceID,
					IdentityID:  sql.NullString{String: identityID, Valid: true},
					Name:        ratelimit.Name,
					Limit:       int32(ratelimit.Limit), // nolint:gosec
					Duration:    ratelimit.Duration,
					CreatedAt:   time.Now().UnixMilli(),
					AutoApply:   ratelimit.AutoApply,
				}

				auditLogs = append(auditLogs, auditlog.AuditLog{
					WorkspaceID: auth.AuthorizedWorkspaceID,
					Event:       auditlog.RatelimitCreateEvent,
					Display:     fmt.Sprintf("Created ratelimit %s.", ratelimitID),
					ActorID:     auth.Key.ID,
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

			err = db.BulkQuery.InsertIdentityRatelimits(
				ctx,
				tx,
				rateLimitsToInsert,
			)
			if err != nil {
				return fault.Wrap(err,
					fault.Internal("unable to create ratelimit"), fault.Public("We're unable to create a ratelimit for the identity."),
				)
			}
		}

		err = h.Auditlogs.Insert(ctx, tx, auditLogs)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.V2IdentitiesCreateIdentityResponseData{IdentityId: identityID},
	})
}
