package oauth

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/clidevice"
)

// AuthorizeHandler is the authorization endpoint. It sends the browser to the
// dashboard consent page or back to the client with an error.
type AuthorizeHandler struct {
	Devices *clidevice.Service
}

// Method returns the HTTP method this route responds to.
func (h *AuthorizeHandler) Method() string { return http.MethodGet }

// Path returns the URL path pattern this route matches.
func (h *AuthorizeHandler) Path() string { return PathAuthorize }

// Handle redirects the user agent, or answers 400 when the client or its
// redirect URI cannot be trusted.
func (h *AuthorizeHandler) Handle(ctx context.Context, s *zen.Session) error {
	s.SkipRequestLog()
	s.AddHeader("Cache-Control", "no-store")
	params := s.Request().URL.Query()
	if err := rejectRepeated(params); err != nil {
		return writeError(s, err)
	}

	location, err := h.Devices.Authorize(ctx, clidevice.AuthorizeRequest{
		ResponseType:        params.Get("response_type"),
		ClientID:            params.Get("client_id"),
		RedirectURI:         params.Get("redirect_uri"),
		CodeChallenge:       params.Get("code_challenge"),
		CodeChallengeMethod: params.Get("code_challenge_method"),
		State:               params.Get("state"),
		Resources:           params["resource"],
		RemoteIP:            s.Location(),
		UserAgent:           s.UserAgent(),
	})
	if err != nil {
		return writeError(s, err)
	}
	s.AddHeader("Location", location)
	return s.Send(http.StatusFound, nil)
}
