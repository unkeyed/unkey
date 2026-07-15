package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/zen"
)

type stubResolver struct {
	principal *principal.Principal
	err       error
	calls     int
}

func (r *stubResolver) Resolve(_ context.Context, _ *zen.Session) (*principal.Principal, error) {
	r.calls++
	return r.principal, r.err
}

// TestServiceAuthenticate_UsesFirstResolvedPrincipal verifies resolver ordering
// is deterministic and stops after the first authenticated principal. This
// matters because multiple credential sources may be configured, but only one
// source may define the request's workspace, subject, and permissions.
func TestServiceAuthenticate_UsesFirstResolvedPrincipal(t *testing.T) {
	t.Parallel()

	first := &stubResolver{}
	second := &stubResolver{principal: testPrincipal("ws_123")}
	third := &stubResolver{principal: testPrincipal("ws_other")}
	service := New(first, second, third)

	got, err := service.Authenticate(context.Background(), &zen.Session{})

	require.NoError(t, err)
	require.Same(t, second.principal, got)
	require.Equal(t, 1, first.calls)
	require.Equal(t, 1, second.calls)
	require.Zero(t, third.calls)
}

// TestServiceAuthenticate_ReusesSessionPrincipal verifies authentication is
// idempotent within a request once a principal is already attached. Reuse keeps
// middleware layers from reinterpreting credentials after the session already
// has the principal that handlers will authorize.
func TestServiceAuthenticate_ReusesSessionPrincipal(t *testing.T) {
	t.Parallel()

	existing := testPrincipal("ws_123")
	resolver := &stubResolver{principal: testPrincipal("ws_other")}
	service := New(resolver)
	sess := &zen.Session{}
	sess.SetPrincipal(existing)

	got, err := service.Authenticate(context.Background(), sess)

	require.NoError(t, err)
	require.Same(t, existing, got)
	require.Zero(t, resolver.calls)
}

// TestServiceAuthenticate_StoresResolvedPrincipalOnSession verifies successful
// authentication publishes the principal for later middleware and handlers.
// The service, not each handler, owns this write so protected route behavior is
// consistent across credential sources.
func TestServiceAuthenticate_StoresResolvedPrincipalOnSession(t *testing.T) {
	t.Parallel()

	want := testPrincipal("ws_123")
	service := New(&stubResolver{principal: want})
	sess := &zen.Session{}

	got, err := service.Authenticate(context.Background(), sess)

	require.NoError(t, err)
	require.Same(t, want, got)
	cached, err := sess.GetPrincipal()
	require.NoError(t, err)
	require.Same(t, want, cached)
}

// TestServiceAuthenticate_ContinuesPastResolverError verifies a rejection by one
// resolver does not block a later resolver that can verify the credential.
// Multiple resolvers may claim the same credential shape (two jwt auth entries
// with different key sources), and each one fully verifies before accepting, so
// continuing is not a bypass.
func TestServiceAuthenticate_ContinuesPastResolverError(t *testing.T) {
	t.Parallel()

	first := &stubResolver{err: errors.New("invalid credential")}
	second := &stubResolver{principal: testPrincipal("ws_123")}
	service := New(first, second)

	got, err := service.Authenticate(context.Background(), &zen.Session{})

	require.NoError(t, err)
	require.Same(t, second.principal, got)
	require.Equal(t, 1, first.calls)
	require.Equal(t, 1, second.calls)
}

// TestServiceAuthenticate_ReturnsFirstResolverErrorWhenNothingResolves verifies
// the most specific rejection is reported when every resolver declines or
// errors, instead of the generic missing-credentials response.
func TestServiceAuthenticate_ReturnsFirstResolverErrorWhenNothingResolves(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("invalid credential")
	first := &stubResolver{err: wantErr}
	second := &stubResolver{err: errors.New("also invalid")}
	third := &stubResolver{}
	service := New(first, second, third)

	got, err := service.Authenticate(context.Background(), &zen.Session{})

	require.ErrorIs(t, err, wantErr)
	require.Nil(t, got)
	require.Equal(t, 1, first.calls)
	require.Equal(t, 1, second.calls)
	require.Equal(t, 1, third.calls)
}

// TestServiceAuthenticate_PrefersInfrastructureErrorOverCredentialRejection
// verifies a resolver that could not determine validity (503) outranks an
// earlier resolver's credential rejection (401), in both orderings. With a
// JWKS-backed and an HS256 jwt entry both claiming the same bearer, an outage
// of one must surface as a retryable failure instead of masking it as an
// invalid token and logging the user out.
func TestServiceAuthenticate_PrefersInfrastructureErrorOverCredentialRejection(t *testing.T) {
	t.Parallel()

	credentialErr := fault.New("bad credential",
		fault.Code(codes.Auth.Authentication.Malformed.URN()),
		fault.Public("Invalid bearer token."),
	)
	infraErr := fault.New("jwks down",
		fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
		fault.Public("Unable to verify the bearer token right now."),
	)

	t.Run("credential rejection first", func(t *testing.T) {
		t.Parallel()

		service := New(&stubResolver{err: credentialErr}, &stubResolver{err: infraErr})

		got, err := service.Authenticate(context.Background(), &zen.Session{})

		require.Nil(t, got)
		code, ok := fault.GetCode(err)
		require.True(t, ok)
		require.Equal(t, codes.App.Internal.ServiceUnavailable.URN(), code)
	})

	t.Run("infrastructure failure first", func(t *testing.T) {
		t.Parallel()

		service := New(&stubResolver{err: infraErr}, &stubResolver{err: credentialErr})

		got, err := service.Authenticate(context.Background(), &zen.Session{})

		require.Nil(t, got)
		code, ok := fault.GetCode(err)
		require.True(t, ok)
		require.Equal(t, codes.App.Internal.ServiceUnavailable.URN(), code)
	})
}

// TestServiceAuthenticate_PrefersAuthorizationOverAuthentication verifies a
// resolver that verified the credential and then denied it (403) outranks a
// sibling resolver's credential rejection (401), in both orderings. With an
// HS256 and a WorkOS JWKS jwt entry both claiming the same bearer, a valid
// WorkOS token whose org has no workspace must surface as the WorkOS resolver's
// 403, not the HS256 resolver's "invalid token" 401 that would log the user out.
func TestServiceAuthenticate_PrefersAuthorizationOverAuthentication(t *testing.T) {
	t.Parallel()

	authnErr := fault.New("bad credential",
		fault.Code(codes.Auth.Authentication.Malformed.URN()),
		fault.Public("Invalid bearer token."),
	)
	authzErr := fault.New("no workspace",
		fault.Code(codes.Auth.Authorization.Forbidden.URN()),
		fault.Public("Your organization does not have an active workspace."),
	)

	t.Run("authentication rejection first", func(t *testing.T) {
		t.Parallel()

		service := New(&stubResolver{err: authnErr}, &stubResolver{err: authzErr})

		got, err := service.Authenticate(context.Background(), &zen.Session{})

		require.Nil(t, got)
		code, ok := fault.GetCode(err)
		require.True(t, ok)
		require.Equal(t, codes.Auth.Authorization.Forbidden.URN(), code)
	})

	t.Run("authorization denial first", func(t *testing.T) {
		t.Parallel()

		service := New(&stubResolver{err: authzErr}, &stubResolver{err: authnErr})

		got, err := service.Authenticate(context.Background(), &zen.Session{})

		require.Nil(t, got)
		code, ok := fault.GetCode(err)
		require.True(t, ok)
		require.Equal(t, codes.Auth.Authorization.Forbidden.URN(), code)
	})
}

// TestServiceAuthenticate_InfrastructureFailureDoesNotMaskSuccess verifies a
// resolver that errors for infrastructure reasons never blocks a later resolver
// that can actually authenticate the request, so a degraded resolver entry
// cannot take down authentication that another entry can complete.
func TestServiceAuthenticate_InfrastructureFailureDoesNotMaskSuccess(t *testing.T) {
	t.Parallel()

	infraErr := fault.New("jwks down",
		fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
	)
	first := &stubResolver{err: infraErr}
	second := &stubResolver{principal: testPrincipal("ws_123")}
	service := New(first, second)

	got, err := service.Authenticate(context.Background(), &zen.Session{})

	require.NoError(t, err)
	require.Same(t, second.principal, got)
	require.Equal(t, 1, first.calls)
	require.Equal(t, 1, second.calls)
}

// TestServiceAuthenticate_ReturnsBearerErrorWhenNoResolverMatches verifies
// unmatched requests still require a bearer credential. This pins the anonymous
// request behavior: no resolver match is an authentication failure, not an empty
// principal with no permissions.
func TestServiceAuthenticate_ReturnsBearerErrorWhenNoResolverMatches(t *testing.T) {
	t.Parallel()

	service := New(&stubResolver{})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	sess := &zen.Session{}
	require.NoError(t, sess.Init(httptest.NewRecorder(), req, 0))

	got, err := service.Authenticate(context.Background(), sess)

	require.Error(t, err)
	require.Nil(t, got)
}

func TestServiceAuthenticate_PreservesMalformedBearerError(t *testing.T) {
	t.Parallel()

	service := New(&stubResolver{})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "not-a-bearer-token")
	sess := &zen.Session{}
	require.NoError(t, sess.Init(httptest.NewRecorder(), req, 0))

	got, err := service.Authenticate(context.Background(), sess)

	require.Error(t, err)
	code, ok := fault.GetCode(err)
	require.True(t, ok)
	require.Equal(t, codes.Auth.Authentication.Malformed.URN(), code)
	require.Nil(t, got)
}

func testPrincipal(workspaceID string) *principal.Principal {
	return &principal.Principal{
		Version: principal.Version,
		Subject: principal.Subject{
			ID:   "user_123",
			Name: "Dashboard User",
			Type: principal.SubjectTypeUser,
		},
		Type:        principal.TypeJWT,
		Source:      principal.JWTSource{},
		WorkspaceID: workspaceID,
		Permissions: []string{"api.*.create_api"},
	}
}
