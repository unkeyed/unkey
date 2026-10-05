package keyauth_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	frontlinev1 "github.com/unkeyed/unkey/gen/proto/frontline/v1"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/frontline/internal/policies"
	"github.com/unkeyed/unkey/svc/frontline/internal/policies/keyauth"
)

func TestAPIExecutor_VerifiesPolicyAndBuildsPrincipal(t *testing.T) {
	t.Parallel()
	type request struct {
		method, path, authorization, contentType, body string
		err                                            error
	}
	requests := make(chan request, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		requests <- request{r.Method, r.URL.Path, r.Header.Get("Authorization"), r.Header.Get("Content-Type"), string(body), err}
		w.Header().Set("Content-Type", "application/json")
		_, err = io.WriteString(w, `{"data":{"valid":true,"code":"VALID","keyId":"key_orders","keyspaceId":"ks_secondary","name":"Orders","expires":1900000000123,"credits":0,"meta":{"plan":"pro"},"roles":["reader"],"permissions":["orders.read"],"identity":{"id":"id_customer","externalId":"customer_42","meta":{"org":"acme"}}}}`)
		if err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	executor, err := keyauth.NewAPI(keyauth.APIConfig{BaseURL: server.URL, RootKey: "root_test", Clock: clock.New()})
	require.NoError(t, err)
	var authenticator policies.KeyAuthenticator = executor
	req := httptest.NewRequest(http.MethodGet, "/orders?api_key=user_test", nil)
	req.Header.Set("Authorization", "Bearer ignored_key")
	sess := &zen.Session{}
	require.NoError(t, sess.Init(httptest.NewRecorder(), req, 0))
	p, err := authenticator.Execute(t.Context(), sess, req, "app_orders", &frontlinev1.KeyAuth{
		Locations: []*frontlinev1.KeyLocation{{Location: &frontlinev1.KeyLocation_QueryParam{
			QueryParam: &frontlinev1.QueryParamKeyLocation{Name: "api_key"},
		}}},
		KeySpaceIds: []string{"ks_primary", "ks_secondary"}, Credits: new(int64(0)),
		PermissionQuery: new("orders.read"),
		Ratelimits:      []*frontlinev1.KeyRatelimit{{Name: "requests", Cost: new(int64(3)), Limit: new(int64(17)), Duration: new(int64(60000))}},
	})
	require.NoError(t, err)
	got := <-requests
	require.NoError(t, got.err)
	require.Equal(t, http.MethodPost, got.method)
	require.Equal(t, "/v2/keys.verifyKey", got.path)
	require.Equal(t, "Bearer root_test", got.authorization)
	require.Equal(t, "application/json", got.contentType)
	require.JSONEq(t, `{"key":"user_test","keyspaces":["ks_primary","ks_secondary"],"credits":{"cost":0},"permissions":"orders.read","ratelimits":[{"name":"requests","cost":3,"limit":17,"duration":60000}]}`, got.body)
	encoded, err := json.Marshal(p)
	require.NoError(t, err)
	require.JSONEq(t, `{"version":"v1","subject":"customer_42","type":"API_KEY","identity":{"externalId":"customer_42","meta":{"org":"acme"}},"source":{"key":{"keyId":"key_orders","keySpaceId":"ks_secondary","name":"Orders","expiresAt":1900000000123,"credits":0,"meta":{"plan":"pro"},"roles":["reader"],"permissions":["orders.read"]}}}`, string(encoded))
}

func TestAPIExecutor_FailsClosed(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name, body string
		code       codes.URN
	}{
		{"not found", `{"data":{"valid":false,"code":"NOT_FOUND"}}`, codes.Frontline.Auth.InvalidKey.URN()},
		{"disabled", `{"data":{"valid":false,"code":"DISABLED"}}`, codes.Frontline.Auth.InvalidKey.URN()},
		{"expired", `{"data":{"valid":false,"code":"EXPIRED"}}`, codes.Frontline.Auth.InvalidKey.URN()},
		{"forbidden", `{"data":{"valid":false,"code":"FORBIDDEN"}}`, codes.Frontline.Auth.InvalidKey.URN()},
		{"permissions", `{"data":{"valid":false,"code":"INSUFFICIENT_PERMISSIONS"}}`, codes.Frontline.Auth.InsufficientPermissions.URN()},
		{"credits", `{"data":{"valid":false,"code":"USAGE_EXCEEDED"}}`, codes.Frontline.Auth.UsageExceeded.URN()},
		{"rate limit", `{"data":{"valid":false,"code":"RATE_LIMITED"}}`, codes.Frontline.Auth.RateLimited.URN()},
		{"contradictory valid", `{"data":{"valid":true,"code":"NOT_FOUND"}}`, codes.Frontline.Internal.InternalServerError.URN()},
		{"contradictory code", `{"data":{"valid":false,"code":"VALID","keyId":"key_orders","keyspaceId":"ks_orders"}}`, codes.Frontline.Internal.InternalServerError.URN()},
		{"unknown code", `{"data":{"valid":false,"code":"NEW_CODE"}}`, codes.Frontline.Internal.InternalServerError.URN()},
		{"missing key ID", `{"data":{"valid":true,"code":"VALID","keyspaceId":"ks_orders"}}`, codes.Frontline.Internal.InternalServerError.URN()},
		{"missing keyspace", `{"data":{"valid":true,"code":"VALID","keyId":"key_orders"}}`, codes.Frontline.Internal.InternalServerError.URN()},
		{"wrong keyspace", `{"data":{"valid":true,"code":"VALID","keyId":"key_orders","keyspaceId":"ks_other"}}`, codes.Frontline.Internal.InternalServerError.URN()},
		{"missing identity subject", `{"data":{"valid":true,"code":"VALID","keyId":"key_orders","keyspaceId":"ks_orders","identity":{"id":"id_customer"}}}`, codes.Frontline.Internal.InternalServerError.URN()},
		{"missing data", `{}`, codes.Frontline.Internal.InternalServerError.URN()},
		{"truncated JSON", `{"data":`, codes.Frontline.Internal.InternalServerError.URN()},
		{"trailing JSON", `{"data":{"valid":true,"code":"VALID","keyId":"key_orders","keyspaceId":"ks_orders"}} {}`, codes.Frontline.Internal.InternalServerError.URN()},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			executor := newAPIExecutor(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if _, err := io.WriteString(w, tt.body); err != nil {
					t.Error(err)
				}
			}))
			req := httptest.NewRequest(http.MethodGet, "/orders", nil)
			req.Header.Set("Authorization", "Bearer user_test")
			sess := &zen.Session{}
			require.NoError(t, sess.Init(httptest.NewRecorder(), req, 0))
			p, err := executor.Execute(t.Context(), sess, req, "app_orders", &frontlinev1.KeyAuth{KeySpaceIds: []string{"ks_orders"}})
			require.Error(t, err)
			require.Nil(t, p)
			code, ok := fault.GetCode(err)
			require.True(t, ok)
			require.Equal(t, tt.code, code)
		})
	}
}

func TestAPIExecutor_EmptyKeyspacesRemainRestricted(t *testing.T) {
	t.Parallel()
	requests := make(chan string, 1)
	executor := newAPIExecutor(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		requests <- string(body)
		if _, err := io.WriteString(w, `{"data":{"valid":false,"code":"NOT_FOUND"}}`); err != nil {
			t.Error(err)
		}
	}))
	req := httptest.NewRequest(http.MethodGet, "/orders", nil)
	req.Header.Set("Authorization", "Bearer user_test")
	sess := &zen.Session{}
	require.NoError(t, sess.Init(httptest.NewRecorder(), req, 0))
	p, err := executor.Execute(t.Context(), sess, req, "app_orders", &frontlinev1.KeyAuth{})
	require.Error(t, err)
	require.Nil(t, p)
	require.JSONEq(t, `{"key":"user_test","keyspaces":[],"credits":{"cost":1}}`, <-requests)
}

func TestAPIExecutor_PrincipalDefaults(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct{ name, identity, want string }{
		{"unlinked", "", `{"version":"v1","subject":"key_orders","type":"API_KEY","source":{"key":{"keyId":"key_orders","keySpaceId":"ks_orders","meta":{}}}}`},
		{"linked", `,"identity":{"id":"id_customer","externalId":"customer_42"}`, `{"version":"v1","subject":"customer_42","type":"API_KEY","identity":{"externalId":"customer_42","meta":{}},"source":{"key":{"keyId":"key_orders","keySpaceId":"ks_orders","meta":{}}}}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			executor := newAPIExecutor(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if _, err := io.WriteString(w, `{"data":{"valid":true,"code":"VALID","keyId":"key_orders","keyspaceId":"ks_orders"`+tt.identity+`}}`); err != nil {
					t.Error(err)
				}
			}))
			req := httptest.NewRequest(http.MethodGet, "/orders", nil)
			req.Header.Set("Authorization", "Bearer user_test")
			sess := &zen.Session{}
			require.NoError(t, sess.Init(httptest.NewRecorder(), req, 0))
			p, err := executor.Execute(t.Context(), sess, req, "app_orders", &frontlinev1.KeyAuth{KeySpaceIds: []string{"ks_orders"}})
			require.NoError(t, err)
			encoded, err := p.Marshal()
			require.NoError(t, err)
			require.JSONEq(t, tt.want, encoded)
		})
	}
}

func TestAPIExecutor_RejectsHTTPFailuresWithoutRetrying(t *testing.T) {
	t.Parallel()
	valid := `{"data":{"valid":true,"code":"VALID","keyId":"key_orders","keyspaceId":"ks_orders"}}`
	for _, tt := range []struct {
		name   string
		status int
		body   string
	}{
		{"redirect", http.StatusFound, valid},
		{"replay redirect", http.StatusTemporaryRedirect, valid},
		{"permanent replay redirect", http.StatusPermanentRedirect, valid},
		{"root key rejected", http.StatusUnauthorized, valid},
		{"root key forbidden", http.StatusForbidden, valid},
		{"workspace throttled", http.StatusTooManyRequests, valid},
		{"API unavailable", http.StatusServiceUnavailable, valid},
		{"oversized body", http.StatusOK, strings.Repeat(" ", 1<<20) + valid},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var requests atomic.Int64
			executor := newAPIExecutor(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				w.Header().Set("Location", "/redirected")
				w.Header().Set("X-RateLimit-Remaining", "999")
				w.WriteHeader(tt.status)
				// The client can close the connection after rejecting the response headers or size.
				if _, err := io.WriteString(w, tt.body); err != nil {
					return
				}
			}))
			req := httptest.NewRequest(http.MethodGet, "/orders", nil)
			req.Header.Set("Authorization", "Bearer user_test")
			w := httptest.NewRecorder()
			sess := &zen.Session{}
			require.NoError(t, sess.Init(w, req, 0))
			p, err := executor.Execute(t.Context(), sess, req, "app_orders", &frontlinev1.KeyAuth{KeySpaceIds: []string{"ks_orders"}})
			require.Error(t, err)
			require.Nil(t, p)
			code, ok := fault.GetCode(err)
			require.True(t, ok)
			require.Equal(t, codes.Frontline.Internal.InternalServerError.URN(), code)
			require.Empty(t, w.Header().Get("X-RateLimit-Remaining"))
			require.EqualValues(t, 1, requests.Load(), "verification can consume quotas and must not be replayed")
		})
	}
}

func TestAPIExecutor_RejectsMissingCredentialsAndInvalidPolicyLocally(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name, authorization string
		policy              *frontlinev1.KeyAuth
		code                codes.URN
	}{
		{"missing key", "", &frontlinev1.KeyAuth{}, codes.Frontline.Auth.MissingCredentials.URN()},
		{"negative credits", "Bearer user_test", &frontlinev1.KeyAuth{Credits: new(int64(-1))}, codes.Frontline.Internal.InvalidConfiguration.URN()},
		{"invalid permissions", "Bearer user_test", &frontlinev1.KeyAuth{PermissionQuery: new("orders.read AND (")}, codes.Frontline.Internal.InvalidConfiguration.URN()},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			executor := newAPIExecutor(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Error("invalid local input reached the verification API")
				w.WriteHeader(http.StatusBadRequest)
			}))
			req := httptest.NewRequest(http.MethodGet, "/orders", nil)
			req.Header.Set("Authorization", tt.authorization)
			sess := &zen.Session{}
			require.NoError(t, sess.Init(httptest.NewRecorder(), req, 0))
			p, err := executor.Execute(t.Context(), sess, req, "app_orders", tt.policy)
			require.Error(t, err)
			require.Nil(t, p)
			code, ok := fault.GetCode(err)
			require.True(t, ok)
			require.Equal(t, tt.code, code)
		})
	}
}

func TestAPIExecutor_RateLimitHeaders(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name, results, limit, remaining, reset, retry string
		valid                                         bool
	}{
		{"lowest remaining", `[{"limit":70,"remaining":9,"reset":1700000004000,"exceeded":false},{"limit":11,"remaining":2,"reset":1700000002124,"exceeded":false}]`, "11", "2", "1700000002", "", true},
		{"denial beats lower remaining", `[{"limit":70,"remaining":0,"reset":1700000004000,"exceeded":false},{"limit":11,"remaining":2,"reset":1700000002124,"exceeded":true}]`, "11", "2", "1700000002", "3", false},
		{"minimum retry", `[{"limit":11,"remaining":0,"reset":1700000000000,"exceeded":true}]`, "11", "0", "1700000000", "1", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			status := "RATE_LIMITED"
			if tt.valid {
				status = "VALID"
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if _, err := fmt.Fprintf(w, `{"data":{"valid":%t,"code":%q,"keyId":"key_orders","keyspaceId":"ks_orders","ratelimits":%s}}`, tt.valid, status, tt.results); err != nil {
					t.Error(err)
				}
			}))
			t.Cleanup(server.Close)
			executor, err := keyauth.NewAPI(keyauth.APIConfig{BaseURL: server.URL, RootKey: "root_test", Clock: clock.NewTestClock(time.UnixMilli(1700000000123))})
			require.NoError(t, err)
			req := httptest.NewRequest(http.MethodGet, "/orders", nil)
			req.Header.Set("Authorization", "Bearer user_test")
			w := httptest.NewRecorder()
			sess := &zen.Session{}
			require.NoError(t, sess.Init(w, req, 0))
			p, err := executor.Execute(t.Context(), sess, req, "app_orders", &frontlinev1.KeyAuth{KeySpaceIds: []string{"ks_orders"}})
			if tt.valid {
				require.NoError(t, err)
				require.NotNil(t, p)
			} else {
				require.Error(t, err)
				require.Nil(t, p)
			}
			require.Equal(t, tt.limit, w.Header().Get("X-RateLimit-Limit"))
			require.Equal(t, tt.remaining, w.Header().Get("X-RateLimit-Remaining"))
			require.Equal(t, tt.reset, w.Header().Get("X-RateLimit-Reset"))
			require.Equal(t, tt.retry, w.Header().Get("Retry-After"))
		})
	}
}

func TestNewAPI_RejectsInvalidConfiguration(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name, baseURL, rootKey, errorMessage string
		clock                                clock.Clock
	}{
		{"malformed URL", "://", "root_test", "verification API requires a valid base URL", clock.New()},
		{"relative URL", "/api", "root_test", "verification API base URL requires a host", clock.New()},
		{"missing host", "https:///api", "root_test", "verification API base URL requires a host", clock.New()},
		{"unsupported scheme", "ftp://example.com", "root_test", "verification API base URL requires http or https", clock.New()},
		{"URL credentials", "https://user:password@example.com", "root_test", "verification API base URL must not carry credentials", clock.New()},
		{"URL query", "https://example.com?token=secret", "root_test", "verification API base URL must not carry a query", clock.New()},
		{"URL fragment", "https://example.com#fragment", "root_test", "verification API base URL must not carry a fragment", clock.New()},
		{"missing root key", "https://example.com", "", "verification API requires a root key", clock.New()},
		{"invalid root key", "https://example.com", "root\r\nHeader:value", "root key must not contain newlines", clock.New()},
		{"missing clock", "https://example.com", "root_test", "clock is required", nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			executor, err := keyauth.NewAPI(keyauth.APIConfig{BaseURL: tt.baseURL, RootKey: tt.rootKey, Clock: tt.clock})
			require.EqualError(t, err, tt.errorMessage)
			require.Nil(t, executor)
		})
	}
}

func TestAPIExecutor_DoesNotReplayInterruptedVerification(t *testing.T) {
	t.Parallel()
	for _, cancelRequest := range []bool{false, true} {
		t.Run(fmt.Sprintf("canceled=%t", cancelRequest), func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithCancel(t.Context())
			t.Cleanup(cancel)
			var requests atomic.Int64
			executor := newAPIExecutor(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if cancelRequest {
					cancel()
					return
				}
				connection, _, err := http.NewResponseController(w).Hijack()
				if err != nil {
					t.Error(err)
					return
				}
				if err := connection.Close(); err != nil {
					t.Error(err)
				}
			}))
			req := httptest.NewRequest(http.MethodGet, "/orders", nil)
			req.Header.Set("Authorization", "Bearer user_test")
			sess := &zen.Session{}
			require.NoError(t, sess.Init(httptest.NewRecorder(), req, 0))
			p, err := executor.Execute(ctx, sess, req, "app_orders", &frontlinev1.KeyAuth{KeySpaceIds: []string{"ks_orders"}})
			require.Error(t, err)
			require.Nil(t, p)
			if cancelRequest {
				require.ErrorIs(t, err, context.Canceled)
			}
			require.EqualValues(t, 1, requests.Load())
		})
	}
}

func newAPIExecutor(t *testing.T, handler http.Handler) *keyauth.APIExecutor {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	executor, err := keyauth.NewAPI(keyauth.APIConfig{BaseURL: server.URL, RootKey: "root_test", Clock: clock.New()})
	require.NoError(t, err)
	return executor
}
