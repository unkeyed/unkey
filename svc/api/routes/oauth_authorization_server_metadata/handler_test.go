package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/oauth"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/oauth_authorization_server_metadata"
)

// TestMetadataAdvertisesTheServer guarantees the RFC 8414 document a client
// discovers the server from: served at the well-known path, every endpoint
// under the issuer, public clients with S256 PKCE only, and cacheable.
func TestMetadataAdvertisesTheServer(t *testing.T) {
	h := testutil.NewHarness(t)
	issuer := "https://api.unkey.test"
	h.Register(&handler.Handler{Metadata: oauth.NewMetadata(issuer)}, h.OAuthMiddleware()...)

	res := testutil.CallRaw[oauth.Metadata](h, httptest.NewRequest(http.MethodGet, "/.well-known/oauth-authorization-server", nil))
	require.Equal(t, http.StatusOK, res.Status, res.RawBody)
	require.Equal(t, "public, max-age=300", res.Headers.Get("Cache-Control"))
	require.Equal(t, oauth.Metadata{
		Issuer:                                     issuer,
		AuthorizationEndpoint:                      issuer + "/oauth/authorize",
		TokenEndpoint:                              issuer + "/oauth/token",
		RegistrationEndpoint:                       issuer + "/oauth/register",
		ScopesSupported:                            []string{"api"},
		ResponseTypesSupported:                     []string{"code"},
		ResponseModesSupported:                     []string{"query"},
		GrantTypesSupported:                        []string{"authorization_code", "refresh_token"},
		TokenEndpointAuthMethodsSupported:          []string{"none"},
		CodeChallengeMethodsSupported:              []string{"S256"},
		AuthorizationResponseIssParameterSupported: true,
		ServiceDocumentation:                       "https://www.unkey.com/docs/platform/ai-coding/mcp-servers",
	}, *res.Body)
}
