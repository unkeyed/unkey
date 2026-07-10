package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	frontlinev1 "github.com/unkeyed/unkey/gen/proto/frontline/v1"
	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/validation"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
	"google.golang.org/protobuf/encoding/protojson"
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

// resolveKeyspaceIDs derives the keyspaces a portal session is scoped to from
// the portal configuration. A config maps to exactly one of a keyspace or an
// app:
//
//   - keyspace-mapped (key_auth_id): the configured keyspace scopes key
//     capabilities directly.
//   - app-mapped (app_id): the app's current deployment carries a sentinel
//     config whose keyauth policies list the keyspaces it verifies keys against
//     at the gateway; those keySpaceIds become the session's keyspaces.
//
// The config is bound to the caller's workspace, so the resolved keyspaces can
// never belong to another workspace.
func (h *Handler) resolveKeyspaceIDs(ctx context.Context, workspaceID string, portalConfig db.PortalConfiguration) ([]string, error) {
	hasKeyspace := portalConfig.KeyAuthID.Valid
	hasApp := portalConfig.AppID.Valid

	// A well-formed config maps to exactly one of a keyspace or an app. Neither
	// or both is a misconfiguration the session can't be scoped from.
	if hasKeyspace == hasApp {
		return nil, fault.New("portal config not mapped to exactly one target",
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Internal("portal config must reference exactly one of key_auth_id or app_id"),
			fault.Public("Portal configuration is invalid."),
		)
	}

	if hasKeyspace {
		return []string{portalConfig.KeyAuthID.String}, nil
	}

	raw, err := db.Query.FindAppSentinelConfigByID(ctx, h.DB.RO(), db.FindAppSentinelConfigByIDParams{
		AppID:       portalConfig.AppID.String,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return nil, fault.New("portal app has no current deployment",
				fault.Code(codes.Auth.Authorization.Forbidden.URN()),
				fault.Internal("app has no current deployment to resolve keyspaces from"),
				fault.Public("Portal is not available: the app has no active deployment."),
			)
		}
		return nil, fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error looking up app sentinel config"),
			fault.Public("Failed to look up portal configuration."),
		)
	}

	keyspaceIDs, err := keyspacesFromSentinelConfig(raw)
	if err != nil {
		return nil, err
	}
	if len(keyspaceIDs) == 0 {
		return nil, fault.New("portal app has no keyauth policies",
			fault.Code(codes.Auth.Authorization.Forbidden.URN()),
			fault.Internal("app sentinel config declares no keyauth keyspaces"),
			fault.Public("Portal is not available: the app has no key verification configured."),
		)
	}

	return keyspaceIDs, nil
}

// keyspacesFromSentinelConfig parses a deployment's sentinel_config and returns
// the deduplicated keyspaces declared across its keyauth policies. Empty or
// legacy empty-object configs yield no keyspaces (mirrors the frontline
// gateway's lenient parsing).
func keyspacesFromSentinelConfig(raw []byte) ([]string, error) {
	if len(raw) == 0 || string(raw) == "{}" {
		return nil, nil
	}

	cfg := &frontlinev1.Config{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(raw, cfg); err != nil {
		return nil, fault.Wrap(err,
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Internal("failed to unmarshal app sentinel config"),
			fault.Public("Portal configuration is invalid."),
		)
	}

	seen := make(map[string]struct{})
	var keyspaceIDs []string
	for _, p := range cfg.GetPolicies() {
		for _, ks := range p.GetKeyauth().GetKeySpaceIds() {
			if _, ok := seen[ks]; ok {
				continue
			}
			seen[ks] = struct{}{}
			keyspaceIDs = append(keyspaceIDs, ks)
		}
	}
	return keyspaceIDs, nil
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

	// The keyspaces a session is scoped to come from the portal configuration,
	// not the public request: the config is already bound to this workspace, so
	// key capabilities can never reach another workspace's keyspaces.
	keyspaceIDs, err := h.resolveKeyspaceIDs(ctx, workspaceID, portalConfig)
	if err != nil {
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
		KeyspaceIDs: keyspaceIDs,
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
