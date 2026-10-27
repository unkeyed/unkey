package clisession

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	tokenjwt "github.com/unkeyed/unkey/pkg/jwt"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
)

const testSecret = "test-cli-signing-secret-32-bytes!"

type fakeSessions struct {
	session Session
	err     error
}

func (f fakeSessions) Allow(context.Context, string) (Session, error) {
	if f.err != nil {
		return Session{}, f.err
	}
	return f.session, nil
}

type tokenFixture struct {
	userID      string
	workspaceID string
	sessionID   string
	permission  string
}

func newTokenFixture() tokenFixture {
	workspaceID := uid.New(uid.WorkspacePrefix)
	return tokenFixture{
		userID:      uid.New(uid.TestPrefix),
		workspaceID: workspaceID,
		sessionID:   uid.New(uid.CLISessionPrefix),
		permission:  "unkey:v1:" + workspaceID + ":projects/*/keyspaces/*/keys/*#read",
	}
}

func (f tokenFixture) claims(clk clock.Clock) Claims {
	return Claims{
		RegisteredClaims: tokenjwt.RegisteredClaims{
			Issuer:    Issuer,
			Subject:   f.userID,
			Audience:  []string{Audience},
			IssuedAt:  clk.Now().Unix(),
			ExpiresAt: clk.Now().Add(time.Hour).Unix(),
		},
		WorkspaceID: f.workspaceID,
		SessionID:   f.sessionID,
		Permissions: []string{f.permission},
		TokenUse:    TokenUse,
		Name:        "Dev",
	}
}

func (f tokenFixture) session(clk clock.Clock) Session {
	return Session{
		UserID:      f.userID,
		WorkspaceID: f.workspaceID,
		DeviceName:  "laptop",
		ExpiresAt:   clk.Now().Add(30 * 24 * time.Hour),
	}
}

func sign(t *testing.T, secret string, claims Claims) string {
	t.Helper()
	signer, err := tokenjwt.NewHS256Signer[Claims]([]byte(secret))
	require.NoError(t, err)
	token, err := signer.Sign(claims)
	require.NoError(t, err)
	return token
}

func bearerSession(t *testing.T, token string) *zen.Session {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	sess := &zen.Session{}
	require.NoError(t, sess.Init(httptest.NewRecorder(), req, 0))
	return sess
}

func requireCode(t *testing.T, err error, want codes.URN) {
	t.Helper()
	require.Error(t, err)
	code, ok := fault.GetCode(err)
	require.True(t, ok)
	require.Equal(t, want, code)
}

func TestResolverBuildsPrincipalFromPermissionsClaim(t *testing.T) {
	clk := clock.NewTestClock()
	fixture := newTokenFixture()
	token := sign(t, testSecret, fixture.claims(clk))

	stored := fixture.session(clk)
	resolver, err := NewResolver(fakeSessions{session: stored, err: nil}, clk, []byte(testSecret))
	require.NoError(t, err)

	got, err := resolver.Resolve(context.Background(), bearerSession(t, token))
	require.NoError(t, err)
	require.Equal(t, principal.TypeJWT, got.Type)
	require.Equal(t, principal.SubjectTypeUser, got.Subject.Type)
	require.Equal(t, fixture.userID, got.Subject.ID)
	require.Equal(t, fixture.workspaceID, got.AuthorizedWorkspaceID)
	require.Equal(t, []string{fixture.permission}, got.Permissions)
	source, ok := got.Source.(principal.CLISessionSource)
	require.True(t, ok)
	require.Equal(t, fixture.sessionID, source.SessionID)
	require.Equal(t, "laptop", source.DeviceName)
	require.True(t, stored.ExpiresAt.Equal(source.ExpiresAt))
}

func TestResolverRejectsRevokedAndExpiredSessions(t *testing.T) {
	clk := clock.NewTestClock()
	fixture := newTokenFixture()
	sess := bearerSession(t, sign(t, testSecret, fixture.claims(clk)))

	resolver, err := NewResolver(fakeSessions{session: Session{}, err: ErrInactive}, clk, []byte(testSecret))
	require.NoError(t, err)
	_, err = resolver.Resolve(context.Background(), sess)
	requireCode(t, err, codes.Auth.Authentication.Malformed.URN())

	clk.Tick(2 * time.Hour)
	expiredResolver, err := NewResolver(fakeSessions{session: fixture.session(clk), err: nil}, clk, []byte(testSecret))
	require.NoError(t, err)
	_, err = expiredResolver.Resolve(context.Background(), sess)
	requireCode(t, err, codes.Auth.Authentication.Malformed.URN())
}

func TestResolverRejectsTokensThatDoNotMatchTheSession(t *testing.T) {
	tests := []struct {
		name     string
		claims   func(Claims) Claims
		session  func(Session) Session
		allowErr error
		want     codes.URN
	}{
		{
			name:     "workspace differs from the session row",
			claims:   func(c Claims) Claims { return c },
			session:  func(s Session) Session { s.WorkspaceID = uid.New(uid.WorkspacePrefix); return s },
			allowErr: nil,
			want:     codes.Auth.Authentication.Malformed.URN(),
		},
		{
			name:     "subject differs from the session row",
			claims:   func(c Claims) Claims { return c },
			session:  func(s Session) Session { s.UserID = uid.New(uid.TestPrefix); return s },
			allowErr: nil,
			want:     codes.Auth.Authentication.Malformed.URN(),
		},
		{
			name:     "empty permissions claim",
			claims:   func(c Claims) Claims { c.Permissions = nil; return c },
			session:  func(s Session) Session { return s },
			allowErr: nil,
			want:     codes.Auth.Authentication.Malformed.URN(),
		},
		{
			name:     "session lookup fails",
			claims:   func(c Claims) Claims { return c },
			session:  func(s Session) Session { return s },
			allowErr: errors.New("database unavailable"),
			want:     codes.App.Internal.ServiceUnavailable.URN(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clk := clock.NewTestClock()
			fixture := newTokenFixture()
			token := sign(t, testSecret, tt.claims(fixture.claims(clk)))
			resolver, err := NewResolver(fakeSessions{session: tt.session(fixture.session(clk)), err: tt.allowErr}, clk, []byte(testSecret))
			require.NoError(t, err)

			got, err := resolver.Resolve(context.Background(), bearerSession(t, token))
			require.Nil(t, got)
			requireCode(t, err, tt.want)
		})
	}
}

func TestResolverAcceptsEveryConfiguredSecret(t *testing.T) {
	secretOld := "test-cli-signing-secret-old-32-b!"
	secretNew := "test-cli-signing-secret-new-32-b!"
	secretOther := "test-cli-signing-secret-other-32!"
	tests := []struct {
		name     string
		secret   string
		accepted bool
	}{
		{name: "old secret", secret: secretOld, accepted: true},
		{name: "new secret", secret: secretNew, accepted: true},
		{name: "unconfigured secret", secret: secretOther, accepted: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clk := clock.NewTestClock()
			fixture := newTokenFixture()
			resolver, err := NewResolver(fakeSessions{session: fixture.session(clk), err: nil}, clk, []byte(secretOld), []byte(secretNew))
			require.NoError(t, err)

			got, err := resolver.Resolve(context.Background(), bearerSession(t, sign(t, tt.secret, fixture.claims(clk))))
			if !tt.accepted {
				require.Nil(t, got)
				requireCode(t, err, codes.Auth.Authentication.Malformed.URN())
				return
			}
			require.NoError(t, err)
			require.Equal(t, fixture.userID, got.Subject.ID)
		})
	}
}

func TestResolverIgnoresOtherBearers(t *testing.T) {
	clk := clock.NewTestClock()
	resolver, err := NewResolver(fakeSessions{session: Session{}, err: nil}, clk, []byte(testSecret))
	require.NoError(t, err)
	got, err := resolver.Resolve(context.Background(), bearerSession(t, "not-a-cli-token"))
	require.NoError(t, err)
	require.Nil(t, got)
}
