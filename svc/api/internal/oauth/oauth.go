// Package oauth is the wire vocabulary of the API's OAuth 2.1 authorization
// server: endpoint paths, single-valued parameter parsing, RFC 6749 and RFC
// 7591 error bodies, token responses, scope and resource rules, and the RFC
// 8414 metadata document. Grant semantics (clients, PKCE, codes, sessions)
// live in package grant; this package never touches the database.
package oauth

import (
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/grant"
)

// Endpoint paths. Route handlers return these from Path() and Metadata builds
// URLs from them.
const (
	MetadataPath  = "/.well-known/oauth-authorization-server"
	AuthorizePath = "/oauth/authorize"
	TokenPath     = "/oauth/token"
	RegisterPath  = "/oauth/register"
)

// ScopeAPI is the only scope. Consent, not scope, picks permissions: the
// approver chooses them in the dashboard, narrowed to their role.
const ScopeAPI = "api"

// ScopesSupported is advertised in metadata.
var ScopesSupported = []string{ScopeAPI}

const (
	formContentType = "application/x-www-form-urlencoded"
	jsonContentType = "application/json"
)

// ErrorCode is an RFC 6749, RFC 7591, or RFC 8707 error code.
type ErrorCode string

const (
	InvalidRequest          ErrorCode = "invalid_request"
	InvalidGrant            ErrorCode = "invalid_grant"
	UnsupportedGrantType    ErrorCode = "unsupported_grant_type"
	UnsupportedResponseType ErrorCode = "unsupported_response_type"
	InvalidScope            ErrorCode = "invalid_scope"
	InvalidTarget           ErrorCode = "invalid_target"
	AccessDenied            ErrorCode = "access_denied"
	ServerError             ErrorCode = "server_error"
	TemporarilyUnavailable  ErrorCode = "temporarily_unavailable"
	InvalidRedirectURI      ErrorCode = "invalid_redirect_uri"
	InvalidClientMetadata   ErrorCode = "invalid_client_metadata"
)

// Error is a protocol error body. Handlers write it and return nil, so the
// API's problem+json middleware never renders it.
type Error struct {
	Code        ErrorCode `json:"error"`
	Description string    `json:"error_description,omitempty"`
}

// WriteError writes e with status and Cache-Control: no-store, as RFC 6749
// section 5.2 requires of the token endpoint.
func WriteError(s *zen.Session, status int, e Error) error {
	noStore(s)
	return s.JSON(status, e)
}

// Params is a query or form in which a parameter may appear at most once
// (RFC 6749 section 3.1). Get returns "" for a repeated parameter; Repeated
// reports it.
type Params struct {
	values url.Values
}

// ParseQuery reads the request's query string.
func ParseQuery(r *http.Request) (Params, bool) {
	values, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return Params{values: url.Values{}}, false
	}
	return Params{values: values}, true
}

// ParseForm reads an application/x-www-form-urlencoded body. Any other
// content type, or a body that does not parse, is invalid_request.
func ParseForm(r *http.Request) (Params, *Error) {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != formContentType {
		return Params{values: url.Values{}}, &Error{Code: InvalidRequest, Description: "The request body must be application/x-www-form-urlencoded."}
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return Params{values: url.Values{}}, &Error{Code: InvalidRequest, Description: "The request body could not be read."}
	}
	values, err := url.ParseQuery(string(body))
	if err != nil {
		return Params{values: url.Values{}}, &Error{Code: InvalidRequest, Description: "The request body is not a valid form."}
	}
	return Params{values: values}, nil
}

// IsJSON reports whether the request declares a JSON body.
func IsJSON(r *http.Request) bool {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	return err == nil && mediaType == jsonContentType
}

// Get returns the parameter's value, or "" when it is absent or repeated.
func (p Params) Get(name string) string {
	if values := p.values[name]; len(values) == 1 {
		return values[0]
	}
	return ""
}

// Has reports whether the parameter appeared at all.
func (p Params) Has(name string) bool {
	_, ok := p.values[name]
	return ok
}

// Repeated reports whether any of names appeared more than once. With no
// names, whether any parameter did.
func (p Params) Repeated(names ...string) bool {
	if len(names) == 0 {
		for _, values := range p.values {
			if len(values) > 1 {
				return true
			}
		}
		return false
	}
	return slices.ContainsFunc(names, func(name string) bool { return len(p.values[name]) > 1 })
}

// ParseScope accepts an absent scope or a space-delimited set within
// ScopesSupported.
func ParseScope(raw string) *Error {
	for _, scope := range strings.Fields(raw) {
		if !slices.Contains(ScopesSupported, scope) {
			return &Error{Code: InvalidScope, Description: "The only supported scope is " + ScopeAPI + "."}
		}
	}
	return nil
}

// ResourceAllowed accepts an absent resource (RFC 8707) or the issuer, with
// or without a trailing slash. Access tokens are always for this API.
func ResourceAllowed(raw string, issuer string) bool {
	return raw == "" || strings.TrimRight(raw, "/") == issuer
}

// TokenResponse is the RFC 6749 section 5.1 body.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

// WriteTokens writes a 200 token response that caches never store.
func WriteTokens(s *zen.Session, t grant.Tokens) error {
	noStore(s)
	return s.JSON(http.StatusOK, TokenResponse{
		AccessToken:  t.AccessToken,
		TokenType:    "Bearer",
		ExpiresIn:    t.ExpiresIn,
		RefreshToken: t.RefreshToken,
		Scope:        ScopeAPI,
	})
}

// Redirect writes a 302 to location that caches never store.
func Redirect(s *zen.Session, location string) error {
	s.AddHeader("Location", location)
	s.AddHeader("Cache-Control", "no-store")
	return s.Send(http.StatusFound, nil)
}

func noStore(s *zen.Session) {
	s.AddHeader("Cache-Control", "no-store")
	s.AddHeader("Pragma", "no-cache")
}

// Metadata is the RFC 8414 authorization server metadata document.
type Metadata struct {
	Issuer                                     string   `json:"issuer"`
	AuthorizationEndpoint                      string   `json:"authorization_endpoint"`
	TokenEndpoint                              string   `json:"token_endpoint"`
	RegistrationEndpoint                       string   `json:"registration_endpoint"`
	ScopesSupported                            []string `json:"scopes_supported"`
	ResponseTypesSupported                     []string `json:"response_types_supported"`
	ResponseModesSupported                     []string `json:"response_modes_supported"`
	GrantTypesSupported                        []string `json:"grant_types_supported"`
	TokenEndpointAuthMethodsSupported          []string `json:"token_endpoint_auth_methods_supported"`
	CodeChallengeMethodsSupported              []string `json:"code_challenge_methods_supported"`
	AuthorizationResponseIssParameterSupported bool     `json:"authorization_response_iss_parameter_supported"`
	ServiceDocumentation                       string   `json:"service_documentation"`
}

// MetadataMaxAge is how long clients may cache the metadata document.
const MetadataMaxAge = 5 * time.Minute

// NewMetadata derives every URL from issuer and the path constants.
func NewMetadata(issuer string) Metadata {
	return Metadata{
		Issuer:                                     issuer,
		AuthorizationEndpoint:                      issuer + AuthorizePath,
		TokenEndpoint:                              issuer + TokenPath,
		RegistrationEndpoint:                       issuer + RegisterPath,
		ScopesSupported:                            ScopesSupported,
		ResponseTypesSupported:                     []string{"code"},
		ResponseModesSupported:                     []string{"query"},
		GrantTypesSupported:                        []string{GrantAuthorizationCode, GrantRefreshToken},
		TokenEndpointAuthMethodsSupported:          []string{AuthMethodNone},
		CodeChallengeMethodsSupported:              []string{"S256"},
		AuthorizationResponseIssParameterSupported: true,
		ServiceDocumentation:                       "https://www.unkey.com/docs/platform/ai-coding/mcp-servers",
	}
}

// Grant types and the client authentication method the server supports.
const (
	GrantAuthorizationCode = "authorization_code"
	GrantRefreshToken      = "refresh_token"
	AuthMethodNone         = "none"
)

// WriteFailure answers err in RFC 6749 shape. A grant refusal is
// invalid_grant with its public message, a rate limit is 429, an unconfigured
// server is 503, and anything else is logged and answered 500 server_error.
// Token endpoint clients cannot parse problem JSON, so no failure reaches the
// API's error middleware.
func WriteFailure(s *zen.Session, err error) error {
	if errors.Is(err, grant.ErrRejected) {
		return WriteError(s, http.StatusBadRequest, Error{Code: InvalidGrant, Description: fault.UserFacingMessage(err)})
	}
	code, _ := fault.GetCode(err)
	//nolint:exhaustive // every other code is an infrastructure failure here
	switch code {
	case codes.User.TooManyRequests.IPRateLimited.URN():
		s.AddHeader("Retry-After", "60")
		return WriteError(s, http.StatusTooManyRequests, Error{Code: InvalidRequest, Description: fault.UserFacingMessage(err)})
	case codes.App.Internal.ServiceUnavailable.URN():
		return WriteError(s, http.StatusServiceUnavailable, Error{Code: TemporarilyUnavailable, Description: fault.UserFacingMessage(err)})
	default:
		logger.Error("oauth endpoint failed",
			"requestId", s.RequestID(),
			"path", s.Request().URL.Path,
			"error", err.Error(),
		)
		return WriteError(s, http.StatusInternalServerError, Error{Code: ServerError, Description: ""})
	}
}
