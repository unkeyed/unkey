package validation

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

var securitySpec = []byte(`
openapi: "3.0.0"
info:
  title: Test
  version: "1.0"
components:
  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer
paths:
  /widgets:
    get:
      security:
        - bearerAuth: []
      responses:
        "200":
          description: ok
`)

// TestValidator_FiltersOnlyIncorrectAuthorizationScheme protects the narrow
// exception in filterIgnoredSecurityErrors. OpenAPI validation normally rejects
// "Authorization: Basic ..." on a bearer-protected route, but Unkey suppresses
// that generic libopenapi error because the auth layer already returns a better
// product-specific malformed-header response. This is not an authentication
// test. It guards against broadening the filter so far that missing
// Authorization headers are ignored too, which would stop the validator from
// enforcing the route's required security scheme.
func TestValidator_FiltersOnlyIncorrectAuthorizationScheme(t *testing.T) {
	t.Parallel()

	validator, err := NewFromBytes(securitySpec)
	require.NoError(t, err)

	t.Run("wrong scheme is ignored", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/widgets", nil)
		req.Header.Set("Authorization", "Basic abc123")

		result := validator.Validate(req)
		require.Nil(t, result)
	})

	t.Run("missing header is still rejected", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/widgets", nil)

		result := validator.Validate(req)
		require.NotNil(t, result)
		require.NotEmpty(t, result.Detail)
	})
}
