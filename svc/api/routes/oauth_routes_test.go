package routes_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/pkg/zen/validation"
	"github.com/unkeyed/unkey/svc/api/internal/clidevice/clidevicetest"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/routes"
	"github.com/unkeyed/unkey/svc/api/routes/oauth"
)

// TestOAuthRoutesAnswerInProtocolShapes guarantees that the production stack
// serves the OAuth endpoints without OpenAPI validation, and only when OAuth
// is configured.
func TestOAuthRoutesAnswerInProtocolShapes(t *testing.T) {
	validator, err := validation.New()
	require.NoError(t, err)
	h := testutil.NewHarness(t)

	enabled, err := zen.New(zen.Config{})
	require.NoError(t, err)
	routes.Register(enabled, &routes.Services{Validator: validator, CLIDevice: clidevicetest.NewOAuth(h, clidevicetest.ClientDocuments{})}, zen.InstanceInfo{})

	rr := httptest.NewRecorder()
	enabled.Mux().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/.well-known/oauth-authorization-server", nil))
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	var metadata oauth.Metadata
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &metadata))
	require.Equal(t, clidevicetest.Issuer, metadata.Issuer)

	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(url.Values{"grant_type": {"password"}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr = httptest.NewRecorder()
	enabled.Mux().ServeHTTP(rr, req)
	require.Equal(t, http.StatusBadRequest, rr.Code, rr.Body.String())
	var oauthErr oauth.ErrorResponse
	decoder := json.NewDecoder(bytes.NewReader(rr.Body.Bytes()))
	decoder.DisallowUnknownFields()
	require.NoError(t, decoder.Decode(&oauthErr))
	require.Equal(t, "unsupported_grant_type", oauthErr.Error)

	disabled, err := zen.New(zen.Config{})
	require.NoError(t, err)
	routes.Register(disabled, &routes.Services{Validator: validator, CLIDevice: clidevicetest.New(h)}, zen.InstanceInfo{})
	for _, path := range []string{"/.well-known/oauth-authorization-server", "/oauth/authorize"} {
		rr = httptest.NewRecorder()
		disabled.Mux().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, path, nil))
		require.Equal(t, http.StatusNotFound, rr.Code, path)
	}
}
