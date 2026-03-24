package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

// Request is the expected JSON body for POST /v2/portal.createSession.
type Request struct {
	ExternalID  string         `json:"externalId"`
	Permissions []string       `json:"permissions"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	Preview     bool           `json:"preview,omitempty"`
}

// Response is the JSON body returned on success.
type Response struct {
	Meta openapi.Meta              `json:"meta"`
	Data CreateSessionResponseData `json:"data"`
}

// CreateSessionResponseData holds the session creation result.
type CreateSessionResponseData struct {
	SessionID string `json:"sessionId"`
	URL       string `json:"url"`
	ExpiresAt int64  `json:"expiresAt"`
}

// Handler implements zen.Route for the portal session creation endpoint.
type Handler struct {
	DB   db.Database
	Keys keys.KeyService
}

func (h *Handler) Method() string { return "POST" }
func (h *Handler) Path() string   { return "/v2/portal.createSession" }

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

	if req.ExternalID == "" {
		return fault.New("externalId is required",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Internal("missing externalId"),
			fault.Public("externalId is required."),
		)
	}

	if len(req.Permissions) == 0 {
		return fault.New("permissions is required",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Internal("missing permissions"),
			fault.Public("permissions array is required."),
		)
	}

	workspaceID := auth.AuthorizedWorkspaceID

	portalConfig, err := db.Query.FindPortalConfigByWorkspaceID(ctx, h.DB.RO(), workspaceID)
	if err != nil {
		if db.IsNotFound(err) {
			return fault.New("portal not found",
				fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
				fault.Internal("no portal config for workspace"),
				fault.Public("No portal is configured for this workspace."),
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

	// Look up the frontline route to construct the portal URL.
	route, err := db.Query.FindFrontlineRouteByPortalConfigID(ctx, h.DB.RO(),
		sql.NullString{String: portalConfig.ID, Valid: true},
	)
	if err != nil {
		if db.IsNotFound(err) {
			return fault.New("portal route not found",
				fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
				fault.Internal("no frontline route for portal config"),
				fault.Public("Portal routing is not configured."),
			)
		}
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error looking up frontline route"),
			fault.Public("Failed to look up portal route."),
		)
	}

	now := time.Now()
	sessionTokenID := uid.New(uid.PortalSessionTokenPrefix)
	expiresAt := now.Add(15 * time.Minute).UnixMilli()

	permissionsJSON, err := json.Marshal(req.Permissions)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Internal("failed to marshal permissions"),
			fault.Public("An internal error occurred."),
		)
	}

	var metadataBytes []byte
	if req.Metadata != nil {
		metadataBytes, err = json.Marshal(req.Metadata)
		if err != nil {
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.UnexpectedError.URN()),
				fault.Internal("failed to marshal metadata"),
				fault.Public("An internal error occurred."),
			)
		}
	}

	err = db.Query.InsertPortalSessionToken(ctx, h.DB.RW(), db.InsertPortalSessionTokenParams{
		ID:             sessionTokenID,
		WorkspaceID:    workspaceID,
		PortalConfigID: portalConfig.ID,
		ExternalID:     req.ExternalID,
		Metadata:       metadataBytes,
		Permissions:    permissionsJSON,
		Preview:        req.Preview,
		ExpiresAt:      expiresAt,
		CreatedAt:      now.UnixMilli(),
	})
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("failed to insert session token"),
			fault.Public("Failed to create session."),
		)
	}

	pathPrefix := "/portal"
	if route.PathPrefix.Valid {
		pathPrefix = route.PathPrefix.String
	}
	portalURL := fmt.Sprintf("https://%s%s?session=%s", route.FullyQualifiedDomainName, pathPrefix, sessionTokenID)

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: CreateSessionResponseData{
			SessionID: sessionTokenID,
			URL:       portalURL,
			ExpiresAt: expiresAt,
		},
	})
}
