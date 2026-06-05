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
	require.Same(t, want, sess.Principal())
}

// TestServiceAuthenticate_StopsOnResolverError verifies verification errors fail
// closed and prevent later resolvers from claiming the request. A resolver error
// means the request contained that resolver's credential shape and verification
// rejected it, so trying a later source would create an auth bypass path.
func TestServiceAuthenticate_StopsOnResolverError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("invalid credential")
	first := &stubResolver{err: wantErr}
	second := &stubResolver{principal: testPrincipal("ws_123")}
	service := New(first, second)

	got, err := service.Authenticate(context.Background(), &zen.Session{})

	require.ErrorIs(t, err, wantErr)
	require.Nil(t, got)
	require.Equal(t, 1, first.calls)
	require.Zero(t, second.calls)
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
