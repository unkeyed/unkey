// Package granttest builds a [grant.Service] for route tests.
package granttest

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auth/clisession"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/auth/workos"
	"github.com/unkeyed/unkey/pkg/cache"
	tokenjwt "github.com/unkeyed/unkey/pkg/jwt"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/grant"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
)

const signingSecret = "test-cli-signing-secret-32-bytes!"

type allowAdmin struct{}

func (allowAdmin) ActiveRoles(context.Context, string, string) ([]string, bool, error) {
	return []string{"admin"}, true, nil
}

// New returns a service that treats every user as an admin member.
func New(h *testutil.Harness) *grant.Service {
	signer, err := tokenjwt.NewHS256Signer[clisession.Claims]([]byte(signingSecret))
	if err != nil {
		panic(err)
	}
	return &grant.Service{
		DB:               h.DB,
		Auditlogs:        h.Auditlogs,
		Clock:            h.Clock,
		Ratelimit:        h.Ratelimit,
		Memberships:      allowAdmin{},
		DashboardBaseURL: "http://dashboard.test",
		Signer:           signer,
		SessionCache:     cache.NewNoopCache[string, grant.SessionVerdict](),
		Issuer:           Issuer,
		RedirectOrigins:  nil,
	}
}

// Issuer is the OAuth issuer of services built by New.
const Issuer = "http://api.test"

// Approver is a dashboard user session with the admin role in the harness's
// user workspace: the principal the consent page approves with.
func Approver(h *testutil.Harness) *principal.Principal {
	workspace := h.Resources().UserWorkspace
	return &principal.Principal{
		Version: principal.Version,
		Type:    principal.TypeJWT,
		Subject: principal.Subject{ID: uid.New(uid.TestPrefix), Name: "Approver", Type: principal.SubjectTypeUser},
		Source: principal.JWTSource{
			Header:    map[string]any{},
			Payload:   map[string]any{},
			Roles:     []string{"admin"},
			Signature: "",
		},
		AuthorizedWorkspaceID: workspace.ID,
		Permissions:           workos.PermissionsForRoles(workspace.ID, []string{"admin"}),
	}
}

// AsPrincipal is the public middleware stack followed by a middleware that
// authenticates every request as p, the way the dashboard proxy does.
func AsPrincipal(h *testutil.Harness, p *principal.Principal) []zen.Middleware {
	return append(h.PublicMiddleware(), func(next zen.HandleFunc) zen.HandleFunc {
		return func(ctx context.Context, session *zen.Session) error {
			session.SetPrincipal(p)
			return next(ctx, session)
		}
	})
}

// OAuthRequest is a pending OAuth authorization request and the secrets only
// its client knows.
type OAuthRequest struct {
	Client       grant.Client
	RedirectURI  string
	UserCode     string
	State        string
	CodeVerifier string
}

// PendingOAuthRequest registers a client and starts an authorization request
// for it, the way /oauth/authorize does.
func PendingOAuthRequest(t *testing.T, svc *grant.Service) OAuthRequest {
	t.Helper()
	redirectURI := "https://gateway.test/oauth/callback"
	client, err := svc.RegisterClient(t.Context(), grant.RegisterClientRequest{
		Name:         "Unkey MCP",
		RedirectURIs: []string{redirectURI},
		RemoteIP:     "",
		UserAgent:    "",
	})
	require.NoError(t, err)
	verified, ok := client.Redirect(redirectURI)
	require.True(t, ok)
	verifier := uid.Secure(64)
	challenge, err := grant.ParseChallenge("S256", S256(verifier))
	require.NoError(t, err)
	state := uid.Secure(16)
	authorized, err := svc.Authorize(t.Context(), grant.AuthorizeRequest{
		Client:      client,
		RedirectURI: verified,
		State:       state,
		Challenge:   challenge,
		RemoteIP:    "",
		UserAgent:   "",
	})
	require.NoError(t, err)
	consent, err := url.Parse(authorized.ConsentURL)
	require.NoError(t, err)
	return OAuthRequest{
		Client:       client,
		RedirectURI:  redirectURI,
		UserCode:     consent.Query().Get("user_code"),
		State:        state,
		CodeVerifier: verifier,
	}
}

// S256 is the PKCE S256 code challenge for verifier.
func S256(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// MustApprove approves a login and fails the test on error.
func MustApprove(t *testing.T, svc *grant.Service, req grant.ApproveRequest) grant.Decided {
	t.Helper()
	decided, err := svc.Approve(t.Context(), req)
	require.NoError(t, err)
	return decided
}

// MustDeny denies a login and fails the test on error.
func MustDeny(t *testing.T, svc *grant.Service, req grant.DenyRequest) grant.Decided {
	t.Helper()
	decided, err := svc.Deny(t.Context(), req)
	require.NoError(t, err)
	return decided
}

// OAuthSession approves req as approver with permissions and redeems the
// code, returning the session the OAuth client holds.
func OAuthSession(t *testing.T, svc *grant.Service, req OAuthRequest, approver *principal.Principal, permissions []string) grant.Tokens {
	t.Helper()
	decided := MustApprove(t, svc, grant.ApproveRequest{
		Principal:   approver,
		UserCode:    req.UserCode,
		Permissions: permissions,
	})
	callback, err := url.Parse(decided.RedirectTo)
	require.NoError(t, err)
	tokens, err := svc.Redeem(t.Context(), grant.RedeemRequest{
		Code:         callback.Query().Get("code"),
		ClientID:     req.Client.ID,
		RedirectURI:  req.RedirectURI,
		CodeVerifier: req.CodeVerifier,
		RemoteIP:     "",
		UserAgent:    "",
	})
	require.NoError(t, err)
	return tokens
}

// SessionMiddleware is the protected stack that also accepts the session
// access tokens svc issues, the way production appends the session resolver
// after the root key resolver.
func SessionMiddleware(h *testutil.Harness, svc *grant.Service) []zen.Middleware {
	resolver, err := clisession.NewResolver(svc, h.Clock, []byte(signingSecret))
	if err != nil {
		panic(err)
	}
	return h.ProtectedMiddleware(resolver)
}
