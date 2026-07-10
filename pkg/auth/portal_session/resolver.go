package portalsession

import (
	"context"
	"errors"
	"net/http"

	"github.com/unkeyed/unkey/internal/services/portal"
	"github.com/unkeyed/unkey/pkg/auth/portalrbac"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/zen"
)

// CookieName is the browser cookie that stores a portal session token.
const CookieName = "portal_session"

// Resolver authenticates customer portal browser sessions into auth principals.
type Resolver struct {
	portal portal.Service
}

// NewResolver creates a portal session resolver backed by the portal service.
func NewResolver(portal portal.Service) *Resolver {
	return &Resolver{portal: portal}
}

// Resolve claims cookie-only portal requests and yields to explicit bearer auth.
func (r *Resolver) Resolve(ctx context.Context, sess *zen.Session) (*principal.Principal, error) {
	if sess == nil || sess.Request() == nil {
		return nil, nil
	}
	if sess.Request().Header.Get("Authorization") != "" {
		return nil, nil
	}

	cookie, err := sess.Request().Cookie(CookieName)
	if errors.Is(err, http.ErrNoCookie) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	session, err := r.portal.GetSession(ctx, cookie.Value)
	if err != nil {
		return nil, err
	}

	// Translate the session's simplified capability model (keyspace ids + verbs)
	// into the RBAC permission strings the shared handlers check. This is the one
	// place the portal capability vocabulary is mapped onto RBAC — see the
	// portalrbac package.
	capabilities, err := portalrbac.ParseAll(session.Permissions)
	if err != nil {
		return nil, err
	}
	permissions := portalrbac.Grant{
		WorkspaceID:  session.WorkspaceID,
		KeyspaceIDs:  session.KeyspaceIDs,
		Capabilities: capabilities,
	}.Expand()

	return &principal.Principal{
		Version: principal.Version,
		Subject: principal.Subject{
			ID:   session.ExternalID,
			Name: session.ExternalID,
			Type: principal.SubjectTypeUser,
		},
		Type: principal.TypePortalSession,
		Source: principal.PortalSessionSource{
			SessionID:      cookie.Value,
			PortalConfigID: session.PortalConfigID,
			ExternalID:     session.ExternalID,
			KeyspaceIDs:    session.KeyspaceIDs,
			Permissions:    permissions,
		},
		WorkspaceID: session.WorkspaceID,
		Permissions: permissions,
	}, nil
}
