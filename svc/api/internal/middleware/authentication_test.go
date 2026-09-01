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
	"github.com/unkeyed/unkey/pkg/batch"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/rbac"
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

// TestWithAuthentication_RecordsRootKeyUsage guarantees successful root key
// authentication creates one row and any later authorization denial updates its
// outcome without adding telemetry code to the handler.
func TestWithAuthentication_RecordsRootKeyUsage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		source         principal.Source
		handler        zen.HandleFunc
		wantKeySpaceID string
		wantOutcome    string
		wantError      bool
	}{
		{
			name:   "records usage without an authorization denial",
			source: principal.KeySource{KeyID: "key_123", KeySpaceID: "ks_123", Permissions: nil},
			handler: func(_ context.Context, _ *zen.Session) error {
				return nil
			},
			wantKeySpaceID: "ks_123",
			wantOutcome:    schema.OutcomeValid,
			wantError:      false,
		},
		{
			name:   "records a returned authorization denial",
			source: principal.KeySource{KeyID: "key_123", KeySpaceID: "ks_123", Permissions: nil},
			handler: func(_ context.Context, sess *zen.Session) error {
				p, err := sess.GetPrincipal()
				if err != nil {
					return err
				}

				return p.Authorize(rbac.T(rbac.Tuple{
					ResourceType: rbac.Api,
					ResourceID:   "*",
					Action:       rbac.CreateAPI,
				}))
			},
			wantKeySpaceID: "ks_123",
			wantOutcome:    schema.OutcomeInsufficientPermissions,
			wantError:      true,
		},
		{
			name:   "records a handled authorization denial",
			source: principal.KeySource{KeyID: "key_123", KeySpaceID: "ks_123", Permissions: nil},
			handler: func(_ context.Context, sess *zen.Session) error {
				p, err := sess.GetPrincipal()
				if err != nil {
					return err
				}

				authorizationErr := p.Authorize(rbac.T(rbac.Tuple{
					ResourceType: rbac.Api,
					ResourceID:   "*",
					Action:       rbac.CreateAPI,
				}))
				if authorizationErr == nil {
					return errors.New("expected authorization denial")
				}
				return nil
			},
			wantKeySpaceID: "ks_123",
			wantOutcome:    schema.OutcomeInsufficientPermissions,
			wantError:      false,
		},
		{
			name: "records empty key space when key metadata is absent",
			source: principal.JWTSource{
				Header:    nil,
				Payload:   nil,
				Signature: "",
			},
			handler: func(_ context.Context, _ *zen.Session) error {
				return nil
			},
			wantKeySpaceID: "",
			wantOutcome:    schema.OutcomeValid,
			wantError:      false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			flushed := make(chan []schema.KeyVerification, 1)
			verifications := batch.New(batch.Config[schema.KeyVerification]{
				Name:          "root_key_usage_test",
				Drop:          false,
				BatchSize:     1,
				BufferSize:    1,
				FlushInterval: time.Hour,
				Consumers:     1,
				Flush: func(_ context.Context, rows []schema.KeyVerification) {
					flushed <- rows
				},
			})
			t.Cleanup(verifications.Close)

			sess := &zen.Session{}
			require.NoError(t, sess.Init(
				httptest.NewRecorder(),
				httptest.NewRequest(http.MethodPost, "/v2/keys.createKey", nil),
				0,
			))

			p := testMiddlewarePrincipal("ws_123")
			p.Source = test.source
			err := WithAuthentication(AuthenticationConfig{
				Auth:             &fakeAuth{principal: p},
				KeyVerifications: verifications,
				Region:           "test-region",
			})(test.handler)(context.Background(), sess)
			if test.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			select {
			case rows := <-flushed:
				require.Len(t, rows, 1)
				verification := rows[0]
				require.Equal(t, sess.RequestID(), verification.RequestID)
				require.Positive(t, verification.Time)
				require.Equal(t, "ws_123", verification.WorkspaceID)
				require.Equal(t, test.wantKeySpaceID, verification.KeySpaceID)
				require.Equal(t, "root_key_123", verification.KeyID)
				require.Equal(t, "test-region", verification.Region)
				require.Equal(t, schema.SourceAPI, verification.Source)
				require.Equal(t, test.wantOutcome, verification.Outcome)
				require.Empty(t, verification.IdentityID)
				require.Empty(t, verification.ExternalID)
				require.Empty(t, verification.AppID)
				require.Empty(t, verification.Tags)
				require.Zero(t, verification.SpentCredits)
				require.GreaterOrEqual(t, verification.Latency, float64(0))
			case <-time.After(time.Second):
				t.Fatal("root key usage did not flush")
			}
		})
	}
}

// TestWithAuthentication_EnforcesWorkspaceRateLimit verifies workspace-level
// API policy is keyed from the authenticated principal and denies the request
// before handler execution. The limits row is preloaded into the cache so the
// test stays focused on middleware ordering and ratelimit request construction,
// without depending on a database fixture. The deferred root key row still
// reports key usage because the workspace limit does not invalidate the key.
func TestWithAuthentication_EnforcesWorkspaceRateLimit(t *testing.T) {
	t.Parallel()

	limitsCache, err := cache.New[string, keysdb.Limit](cache.Config[string, keysdb.Limit]{
		Fresh:    time.Minute,
		Stale:    time.Minute,
		MaxSize:  10,
		Resource: "test_workspace_limits",
		Clock:    clock.NewTestClock(),
	})
	require.NoError(t, err)
	limitsCache.Set(context.Background(), "ws_123", keysdb.Limit{
		Pk:                                    0,
		WorkspaceID:                           "ws_123",
		ApiBillableOperationsCountMaxPerMonth: 0,
		ApiRequestsCountMaxPerMinute:          sql.NullInt32{Int32: 1, Valid: true},
		LogsRetentionDaysMax:                  0,
		LogsAuditRetentionDaysMax:             0,
		TeamEnabled:                           false,
		CpuCoresMax:                           0,
		CpuCoresMaxPerInstance:                0,
		MemoryMibMax:                          0,
		MemoryMibMaxPerInstance:               0,
		StorageMibMax:                         0,
		StorageMibMaxPerInstance:              0,
		BuildsConcurrentMax:                   0,
		CustomDomainsMax:                      0,
		AutoscalingReplicasMax:                0,
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
	flushed := make(chan []schema.KeyVerification, 1)
	verifications := batch.New(batch.Config[schema.KeyVerification]{
		Name:          "root_key_rate_limit_test",
		Drop:          false,
		BatchSize:     1,
		BufferSize:    1,
		FlushInterval: time.Hour,
		Consumers:     1,
		Flush: func(_ context.Context, rows []schema.KeyVerification) {
			flushed <- rows
		},
	})
	t.Cleanup(verifications.Close)

	handlerCalled := false
	sess := &zen.Session{}
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	require.NoError(t, sess.Init(httptest.NewRecorder(), req, 0))

	err = WithAuthentication(AuthenticationConfig{
		Auth:             &fakeAuth{principal: testMiddlewarePrincipal("ws_123")},
		KeyVerifications: verifications,
		Region:           "test-region",
		LimitsCache:      limitsCache,
		Ratelimit:        rl,
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

	select {
	case rows := <-flushed:
		require.Len(t, rows, 1)
		require.Equal(t, schema.OutcomeValid, rows[0].Outcome)
	case <-time.After(time.Second):
		t.Fatal("root key usage did not flush")
	}
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
		Source:      principal.KeySource{KeyID: "key_123", KeySpaceID: "ks_123"},
		WorkspaceID: workspaceID,
		Permissions: []string{"api.*.read_key"},
	}
}
