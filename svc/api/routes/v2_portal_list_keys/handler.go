package handler

import (
	"context"

	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/portalscope"
	listkeys "github.com/unkeyed/unkey/svc/api/routes/v2_apis_list_keys"
)

// Handler serves the portal-scoped variant of apis.listKeys. It authenticates
// only portal sessions and forces the listing to the session's external
// identity.
type Handler struct {
	*listkeys.Handler
}

// Path returns the URL path pattern this route matches.
func (h *Handler) Path() string { return "/v2/portal.listKeys" }

// Handle lists keys scoped to the portal session's external identity.
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	externalID, err := portalscope.ExternalID(s)
	if err != nil {
		return err
	}

	req, err := zen.BindBody[listkeys.Request](s)
	if err != nil {
		return err
	}

	// Force the listing to the session identity, ignoring any externalId the
	// client sent. From here the shared core treats it like any other request.
	req.ExternalId = &externalID

	return h.Handler.ListKeys(ctx, s, req)
}
