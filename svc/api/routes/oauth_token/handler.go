package handler

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/grant"
	"github.com/unkeyed/unkey/svc/api/internal/oauth"
)

type Handler struct {
	Grants *grant.Service
}

func (h *Handler) Method() string { return http.MethodPost }

func (h *Handler) Path() string { return oauth.TokenPath }

// Handle is RFC 6749 sections 4.1.3 and 6 for public clients. It runs on the
// OAuth stack, which has no OpenAPI validation, so form bodies reach it and
// RFC error bodies leave it, and no request metrics, so codes, verifiers, and
// refresh tokens in the form are never stored.
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	form, oerr := oauth.ParseForm(s.Request())
	if oerr != nil {
		return oauth.WriteError(s, http.StatusBadRequest, *oerr)
	}
	if form.Repeated() {
		return invalidRequest(s, "Parameters must not repeat.")
	}
	if !oauth.ResourceAllowed(form.Get("resource"), h.Grants.Issuer) {
		return oauth.WriteError(s, http.StatusBadRequest, oauth.Error{Code: oauth.InvalidTarget, Description: "resource must be " + h.Grants.Issuer + "."})
	}

	var tokens grant.Tokens
	var err error
	switch form.Get("grant_type") {
	case "":
		return invalidRequest(s, "grant_type is required.")
	case oauth.GrantAuthorizationCode:
		tokens, err = h.Grants.Redeem(ctx, grant.RedeemRequest{
			Code:         form.Get("code"),
			ClientID:     grant.ClientID(form.Get("client_id")),
			RedirectURI:  form.Get("redirect_uri"),
			CodeVerifier: form.Get("code_verifier"),
			RemoteIP:     s.Location(),
			UserAgent:    s.UserAgent(),
		})
	case oauth.GrantRefreshToken:
		if scopeErr := oauth.ParseScope(form.Get("scope")); scopeErr != nil {
			return oauth.WriteError(s, http.StatusBadRequest, *scopeErr)
		}
		holder, holderErr := grant.HeldBy(grant.ClientID(form.Get("client_id")))
		if holderErr != nil {
			return invalidRequest(s, "client_id is required.")
		}
		tokens, err = h.Grants.Refresh(ctx, grant.RefreshRequest{
			RefreshToken: form.Get("refresh_token"),
			Holder:       holder,
			RemoteIP:     s.Location(),
			UserAgent:    s.UserAgent(),
		})
	default:
		return oauth.WriteError(s, http.StatusBadRequest, oauth.Error{Code: oauth.UnsupportedGrantType, Description: "grant_type must be authorization_code or refresh_token."})
	}
	if err != nil {
		return oauth.WriteFailure(s, err)
	}
	return oauth.WriteTokens(s, tokens)
}

func invalidRequest(s *zen.Session, description string) error {
	return oauth.WriteError(s, http.StatusBadRequest, oauth.Error{Code: oauth.InvalidRequest, Description: description})
}
