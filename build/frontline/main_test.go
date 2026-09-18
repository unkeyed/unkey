package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommand_LocalDevDoesNotRequireProductionSettings(t *testing.T) {
	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, listener.Close()) })
	_, port, err := net.SplitHostPort(listener.Addr().String())
	require.NoError(t, err)

	err = command().Run(t.Context(), []string{"frontline", "--config", fmt.Sprintf(`
[local-dev]
http_port = %s
root_key = "root_test"
[[local-dev.routes]]
hostname = "localhost"
upstream = "localhost:3000"
`, port)})
	require.ErrorContains(t, err, "address already in use")
}

func TestCommand_RejectsUnknownLocalDevFields(t *testing.T) {
	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, listener.Close()) })
	_, port, err := net.SplitHostPort(listener.Addr().String())
	require.NoError(t, err)
	valid := fmt.Sprintf("[local-dev]\nhttp_port=%s\nroot_key='root_test'\n[[local-dev.routes]]\nhostname='localhost'\nupstream='localhost:3000'\n", port)
	for _, tc := range []struct{ name, content, field string }{
		{"mixed production config", "region='test'\n" + valid, "region"},
		{"root routes", valid + "[[routes]]\nhostname='other.localhost'\nupstream='localhost:4000'", "routes"},
		{"unknown route field", valid + "polciies=[]", "local-dev.routes.polciies"},
		{"unknown local-dev field", valid + "[local-dev.typo]\nvalue=true", "local-dev.typo"},
		{"request timeout override", strings.Replace(valid, "[local-dev]", "[local-dev]\nrequest_timeout='1s'", 1), "local-dev.request_timeout"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := command().Run(t.Context(), []string{"frontline", "--config", tc.content})
			require.ErrorContains(t, err, "unknown local-dev configuration field: "+tc.field)
		})
	}
}

func TestCommand_ValidatesOnlyTheSelectedConfig(t *testing.T) {
	for _, tc := range []struct{ name, content, message string }{
		{"empty local-dev", "[local-dev]\n", "load local-dev config"},
		{"missing root key", "[[local-dev.routes]]\nhostname='localhost'\nupstream='localhost:3000'", "RootKey"},
		{"missing routes", "[local-dev]\nroot_key='root_test'", "Routes"},
		{"negative port", "[local-dev]\nhttp_port=-1", "HTTPPort"},
		{"port out of range", "[local-dev]\nhttp_port=65536", "HTTPPort"},
		{"malformed local-dev", "local-dev=false", "decode config"},
		{"malformed TOML", "[local-dev\n", "decode config"},
		{"production still requires settings", "region='test'", "load production config"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := command().Run(t.Context(), []string{"frontline", "--config", tc.content})
			require.ErrorContains(t, err, tc.message)
		})
	}
}

func TestCommand_LocalDevAcceptsFileAndInlineConfig(t *testing.T) {
	t.Setenv("UNKEY_TEST_ROOT_KEY", "root_$UNKEY_TEST_NESTED")
	t.Setenv("UNKEY_TEST_NESTED", "must-not-expand")
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer root_$UNKEY_TEST_NESTED" {
			t.Errorf("root key was expanded more than once: %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"data":{"valid":true,"code":"VALID","keyId":"key_orders","keyspaceId":"ks_orders","identity":{"externalId":"customer_42"}}}`)
		if err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(api.Close)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.WriteString(w, r.Header.Get("X-Unkey-Principal")); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(upstream.Close)
	t.Setenv("UNKEY_TEST_UPSTREAM", strings.TrimPrefix(upstream.URL, "http://"))
	for _, source := range []string{"file", "inline"} {
		t.Run(source, func(t *testing.T) {
			directory := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(directory, "orders.json"), []byte(`{"openapi":"3.0.3","info":{"title":"Orders","version":"1"},"paths":{"/orders":{"get":{"responses":{"200":{"description":"OK"}}}}}}`), 0600))
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(t, err)
			address := listener.Addr().String()
			_, port, err := net.SplitHostPort(address)
			require.NoError(t, err)
			require.NoError(t, listener.Close())
			content := fmt.Sprintf(`
[local-dev]
http_port = %s
api_url = %q
root_key = "${UNKEY_TEST_ROOT_KEY}"
[[local-dev.routes]]
hostname = "localhost"
upstream = "${UNKEY_TEST_UPSTREAM}"
openapi_spec = "orders.json"
[[local-dev.routes.policies]]
id = "authenticate"
enabled = true
keyauth = { key_space_ids = ["ks_orders"], credits = 0 }
[[local-dev.routes.policies]]
id = "validate"
enabled = true
openapi = {}
`, port, api.URL)
			if source == "file" {
				path := filepath.Join(directory, "gateway.toml")
				require.NoError(t, os.WriteFile(path, []byte(content), 0600))
				t.Setenv("UNKEY_CONFIG", path)
			} else {
				t.Chdir(directory)
				t.Setenv("UNKEY_CONFIG", content)
			}
			ctx, cancel := context.WithCancel(t.Context())
			done := make(chan error, 1)
			go func() { done <- command().Run(ctx, []string{"frontline"}) }()
			t.Cleanup(func() {
				cancel()
				select {
				case err := <-done:
					require.NoError(t, err)
				case <-time.After(5 * time.Second):
					t.Fatal("gateway did not stop")
				}
			})
			client := &http.Client{Timeout: time.Second}
			t.Cleanup(client.CloseIdleConnections)
			require.EventuallyWithT(t, func(t *assert.CollectT) {
				req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+address+"/orders", nil)
				require.NoError(t, err)
				req.Host = "localhost:8080"
				req.Header.Set("Authorization", "Bearer user_test")
				res, err := client.Do(req)
				require.NoError(t, err)
				body, err := io.ReadAll(res.Body)
				require.NoError(t, res.Body.Close())
				require.NoError(t, err)
				require.Equal(t, http.StatusOK, res.StatusCode)
				require.Contains(t, string(body), `"subject":"customer_42"`)
			}, 5*time.Second, 10*time.Millisecond)
		})
	}
}
