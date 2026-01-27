package validation

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestEdgeCases tests various edge cases
func TestEdgeCases(t *testing.T) {
	t.Parallel()
	v := newTestValidator(t)

	t.Run("unicode in string values", func(t *testing.T) {
		t.Parallel()
		body := `{"id": "用户_123", "name": "日本語テスト"}`
		req := makeRequest(http.MethodPost, "/test/user", body, nil)
		resp, valid := v.Validate(context.Background(), req)
		require.True(t, valid, "unicode should be valid, got errors: %+v", resp)
	})

	t.Run("empty string where required", func(t *testing.T) {
		t.Parallel()
		body := `{"id": "", "name": ""}`
		req := makeRequest(http.MethodPost, "/test/user", body, nil)
		// Empty strings are technically valid unless minLength is specified
		_, valid := v.Validate(context.Background(), req)
		require.True(t, valid, "empty strings should be valid for type: string without minLength")
	})

	t.Run("whitespace-only string", func(t *testing.T) {
		t.Parallel()
		body := `{"id": "   ", "name": "   "}`
		req := makeRequest(http.MethodPost, "/test/user", body, nil)
		_, valid := v.Validate(context.Background(), req)
		require.True(t, valid, "whitespace strings should be valid for type: string without pattern")
	})

	t.Run("very large number", func(t *testing.T) {
		t.Parallel()
		body := `{"count": 99999999999999999999}`
		req := makeRequest(http.MethodPost, "/test/minmax", body, nil)
		_, valid := v.Validate(context.Background(), req)
		require.False(t, valid, "number exceeding max should be invalid")
	})

	t.Run("negative zero", func(t *testing.T) {
		t.Parallel()
		body := `{"count": -0}`
		req := makeRequest(http.MethodPost, "/test/minmax", body, nil)
		resp, valid := v.Validate(context.Background(), req)
		require.True(t, valid, "-0 should equal 0 and be valid, got errors: %+v", resp)
	})

	t.Run("floating point for integer field", func(t *testing.T) {
		t.Parallel()
		body := `[1.5, 2.5, 3.5]`
		req := makeRequest(http.MethodPost, "/test/array", body, nil)
		_, valid := v.Validate(context.Background(), req)
		require.False(t, valid, "floats should be invalid for integer array")
	})

	t.Run("null for non-nullable field", func(t *testing.T) {
		t.Parallel()
		body := `{"id": null, "name": "test"}`
		req := makeRequest(http.MethodPost, "/test/user", body, nil)
		_, valid := v.Validate(context.Background(), req)
		require.False(t, valid, "null should be invalid for non-nullable field")
	})
}
