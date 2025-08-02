// nolint:exhaustruct
package zen

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/fault"
)

func TestBearer(t *testing.T) {
	tests := []struct {
		name        string
		headerValue string
		wantToken   string
		wantErr     bool
		code        codes.URN
	}{
		{
			name:        "valid bearer token",
			headerValue: "Bearer abc123xyz",
			wantToken:   "abc123xyz",
			wantErr:     false,
		},
		{
			name:        "empty authorization header",
			headerValue: "",
			wantErr:     true,
			code:        codes.Auth.Authentication.Missing.URN(),
		},
		{
			name:        "missing bearer prefix",
			headerValue: "abc123xyz",
			wantErr:     true,
			code:        codes.Auth.Authentication.Malformed.URN(),
		},
		{
			name:        "bearer with extra spaces",
			headerValue: "  Bearer   abc123xyz  ",
			wantToken:   "abc123xyz",
			wantErr:     false,
		},
		{
			name:        "empty token",
			headerValue: "Bearer ",
			wantErr:     true,
			code:        codes.Auth.Authentication.Malformed.URN(),
		},
		{
			name:        "non-bearer auth type",
			headerValue: "Basic YWxhZGRpbjpvcGVuc2VzYW1l",
			wantErr:     true,
			code:        codes.Auth.Authentication.Malformed.URN(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a request with the Authorization header
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.headerValue != "" {
				req.Header.Set("Authorization", tt.headerValue)
			}

			// Create a session
			sess := &Session{
				r: req,
			}

			// Call Bearer
			token, err := Bearer(sess)

			// Check error conditions
			if tt.wantErr {
				require.Error(t, err)

				code, ok := fault.GetCode(err)
				if tt.code != "" {
					require.True(t, ok)

					require.Equal(t, tt.code, code)
				} else {
					require.False(t, ok)
					require.Equal(t, "", code)
				}
				return
			}

			// Verify no error for positive cases
			require.NoError(t, err)
			require.Equal(t, tt.wantToken, token)
		})
	}
}

func TestBearer_Integration(t *testing.T) {
	// Test with a fully initialized session
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer token123")
	w := httptest.NewRecorder()

	sess := &Session{}
	err := sess.init(w, req)
	require.NoError(t, err)

	token, err := Bearer(sess)
	require.NoError(t, err)
	require.Equal(t, "token123", token)
}

func TestStaticAuth(t *testing.T) {
	tests := []struct {
		name          string
		headerValue   string
		expectedToken string
		wantErr       bool
		code          codes.URN
	}{
		{
			name:          "valid token matches",
			headerValue:   "Bearer secret123",
			expectedToken: "secret123",
			wantErr:       false,
		},
		{
			name:          "token mismatch",
			headerValue:   "Bearer wrong123",
			expectedToken: "secret123",
			wantErr:       true,
			code:          codes.Auth.Authentication.KeyNotFound.URN(),
		},
		{
			name:          "empty authorization header",
			headerValue:   "",
			expectedToken: "secret123",
			wantErr:       true,
			code:          codes.Auth.Authentication.Missing.URN(),
		},
		{
			name:          "malformed authorization header",
			headerValue:   "secret123",
			expectedToken: "secret123",
			wantErr:       true,
			code:          codes.Auth.Authentication.Malformed.URN(),
		},
		{
			name:          "case sensitive token comparison",
			headerValue:   "Bearer Secret123",
			expectedToken: "secret123",
			wantErr:       true,
			code:          codes.Auth.Authentication.KeyNotFound.URN(),
		},
		{
			name:          "whitespace handling",
			headerValue:   "  Bearer   secret123  ",
			expectedToken: "secret123",
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a request with the Authorization header
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			if tt.headerValue != "" {
				req.Header.Set("Authorization", tt.headerValue)
			}

			// Create a session
			sess := &Session{
				r: req,
			}

			// Call StaticAuth
			err := StaticAuth(sess, tt.expectedToken)

			// Check error conditions
			if tt.wantErr {
				require.Error(t, err)

				code, ok := fault.GetCode(err)
				if tt.code != "" {
					require.True(t, ok)
					require.Equal(t, tt.code, code)
				}
				return
			}

			// Verify no error for positive cases
			require.NoError(t, err)
		})
	}
}

func TestStaticAuth_Integration(t *testing.T) {
	// Test with a fully initialized session
	req := httptest.NewRequest(http.MethodPost, "/internal/test", nil)
	req.Header.Set("Authorization", "Bearer internal-token-123")
	w := httptest.NewRecorder()

	sess := &Session{}
	err := sess.init(w, req)
	require.NoError(t, err)

	// Should succeed with correct token
	err = StaticAuth(sess, "internal-token-123")
	require.NoError(t, err)

	// Should fail with wrong token
	err = StaticAuth(sess, "wrong-token")
	require.Error(t, err)
	code, ok := fault.GetCode(err)
	require.True(t, ok)
	require.Equal(t, codes.Auth.Authentication.KeyNotFound.URN(), code)
}
