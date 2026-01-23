package validation

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSecurityScheme_HTTP_Bearer(t *testing.T) {
	scheme := SecurityScheme{
		Type:   SecurityTypeHTTP,
		Scheme: "bearer",
	}

	// Valid bearer token
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer valid_token")
	require.True(t, validateHTTPScheme(req, scheme))

	// Missing header
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	require.False(t, validateHTTPScheme(req, scheme))

	// Wrong scheme
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Basic abc123")
	require.False(t, validateHTTPScheme(req, scheme))

	// Empty token
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer ")
	require.False(t, validateHTTPScheme(req, scheme))
}

func TestSecurityScheme_HTTP_Basic(t *testing.T) {
	scheme := SecurityScheme{
		Type:   SecurityTypeHTTP,
		Scheme: "basic",
	}

	// Valid basic auth
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	require.True(t, validateHTTPScheme(req, scheme))

	// Wrong scheme
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer token")
	require.False(t, validateHTTPScheme(req, scheme))
}

func TestSecurityScheme_APIKey_Header(t *testing.T) {
	scheme := SecurityScheme{
		Type: SecurityTypeAPIKey,
		Name: "X-API-Key",
		In:   LocationHeader,
	}

	// Valid API key in header
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-API-Key", "my-api-key")
	require.True(t, validateAPIKeyScheme(req, scheme))

	// Missing header
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	require.False(t, validateAPIKeyScheme(req, scheme))
}

func TestSecurityScheme_APIKey_Query(t *testing.T) {
	scheme := SecurityScheme{
		Type: SecurityTypeAPIKey,
		Name: "api_key",
		In:   LocationQuery,
	}

	// Valid API key in query
	req := httptest.NewRequest(http.MethodGet, "/test?api_key=my-key", nil)
	require.True(t, validateAPIKeyScheme(req, scheme))

	// Missing query param
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	require.False(t, validateAPIKeyScheme(req, scheme))
}

func TestSecurityScheme_APIKey_Cookie(t *testing.T) {
	scheme := SecurityScheme{
		Type: SecurityTypeAPIKey,
		Name: "session",
		In:   LocationCookie,
	}

	// Valid API key in cookie
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "abc123"})
	require.True(t, validateAPIKeyScheme(req, scheme))

	// Missing cookie
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	require.False(t, validateAPIKeyScheme(req, scheme))
}

func TestSecurityScheme_OAuth2(t *testing.T) {
	// OAuth2 only checks for presence of Authorization header
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer oauth_token")
	require.True(t, validateOAuth2Scheme(req))

	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	require.False(t, validateOAuth2Scheme(req))
}

func TestSecurityRequirement_ORLogic(t *testing.T) {
	schemes := map[string]SecurityScheme{
		"bearerAuth": {
			Type:   SecurityTypeHTTP,
			Scheme: "bearer",
		},
		"apiKey": {
			Type: SecurityTypeAPIKey,
			Name: "X-API-Key",
			In:   LocationHeader,
		},
	}

	// Either bearerAuth OR apiKey should work
	requirements := []SecurityRequirement{
		{Schemes: map[string][]string{"bearerAuth": {}}},
		{Schemes: map[string][]string{"apiKey": {}}},
	}

	// Request with bearer auth
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer valid_token")
	err := ValidateSecurity(req, requirements, schemes, "test-req-id")
	require.Nil(t, err, "bearer auth should satisfy requirements")

	// Request with API key
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-API-Key", "my-key")
	err = ValidateSecurity(req, requirements, schemes, "test-req-id")
	require.Nil(t, err, "API key should satisfy requirements")

	// Request with neither
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	err = ValidateSecurity(req, requirements, schemes, "test-req-id")
	require.NotNil(t, err, "no auth should fail requirements")
}

func TestSecurityRequirement_ANDLogic(t *testing.T) {
	schemes := map[string]SecurityScheme{
		"bearerAuth": {
			Type:   SecurityTypeHTTP,
			Scheme: "bearer",
		},
		"apiKey": {
			Type: SecurityTypeAPIKey,
			Name: "X-API-Key",
			In:   LocationHeader,
		},
	}

	// Both bearerAuth AND apiKey required
	requirements := []SecurityRequirement{
		{Schemes: map[string][]string{
			"bearerAuth": {},
			"apiKey":     {},
		}},
	}

	// Request with both
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer valid_token")
	req.Header.Set("X-API-Key", "my-key")
	err := ValidateSecurity(req, requirements, schemes, "test-req-id")
	require.Nil(t, err, "both auths should satisfy requirements")

	// Request with only bearer
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer valid_token")
	err = ValidateSecurity(req, requirements, schemes, "test-req-id")
	require.NotNil(t, err, "only bearer should fail AND requirements")

	// Request with only API key
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-API-Key", "my-key")
	err = ValidateSecurity(req, requirements, schemes, "test-req-id")
	require.NotNil(t, err, "only API key should fail AND requirements")
}

func TestSecurityRequirement_Empty(t *testing.T) {
	schemes := map[string]SecurityScheme{}
	requirements := []SecurityRequirement{}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	err := ValidateSecurity(req, requirements, schemes, "test-req-id")
	require.Nil(t, err, "empty security should pass")
}

func TestValidateBearerAuth_Detailed(t *testing.T) {
	tests := []struct {
		name        string
		authHeader  string
		expectError bool
		errorType   string
	}{
		{
			name:        "valid bearer token",
			authHeader:  "Bearer valid_token",
			expectError: false,
		},
		{
			name:        "missing auth header",
			authHeader:  "",
			expectError: true,
			errorType:   "https://unkey.com/docs/errors/unkey/authentication/missing",
		},
		{
			name:        "wrong scheme",
			authHeader:  "Basic abc123",
			expectError: true,
			errorType:   "https://unkey.com/docs/errors/unkey/authentication/malformed",
		},
		{
			name:        "empty bearer token",
			authHeader:  "Bearer ",
			expectError: true,
			errorType:   "https://unkey.com/docs/errors/unkey/authentication/malformed",
		},
		{
			name:        "case insensitive bearer",
			authHeader:  "BEARER valid_token",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			err := ValidateBearerAuth(req, "test-req-id")

			if tt.expectError {
				require.NotNil(t, err)
				require.Equal(t, tt.errorType, err.Error.Type)
			} else {
				require.Nil(t, err)
			}
		})
	}
}
