package api

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestConfig_ValidateJWTSecretMinimumLength verifies the API rejects JWT
// secrets shorter than the HS256 entropy requirement (256 bits). Accepting a
// shorter secret would weaken signature security across every token signed or
// verified by this node.
func TestConfig_ValidateJWTSecretMinimumLength(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		secrets []string
		wantErr bool
	}{
		{
			name:    "empty list disables JWT and passes",
			secrets: nil,
			wantErr: false,
		},
		{
			name:    "32 byte secret meets the minimum",
			secrets: []string{strings.Repeat("a", 32)},
			wantErr: false,
		},
		{
			name:    "31 byte secret is rejected",
			secrets: []string{strings.Repeat("a", 31)},
			wantErr: true,
		},
		{
			name:    "second secret too short still rejected",
			secrets: []string{strings.Repeat("a", 32), strings.Repeat("b", 16)},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := &Config{JWTSecrets: tt.secrets}
			err := cfg.Validate()
			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), "jwt_secrets")
				return
			}
			require.NoError(t, err)
		})
	}
}
