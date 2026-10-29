package handler_test

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/grant"
	"github.com/unkeyed/unkey/svc/api/internal/grant/granttest"
	"github.com/unkeyed/unkey/svc/api/internal/oauth"
)

// TestTokenRequestsMustBeForms guarantees RFC 6749 section 4.1.3: the token
// endpoint reads only form bodies, accepts a charset parameter, and answers
// anything else with an RFC error body rather than problem JSON.
func TestTokenRequestsMustBeForms(t *testing.T) {
	h, _ := newTokenHarness(t)
	body, err := json.Marshal(map[string]string{"grant_type": oauth.GrantAuthorizationCode})
	require.NoError(t, err)

	res := postTokenBody(t, h, "application/json", string(body), gatewayIP)
	require.Equal(t, http.StatusBadRequest, res.status)
	require.Equal(t, oauth.InvalidRequest, res.err.Code)

	withCharset := url.Values{"grant_type": {oauth.GrantAuthorizationCode}, "code": {uid.Secure(32)}}
	res = postTokenBody(t, h, "application/x-www-form-urlencoded; charset=UTF-8", withCharset.Encode(), gatewayIP)
	require.Equal(t, http.StatusBadRequest, res.status)
	require.Equal(t, oauth.InvalidGrant, res.err.Code, "a charset parameter does not make the form unreadable")
}

// TestTokenGrantTypeErrors guarantees RFC 6749 section 5.2: a missing
// grant_type is invalid_request, an unknown one is unsupported_grant_type,
// and repeated parameters are invalid_request.
func TestTokenGrantTypeErrors(t *testing.T) {
	h, _ := newTokenHarness(t)
	for _, tt := range []struct {
		name string
		form url.Values
		code oauth.ErrorCode
	}{
		{name: "missing grant_type", form: url.Values{"code": {uid.Secure(32)}}, code: oauth.InvalidRequest},
		{name: "unknown grant_type", form: url.Values{"grant_type": {"password"}}, code: oauth.UnsupportedGrantType},
		{name: "device grant", form: url.Values{"grant_type": {"urn:ietf:params:oauth:grant-type:device_code"}}, code: oauth.UnsupportedGrantType},
		{name: "repeated parameter", form: url.Values{"grant_type": {oauth.GrantAuthorizationCode, oauth.GrantRefreshToken}}, code: oauth.InvalidRequest},
		{name: "refresh without client_id", form: url.Values{"grant_type": {oauth.GrantRefreshToken}, "refresh_token": {uid.Secure(32)}}, code: oauth.InvalidRequest},
		{name: "foreign resource", form: url.Values{"grant_type": {oauth.GrantAuthorizationCode}, "resource": {"https://evil.test"}}, code: oauth.InvalidTarget},
		{name: "unknown scope on refresh", form: url.Values{"grant_type": {oauth.GrantRefreshToken}, "scope": {"admin"}}, code: oauth.InvalidScope},
	} {
		t.Run(tt.name, func(t *testing.T) {
			res := postToken(t, h, tt.form, gatewayIP)
			require.Equal(t, http.StatusBadRequest, res.status)
			require.Equal(t, tt.code, res.err.Code)
		})
	}
}

// TestTokenInvalidGrantIsIndistinguishable guarantees that every way a code
// redemption can fail returns the same invalid_grant body, so a caller probing
// with a stolen code learns nothing about which binding it got wrong.
func TestTokenInvalidGrantIsIndistinguishable(t *testing.T) {
	h, grants := newTokenHarness(t)
	req := granttest.PendingOAuthRequest(t, grants)
	code := approvedCode(t, h, grants, req)
	started, err := grants.Start(t.Context(), grant.StartRequest{DeviceName: "laptop", RemoteIP: "203.0.113.100", UserAgent: ""})
	require.NoError(t, err)

	var bodies []oauth.Error
	for _, tt := range []struct {
		name   string
		mutate func(url.Values)
	}{
		{name: "wrong verifier", mutate: func(f url.Values) { f.Set("code_verifier", uid.Secure(64)) }},
		{name: "missing verifier", mutate: func(f url.Values) { f.Del("code_verifier") }},
		{name: "another client", mutate: func(f url.Values) { f.Set("client_id", uid.New(uid.OAuthClientPrefix, 16)) }},
		{name: "missing client", mutate: func(f url.Values) { f.Del("client_id") }},
		{name: "another redirect uri", mutate: func(f url.Values) { f.Set("redirect_uri", "https://gateway.test/other") }},
		{name: "missing redirect uri it was authorized with", mutate: func(f url.Values) { f.Del("redirect_uri") }},
		{name: "unknown code", mutate: func(f url.Values) { f.Set("code", "ukc_"+uid.Secure(32)) }},
		{name: "missing code", mutate: func(f url.Values) { f.Del("code") }},
		{name: "cli login id as code", mutate: func(f url.Values) { f.Set("code", started.LoginID) }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			form := codeForm(req, code)
			tt.mutate(form)
			res := postToken(t, h, form, gatewayIP)
			require.Equal(t, http.StatusBadRequest, res.status)
			bodies = append(bodies, res.err)
		})
	}
	for _, body := range bodies {
		require.Equal(t, oauth.Error{Code: oauth.InvalidGrant, Description: "The authorization code is invalid, expired, or already used."}, body)
	}

	res := postToken(t, h, codeForm(req, code), gatewayIP)
	require.Equal(t, http.StatusOK, res.status, "failed attempts must not burn the code")
}

// TestTokenRejectsAnExpiredCode guarantees the 60 second code lifetime at the
// token endpoint.
func TestTokenRejectsAnExpiredCode(t *testing.T) {
	h, grants := newTokenHarness(t)
	req := granttest.PendingOAuthRequest(t, grants)
	code := approvedCode(t, h, grants, req)

	h.Clock.Tick(61 * time.Second)
	res := postToken(t, h, codeForm(req, code), gatewayIP)
	require.Equal(t, http.StatusBadRequest, res.status)
	require.Equal(t, oauth.InvalidGrant, res.err.Code)
}

// TestTokenRefreshRotatesOnlyTheClientsOwnTokens guarantees that the token
// endpoint rotates a client's refresh token, refuses a CLI refresh token
// without touching the CLI session, and refuses the client's token when
// another client presents it.
func TestTokenRefreshRotatesOnlyTheClientsOwnTokens(t *testing.T) {
	h, grants := newTokenHarness(t)
	approver := granttest.Approver(h)
	permissions := []string{"unkey:v1:" + h.Resources().UserWorkspace.ID + ":rootKeys/*#read"}
	req := granttest.PendingOAuthRequest(t, grants)
	tokens := granttest.OAuthSession(t, grants, req, approver, permissions)

	started, err := grants.Start(t.Context(), grant.StartRequest{DeviceName: "laptop", RemoteIP: "203.0.113.101", UserAgent: ""})
	require.NoError(t, err)
	granttest.MustApprove(t, grants, grant.ApproveRequest{Principal: approver, UserCode: started.UserCode, Permissions: permissions})
	cli, err := grants.Poll(t.Context(), grant.PollRequest{LoginID: started.LoginID, RemoteIP: "203.0.113.101", UserAgent: ""})
	require.NoError(t, err)

	res := postToken(t, h, refreshForm(req.Client.ID, cli.RefreshToken), gatewayIP)
	require.Equal(t, http.StatusBadRequest, res.status)
	require.Equal(t, oauth.InvalidGrant, res.err.Code)
	session, err := db.Query.FindCLISessionByID(t.Context(), h.DB.RW(), cli.SessionID)
	require.NoError(t, err)
	require.False(t, session.RotatedAt.Valid)
	require.False(t, session.RevokedAt.Valid)

	res = postToken(t, h, refreshForm(grant.ClientID(uid.New(uid.OAuthClientPrefix, 16)), tokens.RefreshToken), gatewayIP)
	require.Equal(t, http.StatusBadRequest, res.status)
	require.Equal(t, oauth.InvalidGrant, res.err.Code)

	res = postToken(t, h, refreshForm(req.Client.ID, tokens.RefreshToken), gatewayIP)
	require.Equal(t, http.StatusOK, res.status)
	require.NotEqual(t, tokens.RefreshToken, res.tokens.RefreshToken)
	require.Equal(t, oauth.ScopeAPI, res.tokens.Scope)
}

// TestFailedRedemptionsAreLimitedPerAddress guarantees that guessing codes is
// bounded per caller address and that only failures count, so one address
// guessing cannot lock out the gateway redeeming real codes, even though the
// gateway's client_id is public.
func TestFailedRedemptionsAreLimitedPerAddress(t *testing.T) {
	h, grants := newTokenHarness(t)
	req := granttest.PendingOAuthRequest(t, grants)
	attacker := "203.0.113.200"
	guess := codeForm(req, "ukc_"+uid.Secure(32))

	limited := 0
	for range 40 {
		res := postToken(t, h, guess, attacker)
		if res.status == http.StatusTooManyRequests {
			limited++
			require.Equal(t, oauth.InvalidRequest, res.err.Code)
			continue
		}
		require.Equal(t, oauth.InvalidGrant, res.err.Code)
	}
	require.Positive(t, limited)

	res := postToken(t, h, codeForm(req, approvedCode(t, h, grants, req)), gatewayIP)
	require.Equal(t, http.StatusOK, res.status)
}

// TestFailedRefreshesAreLimitedPerAddress guarantees that guessing refresh
// tokens is bounded per caller address, and that the limit does not reach a
// client refreshing real tokens from another address.
func TestFailedRefreshesAreLimitedPerAddress(t *testing.T) {
	h, grants := newTokenHarness(t)
	req := granttest.PendingOAuthRequest(t, grants)
	tokens := granttest.OAuthSession(t, grants, req, granttest.Approver(h), []string{"unkey:v1:" + h.Resources().UserWorkspace.ID + ":rootKeys/*#read"})
	attacker := "203.0.113.201"

	limited := 0
	for range 40 {
		res := postToken(t, h, refreshForm(req.Client.ID, "ukr_"+uid.Secure(20)+"_"+uid.Secure(32)), attacker)
		if res.status == http.StatusTooManyRequests {
			limited++
			continue
		}
		require.Equal(t, oauth.InvalidGrant, res.err.Code)
	}
	require.Positive(t, limited)

	res := postToken(t, h, refreshForm(req.Client.ID, tokens.RefreshToken), gatewayIP)
	require.Equal(t, http.StatusOK, res.status)
}
