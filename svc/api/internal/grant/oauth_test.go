package grant

import (
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
)

const testIssuer = "https://api.unkey.test"

func newOAuthService(h *testutil.Harness, membership mem) *Service {
	svc := newService(h, membership)
	svc.Issuer = testIssuer
	return svc
}

func registerTestClient(t *testing.T, svc *Service, redirectURIs ...string) Client {
	t.Helper()
	client, err := svc.RegisterClient(t.Context(), RegisterClientRequest{
		Name:         "Unkey MCP",
		RedirectURIs: redirectURIs,
		RemoteIP:     "203.0.113.50",
		UserAgent:    "gateway",
	})
	require.NoError(t, err)
	return client
}

// oauthRequest is one pending authorization request and the secrets only the
// client knows.
type oauthRequest struct {
	client   Client
	verifier string
	state    string
	userCode string
}

func authorizeRequest(t *testing.T, svc *Service, client Client, redirectURI string) oauthRequest {
	t.Helper()
	verifier := uid.Secure(64)
	sum := sha256.Sum256([]byte(verifier))
	challenge, err := ParseChallenge("S256", base64.RawURLEncoding.EncodeToString(sum[:]))
	require.NoError(t, err)
	verified, ok := client.Redirect(redirectURI)
	require.True(t, ok)
	state := uid.Secure(16)
	authorized, err := svc.Authorize(t.Context(), AuthorizeRequest{
		Client:      client,
		RedirectURI: verified,
		State:       state,
		Challenge:   challenge,
		RemoteIP:    "203.0.113.51",
		UserAgent:   "browser",
	})
	require.NoError(t, err)
	consent, err := url.Parse(authorized.ConsentURL)
	require.NoError(t, err)
	require.Equal(t, consentPath, consent.Path)
	return oauthRequest{client: client, verifier: verifier, state: state, userCode: consent.Query().Get("user_code")}
}

func approveRequest(t *testing.T, svc *Service, h *testutil.Harness, userID string, req oauthRequest) *url.URL {
	t.Helper()
	workspace := h.Resources().UserWorkspace
	decided := mustApprove(t, svc, ApproveRequest{
		Principal:   developerPrincipal(workspace, userID),
		UserCode:    req.userCode,
		Permissions: []string{permissionOnlyDeveloperHas(t, workspace.ID)},
	})
	callback, err := url.Parse(decided.RedirectTo)
	require.NoError(t, err)
	return callback
}

func redeemRequest(req oauthRequest, code string) RedeemRequest {
	return RedeemRequest{
		Code:         code,
		ClientID:     req.client.ID,
		RedirectURI:  req.client.RedirectURIs[0],
		CodeVerifier: req.verifier,
		RemoteIP:     "203.0.113.52",
		UserAgent:    "gateway",
	}
}

// oauthSession runs the whole redirect flow and returns the issued tokens.
func oauthSession(t *testing.T, svc *Service, h *testutil.Harness, userID string) (Client, Tokens) {
	t.Helper()
	client := registerTestClient(t, svc, "https://gateway.test/oauth/callback")
	req := authorizeRequest(t, svc, client, "https://gateway.test/oauth/callback")
	callback := approveRequest(t, svc, h, userID, req)
	tokens, err := svc.Redeem(t.Context(), redeemRequest(req, callback.Query().Get("code")))
	require.NoError(t, err)
	return client, tokens
}

func requireRejectedCode(t *testing.T, err error) {
	t.Helper()
	require.ErrorIs(t, err, ErrRejected)
	require.Equal(t, "The authorization code is invalid, expired, or already used.", fault.UserFacingMessage(err))
}

// TestParseRequestDecidesTheDeliveryFromClientID guarantees that client_id
// alone picks the delivery mode and that a row whose OAuth columns disagree
// with it never reaches the state machine.
func TestParseRequestDecidesTheDeliveryFromClientID(t *testing.T) {
	present := func(value string) sql.NullString { return sql.NullString{String: value, Valid: true} }
	redirect := db.DeviceAuthorization{
		ID:                   uid.New(uid.DeviceAuthorizationPrefix),
		ClientID:             present(uid.New(uid.OAuthClientPrefix, 16)),
		RedirectUri:          present("https://gateway.test/oauth/callback"),
		RedirectUriPresented: sql.NullBool{Bool: true, Valid: true},
		State:                present("state"),
		CodeChallenge:        present(uid.Secure(43)),
	}
	device := db.DeviceAuthorization{ID: uid.New(uid.DeviceAuthorizationPrefix)}

	for _, tt := range []struct {
		name     string
		row      func() db.DeviceAuthorization
		delivery delivery
	}{
		{name: "device login", row: func() db.DeviceAuthorization { return device }, delivery: deviceDelivery{}},
		{name: "oauth request", row: func() db.DeviceAuthorization { return redirect }, delivery: redirectDelivery{}},
		{name: "oauth request without state", row: func() db.DeviceAuthorization {
			r := redirect
			r.State = sql.NullString{}
			return r
		}, delivery: redirectDelivery{}},
		{name: "device login with a redirect uri", row: func() db.DeviceAuthorization {
			r := device
			r.RedirectUri = redirect.RedirectUri
			return r
		}, delivery: nil},
		{name: "device login with a code hash", row: func() db.DeviceAuthorization {
			r := device
			r.CodeHash = present(hashCode(uid.Secure(32)))
			return r
		}, delivery: nil},
		{name: "oauth request without a challenge", row: func() db.DeviceAuthorization {
			r := redirect
			r.CodeChallenge = sql.NullString{}
			return r
		}, delivery: nil},
		{name: "oauth request without a redirect uri", row: func() db.DeviceAuthorization {
			r := redirect
			r.RedirectUri = sql.NullString{}
			return r
		}, delivery: nil},
		{name: "oauth request without the presented flag", row: func() db.DeviceAuthorization {
			r := redirect
			r.RedirectUriPresented = sql.NullBool{}
			return r
		}, delivery: nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := parseRequest(tt.row())
			if tt.delivery == nil {
				require.Error(t, err)
				require.NotErrorIs(t, err, ErrRejected)
				return
			}
			require.NoError(t, err)
			require.IsType(t, tt.delivery, parsed.delivery)
		})
	}
}

// TestApproveSendsTheBrowserBackWithACode guarantees the redirect approval
// contract: the code goes only to the registered redirect URI, with the
// client's state and the issuer, and lives at most codeTTL.
func TestApproveSendsTheBrowserBackWithACode(t *testing.T) {
	h := testutil.NewHarness(t)
	svc := newOAuthService(h, mem{})
	client := registerTestClient(t, svc, "https://gateway.test/oauth/callback?tenant=a")
	req := authorizeRequest(t, svc, client, "https://gateway.test/oauth/callback?tenant=a")
	require.Len(t, req.userCode, consentHandleLength)

	callback := approveRequest(t, svc, h, uid.New(uid.TestPrefix), req)
	require.Equal(t, "https", callback.Scheme)
	require.Equal(t, "gateway.test", callback.Host)
	require.Equal(t, "/oauth/callback", callback.Path)
	require.Equal(t, "a", callback.Query().Get("tenant"))
	require.Equal(t, req.state, callback.Query().Get("state"))
	require.Equal(t, testIssuer, callback.Query().Get("iss"))
	code := callback.Query().Get("code")
	require.True(t, strings.HasPrefix(code, authorizationCodePrefix))

	row, err := db.Query.FindDeviceAuthorizationByUserCode(t.Context(), h.DB.RW(), req.userCode)
	require.NoError(t, err)
	require.NotContains(t, callback.String(), row.ID, "the request id is not the code")
	require.Equal(t, sql.NullString{String: hashCode(code), Valid: true}, row.CodeHash)
	require.True(t, row.ApprovedAt.Valid)
	require.LessOrEqual(t, row.ExpiresAt, row.ApprovedAt.Int64+codeTTL.Milliseconds())
}

// TestDenySendsTheBrowserBackWithAccessDenied guarantees that a denial tells
// the client through its redirect URI, keeps its state, and can be repeated.
func TestDenySendsTheBrowserBackWithAccessDenied(t *testing.T) {
	h := testutil.NewHarness(t)
	svc := newOAuthService(h, mem{})
	client := registerTestClient(t, svc, "https://gateway.test/oauth/callback")
	req := authorizeRequest(t, svc, client, "https://gateway.test/oauth/callback")
	denier := developerPrincipal(h.Resources().UserWorkspace, uid.New(uid.TestPrefix))

	first := mustDeny(t, svc, DenyRequest{Principal: denier, UserCode: req.userCode})
	again := mustDeny(t, svc, DenyRequest{Principal: denier, UserCode: req.userCode})
	require.Equal(t, first, again)
	callback, err := url.Parse(first.RedirectTo)
	require.NoError(t, err)
	require.Equal(t, "gateway.test", callback.Host)
	require.Equal(t, "access_denied", callback.Query().Get("error"))
	require.Equal(t, req.state, callback.Query().Get("state"))
	require.Equal(t, testIssuer, callback.Query().Get("iss"))
	require.Empty(t, callback.Query().Get("code"))
}

// TestDeviceDecisionsHaveNoRedirect guarantees that the CLI flow is unchanged:
// its decisions send the browser nowhere.
func TestDeviceDecisionsHaveNoRedirect(t *testing.T) {
	h := testutil.NewHarness(t)
	svc := newOAuthService(h, mem{})
	workspace := h.Resources().UserWorkspace
	started, err := svc.Start(t.Context(), StartRequest{DeviceName: "laptop", RemoteIP: "203.0.113.53", UserAgent: "unkey-cli"})
	require.NoError(t, err)

	decided := mustApprove(t, svc, ApproveRequest{
		Principal:   developerPrincipal(workspace, uid.New(uid.TestPrefix)),
		UserCode:    started.UserCode,
		Permissions: []string{permissionOnlyDeveloperHas(t, workspace.ID)},
	})
	require.Empty(t, decided.RedirectTo)
	row, err := db.Query.FindDeviceAuthorizationByID(t.Context(), h.DB.RW(), started.LoginID)
	require.NoError(t, err)
	require.False(t, row.CodeHash.Valid)
	require.Greater(t, row.ExpiresAt, row.ApprovedAt.Int64+codeTTL.Milliseconds(), "device logins keep their full TTL")

	authz, err := svc.Get(t.Context(), GetRequest{Principal: developerPrincipal(workspace, uid.New(uid.TestPrefix)), UserCode: started.UserCode})
	require.NoError(t, err)
	require.Nil(t, authz.Client)
}

// TestGetShowsTheClientAsking guarantees the consent page learns which app is
// asking and where approval sends the browser.
func TestGetShowsTheClientAsking(t *testing.T) {
	h := testutil.NewHarness(t)
	svc := newOAuthService(h, mem{})
	client := registerTestClient(t, svc, "https://gateway.test/oauth/callback")
	req := authorizeRequest(t, svc, client, "https://gateway.test/oauth/callback")

	authz, err := svc.Get(t.Context(), GetRequest{
		Principal: developerPrincipal(h.Resources().UserWorkspace, uid.New(uid.TestPrefix)),
		UserCode:  req.userCode,
	})
	require.NoError(t, err)
	require.Equal(t, &ConsentClient{ID: client.ID, Name: "Unkey MCP", RedirectURI: "https://gateway.test/oauth/callback"}, authz.Client)
	require.Equal(t, "pending", authz.Status)
	require.Equal(t, "Unkey MCP", authz.DeviceName)
}

// TestLoginIDsAndCodesAreNotInterchangeable guarantees that a CLI login id
// cannot be redeemed as an authorization code and that an OAuth request
// cannot be polled as a CLI login.
func TestLoginIDsAndCodesAreNotInterchangeable(t *testing.T) {
	h := testutil.NewHarness(t)
	svc := newOAuthService(h, mem{})
	workspace := h.Resources().UserWorkspace
	started, err := svc.Start(t.Context(), StartRequest{DeviceName: "laptop", RemoteIP: "203.0.113.54", UserAgent: "unkey-cli"})
	require.NoError(t, err)
	mustApprove(t, svc, ApproveRequest{
		Principal:   developerPrincipal(workspace, uid.New(uid.TestPrefix)),
		UserCode:    started.UserCode,
		Permissions: []string{permissionOnlyDeveloperHas(t, workspace.ID)},
	})

	client := registerTestClient(t, svc, "https://gateway.test/oauth/callback")
	req := authorizeRequest(t, svc, client, "https://gateway.test/oauth/callback")
	_, err = svc.Redeem(t.Context(), redeemRequest(req, started.LoginID))
	requireRejectedCode(t, err)

	callback := approveRequest(t, svc, h, uid.New(uid.TestPrefix), req)
	row, err := db.Query.FindDeviceAuthorizationByUserCode(t.Context(), h.DB.RW(), req.userCode)
	require.NoError(t, err)
	for _, loginID := range []string{row.ID, callback.Query().Get("code")} {
		_, err = svc.Poll(t.Context(), PollRequest{LoginID: loginID, RemoteIP: "203.0.113.54"})
		require.Equal(t, "This login code was not found. Run unkey login again.", fault.UserFacingMessage(err))
	}

	polled, err := svc.Poll(t.Context(), PollRequest{LoginID: started.LoginID, RemoteIP: "203.0.113.54"})
	require.NoError(t, err)
	require.Equal(t, PollComplete, polled.Status)
}

// TestRedeemBindsTheCodeToClientRedirectAndVerifier guarantees that a stolen
// code is useless without the client's PKCE verifier, client id, and redirect
// URI, that every mismatch looks the same, and that a failed attempt does not
// burn the code for the real client.
func TestRedeemBindsTheCodeToClientRedirectAndVerifier(t *testing.T) {
	h := testutil.NewHarness(t)
	svc := newOAuthService(h, mem{})
	client := registerTestClient(t, svc, "https://gateway.test/oauth/callback")
	other := registerTestClient(t, svc, "https://gateway.test/other/callback")
	req := authorizeRequest(t, svc, client, "https://gateway.test/oauth/callback")
	code := approveRequest(t, svc, h, uid.New(uid.TestPrefix), req).Query().Get("code")

	for _, tt := range []struct {
		name   string
		mutate func(RedeemRequest) RedeemRequest
	}{
		{name: "wrong verifier", mutate: func(r RedeemRequest) RedeemRequest { r.CodeVerifier = uid.Secure(64); return r }},
		{name: "missing verifier", mutate: func(r RedeemRequest) RedeemRequest { r.CodeVerifier = ""; return r }},
		{name: "another client", mutate: func(r RedeemRequest) RedeemRequest { r.ClientID = other.ID; return r }},
		{name: "another redirect uri", mutate: func(r RedeemRequest) RedeemRequest { r.RedirectURI = "https://gateway.test/other/callback"; return r }},
		{name: "redirect uri omitted after it was presented", mutate: func(r RedeemRequest) RedeemRequest { r.RedirectURI = ""; return r }},
		{name: "unknown code", mutate: func(r RedeemRequest) RedeemRequest { r.Code = authorizationCodePrefix + uid.Secure(32); return r }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Redeem(t.Context(), tt.mutate(redeemRequest(req, code)))
			requireRejectedCode(t, err)
		})
	}

	tokens, err := svc.Redeem(t.Context(), redeemRequest(req, code))
	require.NoError(t, err)
	session, err := db.Query.FindCLISessionByID(t.Context(), h.DB.RW(), tokens.SessionID)
	require.NoError(t, err)
	require.Equal(t, sql.NullString{String: string(client.ID), Valid: true}, session.ClientID)
	require.Equal(t, sql.NullString{String: "Unkey MCP", Valid: true}, session.DeviceName)
}

// TestRedeemWithoutRedirectURIWhenAuthorizeOmittedIt guarantees RFC 6749
// section 4.1.3: the token request needs redirect_uri only when the
// authorization request included it.
func TestRedeemWithoutRedirectURIWhenAuthorizeOmittedIt(t *testing.T) {
	h := testutil.NewHarness(t)
	svc := newOAuthService(h, mem{})
	client := registerTestClient(t, svc, "https://gateway.test/oauth/callback")
	req := authorizeRequest(t, svc, client, "")
	callback := approveRequest(t, svc, h, uid.New(uid.TestPrefix), req)
	require.Equal(t, "gateway.test", callback.Host)

	redeem := redeemRequest(req, callback.Query().Get("code"))
	redeem.RedirectURI = ""
	_, err := svc.Redeem(t.Context(), redeem)
	require.NoError(t, err)
}

// TestRedeemAfterTheCodeExpiresIsRejected guarantees the 60 second code
// lifetime counted from approval.
func TestRedeemAfterTheCodeExpiresIsRejected(t *testing.T) {
	h := testutil.NewHarness(t)
	svc := newOAuthService(h, mem{})
	client := registerTestClient(t, svc, "https://gateway.test/oauth/callback")
	req := authorizeRequest(t, svc, client, "https://gateway.test/oauth/callback")
	code := approveRequest(t, svc, h, uid.New(uid.TestPrefix), req).Query().Get("code")

	h.Clock.Tick(codeTTL + time.Second)
	_, err := svc.Redeem(t.Context(), redeemRequest(req, code))
	requireRejectedCode(t, err)
}

// TestReapprovalRetiresTheEarlierCode guarantees that approving twice leaves
// only the newest code redeemable, so a code from a stale approval cannot
// issue a session with the earlier permissions.
func TestReapprovalRetiresTheEarlierCode(t *testing.T) {
	h := testutil.NewHarness(t)
	svc := newOAuthService(h, mem{})
	client := registerTestClient(t, svc, "https://gateway.test/oauth/callback")
	req := authorizeRequest(t, svc, client, "https://gateway.test/oauth/callback")
	userID := uid.New(uid.TestPrefix)
	first := approveRequest(t, svc, h, userID, req).Query().Get("code")
	second := approveRequest(t, svc, h, userID, req).Query().Get("code")
	require.NotEqual(t, first, second)

	_, err := svc.Redeem(t.Context(), redeemRequest(req, first))
	requireRejectedCode(t, err)
	_, err = svc.Redeem(t.Context(), redeemRequest(req, second))
	require.NoError(t, err)
}

// TestReplayedCodeRevokesTheSessionItIssued guarantees RFC 6749 section
// 4.1.2: a second redemption fails and ends the session the first one
// created, because either the client is broken or someone else holds it.
func TestReplayedCodeRevokesTheSessionItIssued(t *testing.T) {
	h := testutil.NewHarness(t)
	svc := newOAuthService(h, mem{})
	client := registerTestClient(t, svc, "https://gateway.test/oauth/callback")
	req := authorizeRequest(t, svc, client, "https://gateway.test/oauth/callback")
	code := approveRequest(t, svc, h, uid.New(uid.TestPrefix), req).Query().Get("code")

	tokens, err := svc.Redeem(t.Context(), redeemRequest(req, code))
	require.NoError(t, err)
	_, err = svc.Redeem(t.Context(), redeemRequest(req, code))
	requireRejectedCode(t, err)

	session, err := db.Query.FindCLISessionByID(t.Context(), h.DB.RW(), tokens.SessionID)
	require.NoError(t, err)
	require.True(t, session.RevokedAt.Valid)
	_, err = svc.Allow(t.Context(), tokens.SessionID)
	require.Error(t, err)
}

// TestConcurrentRedemptionsIssueOneSession guarantees that racing
// redemptions of one code issue exactly one session, that every loser is an
// invalid_grant refusal rather than an infrastructure error, and that the race
// counts as a replay and revokes the winner's session.
func TestConcurrentRedemptionsIssueOneSession(t *testing.T) {
	h := testutil.NewHarness(t)
	svc := newOAuthService(h, mem{})
	userID := uid.New(uid.TestPrefix)
	client := registerTestClient(t, svc, "https://gateway.test/oauth/callback")
	req := authorizeRequest(t, svc, client, "https://gateway.test/oauth/callback")
	code := approveRequest(t, svc, h, userID, req).Query().Get("code")

	const redeemers = 8
	results := make([]Tokens, redeemers)
	errs := make([]error, redeemers)
	var wg sync.WaitGroup
	for i := range redeemers {
		wg.Go(func() {
			results[i], errs[i] = svc.Redeem(t.Context(), redeemRequest(req, code))
		})
	}
	wg.Wait()

	var winner Tokens
	issued := 0
	for i := range redeemers {
		if errs[i] == nil {
			issued++
			winner = results[i]
			continue
		}
		requireRejectedCode(t, errs[i])
	}
	require.Equal(t, 1, issued)

	sessions, err := db.Query.ListCLISessionsByUser(t.Context(), h.DB.RW(), db.ListCLISessionsByUserParams{
		WorkspaceID: h.Resources().UserWorkspace.ID,
		UserID:      userID,
		Now:         0,
	})
	require.NoError(t, err)
	require.Empty(t, sessions, "the only issued session was revoked")
	session, err := db.Query.FindCLISessionByID(t.Context(), h.DB.RW(), winner.SessionID)
	require.NoError(t, err)
	require.True(t, session.RevokedAt.Valid)
}

// TestRefreshTokensWorkOnlyForTheirHolder guarantees that a CLI refresh token
// cannot be rotated by an OAuth client and an OAuth client's refresh token
// cannot be rotated by the CLI or by another client, and that such an attempt
// does not disturb the session.
func TestRefreshTokensWorkOnlyForTheirHolder(t *testing.T) {
	h := testutil.NewHarness(t)
	svc := newOAuthService(h, mem{})
	workspace := h.Resources().UserWorkspace
	cli := login(t, svc, workspace, uid.New(uid.TestPrefix), permissionOnlyDeveloperHas(t, workspace.ID))
	client, oauth := oauthSession(t, svc, h, uid.New(uid.TestPrefix))
	holder, err := HeldBy(client.ID)
	require.NoError(t, err)
	stranger, err := HeldBy(ClientID(uid.New(uid.OAuthClientPrefix, 16)))
	require.NoError(t, err)

	for _, tt := range []struct {
		name   string
		token  string
		holder Holder
	}{
		{name: "cli token at an oauth client", token: cli.RefreshToken, holder: holder},
		{name: "oauth token at the cli", token: oauth.RefreshToken, holder: CLI},
		{name: "oauth token at another client", token: oauth.RefreshToken, holder: stranger},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Refresh(t.Context(), RefreshRequest{RefreshToken: tt.token, Holder: tt.holder, RemoteIP: "203.0.113.55", UserAgent: ""})
			require.ErrorIs(t, err, ErrRejected)
			require.Equal(t, codes.Auth.Authentication.CredentialRejected.URN(), codeOf(t, err))
		})
	}

	_, err = svc.Refresh(t.Context(), RefreshRequest{RefreshToken: cli.RefreshToken, Holder: CLI, RemoteIP: "203.0.113.55", UserAgent: ""})
	require.NoError(t, err)
	rotated, err := svc.Refresh(t.Context(), RefreshRequest{RefreshToken: oauth.RefreshToken, Holder: holder, RemoteIP: "203.0.113.55", UserAgent: ""})
	require.NoError(t, err)
	require.NotEqual(t, oauth.RefreshToken, rotated.RefreshToken)
	require.Equal(t, oauth.SessionID, rotated.SessionID)
}

// TestOAuthRejectionsSendUsersBackToTheApp guarantees that an ended OAuth
// session tells the user to sign in from the app, not to run the CLI.
func TestOAuthRejectionsSendUsersBackToTheApp(t *testing.T) {
	h := testutil.NewHarness(t)
	svc := newOAuthService(h, mem{})
	client, tokens := oauthSession(t, svc, h, uid.New(uid.TestPrefix))
	holder, err := HeldBy(client.ID)
	require.NoError(t, err)

	rotated, err := svc.Refresh(t.Context(), RefreshRequest{RefreshToken: tokens.RefreshToken, Holder: holder, RemoteIP: "203.0.113.56", UserAgent: ""})
	require.NoError(t, err)
	_, err = svc.Refresh(t.Context(), RefreshRequest{RefreshToken: rotated.RefreshToken, Holder: holder, RemoteIP: "203.0.113.56", UserAgent: ""})
	require.NoError(t, err)

	_, err = svc.Refresh(t.Context(), RefreshRequest{RefreshToken: tokens.RefreshToken, Holder: holder, RemoteIP: "203.0.113.56", UserAgent: ""})
	require.ErrorIs(t, err, ErrRejected)
	require.Equal(t, "This refresh token was already used. The session was revoked. Sign in to Unkey again from Unkey MCP.", fault.UserFacingMessage(err))

	session, err := svc.Allow(t.Context(), tokens.SessionID)
	require.Error(t, err)
	require.Equal(t, string(client.ID), session.ClientID)
}

// TestOAuthWorkspaceRefusalsDoNotMentionTheCLI guarantees that a sign-in
// refused because the workspace became unusable is worded for the OAuth client,
// at approval and at redemption.
func TestOAuthWorkspaceRefusalsDoNotMentionTheCLI(t *testing.T) {
	h := testutil.NewHarness(t)
	svc := newOAuthService(h, mem{})
	workspace := h.Resources().UserWorkspace
	client := registerTestClient(t, svc, "https://gateway.test/oauth/callback")
	approved := authorizeRequest(t, svc, client, "https://gateway.test/oauth/callback")
	callback := approveRequest(t, svc, h, uid.New(uid.TestPrefix), approved)
	pending := authorizeRequest(t, svc, client, "https://gateway.test/oauth/callback")

	_, err := db.Query.UpdateWorkspaceEnabled(t.Context(), h.DB.RW(), db.UpdateWorkspaceEnabledParams{Enabled: false, ID: workspace.ID})
	require.NoError(t, err)

	_, err = svc.Redeem(t.Context(), redeemRequest(approved, callback.Query().Get("code")))
	require.ErrorIs(t, err, ErrRejected)
	require.Equal(t, "This workspace cannot approve a sign-in.", fault.UserFacingMessage(err))

	_, err = svc.Approve(t.Context(), ApproveRequest{
		Principal:   developerPrincipal(workspace, uid.New(uid.TestPrefix)),
		UserCode:    pending.userCode,
		Permissions: []string{permissionOnlyDeveloperHas(t, workspace.ID)},
	})
	require.ErrorIs(t, err, ErrRejected)
	require.Equal(t, "This workspace cannot approve a sign-in.", fault.UserFacingMessage(err))
}

// TestSessionAuditFollowsTheHolder guarantees that OAuth sessions are audited
// under their own events with the client named, while CLI sessions keep
// theirs.
func TestSessionAuditFollowsTheHolder(t *testing.T) {
	h := testutil.NewHarness(t)
	svc := newOAuthService(h, mem{})
	workspace := h.Resources().UserWorkspace
	userID := uid.New(uid.TestPrefix)
	client, tokens := oauthSession(t, svc, h, userID)
	holder, err := HeldBy(client.ID)
	require.NoError(t, err)
	_, err = svc.Refresh(t.Context(), RefreshRequest{RefreshToken: tokens.RefreshToken, Holder: holder, RemoteIP: "203.0.113.57", UserAgent: ""})
	require.NoError(t, err)
	require.NoError(t, svc.Revoke(t.Context(), RevokeRequest{Principal: developerPrincipal(workspace, userID), SessionID: tokens.SessionID}))

	events := map[string]bool{}
	for _, log := range h.FindAuditLogsByTargetID(t.Context(), t, tokens.SessionID) {
		events[log.Event] = true
		require.Equal(t, "oauthSession", log.Targets[0].Type)
		require.Equal(t, string(client.ID), log.Targets[0].Meta["client_id"])
	}
	require.Equal(t, map[string]bool{"oauth.session.create": true, "oauth.session.refresh": true, "oauth.session.revoke": true}, events)

	cli := login(t, svc, workspace, userID, permissionOnlyDeveloperHas(t, workspace.ID))
	for _, log := range h.FindAuditLogsByTargetID(t.Context(), t, cli.SessionID) {
		require.Equal(t, "cli.session.create", log.Event)
		require.NotContains(t, log.Targets[0].Meta, "client_id")
	}

	sessions, err := svc.List(t.Context(), ListRequest{Principal: developerPrincipal(workspace, userID)})
	require.NoError(t, err)
	require.Len(t, sessions, 1)
	require.Equal(t, cli.SessionID, sessions[0].ID)
	require.Empty(t, sessions[0].ClientID)
}

// TestRegisterClientDedupesIdenticalMetadata guarantees that repeated
// registrations of the same name and redirect URIs return one client, so an
// unauthenticated endpoint cannot grow the client table without bound.
func TestRegisterClientDedupesIdenticalMetadata(t *testing.T) {
	h := testutil.NewHarness(t)
	svc := newOAuthService(h, mem{})
	callback := "https://gateway.test/" + uid.New(uid.TestPrefix)
	first := registerTestClient(t, svc, callback, callback)
	second := registerTestClient(t, svc, callback)
	require.Equal(t, first, second)
	require.Equal(t, []string{callback}, first.RedirectURIs)

	found, err := svc.FindClient(t.Context(), string(first.ID))
	require.NoError(t, err)
	require.Equal(t, first, found)

	_, err = svc.FindClient(t.Context(), uid.New(uid.OAuthClientPrefix, 16))
	require.ErrorIs(t, err, ErrUnknownClient)
	_, err = svc.FindClient(t.Context(), uid.New(uid.TestPrefix))
	require.ErrorIs(t, err, ErrUnknownClient)
}

// TestRegisterClientRejectsUnsafeRedirectURIs guarantees the redirect URI
// rules that keep codes off plaintext networks and away from schemes and
// hosts the server cannot vouch for.
func TestRegisterClientRejectsUnsafeRedirectURIs(t *testing.T) {
	for _, tt := range []struct {
		name    string
		uri     string
		origins []string
		valid   bool
	}{
		{name: "https", uri: "https://gateway.test/callback", origins: nil, valid: true},
		{name: "http on ipv4 loopback", uri: "http://127.0.0.1:8080/callback", origins: nil, valid: true},
		{name: "http on ipv6 loopback", uri: "http://[::1]/callback", origins: nil, valid: true},
		{name: "http elsewhere", uri: "http://gateway.test/callback", origins: nil, valid: false},
		{name: "http on localhost", uri: "http://localhost/callback", origins: nil, valid: false},
		{name: "custom scheme", uri: "com.example.app:/callback", origins: nil, valid: false},
		{name: "fragment", uri: "https://gateway.test/callback#frag", origins: nil, valid: false},
		{name: "userinfo", uri: "https://user@gateway.test/callback", origins: nil, valid: false},
		{name: "relative", uri: "/callback", origins: nil, valid: false},
		{name: "too long", uri: "https://gateway.test/" + strings.Repeat("a", redirectURIBytesMax), origins: nil, valid: false},
		{name: "allowed origin", uri: "https://gateway.test/callback", origins: []string{"https://gateway.test"}, valid: true},
		{name: "origin outside the allowlist", uri: "https://evil.test/callback", origins: []string{"https://gateway.test"}, valid: false},
		{name: "loopback outside the allowlist", uri: "http://127.0.0.1/callback", origins: []string{"https://gateway.test"}, valid: false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRedirectURI(tt.uri, tt.origins)
			if tt.valid {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, ErrInvalidRedirectURI)
		})
	}

	h := testutil.NewHarness(t)
	svc := newOAuthService(h, mem{})
	_, err := svc.RegisterClient(t.Context(), RegisterClientRequest{Name: "app", RedirectURIs: nil, RemoteIP: "203.0.113.58", UserAgent: ""})
	require.ErrorIs(t, err, ErrInvalidRedirectURI)
	_, err = svc.RegisterClient(t.Context(), RegisterClientRequest{Name: "app\x00", RedirectURIs: []string{"https://gateway.test/callback"}, RemoteIP: "203.0.113.58", UserAgent: ""})
	require.ErrorIs(t, err, ErrInvalidClientMetadata)
}

// TestClientRedirectMatchesTheRegistration guarantees that only registered
// redirect URIs are reachable, with the RFC 8252 loopback port exception and
// the single-registration default.
func TestClientRedirectMatchesTheRegistration(t *testing.T) {
	single := Client{RedirectURIs: []string{"https://gateway.test/callback"}}
	loopback := Client{RedirectURIs: []string{"http://127.0.0.1/callback", "https://gateway.test/callback"}}

	for _, tt := range []struct {
		name      string
		client    Client
		raw       string
		matched   string
		presented bool
		ok        bool
	}{
		{name: "exact", client: single, raw: "https://gateway.test/callback", matched: "https://gateway.test/callback", presented: true, ok: true},
		{name: "omitted with one registered", client: single, raw: "", matched: "https://gateway.test/callback", presented: false, ok: true},
		{name: "omitted with two registered", client: loopback, raw: "", matched: "", presented: false, ok: false},
		{name: "different path", client: single, raw: "https://gateway.test/callback/", matched: "", presented: false, ok: false},
		{name: "different host", client: single, raw: "https://evil.test/callback", matched: "", presented: false, ok: false},
		{name: "loopback on any port", client: loopback, raw: "http://127.0.0.1:53124/callback", matched: "http://127.0.0.1:53124/callback", presented: true, ok: true},
		{name: "loopback on another path", client: loopback, raw: "http://127.0.0.1:53124/other", matched: "", presented: false, ok: false},
		{name: "https port is not relaxed", client: single, raw: "https://gateway.test:8443/callback", matched: "", presented: false, ok: false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := tt.client.Redirect(tt.raw)
			require.Equal(t, tt.ok, ok)
			require.Equal(t, RedirectURI{raw: tt.matched, presented: tt.presented}, got)
		})
	}
}

// TestChallengeAcceptsOnlyS256 guarantees that PKCE cannot be downgraded to
// plain and that only the matching verifier passes.
func TestChallengeAcceptsOnlyS256(t *testing.T) {
	verifier := uid.Secure(64)
	sum := sha256.Sum256([]byte(verifier))
	encoded := base64.RawURLEncoding.EncodeToString(sum[:])

	_, err := ParseChallenge("plain", verifier)
	require.Error(t, err)
	_, err = ParseChallenge("", encoded)
	require.Error(t, err)
	_, err = ParseChallenge("S256", encoded[:42])
	require.Error(t, err)
	_, err = ParseChallenge("S256", strings.Repeat("*", 43))
	require.Error(t, err)

	challenge, err := ParseChallenge("S256", encoded)
	require.NoError(t, err)
	require.True(t, challenge.Verify(verifier))
	require.False(t, challenge.Verify(uid.Secure(64)))
	require.False(t, challenge.Verify(encoded), "the challenge itself is not a verifier")
	require.False(t, challenge.Verify(verifier[:42]))
}

// TestHeldByRejectsIdsRegistrationCannotIssue guarantees that a missing or
// foreign client_id cannot select the CLI holder.
func TestHeldByRejectsIdsRegistrationCannotIssue(t *testing.T) {
	for _, id := range []ClientID{"", ClientID(uid.New(uid.TestPrefix)), "oac"} {
		_, err := HeldBy(id)
		require.Error(t, err)
	}
	holder, err := HeldBy(ClientID(uid.New(uid.OAuthClientPrefix, 16)))
	require.NoError(t, err)
	require.NotEqual(t, CLI, holder)
}

// TestOAuthRequiresAnIssuer guarantees that OAuth stays off until the issuer
// is configured while device login keeps working.
func TestOAuthRequiresAnIssuer(t *testing.T) {
	h := testutil.NewHarness(t)
	svc := newService(h, mem{})
	require.True(t, svc.Enabled())
	require.False(t, svc.OAuthEnabled())
	_, err := svc.RegisterClient(t.Context(), RegisterClientRequest{Name: "app", RedirectURIs: []string{"https://gateway.test/callback"}, RemoteIP: "203.0.113.59", UserAgent: ""})
	require.Equal(t, codes.App.Internal.ServiceUnavailable.URN(), codeOf(t, err))
	require.False(t, errors.Is(err, ErrRejected))
}
