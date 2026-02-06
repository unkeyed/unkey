---
title: certmanager
description: "provides dynamic TLS certificate management for the frontline"
---

Package certmanager provides dynamic TLS certificate management for the frontline.

The certificate manager is responsible for:

  - Loading TLS certificates for custom domains from Vault
  - Caching certificates to minimize Vault lookups
  - Providing certificates to the TLS handshake handler
  - Falling back to a default certificate when domain certificates are unavailable

### Certificate Loading

Certificates are stored in Vault with encryption and retrieved on-demand during TLS handshakes. The manager caches certificates to avoid repeated Vault lookups for the same domain.

### Multi-Tenant Support

Each tenant can have multiple custom domains, each with their own certificate. The manager looks up certificates by SNI (Server Name Indication) from the TLS ClientHello, allowing it to serve the correct certificate for each domain.

### Fallback Behavior

If a certificate cannot be found for a domain, the manager returns a default certificate (e.g., for \*.basedomain.com) to allow the TLS handshake to complete. The request will still be routed correctly, but may show a certificate warning in browsers for custom domains.

## Types

### type Config

```go
type Config struct {
	DB db.Database

	Vault vaultv1connect.VaultServiceClient

	TLSCertificateCache cache.Cache[string, tls.Certificate]
}
```

### type Service

```go
type Service interface {
	// GetCertificate returns a certificate for the given domain.
	GetCertificate(ctx context.Context, domain string) (*tls.Certificate, error)
}
```

