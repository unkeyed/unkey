package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/grant"
	"github.com/unkeyed/unkey/svc/api/internal/oauth"
)

type Handler struct {
	Grants *grant.Service
}

func (h *Handler) Method() string { return http.MethodGet }

func (h *Handler) Path() string { return oauth.AuthorizePath }

// Handle verifies the client and redirect URI first. Until both are verified
// every failure is a 400 page, never a redirect, so the endpoint cannot be
// used to bounce a browser to an unregistered address (RFC 6749 section
// 4.1.2.1). After that every failure redirects to the verified URI with
// error, state, and iss. On success the browser goes to the dashboard consent
// page with one opaque parameter, which survives the dashboard's sign-in
// redirect where raw OAuth parameters would not.
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	query, ok := oauth.ParseQuery(s.Request())
	if !ok || query.Repeated("client_id", "redirect_uri") {
		return page(s, "This sign-in link is malformed. Start sign-in from the app again.")
	}
	client, err := h.Grants.FindClient(ctx, query.Get("client_id"))
	if errors.Is(err, grant.ErrUnknownClient) {
		return page(s, "This app is not registered with Unkey. Start sign-in from the app again.")
	}
	if err != nil {
		return err
	}
	redirectURI, ok := client.Redirect(query.Get("redirect_uri"))
	if !ok {
		return page(s, "This app's redirect URI does not match its registration. Start sign-in from the app again.")
	}

	state := query.Get("state")
	if len(state) > grant.StateBytesMax {
		state = ""
	}
	fail := func(code oauth.ErrorCode, description string) error {
		location, err := redirectURI.WithError(string(code), description, state, h.Grants.Issuer)
		if err != nil {
			return err
		}
		return oauth.Redirect(s, location)
	}

	switch {
	case query.Repeated():
		return fail(oauth.InvalidRequest, "Parameters must not repeat.")
	case len(query.Get("state")) > grant.StateBytesMax:
		return fail(oauth.InvalidRequest, "state is too long.")
	case query.Get("response_type") == "":
		return fail(oauth.InvalidRequest, "response_type is required.")
	case query.Get("response_type") != "code":
		return fail(oauth.UnsupportedResponseType, "response_type must be code.")
	}
	challenge, err := grant.ParseChallenge(query.Get("code_challenge_method"), query.Get("code_challenge"))
	if err != nil {
		return fail(oauth.InvalidRequest, "PKCE with S256 is required.")
	}
	if oerr := oauth.ParseScope(query.Get("scope")); oerr != nil {
		return fail(oerr.Code, oerr.Description)
	}

	authorized, err := h.Grants.Authorize(ctx, grant.AuthorizeRequest{
		Client:      client,
		RedirectURI: redirectURI,
		State:       state,
		Challenge:   challenge,
		RemoteIP:    s.Location(),
		UserAgent:   s.UserAgent(),
	})
	if err != nil {
		if code, _ := fault.GetCode(err); code == codes.User.TooManyRequests.IPRateLimited.URN() {
			return fail(oauth.TemporarilyUnavailable, fault.UserFacingMessage(err))
		}
		logger.Error("oauth authorize failed", "requestId", s.RequestID(), "error", err.Error())
		return fail(oauth.ServerError, "")
	}
	return oauth.Redirect(s, authorized.ConsentURL)
}

func page(s *zen.Session, message string) error {
	s.AddHeader("Cache-Control", "no-store")
	return s.Plain(http.StatusBadRequest, []byte(message))
}
