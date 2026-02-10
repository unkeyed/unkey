package validation

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestPathParameters tests path parameter validation
func TestPathParameters(t *testing.T) {
	t.Parallel()
	v := newTestValidator(t)

	tests := []struct {
		name    string
		path    string
		isValid bool
	}{
		{
			name:    "valid path parameter",
			path:    "/test/params/123",
			isValid: true,
		},
		{
			name:    "valid path parameter with special chars",
			path:    "/test/params/abc-123",
			isValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			req.Header.Set("Authorization", "Bearer test_token")

			resp, valid := v.Validate(context.Background(), req)

			if tt.isValid {
				require.True(t, valid, "expected valid request, got errors: %+v", resp)
			} else {
				require.False(t, valid, "expected invalid request")
			}
		})
	}
}

// TestQueryParameters tests query parameter validation
func TestQueryParameters(t *testing.T) {
	t.Parallel()
	v := newTestValidator(t)

	tests := []struct {
		name    string
		path    string
		isValid bool
	}{
		{
			name:    "valid without optional params",
			path:    "/test/params/123",
			isValid: true,
		},
		{
			name:    "valid with limit param",
			path:    "/test/params/123?limit=10",
			isValid: true,
		},
		{
			name:    "valid with multiple tags (explode)",
			path:    "/test/params/123?tags=a&tags=b&tags=c",
			isValid: true,
		},
		{
			name:    "invalid limit below minimum",
			path:    "/test/params/123?limit=0",
			isValid: false,
		},
		{
			name:    "invalid limit above maximum",
			path:    "/test/params/123?limit=101",
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			req.Header.Set("Authorization", "Bearer test_token")

			resp, valid := v.Validate(context.Background(), req)

			if tt.isValid {
				require.True(t, valid, "expected valid request, got errors: %+v", resp)
			} else {
				require.False(t, valid, "expected invalid request")
			}
		})
	}
}
