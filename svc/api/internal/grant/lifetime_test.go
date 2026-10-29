package grant

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auth/clisession"
	"github.com/unkeyed/unkey/pkg/clock"
	tokenjwt "github.com/unkeyed/unkey/pkg/jwt"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestSessionExpiry(t *testing.T) {
	createdAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	require.Equal(t, createdAt+sessionIdle.Milliseconds(), sessionExpiry(createdAt, createdAt))

	day := (24 * time.Hour).Milliseconds()
	require.Equal(t, createdAt+10*day+sessionIdle.Milliseconds(), sessionExpiry(createdAt, createdAt+10*day))

	capped := createdAt + sessionMax.Milliseconds()
	require.Equal(t, capped, sessionExpiry(createdAt, capped-day))
	require.Equal(t, capped, sessionExpiry(createdAt, capped+day))
}

func TestAccessTokenLastsFifteenMinutes(t *testing.T) {
	require.Equal(t, 15*time.Minute, accessTTL)
	clk := clock.NewTestClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	signer, err := tokenjwt.NewHS256Signer[clisession.Claims]([]byte(testSecret))
	require.NoError(t, err)
	svc := &Service{Clock: clk, Signer: signer}

	token, err := svc.signAccessToken(uid.New(uid.TestPrefix), "Ada", uid.New(uid.WorkspacePrefix), uid.New(uid.CLISessionPrefix), "laptop", []string{"api.*.read_api"})
	require.NoError(t, err)
	verifier, err := tokenjwt.NewHS256Verifier[clisession.Claims]([]byte(testSecret),
		tokenjwt.WithIssuer(clisession.Issuer),
		tokenjwt.WithAudience(clisession.Audience),
		tokenjwt.WithClock(clk),
	)
	require.NoError(t, err)
	claims, err := verifier.Verify(token)
	require.NoError(t, err)
	require.Equal(t, clk.Now().Add(15*time.Minute).Unix(), claims.ExpiresAt)
	require.Equal(t, "Ada", claims.Name)
}
