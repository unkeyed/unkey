package frontline_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/config"
	"github.com/unkeyed/unkey/svc/frontline"
)

func TestLocalDevConfig_Defaults(t *testing.T) {
	t.Parallel()
	cfg, err := config.LoadBytes[frontline.LocalDevConfig]([]byte(`
root_key = "root_test"
[[routes]]
hostname = "localhost"
upstream = "localhost:3000"
`))
	require.NoError(t, err)
	require.Equal(t, 8080, cfg.HTTPPort)
	require.Equal(t, "https://api.unkey.com", cfg.APIBaseURL)
}

func TestServeLocalDev_VerifiesKeyAndForwardsPrincipal(t *testing.T) {
	t.Parallel()

	type observedRequest struct {
		method, path, authorization, principal, body string
		err                                          error
	}
	verified := make(chan observedRequest, 1)
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		verified <- observedRequest{r.Method, r.URL.Path, r.Header.Get("Authorization"), "", string(body), err}
		w.Header().Set("Content-Type", "application/json")
		if _, err := io.WriteString(w, `{"data":{"valid":true,"code":"VALID","keyId":"key_orders","keyspaceId":"ks_orders","permissions":["orders.write"],"identity":{"externalId":"customer_42","meta":{"plan":"pro"}}}}`); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(api.Close)
	forwarded := make(chan observedRequest, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		forwarded <- observedRequest{r.Method, r.URL.RequestURI(), r.Header.Get("Authorization"), r.Header.Get("X-Unkey-Principal"), string(body), err}
		w.WriteHeader(http.StatusCreated)
		if _, err := io.WriteString(w, "created"); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(upstream.Close)
	path := writeLocalDevConfig(t, fmt.Sprintf(`
[[local-dev.routes]]
hostname = "api.localhost"
upstream = %q
[[local-dev.routes.policies]]
id = "authenticate"
enabled = true
keyauth = { key_space_ids = ["ks_orders"], permission_query = "orders.write", credits = 0 }
`, strings.TrimPrefix(upstream.URL, "http://")))
	address := startLocalDev(t, loadLocalDev(t, path, api.URL, "root_test"))
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, address+"/orders?view=full", strings.NewReader(`{"quantity":3}`))
	require.NoError(t, err)
	req.Host = "api.localhost:8080"
	req.Header.Set("Authorization", "Bearer user_test")
	req.Header.Set("X-Unkey-Principal", `{"subject":"forged"}`)
	response, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	body, err := io.ReadAll(response.Body)
	require.NoError(t, response.Body.Close())
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, response.StatusCode)
	require.Equal(t, "created", string(body))

	verification := <-verified
	require.NoError(t, verification.err)
	require.Equal(t, http.MethodPost, verification.method)
	require.Equal(t, "/v2/keys.verifyKey", verification.path)
	require.Equal(t, "Bearer root_test", verification.authorization)
	require.JSONEq(t, `{"key":"user_test","keyspaces":["ks_orders"],"permissions":"orders.write","credits":{"cost":0}}`, verification.body)
	request := <-forwarded
	require.NoError(t, request.err)
	require.Equal(t, http.MethodPost, request.method)
	require.Equal(t, "/orders?view=full", request.path)
	require.Equal(t, "Bearer user_test", request.authorization)
	require.Equal(t, `{"quantity":3}`, request.body)
	require.JSONEq(t, `{"version":"v1","type":"API_KEY","subject":"customer_42","identity":{"externalId":"customer_42","meta":{"plan":"pro"}},"source":{"key":{"keyId":"key_orders","keySpaceId":"ks_orders","permissions":["orders.write"],"meta":{}}}}`, request.principal)
}

func TestServeLocalDev_RejectsRequestsWhenAuthenticationFails(t *testing.T) {
	t.Parallel()

	var forwarded atomic.Int64
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		forwarded.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(upstream.Close)
	path := writeLocalDevConfig(t, fmt.Sprintf(`
[[local-dev.routes]]
hostname = "api.localhost"
upstream = %q
[[local-dev.routes.policies]]
id = "authenticate"
enabled = true
keyauth = { key_space_ids = ["ks_orders"] }
`, strings.TrimPrefix(upstream.URL, "http://")))
	for _, tc := range []struct {
		name, key, response string
		apiStatus, status   int
	}{
		{"missing key", "", "", http.StatusOK, http.StatusUnauthorized},
		{"invalid key", "invalid", `{"data":{"valid":false,"code":"NOT_FOUND"}}`, http.StatusOK, http.StatusUnauthorized},
		{"API unavailable", "user_test", "", http.StatusServiceUnavailable, http.StatusInternalServerError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var verified atomic.Int64
			api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				verified.Add(1)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.apiStatus)
				if _, err := io.WriteString(w, tc.response); err != nil {
					t.Error(err)
				}
			}))
			t.Cleanup(api.Close)
			address := startLocalDev(t, loadLocalDev(t, path, api.URL, "root_test"))
			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, address+"/orders", nil)
			require.NoError(t, err)
			req.Host = "api.localhost"
			if tc.key != "" {
				req.Header.Set("Authorization", "Bearer "+tc.key)
			}
			req.Header.Set("X-Unkey-Principal", `{"subject":"forged"}`)
			response, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			require.NoError(t, response.Body.Close())
			require.Equal(t, tc.status, response.StatusCode)
			if tc.key == "" {
				require.Zero(t, verified.Load())
			} else {
				require.Equal(t, int64(1), verified.Load())
			}
		})
	}
	require.Zero(t, forwarded.Load())
}

func TestServeLocalDev_EnforcesLocalRateLimit(t *testing.T) {
	t.Parallel()

	var calls atomic.Int64
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(upstream.Close)
	path := writeLocalDevConfig(t, fmt.Sprintf(`
[[local-dev.routes]]
hostname = "localhost"
upstream = %q
[[local-dev.routes.policies]]
id = "per-client"
enabled = true
ratelimit = { limit = 1, window_ms = 86400000, identifiers = [{ header = { name = "X-Client-ID" } }] }
`, strings.TrimPrefix(upstream.URL, "http://")))
	address := startLocalDev(t, loadLocalDev(t, path, "http://127.0.0.1:1", "unused"))
	for _, step := range []struct {
		client string
		status int
	}{
		{"alice", http.StatusNoContent},
		{"alice", http.StatusTooManyRequests},
		{"bob", http.StatusNoContent},
	} {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, address+"/orders", nil)
		require.NoError(t, err)
		req.Host = "localhost:8080"
		req.Header.Set("X-Client-ID", step.client)
		response, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.NoError(t, response.Body.Close())
		require.Equal(t, step.status, response.StatusCode)
		require.Equal(t, "1", response.Header.Get("X-RateLimit-Limit"))
		require.Equal(t, "0", response.Header.Get("X-RateLimit-Remaining"))
		if step.status == http.StatusTooManyRequests {
			require.NotEmpty(t, response.Header.Get("Retry-After"))
		}
	}
	require.Equal(t, int64(2), calls.Load())
}

func TestServeLocalDev_StripsUntrustedHeadersOnPublicRoutes(t *testing.T) {
	t.Parallel()

	var calls atomic.Int64
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if err := json.NewEncoder(w).Encode(map[string]string{
			"host": r.Host, "principal": r.Header.Get("X-Unkey-Principal"),
			"metadata": r.Header.Get("X-Unkey-Frontline-Meta"),
			"scheme":   r.Header.Get("X-Forwarded-Proto"), "clientIP": r.Header.Get("X-Forwarded-For"),
		}); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(upstream.Close)
	path := writeLocalDevConfig(t, fmt.Sprintf("[[local-dev.routes]]\nhostname='public.localhost'\nupstream=%q", strings.TrimPrefix(upstream.URL, "http://")))
	address := startLocalDev(t, loadLocalDev(t, path, "http://127.0.0.1:1", "unused"))
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, address+"/public", nil)
	require.NoError(t, err)
	req.Host = "public.localhost:8080"
	req.Header.Set("X-Unkey-Principal", `{"subject":"forged"}`)
	req.Header.Set("X-Unkey-Frontline-Meta", "forged")
	req.Header.Set("X-Forwarded-For", "203.0.113.9")
	req.Header.Set("X-Forwarded-Proto", "https")
	response, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	body, err := io.ReadAll(response.Body)
	require.NoError(t, response.Body.Close())
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.JSONEq(t, `{"host":"public.localhost:8080","principal":"","metadata":"","scheme":"http","clientIP":"127.0.0.1"}`, string(body))

	req.Host = "unknown.localhost:8080"
	response, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	require.Equal(t, http.StatusNotFound, response.StatusCode)
	require.Equal(t, int64(1), calls.Load())
}

func TestServeLocalDev_ValidatesOpenAPIAndPreservesBody(t *testing.T) {
	t.Parallel()

	var calls atomic.Int64
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusCreated)
		if _, err := io.Copy(w, r.Body); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(upstream.Close)
	path := writeLocalDevConfig(t, fmt.Sprintf(`
[[local-dev.routes]]
hostname = "api.localhost"
upstream = %q
openapi_spec = "orders.json"
[[local-dev.routes.policies]]
id = "validate"
enabled = true
openapi = {}
`, strings.TrimPrefix(upstream.URL, "http://")))
	spec := `{
  "openapi": "3.0.3",
  "info": {"title": "Orders", "version": "1"},
  "paths": {
    "/orders": {
      "post": {
        "requestBody": {
          "required": true,
          "content": {"application/json": {"schema": {
            "type": "object", "required": ["quantity"],
            "properties": {"quantity": {"type": "integer", "minimum": 1}}
          }}}
        },
        "responses": {"201": {"description": "Created"}}
      }
    }
  }
}`
	require.NoError(t, os.WriteFile(filepath.Join(filepath.Dir(path), "orders.json"), []byte(spec), 0600))
	address := startLocalDev(t, loadLocalDev(t, path, "http://127.0.0.1:1", "unused"))
	for _, step := range []struct {
		body   string
		status int
	}{
		{`{"quantity":0}`, http.StatusBadRequest},
		{`{"quantity":3}`, http.StatusCreated},
	} {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, address+"/orders", strings.NewReader(step.body))
		require.NoError(t, err)
		req.Host = "api.localhost"
		req.Header.Set("Content-Type", "application/json")
		response, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		body, err := io.ReadAll(response.Body)
		require.NoError(t, response.Body.Close())
		require.NoError(t, err)
		require.Equal(t, step.status, response.StatusCode)
		if step.status == http.StatusCreated {
			require.Equal(t, step.body, string(body))
		}
	}
	require.Equal(t, int64(1), calls.Load())
}

func TestServeLocalDev_ClosesListenerOnStartupFailure(t *testing.T) {
	t.Parallel()

	path := writeLocalDevConfig(t, "[[local-dev.routes]]\nhostname='api.localhost'\nupstream='127.0.0.1:9000'")
	for _, tc := range []struct {
		name, errorMessage string
		config             frontline.LocalDevConfig
	}{
		{"missing routes", "load local-dev routes", frontline.LocalDevConfig{}},
		{"missing root key", "configure key verification", loadLocalDev(t, path, "https://api.unkey.com", "")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(t, err)
			err = frontline.ServeLocalDev(t.Context(), listener, tc.config)
			require.ErrorContains(t, err, tc.errorMessage)
			require.ErrorIs(t, listener.Close(), net.ErrClosed)
		})
	}
}

func loadLocalDev(t *testing.T, path, apiURL, rootKey string) frontline.LocalDevConfig {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	cfg, err := config.LoadBytes[struct {
		LocalDev frontline.LocalDevConfig `toml:"local-dev"`
	}](append([]byte("[local-dev]\nroot_key='root_test'\n"), data...))
	require.NoError(t, err)
	cfg.LocalDev.ConfigDirectory = filepath.Dir(path)
	cfg.LocalDev.APIBaseURL = apiURL
	cfg.LocalDev.RootKey = rootKey
	return cfg.LocalDev
}

func writeLocalDevConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "gateway.toml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))
	return path
}

func startLocalDev(t *testing.T, cfg frontline.LocalDevConfig) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- frontline.ServeLocalDev(ctx, listener, cfg) }()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-done:
			require.NoError(t, err)
		case <-time.After(5 * time.Second):
			t.Fatal("local-dev server did not stop")
		}
	})
	return "http://" + listener.Addr().String()
}
