package handler

import (
	"context"

	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/portalscope"
	"github.com/unkeyed/unkey/svc/api/openapi"
	listkeys "github.com/unkeyed/unkey/svc/api/routes/v2_apis_list_keys"
)

// Request is the portal.listKeys public contract. Unlike apis.listKeys it has no
// externalId: the listing is always scoped to the session's own end user, so
// there is nothing for the caller to pass.
type Request = openapi.V2PortalListKeysRequestBody

// Handler serves the portal-scoped variant of apis.listKeys. It authenticates
// only portal sessions and forces the listing to the session's external
// identity.
//
// The core is held in an unexported field, not embedded: embedding would
// promote the core's Method/Path/Handle onto this type, so a typo or a missing
// override here would silently fall through to the unscoped apis.listKeys
// handler and leak every workspace key. With an explicit field the compiler
// forces us to define every zen.Route method.
type Handler struct {
	listKeys *listkeys.Handler
}

// New builds a portal.listKeys handler over the shared apis.listKeys core.
func New(listKeys *listkeys.Handler) *Handler {
	return &Handler{listKeys: listKeys}
}

// Method returns the HTTP method this route responds to.
func (h *Handler) Method() string { return "POST" }

// Path returns the URL path pattern this route matches.
func (h *Handler) Path() string { return "/v2/portal.listKeys" }

// Handle lists keys scoped to the portal session's external identity.
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	externalID, err := portalscope.ExternalID(s)
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	// The end user identity comes from the session, never the request; the portal
	// contract has no externalId. decrypt and cache revalidation are not exposed
	// to portal sessions.
	return h.listKeys.ListKeys(ctx, s, listkeys.ListKeysParams{
		ApiID:               req.ApiId,
		Limit:               req.Limit,
		Cursor:              req.Cursor,
		ExternalID:          &externalID,
		Decrypt:             nil,
		RevalidateKeysCache: nil,
	})
}
