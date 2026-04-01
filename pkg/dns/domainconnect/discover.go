// Package domainconnect wraps the Domain Connect protocol for custom domain setup.
//
// It uses github.com/railwayapp/domainconnect-go for discovery and URL signing,
// providing a simplified API for Unkey's custom domain flow.
package domainconnect

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"

	dc "github.com/railwayapp/domainconnect-go"
)

const (
	providerID = "unkey.com"
	serviceID  = "custom-domain"
	keyID      = "_dcpubkeyv1"
)

// Result holds the Domain Connect discovery and URL generation output.
type Result struct {
	// ProviderName is the display name of the DNS provider (e.g. "Cloudflare").
	ProviderName string
	// URL is the fully signed Domain Connect redirect URL.
	URL string
}

// Discover checks if the domain's DNS provider supports Domain Connect and,
// if so, builds a signed redirect URL. Returns nil if the provider doesn't
// support Domain Connect.
func Discover(ctx context.Context, domain string, privateKeyPEM []byte, params map[string]string, redirectURL string) (*Result, error) {
	client := dc.New()

	cfg, err := client.GetDomainConfig(ctx, domain)
	if err != nil {
		if errors.Is(err, dc.ErrNoDomainConnectRecord) {
			return nil, nil
		}
		return nil, fmt.Errorf("domain connect discovery: %w", err)
	}

	syncURL, err := client.GetSyncURL(ctx, dc.SyncURLOptions{
		Config:        cfg,
		ProviderID:    providerID,
		ServiceID:     serviceID,
		Params:        params,
		RedirectURL:   redirectURL,
		State:         "",
		GroupIDs:      nil,
		PrivateKey:    privateKeyPEM,
		KeyID:         keyID,
		ForceProvider: false,
	})
	if err != nil {
		return nil, fmt.Errorf("domain connect sync URL: %w", err)
	}

	return &Result{
		ProviderName: cfg.ProviderDisplayName,
		URL:          syncURL,
	}, nil
}

// ValidatePrivateKey checks that the PEM bytes contain a valid RSA private key.
// Supports both PKCS1 and PKCS8 formats.
func ValidatePrivateKey(pemBytes []byte) error {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return errors.New("failed to decode PEM block")
	}
	// Try PKCS1 first, then PKCS8
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
