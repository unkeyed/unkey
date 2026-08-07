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

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/unkeyed/unkey/pkg/fault"
)

// ImageResolver resolves a mutable OCI image reference to an immutable digest.
type ImageResolver interface {
	Resolve(ctx context.Context, imageReference string) (string, error)
}

type ImageResolverFunc func(ctx context.Context, imageReference string) (string, error)

func (f ImageResolverFunc) Resolve(ctx context.Context, imageReference string) (string, error) {
	return f(ctx, imageReference)
}

type ociImageResolver struct {
	internalRepository string
	internalAuth       authn.Authenticator
	publicTransport    http.RoundTripper
	internalTransport  http.RoundTripper
	options            []remote.Option
}

var nonPublicRegistryNetworks = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("10.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("127.0.0.0/8"),
	netip.MustParsePrefix("169.254.0.0/16"),
	netip.MustParsePrefix("172.16.0.0/12"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("192.88.99.0/24"),
	netip.MustParsePrefix("192.168.0.0/16"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("224.0.0.0/4"),
	netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("::/96"),
	netip.MustParsePrefix("::1/128"),
	netip.MustParsePrefix("64:ff9b::/96"),
	netip.MustParsePrefix("64:ff9b:1::/48"),
	netip.MustParsePrefix("100::/64"),
	netip.MustParsePrefix("100:0:0:1::/64"),
	netip.MustParsePrefix("2001:db8::/32"),
	netip.MustParsePrefix("2002::/16"),
	netip.MustParsePrefix("fc00::/7"),
	netip.MustParsePrefix("fec0::/10"),
	netip.MustParsePrefix("fe80::/10"),
	netip.MustParsePrefix("ff00::/8"),
}

type registryDialer struct {
	trustedAddresses map[string]struct{}
	lookupNetIP      func(context.Context, string, string) ([]netip.Addr, error)
	dialContext      func(context.Context, string, string) (net.Conn, error)
}

// NewImageResolver resolves public images anonymously and reuses the existing
// build-registry credentials for historical Unkey-built images.
func NewImageResolver(config RegistryConfig) (ImageResolver, error) {
	resolver := &ociImageResolver{
		internalRepository: "",
		internalAuth:       authn.Anonymous,
		publicTransport:    newRegistryTransport(""),
		internalTransport:  nil,
		options:            nil,
	}
	if config.Repository == "" {
		return resolver, nil
	}

	repository, err := name.NewRepository(config.Repository, name.WeakValidation)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("invalid internal registry repository"))
	}
	resolver.internalRepository = repository.Name()
	resolver.internalTransport = newRegistryTransport(repository.RegistryStr())
	if config.Username != "" || config.Password != "" {
		resolver.internalAuth = authn.FromConfig(authn.AuthConfig{
			Username:      config.Username,
			Password:      config.Password,
			Auth:          "",
			IdentityToken: "",
			RegistryToken: "",
		})
	}
	return resolver, nil
}

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
	for _, address := range addresses {
		address = address.Unmap()
		if !isPublicRegistryAddress(address) {
			continue
		}
		conn, dialErr := d.dialContext(ctx, network, net.JoinHostPort(address.String(), port))
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

func isBlockedRegistryHostname(host string) bool {
	if net.ParseIP(host) != nil {
		return false
	}
	return !strings.Contains(host, ".") ||
		host == "localhost" ||
		strings.HasSuffix(host, ".localhost") ||
		strings.HasSuffix(host, ".local") ||
		strings.HasSuffix(host, ".internal") ||
		strings.HasSuffix(host, ".svc") ||
		strings.HasSuffix(host, ".svc.cluster.local")
}

func isPublicRegistryAddress(address netip.Addr) bool {
	if !address.IsValid() {
		return false
	}
	address = address.Unmap()
	for _, network := range nonPublicRegistryNetworks {
		if network.Contains(address) {
			return false
		}
	}
	return address.IsGlobalUnicast()
}

func (r *ociImageResolver) Resolve(ctx context.Context, imageReference string) (string, error) {
	reference, err := name.ParseReference(imageReference, name.WeakValidation)
	if err != nil {
		return "", fault.Wrap(err, fault.Internal("invalid OCI image reference"))
	}

	authenticator := authn.Anonymous
	transport := r.publicTransport
	if reference.Context().Name() == r.internalRepository {
		authenticator = r.internalAuth
		transport = r.internalTransport
	}
	options := make([]remote.Option, 0, len(r.options)+3)
	if transport != nil {
		options = append(options, remote.WithTransport(transport))
	}
	options = append(options, r.options...)
	options = append(options, remote.WithContext(ctx), remote.WithAuth(authenticator))
	descriptor, err := remote.Get(reference, options...)
	if err != nil {
		return "", fault.Wrap(err, fault.Internal("unable to resolve OCI image reference"))
	}

	return reference.Context().Digest(descriptor.Digest.String()).Name(), nil
}
