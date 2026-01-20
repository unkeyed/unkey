package providers

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-acme/lego/v4/challenge"
)

var _ challenge.Provider = (*MultiProvider)(nil)
var _ challenge.ProviderTimeout = (*MultiProvider)(nil)

// ZoneRoute maps a DNS zone (domain suffix) to a provider.
type ZoneRoute struct {
	Zone     string             // e.g., "unkey.app", "unkey.cloud"
	Provider challenge.Provider // the provider that handles this zone
}

// MultiProvider routes DNS-01 challenges to the appropriate provider based on domain suffix.
// It implements challenge.Provider and challenge.ProviderTimeout.
type MultiProvider struct {
	routes []ZoneRoute
}

// NewMultiProvider creates a provider that routes to different backends based on domain zone.
// Routes are matched by longest suffix match (most specific wins).
func NewMultiProvider(routes []ZoneRoute) (*MultiProvider, error) {
	if len(routes) == 0 {
		return nil, fmt.Errorf("at least one zone route is required")
	}
	return &MultiProvider{routes: routes}, nil
}

// providerFor finds the provider for a domain using longest suffix match.
func (m *MultiProvider) providerFor(domain string) (challenge.Provider, error) {
	domain = strings.TrimPrefix(domain, "*.")
	domain = strings.TrimSuffix(domain, ".")
	domain = strings.ToLower(domain)

	var bestMatch ZoneRoute
	bestLen := 0

	for _, route := range m.routes {
		zone := strings.ToLower(route.Zone)
		if domain == zone || strings.HasSuffix(domain, "."+zone) {
			if len(zone) > bestLen {
				bestMatch = route
				bestLen = len(zone)
			}
		}
	}

	if bestLen == 0 {
		zones := make([]string, len(m.routes))
		for i, r := range m.routes {
			zones[i] = r.Zone
		}
		return nil, fmt.Errorf("no DNS provider mapping for domain=%q (configured zones: %s)", domain, strings.Join(zones, ", "))
	}

	return bestMatch.Provider, nil
}

// Present creates a DNS TXT record by delegating to the appropriate provider.
func (m *MultiProvider) Present(domain, token, keyAuth string) error {
	p, err := m.providerFor(domain)
	if err != nil {
		return err
	}
	return p.Present(domain, token, keyAuth)
}

// CleanUp removes the DNS TXT record by delegating to the appropriate provider.
func (m *MultiProvider) CleanUp(domain, token, keyAuth string) error {
	p, err := m.providerFor(domain)
	if err != nil {
		return err
	}
	return p.CleanUp(domain, token, keyAuth)
}

// Timeout returns the maximum timeout and minimum interval across all providers.
func (m *MultiProvider) Timeout() (timeout, interval time.Duration) {
	var maxTimeout, minInterval time.Duration

	for _, route := range m.routes {
		if pt, ok := route.Provider.(challenge.ProviderTimeout); ok {
			t, i := pt.Timeout()
			if t > maxTimeout {
				maxTimeout = t
			}
			if minInterval == 0 || i < minInterval {
				minInterval = i
			}
		}
	}

	if maxTimeout == 0 {
		maxTimeout = 5 * time.Minute
	}
	if minInterval == 0 {
		minInterval = 10 * time.Second
	}

	return maxTimeout, minInterval
}
