package deploy

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/unkeyed/unkey/pkg/ssrf"
)

// registryDialer resolves registry names itself so only checked addresses can
// reach the network. trustedAddresses is limited to Unkey's build registry.
type registryDialer struct {
	trustedAddresses map[string]struct{}
	lookupNetIP      func(context.Context, string, string) ([]netip.Addr, error)
	dialContext      func(context.Context, string, string) (net.Conn, error)
}

// newRegistryTransport clones the registry client's defaults and replaces its
// dialer so DNS resolution and the connection use the same approved address.
func newRegistryTransport(trustedRegistry string) *http.Transport {
	dialer := &net.Dialer{ //nolint:exhaustruct // Standard-library dialer defaults are intentional.
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	restrictedDialer := &registryDialer{
		trustedAddresses: trustedRegistryAddresses(trustedRegistry),
		lookupNetIP:      net.DefaultResolver.LookupNetIP,
		dialContext:      dialer.DialContext,
	}

	transport := remote.DefaultTransport.(*http.Transport).Clone()
	// A proxy would resolve and connect to the destination outside this dialer.
	transport.Proxy = nil
	transport.DialContext = restrictedDialer.DialContext
	return transport
}

// DialContext resolves untrusted registry hosts, filters every result, and
// dials an approved IP directly to prevent DNS rebinding between checks.
func (d *registryDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("invalid registry address: %w", err)
	}
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	canonicalAddress := net.JoinHostPort(host, port)
	if _, trusted := d.trustedAddresses[canonicalAddress]; trusted {
		return d.dialContext(ctx, network, address)
	}
	if isBlockedRegistryHostname(host) {
		return nil, fmt.Errorf("registry host %q is not public", host)
	}

	addresses, err := d.lookupNetIP(ctx, "ip", host)
	if err != nil {
		return nil, fmt.Errorf("resolve registry host: %w", err)
	}

	var dialErrors []error
	for _, resolvedAddress := range addresses {
		resolvedAddress = resolvedAddress.Unmap()
		if !isPublicRegistryAddress(resolvedAddress) {
			continue
		}
		conn, dialErr := d.dialContext(ctx, network, net.JoinHostPort(resolvedAddress.String(), port))
		if dialErr == nil {
			return conn, nil
		}
		dialErrors = append(dialErrors, dialErr)
	}
	if len(dialErrors) > 0 {
		return nil, fmt.Errorf("connect to public registry host: %w", errors.Join(dialErrors...))
	}
	return nil, fmt.Errorf("registry host %q has no public addresses", host)
}

// trustedRegistryAddresses returns the HTTP endpoints that may bypass public
// address checks because they belong to Unkey's configured build registry.
func trustedRegistryAddresses(registry string) map[string]struct{} {
	addresses := make(map[string]struct{})
	if registry == "" {
		return addresses
	}
	host, port, err := net.SplitHostPort(registry)
	if err == nil {
		addresses[net.JoinHostPort(strings.ToLower(strings.TrimSuffix(host, ".")), port)] = struct{}{}
		return addresses
	}
	host = strings.ToLower(strings.TrimSuffix(registry, "."))
	addresses[net.JoinHostPort(host, "443")] = struct{}{}
	addresses[net.JoinHostPort(host, "80")] = struct{}{}
	return addresses
}

// isBlockedRegistryHostname rejects names reserved for local and internal use.
func isBlockedRegistryHostname(host string) bool {
	if net.ParseIP(host) != nil {
		return false
	}
	return !strings.Contains(host, ".") ||
		strings.HasSuffix(host, ".localhost") ||
		strings.HasSuffix(host, ".local") ||
		strings.HasSuffix(host, ".internal") ||
		strings.HasSuffix(host, ".svc")
}

// isPublicRegistryAddress reports whether an address is safe for public OCI
// registry traffic.
func isPublicRegistryAddress(address netip.Addr) bool {
	return !ssrf.IsForbiddenAddr(address)
}
