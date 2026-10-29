package grant

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auth/clisession"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	tokenjwt "github.com/unkeyed/unkey/pkg/jwt"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestRefreshTokenCarriesItsFamily(t *testing.T) {
	family := newRefreshFamily()
	token := newRefreshToken(family)
	require.True(t, strings.HasPrefix(token, refreshTokenPrefix+family+"_"))
	require.Len(t, token, len(refreshTokenPrefix)+refreshFamilyLength+1+refreshSecretLength)

	parsed, ok := refreshTokenFamily(token)
	require.True(t, ok)
	require.Equal(t, family, parsed)
	require.NotEqual(t, token, newRefreshToken(family))
}

func TestRefreshTokenFamilyRejectsMalformedTokens(t *testing.T) {
	family := uid.Secure(refreshFamilyLength)
	secret := uid.Secure(refreshSecretLength)
	tests := []struct {
		name  string
		token string
	}{
		{name: "empty", token: ""},
		{name: "legacy format without family", token: refreshTokenPrefix + secret},
		{name: "missing prefix", token: family + "_" + secret},
		{name: "wrong prefix", token: "uk_" + family + "_" + secret},
		{name: "short family", token: refreshTokenPrefix + family[1:] + "_" + secret},
		{name: "short secret", token: refreshTokenPrefix + family + "_" + secret[1:]},
		{name: "extra delimiter", token: refreshTokenPrefix + family + "_" + secret[:16] + "_" + secret[17:]},
		{name: "non alphanumeric family", token: refreshTokenPrefix + family[1:] + "-" + "_" + secret},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := refreshTokenFamily(tt.token)
			require.False(t, ok)
		})
	}
}

func TestRefreshRejectsMalformedTokenWithoutRevoking(t *testing.T) {
	signer, err := tokenjwt.NewHS256Signer[clisession.Claims]([]byte(testSecret))
	require.NoError(t, err)
	svc := &Service{
		Clock:            clock.NewTestClock(),
		Memberships:      mem{},
		DashboardBaseURL: "http://localhost:3000",
		Signer:           signer,
	}

	for _, token := range []string{"", "not-a-token", refreshTokenPrefix + uid.Secure(refreshSecretLength)} {
		_, err := svc.Refresh(context.Background(), RefreshRequest{RefreshToken: token})
		require.Equal(t, codes.Auth.Authentication.CredentialRejected.URN(), codeOf(t, err))
		require.Equal(t, "The refresh token is invalid. Run unkey login again.", fault.UserFacingMessage(err))
	}
}
