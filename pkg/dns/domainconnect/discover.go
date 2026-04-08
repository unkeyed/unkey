package domainconnect

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

const (
	// providerID identifies Unkey in the Domain Connect template registry.
	// Must match the providerId in our published template at
	// https://github.com/Domain-Connect/Templates
	providerID = "unkey.com"
	// serviceID identifies the specific service template (one provider can have many).
	serviceID = "custom-domain"
	// keyID is the subdomain under the provider's syncPubKeyDomain where the public
	// key is published as a TXT record. Providers fetch {keyID}.{syncPubKeyDomain}
	// to verify signatures.
	keyID = "_dcpubkeyv1"
)

// Known DNS provider IDs returned by the Domain Connect settings endpoint.
// Values confirmed by querying each provider's /v2/{zone}/settings endpoint.
const (
	ProviderCloudflare = "cloudflare.com"
	ProviderVercel     = "vercel.com"
)

// Result holds the Domain Connect discovery and URL generation output.
type Result struct {
	// ProviderID is the stable machine-readable provider identifier (e.g. "cloudflare.com").
	// Use this for programmatic decisions (e.g. apex domain gating).
	ProviderID string
	// ProviderName is the display name of the DNS provider (e.g. "Cloudflare").
	// Use this for UI display only.
	ProviderName string
	// URL is the fully signed Domain Connect redirect URL.
	URL string
}

// Discover checks if the domain's DNS provider supports Domain Connect and,
// if so, builds a signed redirect URL. Returns nil if the provider doesn't
// support Domain Connect.
func Discover(ctx context.Context, domain string, privateKeyPEM []byte, params map[string]string, redirectURL string) (*Result, error) {
	cfg, err := discoverConfig(ctx, domain)
	if err != nil {
		if errors.Is(err, ErrNoDomainConnectRecord) || errors.Is(err, ErrNoDomainConnectSettings) {
			return nil, nil
		}
		return nil, fmt.Errorf("domain connect discovery: %w", err)
	}

	syncURL, err := buildSyncURL(cfg, params, redirectURL)
	if err != nil {
		return nil, fmt.Errorf("build sync URL: %w", err)
	}

	signedURL, err := signSyncURL(syncURL, privateKeyPEM, keyID)
	if err != nil {
		return nil, fmt.Errorf("sign sync URL: %w", err)
	}

	return &Result{
		ProviderID:   cfg.ProviderID,
		ProviderName: cfg.ProviderDisplayName,
		URL:          signedURL,
	}, nil
}

// buildSyncURL constructs the unsigned Domain Connect sync URL.
func buildSyncURL(cfg *Config, params map[string]string, redirectURL string) (string, error) {
	baseURL := ensureScheme(cfg.URLSyncUX)
	rawURL := fmt.Sprintf("%s/v2/domainTemplates/providers/%s/services/%s/apply",
		baseURL, providerID, serviceID)

	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	q := u.Query()
	q.Set("domain", cfg.DomainRoot)
	if cfg.Host != "" {
		q.Set("host", cfg.Host)
	}
	for k, v := range params {
		q.Set(k, v)
	}
	if redirectURL != "" {
		q.Set("redirect_uri", redirectURL)
	}

	u.RawQuery = q.Encode()
	return u.String(), nil
}

// ensureScheme adds https:// prefix if not already present.
func ensureScheme(urlStr string) string {
	if strings.HasPrefix(urlStr, "https://") || strings.HasPrefix(urlStr, "http://") {
		return urlStr
	}
	return "https://" + urlStr
}

// ValidatePrivateKey checks that the PEM bytes contain a valid RSA private key.
// Supports both PKCS1 and PKCS8 formats.
func ValidatePrivateKey(pemBytes []byte) error {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return errors.New("failed to decode PEM block")
	}
	_, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err == nil {
		return nil
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse private key (tried PKCS1 and PKCS8): %w", err)
	}
	if _, ok := key.(*rsa.PrivateKey); !ok {
		return errors.New("private key is not RSA")
	}
	return nil
}
