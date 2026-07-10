package handler

import (
	"context"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/portalscope"
	rerollkey "github.com/unkeyed/unkey/svc/api/routes/v2_keys_reroll_key"
)

// Handler serves the portal-scoped variant of keys.rerollKey. It authenticates
// only portal sessions and may only reroll keys owned by the session's external
// identity.
//
// It reuses the rerollKey core rather than wrapping it with a scope flag: the
// core does the (large) reroll, and this handler owns the identity guard by
// loading the key up front and refusing anything it does not own.
//
// The core is held in an unexported field, not embedded: embedding would
// promote the core's Method/Path/Handle onto this type, so a typo or a missing
// override here would silently fall through to the unscoped keys.rerollKey
// handler and expose an unauthenticated reroll. With an explicit field the
// compiler forces us to define every zen.Route method.
type Handler struct {
	reroll *rerollkey.Handler
}

// New builds a portal.rerollKey handler over the shared keys.rerollKey core.
func New(reroll *rerollkey.Handler) *Handler {
	return &Handler{reroll: reroll}
}

// Method returns the HTTP method this route responds to.
func (h *Handler) Method() string { return "POST" }

// Path returns the URL path pattern this route matches.
func (h *Handler) Path() string { return "/v2/portal.rerollKey" }

// Handle rerolls a key scoped to the portal session's external identity.
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}

	externalID, err := portalscope.ExternalID(s)
	if err != nil {
		return err
	}

	req, err := zen.BindBody[rerollkey.Request](s)
	if err != nil {
		return err
	}

	key, err := h.reroll.FindLiveKey(ctx, req.KeyId)
	if err != nil {
		return err
	}

	// Ownership guard: a portal caller may only reroll a key that belongs to its
	// own external identity within its own workspace. Fail closed with a 404 so
	// the caller cannot probe for keys it does not own.
	if key.WorkspaceID != principal.WorkspaceID ||
		!key.IdentityExternalID.Valid ||
		key.IdentityExternalID.String != externalID {
		return fault.New("key not found",
			fault.Code(codes.Data.Key.NotFound.URN()),
			fault.Internal("key does not belong to portal session identity"),
			fault.Public("The specified key was not found."),
		)
	}

	return h.reroll.RerollKey(ctx, s, req, key)
}
