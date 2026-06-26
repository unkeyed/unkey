package openapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateSpecPath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{name: "valid simple path", path: "/openapi.json", wantErr: false},
		{name: "valid nested path", path: "/api/v1/openapi.json", wantErr: false},
		{name: "valid path with query", path: "/openapi.json?format=yaml", wantErr: false},

		// Authority-confusion SSRF payloads.
		{name: "at-sign authority confusion", path: "@attacker.com/openapi.json", wantErr: true},
		{name: "at-sign with port", path: "@127.0.0.1:8080/openapi.json", wantErr: true},

		// Scheme-based payloads.
		{name: "absolute URL with https", path: "https://attacker.com/openapi.json", wantErr: true},
		{name: "absolute URL with http", path: "http://attacker.com/openapi.json", wantErr: true},

		// Authority reference payloads.
		{name: "double-slash authority", path: "//attacker.com/openapi.json", wantErr: true},

		// Relative paths (no leading slash).
		{name: "relative path", path: "openapi.json", wantErr: true},
		{name: "empty string", path: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := validateSpecPath(tt.path)
			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, parsed)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, parsed)
		})
	}
}

func TestHTTPClientRefusesRedirects(t *testing.T) {
	internalHit := false
	internal := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		internalHit = true
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(internal.Close)

	deployment := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Redirect(w, &http.Request{}, internal.URL, http.StatusFound)
	}))
	t.Cleanup(deployment.Close)

	client := New(Config{}).httpClient

	resp, err := client.Get(deployment.URL)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })

	require.Equal(t, http.StatusFound, resp.StatusCode)
	require.False(t, internalHit, "client must not follow redirect to internal host")
}
