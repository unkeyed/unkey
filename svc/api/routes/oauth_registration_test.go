package routes_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auth/clisession"
	tokenjwt "github.com/unkeyed/unkey/pkg/jwt"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/pkg/zen/validation"
	"github.com/unkeyed/unkey/svc/api/internal/grant"
	"github.com/unkeyed/unkey/svc/api/internal/oauth"
	"github.com/unkeyed/unkey/svc/api/routes"
)

type anyMember struct{}

func (anyMember) ActiveRoles(context.Context, string, string) ([]string, bool, error) {
	return []string{"admin"}, true, nil
}

// TestOAuthRoutesRegisterOnlyWithAnIssuer guarantees that an API without
// cli_auth.oauth_issuer exposes no OAuth endpoint, and that one with it
// serves them outside the OpenAPI validator, which would reject these
// unspecified paths with problem JSON.
func TestOAuthRoutesRegisterOnlyWithAnIssuer(t *testing.T) {
	signer, err := tokenjwt.NewHS256Signer[clisession.Claims]([]byte("test-cli-signing-secret-32-bytes!"))
	require.NoError(t, err)
	for _, tt := range []struct {
		name   string
		issuer string
		status int
	}{
		{name: "without issuer", issuer: "", status: http.StatusNotFound},
		{name: "with issuer", issuer: "https://api.unkey.test", status: http.StatusOK},
	} {
		t.Run(tt.name, func(t *testing.T) {
			validator, err := validation.New()
			require.NoError(t, err)
			srv, err := zen.New(zen.Config{MaxRequestBodySize: 1 << 20})
			require.NoError(t, err)
			routes.Register(srv, &routes.Services{
				Validator: validator,
				Grants: &grant.Service{
					Memberships:      anyMember{},
					DashboardBaseURL: "https://app.unkey.test",
					Signer:           signer,
					Issuer:           tt.issuer,
				},
			}, zen.InstanceInfo{})

			rr := httptest.NewRecorder()
			srv.Mux().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, oauth.MetadataPath, nil))
			require.Equal(t, tt.status, rr.Code, rr.Body.String())
		})
	}
}
