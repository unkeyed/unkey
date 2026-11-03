package clidevice

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"net/url"
	"strings"
	"time"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/hash"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/uid"
)

const (
	stateLengthMax          = 1024
	codeChallengeLength     = 43
	codeVerifierLengthMin   = 43
	codeVerifierLengthMax   = 128
	codeChallengeMethodS256 = "S256"

	grantAuthorizationCode = "authorization_code"
	grantRefreshToken      = "refresh_token"

	authorizationCodePrefix = "ukc_"
	authorizationCodeLength = 32
	authorizationCodeTTL    = 60 * time.Second
)

// OAuthErrorCode is an error code from RFC 6749 sections 4.1.2.1 and 5.2,
// invalid_target from RFC 8707, or a registration error from RFC 7591.
type OAuthErrorCode string

const (
	OAuthInvalidRequest          OAuthErrorCode = "invalid_request"
	OAuthInvalidGrant            OAuthErrorCode = "invalid_grant"
	OAuthUnsupportedGrantType    OAuthErrorCode = "unsupported_grant_type"
	OAuthUnsupportedResponseType OAuthErrorCode = "unsupported_response_type"
	OAuthInvalidTarget           OAuthErrorCode = "invalid_target"
	OAuthAccessDenied            OAuthErrorCode = "access_denied"
	OAuthInvalidRedirectURI      OAuthErrorCode = "invalid_redirect_uri"
	OAuthInvalidClientMetadata   OAuthErrorCode = "invalid_client_metadata"
)

// OAuthError is an OAuth protocol error the client receives as-is.
// Authorize returns it only when the client or redirect URI cannot be trusted,
// so it must be shown to the user agent and never sent to a redirect URI.
type OAuthError struct {
	Code        OAuthErrorCode
	Description string
}

// Error returns the code and description.
func (e *OAuthError) Error() string {
	return string(e.Code) + ": " + e.Description
}

func oauthError(code OAuthErrorCode, description string) *OAuthError {
	return &OAuthError{Code: code, Description: description}
}

// OAuthServer is the OAuth 2.1 authorization server MCP clients sign in
// through. Unkey is the authorization server, the MCP server is a resource
// server, and there is no Unkey-run OAuth client.
type OAuthServer struct {
	// Issuer is the API's public origin (RFC 8414). It names the server in
	// metadata and on every authorization response (RFC 9207).
	Issuer    string
	Resources ResourcePolicy
	Clients   Directory
}

// AuthorizeRequest is an authorization code request with PKCE (RFC 7636) and
// resource indicators (RFC 8707).
type AuthorizeRequest struct {
	ResponseType        string
	ClientID            string
	RedirectURI         string
	CodeChallenge       string
	CodeChallengeMethod string
	State               string
	Resources           []string
	RemoteIP            string
	UserAgent           string
}

// CodeExchangeRequest redeems an authorization code (RFC 6749 section 4.1.3).
type CodeExchangeRequest struct {
	Code         string
	CodeVerifier string
	ClientID     string
	RedirectURI  string
	Resources    []string
	RemoteIP     string
	UserAgent    string
}

// OAuthRefreshRequest runs the refresh_token grant for an OAuth client.
type OAuthRefreshRequest struct {
	RefreshToken string
	ClientID     string
	Resources    []string
	RemoteIP     string
	UserAgent    string
}

// RegisterRequest is an unauthenticated dynamic client registration.
type RegisterRequest struct {
	Registration ClientRegistration
	RemoteIP     string
}

// OAuthEnabled reports whether the OAuth authorization server is configured.
func (s *Service) OAuthEnabled() bool {
	return s.Enabled() && s.OAuth != nil
}

func (s *Service) requireOAuthEnabled() error {
	if !s.OAuthEnabled() {
		return fault.New("oauth is not configured",
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Public("OAuth is not configured on this API."),
		)
	}
	return nil
}

// Register runs RFC 7591 dynamic client registration.
func (s *Service) Register(ctx context.Context, req RegisterRequest) (OAuthClient, error) {
	if err := s.requireOAuthEnabled(); err != nil {
		return OAuthClient{}, err
	}
	if err := s.limit(ctx, "register", req.RemoteIP, startLimitPerMinute); err != nil {
		return OAuthClient{}, err
	}
	return s.OAuth.Clients.Registrar.Register(req.Registration, s.Clock.Now())
}

// Authorize starts a device authorization for an OAuth client and returns
// where to send the user agent: the dashboard consent page, or the client's
// redirect URI carrying an error. An [OAuthError] means the client_id or
// redirect_uri is not trustworthy and the user agent must not be redirected.
func (s *Service) Authorize(ctx context.Context, req AuthorizeRequest) (string, error) {
	if err := s.requireOAuthEnabled(); err != nil {
		return "", err
	}
	if err := s.limit(ctx, "authorize", req.RemoteIP, startLimitPerMinute); err != nil {
		return "", err
	}

	client, err := s.OAuth.Clients.Lookup(ctx, req.ClientID)
	if err != nil {
		return "", err
	}
	if err := checkRedirectURI(req.RedirectURI); err != nil {
		return "", err
	}
	if !client.allowsRedirect(req.RedirectURI) {
		return "", oauthError(OAuthInvalidRequest, "redirect_uri is not registered for this client.")
	}

	reject := func(code OAuthErrorCode, description string) (string, error) {
		return s.clientRedirect(req.RedirectURI, req.State, url.Values{
			"error":             {string(code)},
			"error_description": {description},
		})
	}
	if req.ResponseType != "code" {
		return reject(OAuthUnsupportedResponseType, "response_type must be code.")
	}
	if req.CodeChallengeMethod != codeChallengeMethodS256 {
		return reject(OAuthInvalidRequest, "code_challenge_method must be S256.")
	}
	if !validCodeChallenge(req.CodeChallenge) {
		return reject(OAuthInvalidRequest, "code_challenge must be a base64url SHA-256 digest.")
	}
	if len(req.State) > stateLengthMax {
		return reject(OAuthInvalidRequest, "state is too long.")
	}
	resource, resourceErr := s.OAuth.Resources.resolve(req.Resources)
	if resourceErr != nil {
		return reject(resourceErr.Code, resourceErr.Description)
	}

	started, err := s.insertAuthorization(ctx, StartRequest{
		DeviceName: client.Name,
		RemoteIP:   req.RemoteIP,
		UserAgent:  req.UserAgent,
	}, &mysqltype.DeviceAuthorizationOAuth{
		ClientID:      req.ClientID,
		RedirectURI:   req.RedirectURI,
		CodeChallenge: req.CodeChallenge,
		State:         req.State,
		Resource:      resource,
	})
	if err != nil {
		return "", err
	}
	return started.VerificationURL, nil
}

// ExchangeCode redeems an approved authorization code for an mcp session.
func (s *Service) ExchangeCode(ctx context.Context, req CodeExchangeRequest) (Tokens, error) {
	if err := s.requireOAuthEnabled(); err != nil {
		return Tokens{}, err
	}
	if req.Code == "" || req.CodeVerifier == "" || req.ClientID == "" || req.RedirectURI == "" {
		return Tokens{}, oauthError(OAuthInvalidRequest, "code, code_verifier, client_id, and redirect_uri are required.")
	}
	if !validCodeVerifier(req.CodeVerifier) {
		return Tokens{}, oauthError(OAuthInvalidRequest, "code_verifier must be 43 to 128 unreserved characters.")
	}

	row, err := db.Query.FindDeviceAuthorizationByCodeHash(ctx, s.DB.RW(), sql.NullString{
		String: hashAuthorizationCode(req.Code),
		Valid:  true,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return Tokens{}, oauthError(OAuthInvalidGrant, "The authorization code is invalid.")
		}
		return Tokens{}, err
	}
	binding := row.Oauth
	if binding == nil || binding.ClientID != req.ClientID {
		return Tokens{}, oauthError(OAuthInvalidGrant, "The authorization code is invalid.")
	}
	if binding.RedirectURI != req.RedirectURI {
		return Tokens{}, oauthError(OAuthInvalidGrant, "redirect_uri does not match the authorization request.")
	}
	if !pkceMatches(req.CodeVerifier, binding.CodeChallenge) {
		return Tokens{}, oauthError(OAuthInvalidGrant, "code_verifier does not match the code challenge.")
	}
	if err := s.OAuth.Resources.requireSame(req.Resources, binding.Resource); err != nil {
		return Tokens{}, err
	}

	switch s.authorizationStatus(row) {
	case db.DeviceAuthorizationsStatusApproved:
		if !row.ApproverUserID.Valid || !row.WorkspaceID.Valid {
			return Tokens{}, oauthError(OAuthInvalidGrant, "The authorization code is not approved.")
		}
	case db.DeviceAuthorizationsStatusConsumed:
		if err := s.revokeIssuedSession(ctx, row); err != nil {
			return Tokens{}, err
		}
		return Tokens{}, oauthError(OAuthInvalidGrant, "The authorization code was already used.")
	case db.DeviceAuthorizationsStatusPending, db.DeviceAuthorizationsStatusDenied, db.DeviceAuthorizationsStatusExpired:
		return Tokens{}, oauthError(OAuthInvalidGrant, "The authorization code is not approved or has expired.")
	default:
		return Tokens{}, oauthError(OAuthInvalidGrant, "The authorization code is invalid.")
	}

	result, err := s.issue(ctx, row, req.RemoteIP, req.UserAgent)
	if err != nil {
		return Tokens{}, grantError(err)
	}
	if result.Status != PollComplete {
		description := result.Detail
		if description == "" {
			description = "The authorization changed while the code was redeemed."
		}
		return Tokens{}, oauthError(OAuthInvalidGrant, description)
	}
	return result.Tokens, nil
}

// revokeIssuedSession ends the session a reused authorization code created
// (RFC 6749 section 4.1.2).
func (s *Service) revokeIssuedSession(ctx context.Context, row db.DeviceAuthorization) error {
	if !row.SessionID.Valid {
		return nil
	}
	session, err := db.Query.FindDeviceSessionByID(ctx, s.DB.RW(), row.SessionID.String)
	if err != nil {
		if db.IsNotFound(err) {
			return nil
		}
		return err
	}
	return s.endSession(ctx, session)
}

// RefreshOAuth runs the refresh_token grant for an mcp session. The client
// must be the one the session was issued to, and a resource, when sent, must
// be the one the session is bound to.
func (s *Service) RefreshOAuth(ctx context.Context, req OAuthRefreshRequest) (Tokens, error) {
	if err := s.requireOAuthEnabled(); err != nil {
		return Tokens{}, err
	}
	if req.RefreshToken == "" || req.ClientID == "" {
		return Tokens{}, oauthError(OAuthInvalidRequest, "refresh_token and client_id are required.")
	}
	resource := ""
	if len(req.Resources) > 0 {
		var resourceErr *OAuthError
		resource, resourceErr = s.OAuth.Resources.resolve(req.Resources)
		if resourceErr != nil {
			return Tokens{}, resourceErr
		}
	}
	tokens, err := s.Refresh(ctx, RefreshRequest{
		RefreshToken: req.RefreshToken,
		ClientID:     req.ClientID,
		Resource:     resource,
		RemoteIP:     req.RemoteIP,
		UserAgent:    req.UserAgent,
	})
	if err != nil {
		return Tokens{}, grantError(err)
	}
	return tokens, nil
}

// grantError reports a rejected credential or a closed login as invalid_grant
// and passes protocol and infrastructure errors through.
func grantError(err error) error {
	code, ok := fault.GetCode(err)
	if ok && (code == codes.Auth.Authentication.CredentialRejected.URN() ||
		code == codes.Auth.Authorization.Forbidden.URN() ||
		code == codes.App.Validation.InvalidInput.URN()) {
		return oauthError(OAuthInvalidGrant, "The grant is invalid, expired, or revoked.")
	}
	return err
}

// clientRedirect adds an authorization response to the client's redirect URI
// and keeps any query the client registered (RFC 6749 section 3.1.2). Every
// response names the issuer (RFC 9207).
func (s *Service) clientRedirect(redirectURI string, state string, params url.Values) (string, error) {
	target, err := url.Parse(redirectURI)
	if err != nil {
		return "", fault.Wrap(err, fault.Internal("parse stored redirect_uri"))
	}
	query := target.Query()
	for key, values := range params {
		query[key] = values
	}
	if state != "" {
		query.Set("state", state)
	}
	query.Set("iss", s.OAuth.Issuer)
	target.RawQuery = query.Encode()
	return target.String(), nil
}

// decisionRedirect is where the dashboard sends the browser after the user
// decides. It is empty for a CLI device login.
func (s *Service) decisionRedirect(binding *mysqltype.DeviceAuthorizationOAuth, params url.Values) (string, error) {
	if binding == nil {
		return "", nil
	}
	if s.OAuth == nil {
		return "", fault.New("oauth login decided while oauth is not configured",
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Public("OAuth is not configured on this API."),
		)
	}
	return s.clientRedirect(binding.RedirectURI, binding.State, params)
}

func newAuthorizationCode() string {
	return authorizationCodePrefix + uid.Secure(authorizationCodeLength)
}

func hashAuthorizationCode(code string) string {
	return hash.Sha256(code)
}

func validCodeChallenge(challenge string) bool {
	if len(challenge) != codeChallengeLength {
		return false
	}
	for _, r := range challenge {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '-' && r != '_' {
			return false
		}
	}
	return true
}

func validCodeVerifier(verifier string) bool {
	if len(verifier) < codeVerifierLengthMin || len(verifier) > codeVerifierLengthMax {
		return false
	}
	for _, r := range verifier {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && !strings.ContainsRune("-._~", r) {
			return false
		}
	}
	return true
}

func pkceMatches(verifier string, challenge string) bool {
	digest := sha256.Sum256([]byte(verifier))
	computed := base64.RawURLEncoding.EncodeToString(digest[:])
	return subtle.ConstantTimeCompare([]byte(computed), []byte(challenge)) == 1
}
