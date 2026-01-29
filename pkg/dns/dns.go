// Package dns provides DNS lookup utilities using Cloudflare's 1.1.1.1 resolver.
package dns

import (
	"context"
	"errors"
	"net"
	"strings"
	"time"
)

const (
	// CloudflareDNS is Cloudflare's public DNS server.
	CloudflareDNS = "1.1.1.1:53"

	// DefaultTimeout is the default timeout for DNS lookups.
	DefaultTimeout = 10 * time.Second
)

// Resolver uses Cloudflare's 1.1.1.1 for consistent lookups across environments.
var Resolver = newResolver(CloudflareDNS, DefaultTimeout)

func newResolver(server string, timeout time.Duration) *net.Resolver {
	return &net.Resolver{
		PreferGo:     true,
		StrictErrors: false,
		Dial: func(ctx context.Context, _, _ string) (net.Conn, error) {
			d := net.Dialer{} //nolint:exhaustruct
			d.Timeout = timeout
			return d.DialContext(ctx, "udp", server)
		},
	}
}

// IsNotFoundError returns true if the error indicates the record doesn't exist
// (NXDOMAIN, no such host, etc.). These are expected when a record hasn't been
// configured yet, not actual failures.
func IsNotFoundError(err error) bool {
	var dnsErr *net.DNSError
	return err != nil && errors.As(err, &dnsErr) && dnsErr.IsNotFound
}

// LookupTXT looks up TXT records for the given domain.
func LookupTXT(ctx context.Context, domain string) ([]string, error) {
	return Resolver.LookupTXT(ctx, domain)
}

// LookupCNAME looks up the CNAME record for the given domain.
// The returned CNAME is normalized (trailing dot removed, lowercased).
func LookupCNAME(ctx context.Context, domain string) (string, error) {
	cname, err := Resolver.LookupCNAME(ctx, domain)
	if err != nil {
		return "", err
	}

	return strings.ToLower(strings.TrimSuffix(cname, ".")), nil
}
