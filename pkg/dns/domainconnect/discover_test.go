package domainconnect

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildSyncURL_ApexDomain(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Domain:     "example.com",
		DomainRoot: "example.com",
		Host:       "",
		URLSyncUX:  "https://dash.cloudflare.com/cdn-cgi/access/domain-connect",
	}

	syncURL, err := buildSyncURL(cfg, map[string]string{
		"target":            "abc123.unkey-dns.com",
		"verificationToken": "tok_secret",
	}, "https://app.unkey.com/ws/projects/p1/settings")
	require.NoError(t, err)

	u, err := url.Parse(syncURL)
	require.NoError(t, err)
	q := u.Query()

	require.Equal(t, "example.com", q.Get("domain"))
	require.Empty(t, q.Get("host"), "apex domain must not have host param")
	require.Equal(t, "abc123.unkey-dns.com", q.Get("target"))
	require.Equal(t, "tok_secret", q.Get("verificationToken"))
	require.Equal(t, "https://app.unkey.com/ws/projects/p1/settings", q.Get("redirect_uri"))
	require.Contains(t, u.Path, "/v2/domainTemplates/providers/unkey.com/services/custom-domain/apply")
}

func TestBuildSyncURL_Subdomain(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Domain:     "api.example.com",
		DomainRoot: "example.com",
		Host:       "api",
		URLSyncUX:  "https://dash.cloudflare.com/cdn-cgi/access/domain-connect",
	}

	syncURL, err := buildSyncURL(cfg, map[string]string{
		"target": "abc123.unkey-dns.com",
	}, "")
	require.NoError(t, err)

	u, err := url.Parse(syncURL)
	require.NoError(t, err)
	q := u.Query()

	require.Equal(t, "example.com", q.Get("domain"))
	require.Equal(t, "api", q.Get("host"))
	require.Equal(t, "abc123.unkey-dns.com", q.Get("target"))
	require.Empty(t, q.Get("redirect_uri"), "empty redirect URL must not be added")
}

func TestBuildSyncURL_EnsureScheme(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Domain:     "example.com",
		DomainRoot: "example.com",
		Host:       "",
		URLSyncUX:  "dash.cloudflare.com/cdn-cgi/access/domain-connect",
	}

	syncURL, err := buildSyncURL(cfg, nil, "")
	require.NoError(t, err)

	u, err := url.Parse(syncURL)
	require.NoError(t, err)
	require.Equal(t, "https", u.Scheme, "must add https scheme when missing")
}

func TestValidatePrivateKey_PKCS8(t *testing.T) {
	t.Parallel()
	pemBytes, _ := generateTestKey(t)
	require.NoError(t, ValidatePrivateKey(pemBytes))
}

func TestValidatePrivateKey_PKCS1(t *testing.T) {
	t.Parallel()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	require.NoError(t, ValidatePrivateKey(pemBytes))
}

func TestValidatePrivateKey_Invalid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		pem  []byte
	}{
		{"empty", nil},
		{"garbage", []byte("not a pem")},
		{"wrong block", []byte("-----BEGIN CERTIFICATE-----\nMQ==\n-----END CERTIFICATE-----")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Error(t, ValidatePrivateKey(tt.pem))
		})
	}
}

func TestEnsureScheme(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{"example.com", "https://example.com"},
		{"https://example.com", "https://example.com"},
		{"http://example.com", "http://example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.expected, ensureScheme(tt.input))
		})
	}
}
