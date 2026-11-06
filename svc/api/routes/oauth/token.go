package oauth

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/clidevice"
)

const (
	grantAuthorizationCode = "authorization_code"
	grantRefreshToken      = "refresh_token"
)

// TokenResponse is an RFC 6749 section 5.1 access token response.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

// TokenHandler is the token endpoint for public clients.
type TokenHandler struct {
	Devices *clidevice.Service
}

// Method returns the HTTP method this route responds to.
func (h *TokenHandler) Method() string { return http.MethodPost }

// Path returns the URL path pattern this route matches.
func (h *TokenHandler) Path() string { return PathToken }

// Handle redeems an authorization code or rotates a refresh token. It reads
// only the form body, so codes and verifiers never ride in a URL that access
// logs keep.
func (h *TokenHandler) Handle(ctx context.Context, s *zen.Session) error {
	s.SkipRequestLog()
	s.AddHeader("Cache-Control", "no-store")
	r := s.Request()
	if err := r.ParseForm(); err != nil {
		return writeError(s, &clidevice.OAuthError{
			Code:        clidevice.OAuthInvalidRequest,
			Description: "The request body must be application/x-www-form-urlencoded.",
		})
	}
	params := r.PostForm
	if err := rejectRepeated(params); err != nil {
		return writeError(s, err)
	}

	var tokens clidevice.Tokens
	var err error
	switch grant := params.Get("grant_type"); grant {
	case grantAuthorizationCode:
		tokens, err = h.Devices.ExchangeCode(ctx, clidevice.CodeExchangeRequest{
			Code:         params.Get("code"),
			CodeVerifier: params.Get("code_verifier"),
			ClientID:     params.Get("client_id"),
			RedirectURI:  params.Get("redirect_uri"),
			Resources:    params["resource"],
			RemoteIP:     s.Location(),
			UserAgent:    s.UserAgent(),
		})
	case grantRefreshToken:
		tokens, err = h.Devices.RefreshOAuth(ctx, clidevice.OAuthRefreshRequest{
			RefreshToken: params.Get("refresh_token"),
			ClientID:     params.Get("client_id"),
			Resources:    params["resource"],
			RemoteIP:     s.Location(),
			UserAgent:    s.UserAgent(),
		})
	case "":
		err = &clidevice.OAuthError{Code: clidevice.OAuthInvalidRequest, Description: "grant_type is required."}
	default:
		err = &clidevice.OAuthError{Code: clidevice.OAuthUnsupportedGrantType, Description: grant + " is not supported."}
	}
	if err != nil {
		return writeError(s, err)
	}

	return s.JSON(http.StatusOK, TokenResponse{
		AccessToken:  tokens.AccessToken,
		TokenType:    "Bearer",
		ExpiresIn:    tokens.ExpiresIn,
		RefreshToken: tokens.RefreshToken,
	})
}
