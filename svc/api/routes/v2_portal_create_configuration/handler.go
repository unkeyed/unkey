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
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/validation"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/portalconfig"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2PortalCreateConfigurationRequestBody
	Response = openapi.V2PortalCreateConfigurationResponseBody
)

// Handler implements zen.Route for creating a portal configuration. This is an
// operator action authenticated by a root key; the configuration is created in
// the root key's workspace.
type Handler struct {
	DB        db.Database
	Auditlogs auditlogs.AuditLogService
}

func (h *Handler) Method() string { return "POST" }
func (h *Handler) Path() string   { return "/v2/portal.createConfiguration" }

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

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	returnURL := sql.NullString{}
	if req.ReturnUrl != nil && *req.ReturnUrl != "" {
		returnURL = sql.NullString{String: *req.ReturnUrl, Valid: true}
	}

	now := time.Now().UnixMilli()
	configID := string(uid.PortalConfigPrefix) + "_" + uid.Secure()

	// Branding columns echoed back on the response; also written when provided.
	var logoURL, primaryColor sql.NullString
	if req.Branding != nil {
		logoURL = sql.NullString{String: req.Branding.LogoUrl, Valid: req.Branding.LogoUrl != ""}
		primaryColor = sql.NullString{String: req.Branding.PrimaryColor, Valid: req.Branding.PrimaryColor != ""}
	}

	err = db.Tx(ctx, h.DB.RW(), func(txCtx context.Context, tx db.DBTX) error {
		txErr := db.Query.InsertPortalConfig(txCtx, tx, db.InsertPortalConfigParams{
			ID:          configID,
			WorkspaceID: workspaceID,
			Slug:        req.Slug,
			DisplayName: req.DisplayName,
			AppID:       appCol,
			KeyAuthID:   keyAuthCol,
			Enabled:     enabled,
			ReturnUrl:   returnURL,
			CreatedAt:   now,
			UpdatedAt:   sql.NullInt64{},
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
				fault.Internal("failed to insert portal config"),
				fault.Public("Failed to create portal configuration."),
			)
		}

		if req.Branding != nil {
			if txErr := db.Query.UpsertPortalBranding(txCtx, tx, db.UpsertPortalBrandingParams{
				PortalConfigID: configID,
				LogoUrl:        logoURL,
				PrimaryColor:   primaryColor,
				CreatedAt:      now,
				UpdatedAt:      sql.NullInt64{},
			}); txErr != nil {
				return fault.Wrap(txErr,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("failed to upsert portal branding"),
					fault.Public("Failed to create portal configuration."),
				)
			}
		}

		return h.Auditlogs.Insert(txCtx, tx, []auditlog.AuditLog{
			{
				Event:         auditlog.PortalConfigCreateEvent,
				WorkspaceID:   workspaceID,
				ActorType:     auditlog.AuditLogActor(principal.Subject.Type),
				ActorID:       principal.Subject.ID,
				ActorName:     principal.Subject.Name,
				ActorMeta:     map[string]any{},
				Display:       fmt.Sprintf("Created portal configuration %s", req.Slug),
				RemoteIP:      s.Location(),
				UserAgent:     s.UserAgent(),
				CorrelationID: "",
				Resources: []auditlog.AuditLogResource{
					{
						ID:          configID,
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

	// Pk is the DB autoincrement, unknown without a read-back and never surfaced
	// in the response; ToResponse ignores it.
	config := db.PortalConfiguration{
		Pk:          0,
		ID:          configID,
		WorkspaceID: workspaceID,
		Slug:        req.Slug,
		DisplayName: req.DisplayName,
		AppID:       appCol,
		KeyAuthID:   keyAuthCol,
		Enabled:     enabled,
		ReturnUrl:   returnURL,
		CreatedAt:   now,
		UpdatedAt:   sql.NullInt64{},
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: portalconfig.ToResponse(config, logoURL, primaryColor),
	})
}
