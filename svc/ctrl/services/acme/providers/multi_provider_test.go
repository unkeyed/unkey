package providers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type mockProvider struct {
	zone     string
	timeout  time.Duration
	interval time.Duration
}

func (m *mockProvider) Present(_, _, _ string) error { return nil }
func (m *mockProvider) CleanUp(_, _, _ string) error { return nil }
func (m *mockProvider) Timeout() (time.Duration, time.Duration) {
	return m.timeout, m.interval
}

func TestMultiProvider_ProviderFor(t *testing.T) {
	cfProvider := &mockProvider{zone: "cloudflare", timeout: 5 * time.Minute, interval: 10 * time.Second}
	r53Provider := &mockProvider{zone: "route53", timeout: 3 * time.Minute, interval: 5 * time.Second}

	mp, err := NewMultiProvider([]ZoneRoute{
		{Zone: "example.com", Provider: cfProvider},
		{Zone: "example.org", Provider: r53Provider},
	})
	require.NoError(t, err)

	tests := []struct {
		name     string
		domain   string
		expected *mockProvider
	}{
		{"exact match cloudflare", "example.com", cfProvider},
		{"exact match route53", "example.org", r53Provider},
		{"subdomain cloudflare", "foo.example.com", cfProvider},
		{"subdomain route53", "bar.example.org", r53Provider},
		{"wildcard cloudflare", "*.example.com", cfProvider},
		{"wildcard route53", "*.example.org", r53Provider},
		{"deep subdomain", "a.b.c.example.com", cfProvider},
		{"regional subdomain", "us-west-2.aws.example.org", r53Provider},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := mp.providerFor(tt.domain)
			require.NoError(t, err)
			require.Equal(t, tt.expected, p)
		})
	}
}

func TestMultiProvider_ProviderFor_NoMatch(t *testing.T) {
	mp, err := NewMultiProvider([]ZoneRoute{
		{Zone: "example.com", Provider: &mockProvider{}},
	})
	require.NoError(t, err)

	_, err = mp.providerFor("other.com")
	require.Error(t, err)
	require.Contains(t, err.Error(), "no DNS provider mapping")
}

func TestMultiProvider_LongestSuffixMatch(t *testing.T) {
	shortProvider := &mockProvider{zone: "short"}
	longProvider := &mockProvider{zone: "long"}

	mp, err := NewMultiProvider([]ZoneRoute{
		{Zone: "org", Provider: shortProvider},
		{Zone: "example.org", Provider: longProvider},
	})
	require.NoError(t, err)

	p, err := mp.providerFor("foo.example.org")
	require.NoError(t, err)
	require.Equal(t, longProvider, p, "should pick longest matching suffix")
}

func TestMultiProvider_Timeout(t *testing.T) {
	mp, err := NewMultiProvider([]ZoneRoute{
		{Zone: "a.com", Provider: &mockProvider{timeout: 3 * time.Minute, interval: 10 * time.Second}},
		{Zone: "b.com", Provider: &mockProvider{timeout: 5 * time.Minute, interval: 5 * time.Second}},
	})
	require.NoError(t, err)

	timeout, interval := mp.Timeout()
	require.Equal(t, 5*time.Minute, timeout, "should use max timeout")
	require.Equal(t, 5*time.Second, interval, "should use min interval")
}

func TestNewMultiProvider_RequiresRoutes(t *testing.T) {
	_, err := NewMultiProvider(nil)
	require.Error(t, err)

	_, err = NewMultiProvider([]ZoneRoute{})
	require.Error(t, err)
}
