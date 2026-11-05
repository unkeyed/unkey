package clisession

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/auth"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	tokenjwt "github.com/unkeyed/unkey/pkg/jwt"
	"github.com/unkeyed/unkey/pkg/zen"
)

// ErrInactive means the session row is missing, revoked, or expired.
var ErrInactive = errors.New("cli session is not active")

// Session is the non-secret state a resolver needs from the session row.
type Session struct {
	UserID      string
	WorkspaceID string
	DeviceName  string
	ExpiresAt   time.Time
}

// Sessions loads the live session behind an access token.
// Implementations should cache for a few seconds. A cache hit must not outlive
// a revocation by more than that window.
type Sessions interface {
	Allow(ctx context.Context, sessionID string) (Session, error)
}

// Resolver authenticates CLI access tokens into user principals.
type Resolver struct {
	verifiers []tokenjwt.Verifier[Claims]
	sessions  Sessions
}

// NewResolver verifies CLI access tokens with any of the configured secrets.
// The first secret is the one the API signs with. Later secrets verify tokens
// signed before a rotation.
func NewResolver(sessions Sessions, clk clock.Clock, secrets ...[]byte) (*Resolver, error) {
	if err := assert.All(
		assert.NotNil(sessions, "cli session lookup is required"),
		assert.NotNil(clk, "clock is required"),
		assert.Greater(len(secrets), 0, "at least one CLI signing secret is required"),
	); err != nil {
		return nil, err
	}

	verifiers := make([]tokenjwt.Verifier[Claims], 0, len(secrets))
	for i, secret := range secrets {
		verifier, err := tokenjwt.NewHS256Verifier[Claims](secret,
			tokenjwt.WithIssuer(Issuer),
			tokenjwt.WithAudience(Audience),
			tokenjwt.WithClock(clk),
		)
		if err != nil {
			return nil, fault.Wrap(err,
				fault.Internal(fmt.Sprintf("create cli verifier for secret %d", i)),
			)
		}

		verifiers = append(verifiers, verifier)
	}

	return &Resolver{
		verifiers: verifiers,
		sessions:  sessions,
	}, nil
}

// Resolve claims CLI access tokens and leaves every other bearer for later
// resolvers. Once the token_use claim marks a token as a CLI access token,
// every refusal is CredentialRejected, so the chain reports it over a sibling
// resolver's rejection of the same token.
func (r *Resolver) Resolve(ctx context.Context, sess *zen.Session) (*principal.Principal, error) {
	token, err := zen.Bearer(sess)
	if err != nil || tokenUse(token) != TokenUse {
		return nil, nil
	}

	claims, err := r.verify(token)
	if err != nil {
		return nil, fault.Wrap(err,
			fault.Code(codes.Auth.Authentication.CredentialRejected.URN()),
			fault.Internal("failed to verify cli access token"),
			fault.Public("Invalid bearer token."),
		)
	}

	if claims.Subject == "" || claims.SessionID == "" ||
		claims.WorkspaceID == "" || len(claims.Permissions) == 0 {
		return nil, fault.New("invalid cli access token",
			fault.Code(codes.Auth.Authentication.CredentialRejected.URN()),
			fault.Internal("cli token is missing subject, session, workspace, or permissions"),
			fault.Public("Invalid bearer token."),
		)
	}

	session, err := r.sessions.Allow(ctx, claims.SessionID)
	if err != nil {
		if errors.Is(err, ErrInactive) {
			return nil, fault.Wrap(err,
				fault.Code(codes.Auth.Authentication.CredentialRejected.URN()),
				fault.Internal("cli session is revoked, expired, or unknown"),
				fault.Public(sessionEndedMessage(claims)),
			)
		}

		return nil, fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("failed to load cli session"),
			fault.Public("Unable to verify the bearer token right now."),
		)
	}

	if session.UserID != claims.Subject || session.WorkspaceID != claims.WorkspaceID {
		return nil, fault.New("cli token does not match the session",
			fault.Code(codes.Auth.Authentication.CredentialRejected.URN()),
			fault.Internal("cli token subject or workspace does not match the session row"),
			fault.Public("Invalid bearer token."),
		)
	}

	return &principal.Principal{
		Version: principal.Version,
		Subject: principal.Subject{
			ID:   claims.Subject,
			Name: claims.Name,
			Type: principal.SubjectTypeUser,
		},
		Type: principal.TypeJWT,
		Source: principal.CLISessionSource{
			SessionID:  claims.SessionID,
			DeviceName: session.DeviceName,
			ExpiresAt:  session.ExpiresAt,
		},
		AuthorizedWorkspaceID: claims.WorkspaceID,
		Permissions:           claims.Permissions,
	}, nil
}

func sessionEndedMessage(claims Claims) string {
	if claims.issuedToMCPClient() {
		return "This MCP session is no longer valid. Sign in to the Unkey MCP server again from your MCP client."
	}
	return "This CLI session is no longer valid. Run unkey login again."
}

func (r *Resolver) verify(token string) (Claims, error) {
	var lastErr error
	for _, verifier := range r.verifiers {
		claims, err := verifier.Verify(token)
		if err == nil {
			return claims, nil
		}

		lastErr = err
	}

	var zero Claims
	return zero, lastErr
}

type tokenUseClaims struct {
	TokenUse string `json:"token_use"`
}

func tokenUse(token string) string {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return ""
	}

	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}

	var peek tokenUseClaims
	if err := json.Unmarshal(raw, &peek); err != nil {
		return ""
	}

	return peek.TokenUse
}

var _ auth.Resolver = (*Resolver)(nil)
