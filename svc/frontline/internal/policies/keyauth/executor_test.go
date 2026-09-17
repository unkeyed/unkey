package keyauth_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	frontlinev1 "github.com/unkeyed/unkey/gen/proto/frontline/v1"
	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/internal/services/ratelimit"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
	"github.com/unkeyed/unkey/svc/frontline/internal/policies/keyauth"
	"github.com/unkeyed/unkey/svc/frontline/internal/policies/principal"
)

type verifierFunc func(context.Context, *zen.Session, keyauth.VerifyRequest) (keyauth.VerifyResult, error)

func (f verifierFunc) Verify(ctx context.Context, sess *zen.Session, req keyauth.VerifyRequest) (keyauth.VerifyResult, error) {
	return f(ctx, sess, req)
}

func TestExecutor_UsesVerificationBackend(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-API-Key", "test_credential")
	w := httptest.NewRecorder()
	sess := &zen.Session{}
	require.NoError(t, sess.Init(w, req, 0))

	wantPrincipal := &principal.Principal{Subject: "external_customer"}
	var received keyauth.VerifyRequest
	backend := verifierFunc(func(_ context.Context, _ *zen.Session, req keyauth.VerifyRequest) (keyauth.VerifyResult, error) {
		received = req
		return keyauth.VerifyResult{
			Status:    keys.StatusValid,
			Principal: wantPrincipal,
			Ratelimits: map[string]keys.RatelimitConfigAndResult{
				"requests": {Response: &ratelimit.RatelimitResponse{
					Success: true, Limit: 12, Remaining: 7, Reset: time.Unix(1_800_000_000, 0),
				}},
			},
		}, nil
	})

	executor := keyauth.New(backend, clock.New())
	result, err := executor.Execute(t.Context(), sess, req, "app_test", &frontlinev1.KeyAuth{
		KeySpaceIds:     []string{"ks_first", "ks_second"},
		Credits:         ptr.P(int64(0)),
		PermissionQuery: ptr.P("documents.read"),
		Locations: []*frontlinev1.KeyLocation{{
			Location: &frontlinev1.KeyLocation_Header{Header: &frontlinev1.HeaderKeyLocation{Name: "X-API-Key"}},
		}},
		Ratelimits: []*frontlinev1.KeyRatelimit{{Name: "requests", Limit: ptr.P(int64(12)), Duration: ptr.P(int64(60_000)), Cost: ptr.P(int64(5))}},
	})
	require.NoError(t, err)
	require.Equal(t, wantPrincipal, result)
	require.Equal(t, keyauth.VerifyRequest{
		RawKey: "test_credential", AppID: "app_test", Keyspaces: []string{"ks_first", "ks_second"}, Credits: 0,
		PermissionQuery: "documents.read",
		Ratelimits:      []openapi.KeysVerifyKeyRatelimit{{Name: "requests", Limit: ptr.P(12), Duration: ptr.P(60_000), Cost: ptr.P(5)}},
	}, received)
	require.Equal(t, "12", w.Header().Get("X-RateLimit-Limit"))
	require.Equal(t, "7", w.Header().Get("X-RateLimit-Remaining"))
	require.Equal(t, "1800000000", w.Header().Get("X-RateLimit-Reset"))
}

func TestExecutor_VerificationFailuresPreserveHeaders(t *testing.T) {
	t.Parallel()

	backendErr := fault.New("principal construction failed", fault.Code(codes.Frontline.Internal.InternalServerError.URN()))
	for _, tt := range []struct {
		name   string
		status keys.KeyStatus
		err    error
		code   codes.URN
	}{
		{name: "invalid key", status: keys.StatusNotFound, code: codes.Frontline.Auth.InvalidKey.URN()},
		{name: "permissions", status: keys.StatusInsufficientPermissions, code: codes.Frontline.Auth.InsufficientPermissions.URN()},
		{name: "credits", status: keys.StatusUsageExceeded, code: codes.Frontline.Auth.UsageExceeded.URN()},
		{name: "rate limit", status: keys.StatusRateLimited, code: codes.Frontline.Auth.RateLimited.URN()},
		{name: "error after enforcement", status: keys.StatusValid, err: backendErr, code: codes.Frontline.Internal.InternalServerError.URN()},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			clk := clock.NewTestClock(time.Unix(1_800_000_000, 0))
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Bearer test_credential")
			w := httptest.NewRecorder()
			sess := &zen.Session{}
			require.NoError(t, sess.Init(w, req, 0))
			backend := verifierFunc(func(_ context.Context, _ *zen.Session, req keyauth.VerifyRequest) (keyauth.VerifyResult, error) {
				require.Equal(t, int64(1), req.Credits)
				return keyauth.VerifyResult{
					Status: tt.status,
					Ratelimits: map[string]keys.RatelimitConfigAndResult{
						"requests": {Response: &ratelimit.RatelimitResponse{
							Success: tt.status != keys.StatusRateLimited, Limit: 9, Remaining: 0, Reset: clk.Now().Add(5 * time.Second),
						}},
					},
				}, tt.err
			})

			result, err := keyauth.New(backend, clk).Execute(t.Context(), sess, req, "app_test", &frontlinev1.KeyAuth{})
			require.Error(t, err)
			require.Nil(t, result)
			code, ok := fault.GetCode(err)
			require.True(t, ok)
			require.Equal(t, tt.code, code)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
			}
			require.Equal(t, "9", w.Header().Get("X-RateLimit-Limit"))
			require.Equal(t, "0", w.Header().Get("X-RateLimit-Remaining"))
			require.Equal(t, "1800000005", w.Header().Get("X-RateLimit-Reset"))
			if tt.status == keys.StatusRateLimited {
				require.Equal(t, "5", w.Header().Get("Retry-After"))
			} else {
				require.Empty(t, w.Header().Get("Retry-After"))
			}
		})
	}
}

func TestExecutor_RejectsInvalidBackendResult(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name   string
		result keyauth.VerifyResult
	}{
		{name: "empty result"},
		{name: "success without principal", result: keyauth.VerifyResult{Status: keys.StatusValid}},
		{name: "unknown status with principal", result: keyauth.VerifyResult{Status: "UNKNOWN", Principal: &principal.Principal{Subject: "customer"}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Bearer test_credential")
			sess := &zen.Session{}
			require.NoError(t, sess.Init(httptest.NewRecorder(), req, 0))
			backend := verifierFunc(func(context.Context, *zen.Session, keyauth.VerifyRequest) (keyauth.VerifyResult, error) {
				return tt.result, nil
			})

			result, err := keyauth.New(backend, clock.New()).Execute(t.Context(), sess, req, "app_test", &frontlinev1.KeyAuth{})
			require.Error(t, err)
			require.Nil(t, result)
			code, ok := fault.GetCode(err)
			require.True(t, ok)
			require.Equal(t, codes.Frontline.Internal.InternalServerError.URN(), code)
		})
	}
}
