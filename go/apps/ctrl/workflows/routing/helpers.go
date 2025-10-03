package routing

import (
	"strings"

	hydrav1 "github.com/unkeyed/unkey/go/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/go/pkg/db"
)

// parseDomainSticky converts a protobuf DomainSticky enum to database representation.
//
// Sticky domains automatically follow the latest deployment for their scope (branch,
// environment, or live), while non-sticky domains remain immutable.
func parseDomainSticky(sticky hydrav1.DomainSticky) db.NullDomainsSticky {
	switch sticky {
	case hydrav1.DomainSticky_DOMAIN_STICKY_BRANCH:
		return db.NullDomainsSticky{Valid: true, DomainsSticky: db.DomainsStickyBranch}
	case hydrav1.DomainSticky_DOMAIN_STICKY_ENVIRONMENT:
		return db.NullDomainsSticky{Valid: true, DomainsSticky: db.DomainsStickyEnvironment}
	case hydrav1.DomainSticky_DOMAIN_STICKY_LIVE:
		return db.NullDomainsSticky{Valid: true, DomainsSticky: db.DomainsStickyLive}
	default:
		return db.NullDomainsSticky{Valid: false}
	}
}

// isLocalHostname checks if a hostname should be skipped from gateway config creation.
//
// Returns true for localhost and development domains (.local, .test TLDs) that should
// not get gateway configurations. Hostnames using the default domain (e.g., *.unkey.app)
// return false, as they represent production/staging environments and need gateway configs.
func isLocalHostname(hostname, defaultDomain string) bool {
	hostname = strings.ToLower(hostname)
	defaultDomain = strings.ToLower(defaultDomain)

	// Exact matches for localhost
	if hostname == "localhost" || hostname == "127.0.0.1" {
		return true
	}

	// If hostname uses the default domain, it's NOT local
	if strings.HasSuffix(hostname, "."+defaultDomain) || hostname == defaultDomain {
		return false
	}

	// Check for local-only TLD suffixes
	localSuffixes := []string{".local", ".test"}
	for _, suffix := range localSuffixes {
		if strings.HasSuffix(hostname, suffix) {
			return true
		}
	}

	return false
}
