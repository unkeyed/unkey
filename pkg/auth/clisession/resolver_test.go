package clisession

import (
	"context"
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

func TestResolverBuildsPrincipalFromPermissionsClaim(t *testing.T) {
	clk := clock.NewTestClock()
	signer, err := tokenjwt.NewHS256Signer[Claims]([]byte(testSecret))
	require.NoError(t, err)
	token, err := signer.Sign(Claims{
		RegisteredClaims: tokenjwt.RegisteredClaims{
			Issuer:    Issuer,
			Subject:   "user_dev",
			Audience:  []string{Audience},
			IssuedAt:  clk.Now().Unix(),
			ExpiresAt: clk.Now().Add(time.Hour).Unix(),
		},
		WorkspaceID: "ws_123",
		SessionID:   "clis_123",
		Permissions: []string{"unkey:v1:ws_123:projects/*/keyspaces/*/keys/*#read"},
		TokenUse:    TokenUse,
		Name:        "Dev",
	})
	require.NoError(t, err)

	resolver, err := NewResolver(fakeSessions{session: Session{
		UserID:      "user_dev",
		WorkspaceID: "ws_123",
		DeviceName:  "laptop",
	}}, clk, []byte(testSecret))
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	sess := &zen.Session{}
	require.NoError(t, sess.Init(httptest.NewRecorder(), req, 0))

	got, err := resolver.Resolve(context.Background(), sess)
	require.NoError(t, err)
	require.Equal(t, principal.TypeJWT, got.Type)
	require.Equal(t, principal.SubjectTypeUser, got.Subject.Type)
	require.Equal(t, "user_dev", got.Subject.ID)
	require.Equal(t, "ws_123", got.AuthorizedWorkspaceID)
	require.Equal(t, []string{"unkey:v1:ws_123:projects/*/keyspaces/*/keys/*#read"}, got.Permissions)
	source, ok := got.Source.(principal.CLISessionSource)
	require.True(t, ok)
	require.Equal(t, "clis_123", source.SessionID)
	require.Equal(t, "laptop", source.DeviceName)
}

func TestResolverRejectsRevokedAndExpiredSessions(t *testing.T) {
	clk := clock.NewTestClock()
	signer, err := tokenjwt.NewHS256Signer[Claims]([]byte(testSecret))
	require.NoError(t, err)
	token, err := signer.Sign(Claims{
		RegisteredClaims: tokenjwt.RegisteredClaims{
			Issuer:    Issuer,
			Subject:   "user_dev",
			Audience:  []string{Audience},
			IssuedAt:  clk.Now().Unix(),
			ExpiresAt: clk.Now().Add(time.Hour).Unix(),
		},
		WorkspaceID: "ws_123",
		SessionID:   "clis_123",
		Permissions: []string{"unkey:v1:ws_123:projects/*/keyspaces/*/keys/*#read"},
		TokenUse:    TokenUse,
	})
	require.NoError(t, err)

	resolver, err := NewResolver(fakeSessions{err: ErrInactive}, clk, []byte(testSecret))
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	sess := &zen.Session{}
	require.NoError(t, sess.Init(httptest.NewRecorder(), req, 0))

	_, err = resolver.Resolve(context.Background(), sess)
	code, ok := fault.GetCode(err)
	require.True(t, ok)
	require.Equal(t, codes.Auth.Authentication.Malformed.URN(), code)

	clk.Tick(2 * time.Hour)
	expiredResolver, err := NewResolver(fakeSessions{session: Session{UserID: "user_dev", WorkspaceID: "ws_123"}}, clk, []byte(testSecret))
	require.NoError(t, err)
	_, err = expiredResolver.Resolve(context.Background(), sess)
	code, ok = fault.GetCode(err)
	require.True(t, ok)
	require.Equal(t, codes.Auth.Authentication.Malformed.URN(), code)
}

func TestResolverIgnoresOtherBearers(t *testing.T) {
	clk := clock.NewTestClock()
	resolver, err := NewResolver(fakeSessions{}, clk, []byte(testSecret))
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Authorization", "Bearer not-a-cli-token")
	sess := &zen.Session{}
	require.NoError(t, sess.Init(httptest.NewRecorder(), req, 0))
	got, err := resolver.Resolve(context.Background(), sess)
	require.NoError(t, err)
	require.Nil(t, got)
}
