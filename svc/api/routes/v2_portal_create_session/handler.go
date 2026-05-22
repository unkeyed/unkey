package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/validation"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2PortalCreateSessionRequestBody
	Response = openapi.V2PortalCreateSessionResponseBody
)

// Handler implements zen.Route for the portal session creation endpoint.
type Handler struct {
	DB            db.Database
	Keys          keys.KeyService
	Auditlogs     auditlogs.AuditLogService
	PortalBaseURL string
}

func (h *Handler) Method() string { return "POST" }
func (h *Handler) Path() string   { return "/v2/portal.createSession" }

// validatePermissionFormat checks that each permission string is a valid
// RBAC tuple: exactly 3 dot-separated non-empty segments.
func validatePermissionFormat(permissions []string) error {
	for _, p := range permissions {
		parts := strings.Split(p, ".")
		if len(parts) != 3 {
			return fault.New("invalid permission format",
				fault.Code(codes.App.Validation.InvalidInput.URN()),
				fault.Internal(fmt.Sprintf("permission %q does not have 3 dot-separated segments", p)),
				fault.Public(fmt.Sprintf("Permission %q is invalid. Expected format: {resourceType}.{resourceId}.{action}", p)),
			)
		}
		for _, segment := range parts {
			if segment == "" {
				return fault.New("invalid permission format",
					fault.Code(codes.App.Validation.InvalidInput.URN()),
					fault.Internal(fmt.Sprintf("permission %q contains an empty segment", p)),
					fault.Public(fmt.Sprintf("Permission %q is invalid. Segments must not be empty.", p)),
				)
			}
		}
	}
	return nil
}

func delegatedPermissionChecks(permissions []string) ([]rbac.PermissionQuery, error) {
	checks := make([]rbac.PermissionQuery, 0, len(permissions))
	for _, permission := range permissions {
		tuple, err := rbac.TupleFromString(permission)
		if err != nil {
			return nil, fault.Wrap(err,
				fault.Code(codes.App.Validation.InvalidInput.URN()),
				fault.Internal(fmt.Sprintf("permission %q failed tuple parsing", permission)),
				fault.Public(fmt.Sprintf("Permission %q is invalid. Expected format: {resourceType}.{resourceId}.{action}", permission)),
			)
		}

		checks = append(checks, rbac.Or(
			rbac.S(permission),
			rbac.T(rbac.Tuple{
				ResourceType: tuple.ResourceType,
				ResourceID:   "*",
				Action:       tuple.Action,
			}),
		))
	}

	return checks, nil
}

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

	if err := validatePermissionFormat(req.Permissions); err != nil {
		return err
	}
	delegatedChecks, err := delegatedPermissionChecks(req.Permissions)
	if err != nil {
		return err
	}

	workspaceID := auth.AuthorizedWorkspaceID

	if !validation.ValidateSlug(req.Slug) {
		return fault.New("invalid slug",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Internal(fmt.Sprintf("slug %q failed validation", req.Slug)),
			fault.Public(validation.ErrMsgInvalidSlug),
		)
	}

	portalConfig, err := db.Query.FindPortalConfigByWorkspaceAndSlug(ctx, h.DB.RO(), db.FindPortalConfigByWorkspaceAndSlugParams{
		WorkspaceID: workspaceID,
		Slug:        req.Slug,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return fault.New("portal config not found",
				fault.Code(codes.Data.PortalConfig.NotFound.URN()),
				fault.Internal("no portal config found for the given slug"),
				fault.Public("Portal configuration not found."),
			)
		}
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error looking up portal config"),
			fault.Public("Failed to look up portal configuration."),
		)
	}

	if err := auth.VerifyRootKey(ctx, keys.WithPermissions(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Portal,
			ResourceID:   portalConfig.ID,
			Action:       rbac.CreatePortalSession,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Portal,
			ResourceID:   "*",
			Action:       rbac.CreatePortalSession,
		}),
	))); err != nil {
		return err
	}
	if err := auth.VerifyRootKey(ctx, keys.WithPermissions(rbac.And(delegatedChecks...))); err != nil {
		return err
	}

	if !portalConfig.Enabled {
		return fault.New("portal is disabled",
			fault.Code(codes.Auth.Authorization.Forbidden.URN()),
			fault.Internal("portal config is disabled"),
			fault.Public("Portal is disabled."),
		)
	}

	// Determine the portal URL: prefer a verified custom domain for the app,
	// fall back to the configured base URL (e.g. https://portal.unkey.com).
	portalBaseURL := h.PortalBaseURL
	if portalConfig.AppID.Valid {
		customDomain, cdErr := db.Query.FindVerifiedCustomDomainByAppID(ctx, h.DB.RO(), portalConfig.AppID.String)
		if cdErr != nil && !db.IsNotFound(cdErr) {
			return fault.Wrap(cdErr,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error looking up custom domain for portal app"),
				fault.Public("Failed to look up portal configuration."),
			)
		}
		if cdErr == nil {
			portalBaseURL = fmt.Sprintf("https://%s", customDomain.Domain)
		}
	}

	now := time.Now()
	sessionTokenID := string(uid.PortalSessionTokenPrefix) + "_" + uid.Secure()
	expiresAt := now.Add(15 * time.Minute).UnixMilli()

	permissionsJSON, err := json.Marshal(req.Permissions)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Internal("failed to marshal permissions"),
			fault.Public("An internal error occurred."),
		)
	}

	preview := false
	if req.Preview != nil {
		preview = *req.Preview
	}

	err = db.Tx(ctx, h.DB.RW(), func(txCtx context.Context, tx db.DBTX) error {
		if txErr := db.Query.InsertPortalSessionToken(txCtx, tx, db.InsertPortalSessionTokenParams{
			ID:             sessionTokenID,
			WorkspaceID:    workspaceID,
			PortalConfigID: portalConfig.ID,
			ExternalID:     req.ExternalId,
			Permissions:    permissionsJSON,
			Preview:        preview,
			ExpiresAt:      expiresAt,
			CreatedAt:      now.UnixMilli(),
		}); txErr != nil {
			return fault.Wrap(txErr,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("failed to insert session token"),
				fault.Public("Failed to create session."),
			)
		}

		if txErr := h.Auditlogs.Insert(txCtx, tx, []auditlog.AuditLog{
			{
				Event:         auditlog.PortalSessionCreateEvent,
				WorkspaceID:   workspaceID,
				ActorType:     auditlog.RootKeyActor,
				ActorID:       auth.Key.ID,
				ActorName:     "root key",
				ActorMeta:     map[string]any{},
				Display:       fmt.Sprintf("Created portal session for %s", req.ExternalId),
				RemoteIP:      s.Location(),
				UserAgent:     s.UserAgent(),
				CorrelationID: "",
				Resources: []auditlog.AuditLogResource{
					{
						ID:          sessionTokenID,
						DisplayName: req.ExternalId,
						Name:        req.ExternalId,
						Meta:        map[string]any{"portalConfigId": portalConfig.ID, "slug": req.Slug},
						Type:        auditlog.PortalSessionResourceType,
					},
				},
			},
		}); txErr != nil {
			return fault.Wrap(txErr,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("failed to insert audit log"),
				fault.Public("Failed to create session."),
			)
		}

		return nil
	})
	if err != nil {
		return err
	}

	portalURL := fmt.Sprintf("%s/?session=%s", portalBaseURL, sessionTokenID)

	s.ResponseWriter().Header().Set("Cache-Control", "no-store")
	s.ResponseWriter().Header().Set("Pragma", "no-cache")

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: openapi.V2PortalCreateSessionResponseData{
			SessionId: sessionTokenID,
			Url:       portalURL,
		},
	})
}
