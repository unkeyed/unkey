// nolint:exhaustruct
package zen

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/fault"
)

func TestBearer(t *testing.T) {
	tests := []struct {
		name        string
		headerValue string
		wantToken   string
		wantErr     bool
		errTag      fault.Tag
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
			errTag:      fault.UNAUTHORIZED,
		},
		{
			name:        "missing bearer prefix",
			headerValue: "abc123xyz",
			wantErr:     true,
			errTag:      fault.UNAUTHORIZED,
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
			errTag:      fault.UNAUTHORIZED,
		},
		{
			name:        "non-bearer auth type",
			headerValue: "Basic YWxhZGRpbjpvcGVuc2VzYW1l",
			wantErr:     true,
			errTag:      fault.UNAUTHORIZED,
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
				if tt.errTag != "" {
					assert.Equal(t, tt.errTag, fault.GetTag(err))
				}
				return
			}

			// Verify no error for positive cases
			require.NoError(t, err)
			assert.Equal(t, tt.wantToken, token)
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
	assert.Equal(t, "token123", token)
}
