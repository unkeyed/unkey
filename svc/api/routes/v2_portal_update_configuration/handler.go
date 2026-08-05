package handler

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/validation"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/portalconfig"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2PortalUpdateConfigurationRequestBody
	Response = openapi.V2PortalUpdateConfigurationResponseBody
)

// Handler implements zen.Route for updating a portal configuration. It replaces
// the configuration's mutable state (slug, enabled, return URL, keyspace/app
// mapping) and upserts branding when provided. It is an operator action
// authenticated by a root key and scoped to the root key's workspace, so a
// caller can never mutate another workspace's configuration.
type Handler struct {
	DB        db.Database
	Auditlogs auditlogs.AuditLogService
}

func (h *Handler) Method() string { return "POST" }
func (h *Handler) Path() string   { return "/v2/portal.updateConfiguration" }

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	workspaceID := principal.WorkspaceID

	if !validation.ValidateSlug(req.Slug) {
		return fault.New("invalid slug",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Internal(fmt.Sprintf("slug %q failed validation", req.Slug)),
			fault.Public(validation.ErrMsgInvalidSlug),
		)
	}

	appCol, keyAuthCol, err := portalconfig.Mapping(req.AppId, req.KeyspaceId)
	if err != nil {
		return err
	}

	// Verify the config exists and belongs to the caller's workspace before
	// mutating it. The scoped UPDATE below also filters by workspace, but this
	// gives a clean 404 (rather than a silent no-op) when the id is unknown or
	// owned by another workspace.
	existing, err := db.Query.FindPortalConfigByID(ctx, h.DB.RW(), db.FindPortalConfigByIDParams{
		ID:          req.ConfigId,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return fault.New("portal config not found",
				fault.Code(codes.Data.PortalConfig.NotFound.URN()),
				fault.Internal("no portal config found for the given id"),
				fault.Public("Portal configuration not found."),
			)
		}
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error looking up portal config"),
			fault.Public("Failed to look up portal configuration."),
		)
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	returnURL := sql.NullString{}
	if req.ReturnUrl != nil && *req.ReturnUrl != "" {
		returnURL = sql.NullString{String: *req.ReturnUrl, Valid: true}
	}

	now := time.Now().UnixMilli()

	err = db.Tx(ctx, h.DB.RW(), func(txCtx context.Context, tx db.DBTX) error {
		txErr := db.Query.UpdatePortalConfig(txCtx, tx, db.UpdatePortalConfigParams{
			Slug:        req.Slug,
			AppID:       appCol,
			KeyAuthID:   keyAuthCol,
			Enabled:     enabled,
			ReturnUrl:   returnURL,
			UpdatedAt:   sql.NullInt64{Int64: now, Valid: true},
			ID:          req.ConfigId,
			WorkspaceID: workspaceID,
		})
		if txErr != nil {
			if db.IsDuplicateKeyError(txErr) {
				return fault.Wrap(txErr,
					fault.Code(codes.Data.PortalConfig.Duplicate.URN()),
					fault.Internal("portal config slug/app/keyspace collision"),
					fault.Public("A portal configuration with this slug, keyspace, or app already exists."),
				)
			}
			return fault.Wrap(txErr,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("failed to update portal config"),
				fault.Public("Failed to update portal configuration."),
			)
		}

		if req.Branding != nil {
			if txErr := db.Query.UpsertPortalBranding(txCtx, tx, db.UpsertPortalBrandingParams{
				PortalConfigID: req.ConfigId,
				LogoUrl:        sql.NullString{String: req.Branding.LogoUrl, Valid: req.Branding.LogoUrl != ""},
				PrimaryColor:   sql.NullString{String: req.Branding.PrimaryColor, Valid: req.Branding.PrimaryColor != ""},
				CreatedAt:      now,
				UpdatedAt:      sql.NullInt64{Int64: now, Valid: true},
			}); txErr != nil {
				return fault.Wrap(txErr,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("failed to upsert portal branding"),
					fault.Public("Failed to update portal configuration."),
				)
			}
		}

		return h.Auditlogs.Insert(txCtx, tx, []auditlog.AuditLog{
			{
				Event:         auditlog.PortalConfigUpdateEvent,
				WorkspaceID:   workspaceID,
				ActorType:     auditlog.AuditLogActor(principal.Subject.Type),
				ActorID:       principal.Subject.ID,
				ActorName:     principal.Subject.Name,
				ActorMeta:     map[string]any{},
				Display:       fmt.Sprintf("Updated portal configuration %s", req.Slug),
				RemoteIP:      s.Location(),
				UserAgent:     s.UserAgent(),
				CorrelationID: "",
				Resources: []auditlog.AuditLogResource{
					{
						ID:          req.ConfigId,
						DisplayName: req.Slug,
						Name:        req.Slug,
						Meta:        map[string]any{"slug": req.Slug},
						Type:        auditlog.PortalConfigResourceType,
					},
				},
			},
		})
	})
	if err != nil {
		return err
	}

	// Reflect the stored branding in the response. Read from the primary so the
	// just-written row is visible; a missing row means the config has no branding.
	var logoURL, primaryColor sql.NullString
	branding, brandingErr := db.Query.FindPortalBrandingByConfigID(ctx, h.DB.RW(), req.ConfigId)
	if brandingErr != nil && !db.IsNotFound(brandingErr) {
		return fault.Wrap(brandingErr,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error looking up portal branding"),
			fault.Public("Failed to update portal configuration."),
		)
	}
	if brandingErr == nil {
		logoURL = branding.LogoUrl
		primaryColor = branding.PrimaryColor
	}

	config := db.PortalConfiguration{
		Pk:          existing.Pk,
		ID:          req.ConfigId,
		WorkspaceID: workspaceID,
		Slug:        req.Slug,
		AppID:       appCol,
		KeyAuthID:   keyAuthCol,
		Enabled:     enabled,
		ReturnUrl:   returnURL,
		CreatedAt:   existing.CreatedAt,
		UpdatedAt:   sql.NullInt64{Int64: now, Valid: true},
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: portalconfig.ToResponse(config, logoURL, primaryColor),
	})
}
