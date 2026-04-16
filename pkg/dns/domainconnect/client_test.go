package domainconnect

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateDomainConnectHost(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		dcHost   string
		wantErr  bool
		wantBase string
	}{
		{name: "cloudflare bare host", dcHost: "api.cloudflare.com", wantBase: "api.cloudflare.com"},
		{name: "cloudflare with path", dcHost: "api.cloudflare.com/client/v4/dns/domainconnect", wantBase: "api.cloudflare.com/client/v4/dns/domainconnect"},
		{name: "vercel bare host", dcHost: "domainconnect.vercel.com", wantBase: "domainconnect.vercel.com"},

		{name: "cloudflare uppercase rejected", dcHost: "API.CLOUDFLARE.COM", wantErr: true},
		{name: "vercel mixed case rejected", dcHost: "DomainConnect.Vercel.com", wantErr: true},
		{name: "query stripped", dcHost: "api.cloudflare.com?evil=1", wantBase: "api.cloudflare.com"},
		{name: "fragment stripped", dcHost: "api.cloudflare.com#evil", wantBase: "api.cloudflare.com"},
		{name: "query after path stripped", dcHost: "api.cloudflare.com/client/v4/dns/domainconnect?evil=1", wantBase: "api.cloudflare.com/client/v4/dns/domainconnect"},

		{name: "attacker host", dcHost: "attacker.example", wantErr: true},
		{name: "attacker with allowed host in path", dcHost: "attacker.example/api.cloudflare.com", wantErr: true},
		{name: "embedded userinfo", dcHost: "api.cloudflare.com@attacker.example", wantErr: true},
		{name: "port specified", dcHost: "api.cloudflare.com:8443", wantErr: true},
		{name: "loopback ipv4", dcHost: "127.0.0.1", wantErr: true},
		{name: "private ipv4", dcHost: "169.254.169.254", wantErr: true},
		{name: "ipv6 literal", dcHost: "[::1]", wantErr: true},
		{name: "empty", dcHost: "", wantErr: true},
		{name: "sibling domain", dcHost: "cloudflare.com", wantErr: true},
		{name: "suffix trick", dcHost: "api.cloudflare.com.attacker.example", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			base, err := validateDomainConnectHost(tt.dcHost)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantBase, base)
		})
	}
}
