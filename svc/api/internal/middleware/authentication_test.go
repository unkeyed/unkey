package middleware

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	keysdb "github.com/unkeyed/unkey/internal/services/keys/db"
	"github.com/unkeyed/unkey/internal/services/ratelimit"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/zen"
)

type fakeAuth struct {
	principal *principal.Principal
	err       error
	calls     int
}

func (a *fakeAuth) Authenticate(_ context.Context, sess *zen.Session) (*principal.Principal, error) {
	a.calls++
	if a.err != nil {
		return nil, a.err
	}
	sess.SetPrincipal(a.principal)
	return a.principal, nil
}

type fakeRatelimit struct {
	response ratelimit.RatelimitResponse
	err      error
	calls    int
	request  ratelimit.RatelimitRequest
}

func (r *fakeRatelimit) Ratelimit(_ context.Context, req ratelimit.RatelimitRequest) (ratelimit.RatelimitResponse, error) {
	r.calls++
	r.request = req
	return r.response, r.err
}

func (r *fakeRatelimit) RatelimitMany(_ context.Context, _ []ratelimit.RatelimitRequest) ([]ratelimit.RatelimitResponse, error) {
	return nil, errors.New("not implemented")
}

// TestWithAuthentication_StopsOnAuthError verifies authentication is the first
// protected-route gate. A failed credential check must return immediately,
// before workspace policy can spend quota and before handler code can observe
// any request state.
func TestWithAuthentication_StopsOnAuthError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("invalid auth")
	auth := &fakeAuth{err: wantErr}
	rl := &fakeRatelimit{}
	handlerCalled := false

	err := WithAuthentication(AuthenticationConfig{
		Auth:      auth,
		Ratelimit: rl,
	})(func(_ context.Context, _ *zen.Session) error {
		handlerCalled = true
		return nil
	})(context.Background(), &zen.Session{})

	require.ErrorIs(t, err, wantErr)
	require.Equal(t, 1, auth.calls)
	require.Zero(t, rl.calls)
	require.False(t, handlerCalled)
}

// TestWithAuthentication_PublishesPrincipalToHandler verifies the middleware
// preserves the auth service contract that downstream handlers read principals
// from the session. This catches regressions where authentication succeeds but
// handlers fail closed because no principal was attached.
func TestWithAuthentication_PublishesPrincipalToHandler(t *testing.T) {
	t.Parallel()

	want := testMiddlewarePrincipal("ws_123")
	auth := &fakeAuth{principal: want}

	err := WithAuthentication(AuthenticationConfig{
		Auth: auth,
	})(func(_ context.Context, sess *zen.Session) error {
		got, err := sess.GetPrincipal()
		require.NoError(t, err)
		require.Same(t, want, got)
		return nil
	})(context.Background(), &zen.Session{})

	require.NoError(t, err)
	require.Equal(t, 1, auth.calls)
}

// TestWithAuthentication_EnforcesWorkspaceRateLimit verifies workspace-level
// API policy is keyed from the authenticated principal and denies the request
// before handler execution. The quota is preloaded into the cache so the test
// stays focused on middleware ordering and ratelimit request construction,
// without depending on a database fixture.
func TestWithAuthentication_EnforcesWorkspaceRateLimit(t *testing.T) {
	t.Parallel()

	quotaCache, err := cache.New[string, keysdb.Quotas](cache.Config[string, keysdb.Quotas]{
		Fresh:    time.Minute,
		Stale:    time.Minute,
		MaxSize:  10,
		Resource: "test_workspace_quota",
		Clock:    clock.NewTestClock(),
	})
	require.NoError(t, err)
	quotaCache.Set(context.Background(), "ws_123", keysdb.Quotas{
		Pk:                          0,
		WorkspaceID:                 "ws_123",
		RequestsPerMonth:            0,
		LogsRetentionDays:           0,
		AuditLogsRetentionDays:      0,
		Team:                        false,
		RatelimitApiLimit:           sql.NullInt32{Int32: 1, Valid: true},
		RatelimitApiDuration:        sql.NullInt32{Int32: 60_000, Valid: true},
		AllocatedCpuMillicoresTotal: 0,
		AllocatedMemoryMibTotal:     0,
		AllocatedStorageMibTotal:    0,
		MaxCpuMillicoresPerInstance: 0,
		MaxMemoryMibPerInstance:     0,
		MaxStorageMibPerInstance:    0,
		MaxConcurrentBuilds:         0,
	})
	rl := &fakeRatelimit{
		response: ratelimit.RatelimitResponse{
			Limit:     1,
			Remaining: 0,
			Reset:     time.Now().Add(time.Minute),
			Success:   false,
			Current:   1,
		},
	}
	handlerCalled := false
	sess := &zen.Session{}
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	require.NoError(t, sess.Init(httptest.NewRecorder(), req, 0))

	err = WithAuthentication(AuthenticationConfig{
		Auth:       &fakeAuth{principal: testMiddlewarePrincipal("ws_123")},
		QuotaCache: quotaCache,
		Ratelimit:  rl,
	})(func(_ context.Context, _ *zen.Session) error {
		handlerCalled = true
		return nil
	})(context.Background(), sess)

	require.Error(t, err)
	require.False(t, handlerCalled)
	require.Equal(t, 1, rl.calls)
	require.Equal(t, "ws_123", rl.request.WorkspaceID)
	require.Equal(t, "ws_123", rl.request.Identifier)
	require.Equal(t, int64(1), rl.request.Cost)
}

func testMiddlewarePrincipal(workspaceID string) *principal.Principal {
	return &principal.Principal{
		Version: principal.Version,
		Subject: principal.Subject{
			ID:   "root_key_123",
			Name: "Root Key",
			Type: principal.SubjectTypeRootKey,
		},
		Type:        principal.TypeAPIKey,
		Source:      principal.Source{Key: &principal.KeySource{KeyID: "key_123", KeySpaceID: "ks_123"}},
		WorkspaceID: workspaceID,
		Permissions: []string{"api.*.read_key"},
	}
}
