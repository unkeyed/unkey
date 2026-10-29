package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"slices"

	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/grant"
	"github.com/unkeyed/unkey/svc/api/internal/oauth"
)

// Request is the subset of RFC 7591 client metadata the server reads.
// Unknown members are ignored, as RFC 7591 section 2 allows.
type Request struct {
	ClientName              string   `json:"client_name"`
	RedirectURIs            []string `json:"redirect_uris"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
	Scope                   string   `json:"scope"`
}

// Response is the RFC 7591 section 3.2.1 body. There is no client_secret
// and no registration_access_token: clients are public and never managed.
type Response struct {
	ClientID                string   `json:"client_id"`
	ClientIDIssuedAt        int64    `json:"client_id_issued_at"`
	ClientName              string   `json:"client_name"`
	RedirectURIs            []string `json:"redirect_uris"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	Scope                   string   `json:"scope"`
}

type Handler struct {
	Grants *grant.Service
}

func (h *Handler) Method() string { return http.MethodPost }

func (h *Handler) Path() string { return oauth.RegisterPath }

// Handle registers a public client (RFC 7591).
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	if !oauth.IsJSON(s.Request()) {
		return invalidMetadata(s, "The request body must be application/json.")
	}
	body, err := io.ReadAll(s.Request().Body)
	if err != nil {
		return invalidMetadata(s, "The request body could not be read.")
	}
	var req Request
	if err := json.Unmarshal(body, &req); err != nil {
		return invalidMetadata(s, "The request body must be a JSON object of client metadata.")
	}
	if req.TokenEndpointAuthMethod != "" && req.TokenEndpointAuthMethod != oauth.AuthMethodNone {
		return invalidMetadata(s, "Only public clients are supported: token_endpoint_auth_method must be none.")
	}
	supportedGrants := []string{oauth.GrantAuthorizationCode, oauth.GrantRefreshToken}
	if len(req.GrantTypes) > 0 && (!slices.Contains(req.GrantTypes, oauth.GrantAuthorizationCode) ||
		slices.ContainsFunc(req.GrantTypes, func(g string) bool { return !slices.Contains(supportedGrants, g) })) {
		return invalidMetadata(s, "grant_types must include authorization_code and may include refresh_token.")
	}
	if slices.ContainsFunc(req.ResponseTypes, func(r string) bool { return r != "code" }) {
		return invalidMetadata(s, "response_types may only include code.")
	}
	if oerr := oauth.ParseScope(req.Scope); oerr != nil {
		return invalidMetadata(s, oerr.Description)
	}

	client, err := h.Grants.RegisterClient(ctx, grant.RegisterClientRequest{
		Name:         req.ClientName,
		RedirectURIs: req.RedirectURIs,
		RemoteIP:     s.Location(),
		UserAgent:    s.UserAgent(),
	})
	switch {
	case errors.Is(err, grant.ErrInvalidRedirectURI):
		return oauth.WriteError(s, http.StatusBadRequest, oauth.Error{Code: oauth.InvalidRedirectURI, Description: fault.UserFacingMessage(err)})
	case errors.Is(err, grant.ErrInvalidClientMetadata):
		return invalidMetadata(s, fault.UserFacingMessage(err))
	case err != nil:
		return oauth.WriteFailure(s, err)
	}

	s.AddHeader("Cache-Control", "no-store")
	return s.JSON(http.StatusCreated, Response{
		ClientID:                string(client.ID),
		ClientIDIssuedAt:        client.IssuedAt,
		ClientName:              client.Name,
		RedirectURIs:            client.RedirectURIs,
		GrantTypes:              supportedGrants,
		ResponseTypes:           []string{"code"},
		TokenEndpointAuthMethod: oauth.AuthMethodNone,
		Scope:                   oauth.ScopeAPI,
	})
}

func invalidMetadata(s *zen.Session, description string) error {
	return oauth.WriteError(s, http.StatusBadRequest, oauth.Error{Code: oauth.InvalidClientMetadata, Description: description})
}
