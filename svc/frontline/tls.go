package frontline

import (
	"context"
	"crypto/tls"
	"fmt"

	"github.com/unkeyed/unkey/pkg/logger"
	pkgtls "github.com/unkeyed/unkey/pkg/tls"
	"github.com/unkeyed/unkey/svc/frontline/services/certmanager"
)

// buildTlsConfig creates a TLS configuration for the frontline server.
//
// The function supports three modes:
//   - Disabled: TLS is explicitly disabled via config
//   - Dynamic: Certificates are fetched from Vault via the cert manager (production)
//   - Static: Certificates are loaded from filesystem (development)
//
// Dynamic certificates are preferred when Vault is configured because they support
// per-domain certificates without server restarts. Static files are a fallback for
// development environments or when Vault is unavailable.
//
// Returns nil TLS config when disabled, or an error if TLS is required but no
// certificate source is configured.
func buildTlsConfig(cfg Config, certManager certmanager.Service) (*tls.Config, error) {

	tlsDisabled := cfg.TLS != nil && cfg.TLS.Disabled

	if tlsDisabled {
		logger.Warn("TLS explicitly disabled via config")

		return nil, nil
	}

	if certManager != nil {
		// Production mode: dynamic certificates from database/vault

		logger.Info("TLS configured with dynamic certificate manager")

		//nolint:exhaustruct
		return &tls.Config{
			GetCertificate: func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
				return certManager.GetCertificate(context.Background(), hello.ServerName)
			},
			MinVersion: tls.VersionTLS12,
			// Enable session resumption for faster subsequent connections
			// Session tickets allow clients to skip the full TLS handshake
			SessionTicketsDisabled: false,
			// Let Go's TLS implementation choose optimal cipher suites
			// This prefers TLS 1.3 when available (1-RTT vs 2-RTT for TLS 1.2)
			PreferServerCipherSuites: false,
		}, nil
	}
	if cfg.TLS != nil && cfg.TLS.CertFile != "" && cfg.TLS.KeyFile != "" {
		// Dev mode: static file-based certificate
		logger.Info("TLS configured with static certificate files",
			"certFile", cfg.TLS.CertFile,
			"keyFile", cfg.TLS.KeyFile)
		return pkgtls.NewFromFiles(cfg.TLS.CertFile, cfg.TLS.KeyFile)
	}

	return nil, fmt.Errorf("TLS is required but no certificate source configured: " +
		"either enable Vault for dynamic certificates, provide [tls] cert_file and key_file, " +
		"or explicitly disable TLS with [tls] disabled = true")

}
