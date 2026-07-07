package handler

import (
	"context"

	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/portalscope"
	createkey "github.com/unkeyed/unkey/svc/api/routes/v2_keys_create_key"
)

// Handler serves the portal-scoped variant of keys.createKey. It authenticates
// only portal sessions and forces the created key to belong to the session's
// external identity. It reuses the createKey core unchanged, supplying the
// external identity as a normal request field.
type Handler struct {
	*createkey.Handler
}

// Path returns the URL path pattern this route matches.
func (h *Handler) Path() string { return "/v2/portal.createKey" }

// Handle creates a key scoped to the portal session's external identity.
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	externalID, err := portalscope.ExternalID(s)
	if err != nil {
		return err
	}

	req, err := zen.BindBody[createkey.Request](s)
	if err != nil {
		return err
	}

	// Force ownership to the session identity, ignoring any externalId the
	// client sent. From here the shared core treats it like any other request.
	req.ExternalId = &externalID

	return h.Handler.CreateKey(ctx, s, req)
}
