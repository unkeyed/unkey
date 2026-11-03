package domainverify

import (
	"context"

	"github.com/unkeyed/unkey/pkg/dns"
)

// Resolver is the DNS surface verification reads. It is an interface so tests
// can decide what a hostname resolves to without touching the network.
type Resolver interface {
	LookupTXT(ctx context.Context, name string) ([]string, error)
	// LookupCNAME returns the target normalized to lowercase without a trailing dot.
	LookupCNAME(ctx context.Context, name string) (string, error)
	LookupHost(ctx context.Context, name string) ([]string, error)
}

// SystemResolver resolves through [dns], which queries Cloudflare's public resolver.
type SystemResolver struct{}

var _ Resolver = SystemResolver{}

// LookupTXT returns the TXT records published at name.
func (SystemResolver) LookupTXT(ctx context.Context, name string) ([]string, error) {
	return dns.LookupTXT(ctx, name)
}

// LookupCNAME returns the normalized CNAME target of name.
func (SystemResolver) LookupCNAME(ctx context.Context, name string) (string, error) {
	return dns.LookupCNAME(ctx, name)
}

// LookupHost returns the A and AAAA addresses name resolves to.
func (SystemResolver) LookupHost(ctx context.Context, name string) ([]string, error) {
	return dns.LookupHost(ctx, name)
}
