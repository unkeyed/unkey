package handler

import (
	"context"
	"encoding/json"
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
	"github.com/unkeyed/unkey/svc/api/openapi"
)

// storedGrant is the JSON shape persisted in the portal session's permissions
// column: the simplified capability model the resolver later expands into RBAC
// via portalrbac. It must stay in sync with the shape parsed in
// internal/services/portal.GetSession.
type storedGrant struct {
	KeyspaceIDs []string `json:"keyspaceIds"`
	Permissions []string `json:"permissions"`
}

type (
	Request  = openapi.V2PortalCreateSessionRequestBody
	Response = openapi.V2PortalCreateSessionResponseBody
)

// Handler implements zen.Route for the portal session creation endpoint.
type Handler struct {
	DB            db.Database
	Auditlogs     auditlogs.AuditLogService
	PortalBaseURL string
}

func (h *Handler) Method() string { return "POST" }
func (h *Handler) Path() string   { return "/v2/portal.createSession" }

// validateKeyspacesOwned confirms every keyspace id belongs to the workspace, so
// a portal session can never be scoped to keyspaces the caller does not own.
func (h *Handler) validateKeyspacesOwned(ctx context.Context, workspaceID string, keyspaceIDs []string) error {
	rows, err := db.Query.FindKeyAuthsByKeyAuthIds(ctx, h.DB.RO(), db.FindKeyAuthsByKeyAuthIdsParams{
		WorkspaceID: workspaceID,
		KeyAuthIds:  keyspaceIDs,
	})
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error looking up keyspaces"),
			fault.Public("Failed to validate keyspaces."),
		)
	}

	owned := make(map[string]struct{}, len(rows))
	for _, r := range rows {
		owned[r.KeyAuthID] = struct{}{}
	}

	for _, id := range keyspaceIDs {
		if _, ok := owned[id]; !ok {
			return fault.New("keyspace not found",
				fault.Code(codes.Data.KeySpace.NotFound.URN()),
				fault.Internal(fmt.Sprintf("keyspace %q not found in workspace", id)),
				fault.Public(fmt.Sprintf("Keyspace %q was not found.", id)),
			)
		}
	}

	return nil
}

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

	if !portalConfig.Enabled {
		return fault.New("portal is disabled",
			fault.Code(codes.Auth.Authorization.Forbidden.URN()),
			fault.Internal("portal config is disabled"),
			fault.Public("Portal is disabled."),
		)
	}

	// The capability vocabulary is enforced by the OpenAPI enum at the request
	// boundary; here we confirm the caller owns every keyspace it is scoping the
	// session to, so a session can never be minted against another workspace's
	// keyspaces.
	if err := h.validateKeyspacesOwned(ctx, workspaceID, req.KeyspaceIds); err != nil {
		return err
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

	verbs := make([]string, len(req.Permissions))
	for i, p := range req.Permissions {
		verbs[i] = string(p)
	}

	permissionsJSON, err := json.Marshal(storedGrant{
		KeyspaceIDs: req.KeyspaceIds,
		Permissions: verbs,
	})
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Internal("failed to marshal portal session grant"),
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
				ActorType:     auditlog.AuditLogActor(principal.Subject.Type),
				ActorID:       principal.Subject.ID,
				ActorName:     principal.Subject.Name,
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
