package domainconnect

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateDomainConnectHost(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		dcHost  string
		wantErr bool
	}{
		{"cloudflare bare host", "api.cloudflare.com", false},
		{"cloudflare with path", "api.cloudflare.com/client/v4/dns/domainconnect", false},
		{"cloudflare uppercase", "API.CLOUDFLARE.COM", false},
		{"vercel bare host", "domainconnect.vercel.com", false},
		{"vercel mixed case", "DomainConnect.Vercel.com", false},

		{"attacker host", "attacker.example", true},
		{"attacker with allowed host in path", "attacker.example/api.cloudflare.com", true},
		{"embedded userinfo", "api.cloudflare.com@attacker.example", true},
		{"port specified", "api.cloudflare.com:8443", true},
		{"loopback ipv4", "127.0.0.1", true},
		{"private ipv4", "169.254.169.254", true},
		{"ipv6 literal", "[::1]", true},
		{"empty", "", true},
		{"sibling domain", "cloudflare.com", true},
		{"suffix trick", "api.cloudflare.com.attacker.example", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateDomainConnectHost(tt.dcHost)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
