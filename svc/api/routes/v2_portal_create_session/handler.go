package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/policyconfig"
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

// resolveKeyspaceIDs derives the keyspaces a portal session is scoped to from
// the portal. A portal maps to exactly one of a keyspace or an
// app:
//
//   - keyspace-mapped (keyspace_id): the configured keyspace scopes key
//     capabilities directly.
//   - app-mapped (app_id): the app's current deployment carries a sentinel
//     config whose keyauth policies list the keyspaces it verifies keys against
//     at the gateway; those keySpaceIds become the session's keyspaces.
//
// The portal is bound to the caller's workspace, so the resolved keyspaces can
// never belong to another workspace.
func (h *Handler) resolveKeyspaceIDs(ctx context.Context, workspaceID string, portal db.Portal) ([]string, error) {
	hasKeyspace := portal.KeyspaceID.Valid
	hasApp := portal.AppID.Valid

	// A well-formed portal maps to exactly one of a keyspace or an app. Neither
	// or both is a misconfiguration the session can't be scoped from.
	if hasKeyspace == hasApp {
		return nil, fault.New("portal not mapped to exactly one target",
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Internal("portal must reference exactly one of keyspace_id or app_id"),
			fault.Public("Portal is invalid."),
		)
	}

	if hasKeyspace {
		return []string{portal.KeyspaceID.String}, nil
	}

	raw, err := db.Query.FindAppSentinelConfigByID(ctx, h.DB.RO(), db.FindAppSentinelConfigByIDParams{
		AppID:       portal.AppID.String,
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
			fault.Public("Failed to look up portal."),
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
// legacy empty-object configs yield no keyspaces.
func keyspacesFromSentinelConfig(raw []byte) ([]string, error) {
	cfg, err := policyconfig.Parse(raw)
	if err != nil {
		return nil, fault.Wrap(err,
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Internal("failed to unmarshal app sentinel config"),
			fault.Public("Portal is invalid."),
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

	portal, err := db.Query.FindPortalByWorkspaceAndIdOrSlug(ctx, h.DB.RO(), db.FindPortalByWorkspaceAndIdOrSlugParams{
		WorkspaceID: workspaceID,
		Portal:      req.Portal,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return fault.New("portal not found",
				fault.Code(codes.Data.Portal.NotFound.URN()),
				fault.Internal("no portal found for the given identifier"),
				fault.Public("Portal not found."),
			)
		}
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error looking up portal"),
			fault.Public("Failed to look up portal."),
		)
	}

	if !portal.Enabled {
		return fault.New("portal is disabled",
			fault.Code(codes.Auth.Authorization.Forbidden.URN()),
			fault.Internal("portal is disabled"),
			fault.Public("Portal is disabled."),
		)
	}

	// The keyspaces a session is scoped to come from the portal,
	// not the public request: the config is already bound to this workspace, so
	// key capabilities can never reach another workspace's keyspaces.
	keyspaceIDs, err := h.resolveKeyspaceIDs(ctx, workspaceID, portal)
	if err != nil {
		return err
	}

	// Determine the portal URL: prefer a verified custom domain for the app,
	// fall back to the configured base URL (e.g. https://portal.unkey.com).
	portalBaseURL := h.PortalBaseURL
	if portal.AppID.Valid {
		customDomain, cdErr := db.Query.FindVerifiedCustomDomainByAppID(ctx, h.DB.RO(), portal.AppID.String)
		if cdErr != nil && !db.IsNotFound(cdErr) {
			return fault.Wrap(cdErr,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error looking up custom domain for portal app"),
				fault.Public("Failed to look up portal."),
			)
		}
		if cdErr == nil {
			portalBaseURL = fmt.Sprintf("https://%s", customDomain.Domain)
		}
	}

	now := time.Now()
	sessionID := uid.New(uid.PortalSessionPrefix)
	exchangeCode := uid.Secure()
	exchangeCodeExpiresAt := now.Add(15 * time.Minute).UnixMilli()

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

	err = db.Tx(ctx, h.DB.RW(), func(txCtx context.Context, tx db.DBTX) error {
		if txErr := db.Query.InsertPortalSession(txCtx, tx, db.InsertPortalSessionParams{
			ID:                    sessionID,
			WorkspaceID:           workspaceID,
			PortalID:              portal.ID,
			ExternalID:            req.ExternalId,
			Permissions:           permissionsJSON,
			ExchangeCodeHash:      sql.NullString{String: hash.Sha256(exchangeCode), Valid: true},
			ExchangeCodeExpiresAt: exchangeCodeExpiresAt,
			AccessTokenHash:       sql.NullString{},
			AccessTokenCreatedAt:  sql.NullInt64{},
			AccessTokenExpiresAt:  sql.NullInt64{},
			CreatedAt:             now.UnixMilli(),
		}); txErr != nil {
			return fault.Wrap(txErr,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("failed to insert portal session"),
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
						ID:          sessionID,
						DisplayName: req.ExternalId,
						Name:        req.ExternalId,
						Meta:        map[string]any{"portalId": portal.ID, "slug": portal.Slug},
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

	portalURL := fmt.Sprintf("%s/?code=%s", portalBaseURL, exchangeCode)

	s.ResponseWriter().Header().Set("Cache-Control", "no-store")
	s.ResponseWriter().Header().Set("Pragma", "no-cache")

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: openapi.V2PortalCreateSessionResponseData{
			SessionId: sessionID,
			Url:       portalURL,
		},
	})
}
