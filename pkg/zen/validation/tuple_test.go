package validation

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestTupleValidation tests prefixItems (tuple) validation which is supported by jsonschema/v6
func TestTupleValidation(t *testing.T) {
	t.Parallel()

	validator := newTestValidator(t)

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "valid tuple - [string, integer, boolean]",
			body:       `["hello", 42, true]`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "wrong type at index 0 - integer instead of string",
			body:       `[123, 42, true]`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "wrong type at index 1 - string instead of integer",
			body:       `["hello", "not-int", true]`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "wrong type at index 2 - string instead of boolean",
			body:       `["hello", 42, "not-bool"]`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "too few items",
			body:       `["hello", 42]`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "too many items",
			body:       `["hello", 42, true, "extra"]`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "empty array",
			body:       `[]`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "null values not allowed",
			body:       `[null, 42, true]`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := makeRequest(http.MethodPost, "/test/tuple", tt.body, nil)
			resp, ok := validator.Validate(context.Background(), req)

			if tt.wantStatus == http.StatusOK {
				require.True(t, ok, "expected validation to pass")
				require.Nil(t, resp, "expected no error response")
			} else {
				require.False(t, ok, "expected validation to fail")
				require.NotNil(t, resp, "expected error response")
			}
		})
	}
}

// TestTupleWithAdditionalItems tests prefixItems with additional items allowed
func TestTupleWithAdditionalItems(t *testing.T) {
	t.Parallel()

	validator := newTestValidator(t)

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "valid - prefix items only",
			body:       `["hello", 42]`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "valid - with additional boolean items",
			body:       `["hello", 42, true, false, true]`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid - wrong prefix type",
			body:       `[123, 42]`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid - additional item is not boolean",
			body:       `["hello", 42, "not-a-bool"]`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "valid - empty additional items",
			body:       `["hello", 42]`,
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := makeRequest(http.MethodPost, "/test/tuple-additional", tt.body, nil)
			resp, ok := validator.Validate(context.Background(), req)

			if tt.wantStatus == http.StatusOK {
				require.True(t, ok, "expected validation to pass")
				require.Nil(t, resp, "expected no error response")
			} else {
				require.False(t, ok, "expected validation to fail")
				require.NotNil(t, resp, "expected error response")
			}
		})
	}
}
