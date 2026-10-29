package oauth

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestParseFormAcceptsOnlyFormBodies guarantees that the token endpoint reads
// a form whatever its media type parameters, and refuses other media types
// instead of guessing.
func TestParseFormAcceptsOnlyFormBodies(t *testing.T) {
	for _, tt := range []struct {
		name        string
		contentType string
		ok          bool
	}{
		{name: "form", contentType: "application/x-www-form-urlencoded", ok: true},
		{name: "form with charset", contentType: "application/x-www-form-urlencoded; charset=UTF-8", ok: true},
		{name: "form in upper case", contentType: "Application/X-WWW-Form-Urlencoded", ok: true},
		{name: "json", contentType: "application/json", ok: false},
		{name: "multipart", contentType: "multipart/form-data; boundary=x", ok: false},
		{name: "missing", contentType: "", ok: false},
		{name: "prefix only", contentType: "application/x-www-form-urlencoded-evil", ok: false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, TokenPath, strings.NewReader("grant_type=refresh_token"))
			req.Header.Set("Content-Type", tt.contentType)
			params, oerr := ParseForm(req)
			if !tt.ok {
				require.NotNil(t, oerr)
				require.Equal(t, InvalidRequest, oerr.Code)
				return
			}
			require.Nil(t, oerr)
			require.Equal(t, GrantRefreshToken, params.Get("grant_type"))
		})
	}
}

// TestParamsNeverResolveRepeatedValues guarantees RFC 6749 section 3.1: a
// parameter sent twice is not silently resolved to one of its values.
func TestParamsNeverResolveRepeatedValues(t *testing.T) {
	params := Params{values: url.Values{"state": {"a", "b"}, "code": {"c"}}}
	require.Empty(t, params.Get("state"))
	require.Equal(t, "c", params.Get("code"))
	require.True(t, params.Repeated())
	require.True(t, params.Repeated("state"))
	require.False(t, params.Repeated("code", "client_id"))
	require.True(t, params.Has("code"))
	require.False(t, params.Has("client_id"))

	resources := Params{values: url.Values{"resource": {"https://a.test", "https://b.test"}}}
	require.False(t, resources.Repeated(), "RFC 8707 lets resource repeat")
}

// TestScopeRules guarantees that only the advertised scope is accepted.
func TestScopeRules(t *testing.T) {
	require.Nil(t, ParseScope(""))
	require.Nil(t, ParseScope("api"))
	require.Nil(t, ParseScope(" api  api "))
	require.Equal(t, InvalidScope, ParseScope("api admin").Code)
}
