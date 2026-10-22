package jwt

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegisteredClaimsUnmarshalAudience(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name    string
		payload string
		want    []string
	}{
		{
			name:    "scalar",
			payload: `{"aud":"https://api.unkey.com/oidc/vercel/vco_test"}`,
			want:    []string{"https://api.unkey.com/oidc/vercel/vco_test"},
		},
		{
			name:    "array",
			payload: `{"aud":["api.unkey.com","https://api.unkey.com"]}`,
			want:    []string{"api.unkey.com", "https://api.unkey.com"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var claims RegisteredClaims
			require.NoError(t, json.Unmarshal([]byte(test.payload), &claims))
			require.Equal(t, test.want, []string(claims.Audience))
		})
	}
}

func TestRegisteredClaimsRejectsInvalidAudienceShape(t *testing.T) {
	t.Parallel()

	var claims RegisteredClaims
	require.Error(t, json.Unmarshal([]byte(`{"aud":42}`), &claims))
}
