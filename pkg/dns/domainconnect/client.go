package domainconnect

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/publicsuffix"
)

// The _domainconnect TXT value is set by whoever controls DNS for the user-
// submitted domain, so it is attacker-controlled. Without this allowlist, the
// control plane would fetch https://<any-host>/v2/<domain>/settings from its
// own network, i.e. SSRF. Add a new entry only when onboarding a provider.
var allowedDomainConnectHosts = map[string]struct{}{
	"api.cloudflare.com":       {},
	"domainconnect.vercel.com": {},
}

// Config holds the Domain Connect settings discovered for a domain.
// These fields are populated from two sources: DNS lookup (domain structure)
// and the provider's /v2/{zone}/settings JSON endpoint.
type Config struct {
	// Domain is the full domain as entered by the user (e.g. "api.example.com").
	Domain string
	// DomainRoot is the registrable domain / zone apex (eTLD+1), e.g. "example.com".
	// Used as the "domain" query parameter in the sync URL.
	DomainRoot string
	// Host is the subdomain part relative to DomainRoot (e.g. "api" for "api.example.com").
	// Empty for apex domains. Used as the "host" query parameter in the sync URL.
	Host string
	// URLSyncUX is the base URL for the provider's sync consent page, from the settings endpoint.
	// Example: "https://dash.cloudflare.com/domainconnect"
	URLSyncUX string
	// ProviderID is the stable machine-readable provider identifier (e.g. "cloudflare.com").
	// Use this for logic (e.g. apex domain gating), not the display name.
	ProviderID string
	// ProviderDisplayName is the human-readable provider name (e.g. "Cloudflare"),
	// from the settings endpoint. Shown to users in the dashboard.
	ProviderDisplayName string
}

// ProviderSettings is the response from a Domain Connect provider's settings endpoint
// (GET https://{host}/v2/{zone}/settings).
// Spec: https://github.com/Domain-Connect/spec/blob/master/Domain%20Connect%20Spec%20Draft.adoc
type ProviderSettings struct {
	// ProviderID is the unique identifier for the DNS provider (e.g. "cloudflare.com").
	ProviderID string `json:"providerId"`
	// ProviderName is the display name of the DNS provider (e.g. "Cloudflare").
	ProviderName string `json:"providerName"`
	// ProviderDisplayName is an optional per-domain display name for multi-brand providers.
	ProviderDisplayName string `json:"providerDisplayName"`
	// URLSyncUX is the base URL for the synchronous flow consent page.
	// Empty if the provider does not support synchronous flows.
	URLSyncUX string `json:"urlSyncUX"`
	// URLAsyncUX is the base URL for the asynchronous (OAuth) flow.
	// Empty if the provider does not support async flows.
	URLAsyncUX string `json:"urlAsyncUX"`
	// URLAPI is the URL prefix for the REST API used in async flows.
	URLAPI string `json:"urlAPI"`
	// URLControlPanel is a link to the provider's DNS management UI.
	// May contain %domain% as a placeholder for deep linking.
	URLControlPanel string `json:"urlControlPanel"`
	// Width is the desired popup window width in pixels (default 750).
	Width int `json:"width,omitempty"`
	// Height is the desired popup window height in pixels (default 750).
	Height int `json:"height,omitempty"`
	// NameServers lists the preferred authoritative nameservers for the zone.
	NameServers []string `json:"nameServers,omitempty"`
}

// discoverConfig performs Domain Connect discovery for the given domain.
// It looks up the _domainconnect TXT record, fetches provider settings,
// and returns the configuration needed to build a sync URL.
func discoverConfig(ctx context.Context, domain string) (*Config, error) {
	domain = normalizeDomain(domain)

	domainRoot, err := publicsuffix.EffectiveTLDPlusOne(domain)
	if err != nil {
		return nil, fmt.Errorf("identify domain root: %w", err)
	}

	host := ""
	if domain != domainRoot {
		host = strings.TrimSuffix(domain, "."+domainRoot)
	}

	dcHost, err := lookupDomainConnectRecord(ctx, domainRoot)
	if err != nil {
		return nil, err
	}

	if err := validateDomainConnectHost(dcHost); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnsupportedDomainConnectHost, err)
	}

	settingsURL := fmt.Sprintf("https://%s/v2/%s/settings", dcHost, domainRoot)
	var settings ProviderSettings
	if err := doJSON(ctx, settingsURL, &settings); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNoDomainConnectSettings, err)
	}

	if settings.URLSyncUX == "" {
		return nil, fmt.Errorf("%w: missing urlSyncUX", ErrNoDomainConnectSettings)
	}

	displayName := settings.ProviderDisplayName
	if displayName == "" {
		displayName = settings.ProviderName
	}

	return &Config{
		Domain:              domain,
		DomainRoot:          domainRoot,
		Host:                host,
		URLSyncUX:           settings.URLSyncUX,
		ProviderID:          settings.ProviderID,
		ProviderDisplayName: displayName,
	}, nil
}

// validateDomainConnectHost checks the TXT value against allowedDomainConnectHosts.
// The value may include a path (Cloudflare's is "api.cloudflare.com/client/v4/dns/domainconnect"),
// so we parse it as a URL and match u.Host exactly.
func validateDomainConnectHost(dcHost string) error {
	u, err := url.Parse("https://" + dcHost)
	if err != nil {
		return fmt.Errorf("invalid domainconnect host: %w", err)
	}
	host := strings.ToLower(u.Host)
	if _, ok := allowedDomainConnectHosts[host]; !ok {
		return fmt.Errorf("unsupported domainconnect host: %q", host)
	}
	return nil
}

// lookupDomainConnectRecord looks up the _domainconnect TXT record for a domain.
func lookupDomainConnectRecord(ctx context.Context, domain string) (string, error) {
	name := "_domainconnect." + domain

	records, err := net.DefaultResolver.LookupTXT(ctx, name)
	if err != nil {
		var dnsErr *net.DNSError
		if errors.As(err, &dnsErr) && dnsErr.IsNotFound {
			return "", ErrNoDomainConnectRecord
		}
		return "", err
	}

	if len(records) == 0 {
		return "", ErrNoDomainConnectRecord
	}

	return strings.Join(records, ""), nil
}

// domainConnectHTTPClient refuses redirects so an allowlisted provider host
// cannot bounce the request to a non-allowlisted destination.
var domainConnectHTTPClient = &http.Client{
	Timeout: 10 * time.Second,
	CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

// doJSON performs a GET request and decodes the JSON response.
func doJSON(ctx context.Context, url string, result any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := domainConnectHTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(result)
}
