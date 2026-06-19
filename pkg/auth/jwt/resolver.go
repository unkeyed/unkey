package jwt

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	tokenjwt "github.com/unkeyed/unkey/pkg/jwt"
	"github.com/unkeyed/unkey/pkg/zen"
)

// ErrWorkspaceNotFound is returned by [WorkspaceLookup] implementations when
// the organization has no usable workspace because it is missing or
// soft-deleted. The token itself is valid, so the resolver reports this as a
// forbidden (403) condition rather than an invalid credential (401): the caller
// authenticated, but the organization has no workspace to act in. Reporting it
// as a credential failure would make the dashboard treat a valid session as
// expired and log the user out. Every other lookup error is treated as an
// infrastructure failure, not a credential problem.
var ErrWorkspaceNotFound = errors.New("workspace not found")

// ErrWorkspaceDisabled is returned by [WorkspaceLookup] implementations when
// the organization belongs to a disabled workspace.
var ErrWorkspaceDisabled = errors.New("workspace disabled")

// WorkspaceLookup resolves a JWT organization claim to a workspace ID.
type WorkspaceLookup interface {
	FindWorkspaceIDByOrgID(ctx context.Context, orgID string) (string, error)
}

// WorkspaceLookupFunc adapts a function to the [WorkspaceLookup] interface.
type WorkspaceLookupFunc func(ctx context.Context, orgID string) (string, error)

// FindWorkspaceIDByOrgID implements [WorkspaceLookup] by calling the function.
func (f WorkspaceLookupFunc) FindWorkspaceIDByOrgID(ctx context.Context, orgID string) (string, error) {
	return f(ctx, orgID)
}

// claimsVerifier verifies a bearer token's signature and registered claims
// into Claims. Implementations classify their own failures with fault codes,
// credential rejections as Auth.Authentication.Malformed and key availability
// problems as App.Internal.ServiceUnavailable, so [Resolver.Resolve] passes
// their errors through unchanged.
type claimsVerifier interface {
	verify(ctx context.Context, token string) (Claims, error)
}

// Resolver authenticates bearer JWTs into auth principals.
type Resolver struct {
	verifier        claimsVerifier
	workspaceLookup WorkspaceLookup
}

// NewResolver creates a resolver that verifies HS256 JWTs with any configured secret.
//
// Secrets are tried in order. Callers should put the active signing secret first
// and keep recently retired secrets later in the list until every token signed
// with those secrets has expired.
func NewResolver(workspaceLookup WorkspaceLookup, issuer string, audience string, secrets ...[]byte) (*Resolver, error) {
	if err := assert.All(
		assert.Greater(len(secrets), 0, "at least one JWT secret is required"),
		assert.NotNil(workspaceLookup, "workspace lookup is required"),
		assert.NotEmpty(strings.TrimSpace(issuer), "JWT issuer is required"),
	); err != nil {
		return nil, err
	}

	verifiers := make([]tokenjwt.Verifier[Claims], 0, len(secrets))
	for i, secret := range secrets {
		verifier, err := tokenjwt.NewHS256Verifier[Claims](secret, verifierOptions(issuer, audience)...)
		if err != nil {
			return nil, fmt.Errorf("create verifier for jwt secret %d: %w", i, err)
		}
		verifiers = append(verifiers, verifier)
	}

	return &Resolver{
		verifier:        &secretsVerifier{verifiers: verifiers},
		workspaceLookup: workspaceLookup,
	}, nil
}

// NewResolverWithJWKSURL creates a resolver that verifies RS256 JWTs signed by
// keys from the configured JWKS endpoint.
//
// The key set is fetched lazily on first use and refetched when a token's
// signing key may be missing from the cached set, so signing-key rotations are
// picked up without a restart. A temporarily unreachable JWKS endpoint does
// not prevent construction; affected requests fail until the endpoint recovers.
func NewResolverWithJWKSURL(workspaceLookup WorkspaceLookup, issuer string, audience string, jwksURL string) (*Resolver, error) {
	if err := assert.All(
		assert.NotNil(workspaceLookup, "workspace lookup is required"),
		assert.NotEmpty(strings.TrimSpace(issuer), "JWT issuer is required"),
		assert.NotEmpty(strings.TrimSpace(jwksURL), "JWKS URL must not be empty"),
	); err != nil {
		return nil, err
	}

	return &Resolver{
		verifier: &jwksVerifier{
			jwksURL:     jwksURL,
			issuer:      issuer,
			audience:    audience,
			clock:       clock.New(),
			current:     atomic.Pointer[keySet]{},
			fetchMu:     sync.Mutex{},
			lastAttempt: time.Time{},
		},
		workspaceLookup: workspaceLookup,
	}, nil
}

// Resolve claims JWT-shaped bearer tokens and leaves other credentials to later resolvers.
func (r *Resolver) Resolve(ctx context.Context, sess *zen.Session) (*principal.Principal, error) {
	token, err := zen.Bearer(sess)
	if err != nil {
		return nil, nil
	}
	if strings.Count(token, ".") != 2 {
		return nil, nil
	}
	segments := strings.Split(token, ".")

	claims, err := r.verifier.verify(ctx, token)
	if err != nil {
		return nil, err
	}
	orgID := claims.organizationID()
	subjectID := claims.subjectID()
	if subjectID == "" || orgID == "" || claims.ExpiresAt == 0 || claims.IssuedAt == 0 {
		return nil, fault.New("invalid JWT claims",
			fault.Code(codes.Auth.Authentication.Malformed.URN()),
			fault.Internal("JWT subject, organization id, exp, and iat are required"),
			fault.Public("Invalid bearer token."),
		)
	}
	workspaceID, err := r.workspaceLookup.FindWorkspaceIDByOrgID(ctx, orgID)
	if err != nil {
		if errors.Is(err, ErrWorkspaceNotFound) {
			return nil, fault.Wrap(err,
				fault.Code(codes.Auth.Authorization.Forbidden.URN()),
				fault.Internal("JWT organization has no usable workspace"),
				fault.Public("Your organization does not have an active workspace."),
			)
		}
		if errors.Is(err, ErrWorkspaceDisabled) {
			return nil, fault.Wrap(err,
				fault.Code(codes.Auth.Authorization.WorkspaceDisabled.URN()),
				fault.Internal("JWT organization workspace is disabled"),
				fault.Public("Your organization does not have an active workspace."),
			)
		}
		return nil, fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("failed to resolve JWT organization to workspace"),
			fault.Public("Unable to verify the bearer token right now."),
		)
	}
	if workspaceID == "" {
		return nil, fault.New("invalid JWT organization",
			fault.Code(codes.Auth.Authentication.Malformed.URN()),
			fault.Internal("JWT organization resolved to empty workspace id"),
			fault.Public("Invalid bearer token."),
		)
	}
	header, err := decodeSegment(segments[0])
	if err != nil {
		return nil, fault.Wrap(err,
			fault.Code(codes.Auth.Authentication.Malformed.URN()),
			fault.Internal("failed to decode JWT header"),
			fault.Public("Invalid bearer token."),
		)
	}
	payload, err := decodeSegment(segments[1])
	if err != nil {
		return nil, fault.Wrap(err,
			fault.Code(codes.Auth.Authentication.Malformed.URN()),
			fault.Internal("failed to decode JWT payload"),
			fault.Public("Invalid bearer token."),
		)
	}

	return &principal.Principal{
		Version: principal.Version,
		Subject: principal.Subject{
			ID:   subjectID,
			Name: claims.subjectName(),
			Type: principal.SubjectTypeUser,
		},
		Type: principal.TypeJWT,
		Source: principal.JWTSource{
			Header:    header,
			Payload:   payload,
			Signature: segments[2],
		},
		WorkspaceID: workspaceID,
		Permissions: claims.permissions(),
	}, nil
}

// secretsVerifier verifies HS256 tokens against an ordered secret list so the
// dashboard can rotate its signing secret without invalidating live tokens.
type secretsVerifier struct {
	verifiers []tokenjwt.Verifier[Claims]
}

var _ claimsVerifier = (*secretsVerifier)(nil)

func (s *secretsVerifier) verify(_ context.Context, token string) (Claims, error) {
	var lastErr error
	for _, verifier := range s.verifiers {
		claims, err := verifier.Verify(token)
		if err == nil {
			return claims, nil
		}
		lastErr = err
	}

	var zero Claims
	return zero, fault.Wrap(lastErr,
		fault.Code(codes.Auth.Authentication.Malformed.URN()),
		fault.Internal("failed to verify JWT"),
		fault.Public("Invalid bearer token."),
	)
}

// decodeSegment base64url-decodes one JWT segment into a generic map for the
// principal source, never returning a nil map.
func decodeSegment(segment string) (map[string]any, error) {
	raw, err := base64.RawURLEncoding.DecodeString(segment)
	if err != nil {
		return nil, err
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, err
	}
	if decoded == nil {
		return map[string]any{}, nil
	}
	return decoded, nil
}

// verifierOptions binds verification to the configured issuer and, when set,
// the audience.
func verifierOptions(issuer string, audience string) []tokenjwt.VerifyOption {
	options := []tokenjwt.VerifyOption{tokenjwt.WithIssuer(issuer)}
	if strings.TrimSpace(audience) != "" {
		options = append(options, tokenjwt.WithAudience(audience))
	}
	return options
}
