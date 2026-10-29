package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/grant"
	"github.com/unkeyed/unkey/svc/api/internal/grant/granttest"
	"github.com/unkeyed/unkey/svc/api/internal/oauth"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/oauth_token"
)

const gatewayIP = "198.51.100.10"

func newTokenHarness(t *testing.T) (*testutil.Harness, *grant.Service) {
	t.Helper()
	h := testutil.NewHarness(t)
	grants := granttest.New(h)
	h.Register(&handler.Handler{Grants: grants}, h.OAuthMiddleware()...)
	return h, grants
}

// tokenResult is a token endpoint response decoded as both a success and an
// RFC 6749 error body; the status says which applies.
type tokenResult struct {
	status  int
	headers http.Header
	tokens  oauth.TokenResponse
	err     oauth.Error
}

func postToken(t *testing.T, h *testutil.Harness, form url.Values, remoteIP string) tokenResult {
	t.Helper()
	return postTokenBody(t, h, "application/x-www-form-urlencoded", form.Encode(), remoteIP)
}

func postTokenBody(t *testing.T, h *testutil.Harness, contentType string, body string, remoteIP string) tokenResult {
	t.Helper()
	return decodeToken(t, sendToken(h, contentType, body, remoteIP))
}

// sendToken makes a token request without asserting, so it is safe to call
// from goroutines other than the test's.
func sendToken(h *testutil.Harness, contentType string, body string, remoteIP string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, oauth.TokenPath, strings.NewReader(body))
	req.Header.Set("Content-Type", contentType)
	req.RemoteAddr = remoteIP + ":443"
	rr := httptest.NewRecorder()
	h.Mux().ServeHTTP(rr, req)
	return rr
}

func decodeToken(t *testing.T, rr *httptest.ResponseRecorder) tokenResult {
	t.Helper()
	res := tokenResult{status: rr.Code, headers: rr.Header(), tokens: oauth.TokenResponse{}, err: oauth.Error{}}
	require.Equal(t, "no-store", rr.Header().Get("Cache-Control"), rr.Body.String())
	if rr.Code == http.StatusOK {
		require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &res.tokens))
		return res
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &res.err), rr.Body.String())
	require.NotEmpty(t, res.err.Code, rr.Body.String())
	return res
}

// approvedCode approves req as an admin of the harness workspace and returns
// the authorization code the browser would carry back.
func approvedCode(t *testing.T, h *testutil.Harness, grants *grant.Service, req granttest.OAuthRequest, permissions ...string) string {
	t.Helper()
	if len(permissions) == 0 {
		permissions = []string{"unkey:v1:" + h.Resources().UserWorkspace.ID + ":rootKeys/*#read"}
	}
	decided := granttest.MustApprove(t, grants, grant.ApproveRequest{
		Principal:   granttest.Approver(h),
		UserCode:    req.UserCode,
		Permissions: permissions,
	})
	callback, err := url.Parse(decided.RedirectTo)
	require.NoError(t, err)
	return callback.Query().Get("code")
}

func codeForm(req granttest.OAuthRequest, code string) url.Values {
	return url.Values{
		"grant_type":    {oauth.GrantAuthorizationCode},
		"code":          {code},
		"client_id":     {string(req.Client.ID)},
		"redirect_uri":  {req.RedirectURI},
		"code_verifier": {req.CodeVerifier},
	}
}

func refreshForm(clientID grant.ClientID, refreshToken string) url.Values {
	return url.Values{
		"grant_type":    {oauth.GrantRefreshToken},
		"refresh_token": {refreshToken},
		"client_id":     {string(clientID)},
	}
}
